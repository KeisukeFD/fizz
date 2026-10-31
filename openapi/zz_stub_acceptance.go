//go:build acceptance

// Stub declarations for symbols whose tasks are still open (ADR 0001 §5, §6).
// Each declaration is deleted by the task named in its FixedBy line.

package openapi

import (
	"errors"
	"reflect"
)

// SchemaContext is a stub type.
// FixedBy: P6.4
type SchemaContext struct {
	Type reflect.Type
	Name string
}

// FieldContext is a stub type.
// FixedBy: P6.4
type FieldContext struct {
	Parent      reflect.Type
	StructField reflect.StructField
	Name        string
}

// ParameterContext is a stub type.
// FixedBy: P6.4
type ParameterContext struct {
	OperationID, Method, Path, In string
	StructField                   reflect.StructField
}

// OperationContext is a stub type.
// FixedBy: P6.4
type OperationContext struct {
	OperationID, Method, Path string
}

// SchemaHook is a stub type.
// FixedBy: P6.4
type SchemaHook func(*SchemaContext, *Schema)

// FieldHook is a stub type.
// FixedBy: P6.4
type FieldHook func(*FieldContext, *Schema) (skip bool)

// ParameterHook is a stub type.
// FixedBy: P6.4
type ParameterHook func(*ParameterContext, *Parameter) (skip bool)

// OperationHook is a stub type.
// FixedBy: P6.4
type OperationHook func(*OperationContext, *Operation) error

// DocumentHook is a stub type.
// FixedBy: P6.4
type DocumentHook func(*OpenAPI) error

// Hooks is a stub type.
// FixedBy: P6.4
type Hooks struct {
	Schema    SchemaHook
	Field     FieldHook
	Parameter ParameterHook
	Operation OperationHook
	Document  DocumentHook
}

// SetHooks is a stub.
// FixedBy: P6.4
func (g *Generator) SetHooks(Hooks) {
}

// SetExtension is a stub.
// FixedBy: P6.5
func (g *Generator) SetExtension(key string, value any) error {
	return errors.New("stub: P6.5")
}

// RegisterSchema is a stub.
// FixedBy: P6.5
func (g *Generator) RegisterSchema(name string, s *Schema) error {
	return errors.New("stub: P6.5")
}

// RegisterParameter is a stub.
// FixedBy: P6.5
func (g *Generator) RegisterParameter(name string, p *Parameter) error {
	return errors.New("stub: P6.5")
}

// RegisterResponse is a stub.
// FixedBy: P6.5
func (g *Generator) RegisterResponse(name string, r *Response) error {
	return errors.New("stub: P6.5")
}

// RegisterExample is a stub.
// FixedBy: P6.5
func (g *Generator) RegisterExample(name string, e *Example) error {
	return errors.New("stub: P6.5")
}

// RegisterRequestBody is a stub.
// FixedBy: P6.5
func (g *Generator) RegisterRequestBody(name string, rb *RequestBody) error {
	return errors.New("stub: P6.5")
}

// RegisterHeader is a stub.
// FixedBy: P6.5
func (g *Generator) RegisterHeader(name string, h *Header) error {
	return errors.New("stub: P6.5")
}

// RegisterType is a stub.
// FixedBy: P6.5
func (g *Generator) RegisterType(t reflect.Type) error {
	return errors.New("stub: P6.5")
}

// SetTag is a stub.
// FixedBy: P6.5
func (g *Generator) SetTag(t *Tag) {
}

// SetTagGroups is a stub.
// FixedBy: P6.5
func (g *Generator) SetTagGroups(groups []*XTagGroup) {
}

// SetSelf is a stub.
// FixedBy: P6.5
func (g *Generator) SetSelf(uri string) {
}

// SetJSONSchemaDialect is a stub.
// FixedBy: P6.5
func (g *Generator) SetJSONSchemaDialect(uri string) {
}

// SetExternalDocs is a stub.
// FixedBy: P6.5
func (g *Generator) SetExternalDocs(d *ExternalDocs) {
}
