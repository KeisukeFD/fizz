package openapi

import (
	"encoding/json"
	"fmt"
	"maps"
	"math"
	"reflect"
	"slices"
	"strconv"
)

// The three interfaces below run in the component stage of a named type
// (PLAN-v2 §1), in this order:
//
//	reflection or SchemaProvider -> Enumer -> SchemaCustomizer -> SchemaHook
//
// A type that implements any of them becomes a component, even a named
// scalar (a plain named scalar stays inline). They can be declared on the
// value or on the pointer receiver. The stage runs once per type: the
// use sites only reference the component, and a use-site annotation (a
// field "enum" tag, for example) never changes it.

// SchemaProvider is implemented by types that describe their own schema.
//
// The returned schema replaces the reflected one entirely; the type is
// not inspected. The schema is deep-copied, so the method may return a
// shared value. If the method returns nil, a *TypeError is recorded and
// the schema is reflected as if the interface was not implemented.
// Enumer and SchemaCustomizer still apply to the provided schema.
type SchemaProvider interface {
	OpenAPISchema() *Schema
}

// SchemaCustomizer is implemented by types that adjust their component
// schema once it has been built (by reflection or by a SchemaProvider,
// then by an Enumer).
//
// CustomizeSchema receives a private copy of the schema. If it returns an
// error, the error is recorded and the component keeps the schema it had
// before the call: none of the changes made by the failed call is kept.
// A panic is not recovered.
type SchemaCustomizer interface {
	CustomizeSchema(*Schema) error
}

// Enumer is implemented by types that list their valid values, typically
// a named string or integer type.
//
// The values are converted to the kind of the type, like the values of an
// "enum" tag, and set as the "enum" of the component. If a value does
// not match the type, an error is recorded for it and the enum is
// omitted. A field "enum" tag overrides the enum at the use site.
type Enumer interface {
	Enum() []any
}

var (
	tofSchemaProvider   = reflect.TypeFor[SchemaProvider]()
	tofSchemaCustomizer = reflect.TypeFor[SchemaCustomizer]()
	tofEnumer           = reflect.TypeFor[Enumer]()
)

// hasStageInterfaces reports whether the named type t implements, on the
// value or the pointer receiver, one of the component stage interfaces.
func hasStageInterfaces(t reflect.Type) bool {
	if t.Name() == "" {
		return false
	}
	return implementsAny(t, tofSchemaProvider) ||
		implementsAny(t, tofSchemaCustomizer) ||
		implementsAny(t, tofEnumer)
}

// stagedSchema returns the use-site schema of the named type t that
// implements a stage interface: always a $ref to a component, unless the
// component name belongs to another type, in which case the schema is
// inlined.
func (g *Generator) stagedSchema(t reflect.Type) *Schema {
	name, ok := g.componentName(t)
	if !ok {
		if g.inlining[t] {
			// Recursive type that cannot be a $ref.
			return &Schema{}
		}
		if g.inlining == nil {
			g.inlining = make(map[reflect.Type]bool)
		}
		g.inlining[t] = true
		defer delete(g.inlining, t)
		return g.componentStage(t)
	}
	// Already registered, or being built: reference it.
	if _, ok := g.schemaTypes[t]; ok {
		return &Schema{Ref: componentsSchemaPath + name}
	}
	g.schemaTypes[t] = struct{}{}

	// A component cuts any cycle with a $ref.
	outer := g.stack
	g.stack = nil
	defer func() { g.stack = outer }()

	s := g.componentStage(t)
	if s == nil {
		delete(g.schemaTypes, t)
		return nil
	}
	g.api.Components.Schemas[name] = s
	return &Schema{Ref: componentsSchemaPath + name}
}

// componentStage runs the component stage of t: reflection or
// SchemaProvider, then Enumer, then SchemaCustomizer.
func (g *Generator) componentStage(t reflect.Type) *Schema {
	var s *Schema
	if implementsAny(t, tofSchemaProvider) {
		// reflect.New yields a *T, whose method set holds the value
		// and the pointer receivers.
		if p := reflect.New(t).Interface().(SchemaProvider).OpenAPISchema(); p != nil {
			s = copySchemaTree(p)
		} else {
			g.error(&TypeError{
				Message: "OpenAPISchema returned nil: the schema is reflected instead",
				Type:    t,
			})
		}
	}
	if s == nil {
		s = g.reflectComponent(t)
		if s == nil {
			return nil
		}
	}
	if implementsAny(t, tofEnumer) {
		if enum, ok := g.enumValues(t, reflect.New(t).Interface().(Enumer).Enum()); ok {
			s.Enum = enum
		}
	}
	if implementsAny(t, tofSchemaCustomizer) {
		c := copySchemaTree(s)
		if err := reflect.New(t).Interface().(SchemaCustomizer).CustomizeSchema(c); err != nil {
			g.error(&TypeError{
				Message: fmt.Sprintf("CustomizeSchema failed, its changes are discarded: %s", err),
				Type:    t,
			})
		} else {
			s = c
		}
	}
	return s
}

// reflectComponent returns the reflected schema of the named type t, to
// be stored as a component.
func (g *Generator) reflectComponent(t reflect.Type) *Schema {
	if t.Kind() == reflect.Struct && g.datatype(t) == TypeComplex {
		return g.flattenStructSchema(t, t, &Schema{
			Type:       Types("object"),
			Properties: make(map[string]*Schema),
		})
	}
	return g.reflectSchema(t)
}

// enumValues converts the values of an Enumer to the kind of t. Any
// invalid value is recorded and makes the whole enum omitted.
func (g *Generator) enumValues(t reflect.Type, values []any) ([]any, bool) {
	out := make([]any, 0, len(values))
	ok := true
	for _, v := range values {
		c, err := enumValue(v, t)
		if err != nil {
			g.error(&TypeError{
				Message: fmt.Sprintf("invalid Enum() value %v (%T): %s: the enum is omitted", v, v, err),
				Type:    t,
			})
			ok = false
			continue
		}
		out = append(out, c)
	}
	return out, ok && len(out) > 0
}

// enumValue converts v to the value that encodes, as JSON, like a value
// of the kind of t does. A number must be representable by t.
func enumValue(v any, t reflect.Type) (any, error) {
	if v == nil {
		return nil, fmt.Errorf("nil value")
	}
	rv := reflect.ValueOf(v)
	switch t.Kind() {
	case reflect.String:
		if rv.Kind() == reflect.String {
			return rv.String(), nil
		}
	case reflect.Bool:
		if rv.Kind() == reflect.Bool {
			return rv.Bool(), nil
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		var n int64
		switch {
		case rv.CanInt():
			n = rv.Int()
		case rv.CanUint() && rv.Uint() <= math.MaxInt64:
			n = int64(rv.Uint())
		case rv.CanFloat() && rv.Float() == math.Trunc(rv.Float()) && math.Abs(rv.Float()) < 1<<63:
			n = int64(rv.Float())
		default:
			return nil, fmt.Errorf("not an integer for type %s", t)
		}
		if reflect.Zero(t).OverflowInt(n) {
			return nil, fmt.Errorf("%d overflows type %s", n, t)
		}
		return n, nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		var n uint64
		switch {
		case rv.CanUint():
			n = rv.Uint()
		case rv.CanInt() && rv.Int() >= 0:
			n = uint64(rv.Int())
		case rv.CanFloat() && rv.Float() >= 0 && rv.Float() == math.Trunc(rv.Float()) && rv.Float() < 1<<64:
			n = uint64(rv.Float())
		default:
			return nil, fmt.Errorf("not an unsigned integer for type %s", t)
		}
		if reflect.Zero(t).OverflowUint(n) {
			return nil, fmt.Errorf("%d overflows type %s", n, t)
		}
		return n, nil
	case reflect.Float32, reflect.Float64:
		var f float64
		switch {
		case rv.CanFloat():
			f = rv.Float()
		case rv.CanInt():
			f = float64(rv.Int())
		case rv.CanUint():
			f = float64(rv.Uint())
		default:
			return nil, fmt.Errorf("not a number for type %s", t)
		}
		if math.IsInf(f, 0) || math.IsNaN(f) {
			return nil, fmt.Errorf("%v is not a finite number", f)
		}
		if t.Kind() == reflect.Float32 {
			// Like convertValue: keep the float32 literal.
			return json.Number(strconv.FormatFloat(f, 'g', -1, 32)), nil
		}
		return f, nil
	default:
		return nil, fmt.Errorf("enum values are not supported for kind %s", t.Kind())
	}
	return nil, fmt.Errorf("does not match kind %s of type %s", t.Kind(), t)
}

// copySchemaTree returns a deep copy of s. Values held by interface
// members (enum, default, examples, extensions) are copied when they are
// maps or slices of any.
func copySchemaTree(s *Schema) *Schema {
	if s == nil {
		return nil
	}
	c := *s
	c.Bool = copyPtr(s.Bool)
	c.Type = slices.Clone(s.Type)
	c.AllOf = copySchemas(s.AllOf)
	c.OneOf = copySchemas(s.OneOf)
	c.AnyOf = copySchemas(s.AnyOf)
	c.PrefixItems = copySchemas(s.PrefixItems)
	c.Not = copySchemaTree(s.Not)
	c.Items = copySchemaTree(s.Items)
	c.AdditionalProperties = copySchemaTree(s.AdditionalProperties)
	c.PropertyNames = copySchemaTree(s.PropertyNames)
	if s.Properties != nil {
		c.Properties = make(map[string]*Schema, len(s.Properties))
		for k, v := range s.Properties {
			c.Properties[k] = copySchemaTree(v)
		}
	}
	c.Default = copyAny(s.Default)
	c.Const = copyAny(s.Const)
	c.Examples = copyAnys(s.Examples)
	c.Enum = copyAnys(s.Enum)
	c.Required = slices.Clone(s.Required)
	c.MaxLength = copyPtr(s.MaxLength)
	c.MinLength = copyPtr(s.MinLength)
	c.MaxItems = copyPtr(s.MaxItems)
	c.MinItems = copyPtr(s.MinItems)
	c.MaxProperties = copyPtr(s.MaxProperties)
	c.MinProperties = copyPtr(s.MinProperties)
	if s.Discriminator != nil {
		d := *s.Discriminator
		d.Mapping = maps.Clone(d.Mapping)
		d.Extensions = copyMap(d.Extensions)
		c.Discriminator = &d
	}
	if s.XML != nil {
		x := *s.XML
		x.Extensions = copyMap(x.Extensions)
		c.XML = &x
	}
	if s.ExternalDocs != nil {
		e := *s.ExternalDocs
		e.Extensions = copyMap(e.Extensions)
		c.ExternalDocs = &e
	}
	c.Extensions = copyMap(s.Extensions)
	return &c
}

func copySchemas(in []*Schema) []*Schema {
	if in == nil {
		return nil
	}
	out := make([]*Schema, len(in))
	for i, s := range in {
		out[i] = copySchemaTree(s)
	}
	return out
}

func copyPtr[T any](p *T) *T {
	if p == nil {
		return nil
	}
	v := *p
	return &v
}

func copyAnys(in []any) []any {
	if in == nil {
		return nil
	}
	out := make([]any, len(in))
	for i, v := range in {
		out[i] = copyAny(v)
	}
	return out
}

func copyMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = copyAny(v)
	}
	return out
}

func copyAny(v any) any {
	switch x := v.(type) {
	case map[string]any:
		return copyMap(x)
	case []any:
		return copyAnys(x)
	}
	return v
}
