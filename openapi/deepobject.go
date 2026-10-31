package openapi

import (
	"encoding"
	"fmt"
	"reflect"
	"sort"
	"strings"
)

// styleDeepObject is the value of the style tag that selects the
// deepObject serialization of a query parameter.
const styleDeepObject = "deepObject"

var textUnmarshaler = reflect.TypeFor[encoding.TextUnmarshaler]()

// isDeepObject reports whether field is a query parameter that asks for
// the deepObject style.
func (g *Generator) isDeepObject(field reflect.StructField, location string) bool {
	if location != BinderTags.Query {
		return false
	}
	v, ok := tagLookup(field, g.tags.Style)
	return ok && v == styleDeepObject
}

// deepObjectSchema returns the inline object schema of a deepObject
// query parameter of type t: a map[string]S, a struct, or a pointer to
// either, where S is a scalar or a slice of scalars. These are the shapes
// the bind package decodes (bind/plan.go). The schema is never nullable,
// even for a pointer: an absent parameter is a nil value, not null.
// Struct members follow the rules of bind: unexported fields and
// fields tagged json:"-" are skipped, and the name is the json name.
func (g *Generator) deepObjectSchema(t reflect.Type) (*Schema, error) {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	switch t.Kind() {
	case reflect.Map:
		if t.Key().Kind() != reflect.String {
			return nil, fmt.Errorf("%s parameter: map key must be a string, got %s", styleDeepObject, t.Key())
		}
		if !deepObjectValue(t.Elem()) {
			return nil, fmt.Errorf("%s parameter: map value %s must be a scalar or a slice of scalars", styleDeepObject, t.Elem())
		}
		vs := g.newSchemaFromType(t.Elem())
		if vs == nil {
			return nil, fmt.Errorf("%s parameter: map value %s has no schema", styleDeepObject, t.Elem())
		}
		return &Schema{Type: Types("object"), AdditionalProperties: vs}, nil
	case reflect.Struct:
		props := make(map[string]*Schema)
		var required []string
		for i := range t.NumField() {
			sf := t.Field(i)
			if !sf.IsExported() {
				continue
			}
			name, _, _ := strings.Cut(sf.Tag.Get("json"), ",")
			if name == "-" {
				continue
			}
			if name == "" {
				name = sf.Name
			}
			if !deepObjectValue(sf.Type) {
				return nil, fmt.Errorf("%s parameter: member %s has type %s, want a scalar or a slice of scalars", styleDeepObject, sf.Name, sf.Type)
			}
			ms := g.fieldSchema(sf, false, name, t, valueForm{param: true})
			if ms == nil {
				return nil, fmt.Errorf("%s parameter: member %s of type %s has no schema", styleDeepObject, sf.Name, sf.Type)
			}
			props[name] = ms
			// validator checks the fields of a nested struct.
			if g.isStructFieldRequired(sf) {
				required = append(required, name)
			}
		}
		s := &Schema{Type: Types("object")}
		if len(props) > 0 {
			s.Properties = props
		}
		sort.Strings(required)
		s.Required = required
		return s, nil
	}
	return nil, fmt.Errorf("%s parameter: type %s must be a map[string]T, a struct or a pointer to one", styleDeepObject, t)
}

// deepObjectValue reports whether t is a scalar or a slice of scalars.
func deepObjectValue(t reflect.Type) bool {
	return deepObjectScalar(t) || (t.Kind() == reflect.Slice && deepObjectScalar(t.Elem()))
}

// deepObjectScalar mirrors bind's notion of a scalar: a type that
// decodes from one string.
func deepObjectScalar(t reflect.Type) bool {
	if reflect.PointerTo(t).Implements(textUnmarshaler) {
		return true
	}
	switch t.Kind() {
	case reflect.String, reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	}
	return false
}
