package openapi

import (
	"errors"
	"fmt"
	"reflect"
	"slices"
)

// SchemaContext describes the component schema handed to a SchemaHook.
type SchemaContext struct {
	// Type is the Go type the schema was generated from.
	Type reflect.Type
	// Name is the name of the component in components/schemas.
	Name string
}

// FieldContext describes the struct field handed to a FieldHook.
type FieldContext struct {
	// Parent is the struct type being described: the component, the
	// input type of an operation, or the outermost struct for a field
	// promoted from an embedded struct.
	Parent reflect.Type
	// StructField is the Go struct field.
	StructField reflect.StructField
	// Name is the name of the property, as encoded.
	Name string
}

// ParameterContext describes the parameter handed to a ParameterHook.
type ParameterContext struct {
	// OperationID, Method and Path identify the operation being added.
	// Path is the OpenAPI path ("/pets/{id}"), never the gin one.
	OperationID, Method, Path string
	// In is the location of the parameter: "path", "query", "header"
	// or "cookie".
	In string
	// StructField is the Go struct field the parameter comes from.
	StructField reflect.StructField
}

// OperationContext identifies the operation handed to an OperationHook.
type OperationContext struct {
	OperationID, Method, Path string
}

// SchemaHook adjusts the schema of a component once everything else has
// built it (reflection or SchemaProvider, Enumer, SchemaCustomizer). It
// is called once for every schema that the generator writes to
// components/schemas from a Go type, and changes the schema in place.
// It is not called for a schema given to RegisterSchema, nor for a type
// whose schema is inlined because its component name belongs to another
// type.
type SchemaHook func(*SchemaContext, *Schema)

// FieldHook adjusts the schema of a struct field that becomes a
// property, after its struct tags have been applied. It returns true to
// leave the field out: the property is then omitted, and removed from
// the required list of its parent.
//
// The schema belongs to the use site of the field: changing it never
// changes a component. A bare $ref receives annotations as siblings,
// whatever the AnnotationStyle. The hook is not called for the fields
// that are parameters (path, query, header, cookie).
type FieldHook func(*FieldContext, *Schema) (skip bool)

// ParameterHook adjusts a parameter once it has been built from the
// struct tags, before it is added to the operation. It returns true to
// leave the parameter out, except for a path parameter, which the URL
// template requires: the skip is ignored and a *Warning is recorded. The
// hook is not called for a path parameter that the generator synthesizes
// because the input does not declare it.
type ParameterHook func(*ParameterContext, *Parameter) (skip bool)

// OperationHook adjusts an operation once everything else has been
// applied. It receives a copy: if it returns nil the copy replaces the
// operation; if it returns an error, the changes are discarded, the
// error is recorded with the operation ID and the operation is kept as
// it was built. It cannot change the operation ID: a new one is reverted
// and a *Warning is recorded.
type OperationHook func(*OperationContext, *Operation) error

// DocumentHook adjusts a deep copy of the document, when it is
// served or returned by Generator.Document. It never runs for
// Generator.API or Generator.Snapshot. An error makes the document
// unavailable.
type DocumentHook func(*OpenAPI) error

// Hooks groups the hooks of a Generator. A nil member is not called.
type Hooks struct {
	Schema    SchemaHook
	Field     FieldHook
	Parameter ParameterHook
	Operation OperationHook
	Document  DocumentHook
}

// SetHooks replaces the hooks of the generator.
//
// The hooks affect what is generated after the call: components that
// have been built are not generated again. Set them before adding the
// operations.
//
// The Schema, Field, Parameter and Operation hooks run while the
// generator is locked, so they must not call methods of the Generator.
// A panic in a hook is not recovered.
func (g *Generator) SetHooks(h Hooks) {
	defer g.mutate()()
	g.hooks = h
}

// ErrDocumentHook is wrapped by the error of Generator.Document when the
// DocumentHook fails.
var ErrDocumentHook = errors.New("DocumentHook failed")

// Document returns a deep copy of the document, taken under the lock of
// the generator, with the DocumentHook applied, and the generation the
// copy was taken at. The hook runs after the lock is released, so a
// slow hook does not block the registration of operations. An error of
// the hook is returned wrapping ErrDocumentHook, and not recorded in
// Errors. A failure to copy the document is returned and recorded once
// per generation, like Snapshot does.
func (g *Generator) Document() (*OpenAPI, uint64, error) {
	g.mu.Lock()
	api, gen, err := g.cloneLocked()
	hook := g.hooks.Document
	g.mu.Unlock()
	if err != nil {
		return nil, gen, err
	}
	if hook != nil {
		if err := hook(api); err != nil {
			return nil, gen, fmt.Errorf("%w: %w", ErrDocumentHook, err)
		}
	}
	return api, gen, nil
}

// schemaHook runs the SchemaHook on the component s of type t.
func (g *Generator) schemaHook(t reflect.Type, name string, s *Schema) {
	if h := g.hooks.Schema; h != nil && s != nil {
		h(&SchemaContext{Type: t, Name: name}, s)
	}
}

// fieldHook runs the FieldHook on the schema s of the property f of
// parent. It reports whether the property is skipped.
func (g *Generator) fieldHook(f field, parent reflect.Type, s *Schema) bool {
	if h := g.hooks.Field; h != nil && s != nil {
		return h(&FieldContext{Parent: parent, StructField: f.sf, Name: f.name}, s)
	}
	return false
}

// parameterHook runs the ParameterHook on p. It reports whether p is
// left out of the operation.
func (g *Generator) parameterHook(sf reflect.StructField, p *Parameter) bool {
	h := g.hooks.Parameter
	if h == nil {
		return false
	}
	ctx := &ParameterContext{In: p.In, StructField: sf}
	if g.cur != nil {
		ctx.OperationID, ctx.Method, ctx.Path = g.cur.id, g.cur.method, g.cur.path
	}
	if !h(ctx, p) {
		return false
	}
	if p.In == "path" {
		g.warn(fmt.Sprintf("ParameterHook: path parameter %q cannot be skipped, the skip is ignored", p.Name))
		return false
	}
	return true
}

// operationHook runs the OperationHook on a copy of op and returns the
// operation to use: the copy, or op if the hook fails.
func (g *Generator) operationHook(op *Operation, method, path string) *Operation {
	h := g.hooks.Operation
	if h == nil {
		return op
	}
	return g.customizeOperation("OperationHook", op, func(c *Operation) error {
		return h(&OperationContext{OperationID: op.ID, Method: method, Path: path}, c)
	})
}

// customizeOperation runs fn on a copy of op and returns the operation
// to use: the copy, or op if fn fails (the error is recorded, prefixed by
// what). The operation ID cannot be changed: a new one is reverted and a
// *Warning is recorded.
func (g *Generator) customizeOperation(what string, op *Operation, fn func(*Operation) error) *Operation {
	c, err := op.Clone()
	if err != nil {
		g.error(fmt.Errorf("%s: %w", what, err))
		return op
	}
	if err := fn(c); err != nil {
		g.error(fmt.Errorf("%s: %w", what, err))
		return op
	}
	if c.ID != op.ID {
		g.warn(fmt.Sprintf("%s: the operation ID cannot be changed (%q to %q): reverted", what, op.ID, c.ID))
		c.ID = op.ID
	}
	return c
}

// dropRequired removes name from the required list of s.
func dropRequired(s *Schema, name string) {
	s.Required = slices.DeleteFunc(s.Required, func(n string) bool { return n == name })
	if len(s.Required) == 0 {
		s.Required = nil
	}
}
