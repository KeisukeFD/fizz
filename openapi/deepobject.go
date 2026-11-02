package openapi

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/KeisukeFD/fizz/internal/scalar"
)

// styleDeepObject is the value of the style tag that selects the
// deepObject serialization of a query parameter.
const styleDeepObject = "deepObject"

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
		if !scalar.IsValue(t.Elem()) {
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
			if !scalar.IsValue(sf.Type) {
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
