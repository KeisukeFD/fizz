package openapi

import (
	"reflect"
	"testing"
)

// TestYAMLMarshalingRefs tests that spec types
// that contains embedded references  are properly
// marshaled to YAML.
func TestYAMLMarshalingRefs(t *testing.T) {
	// tests for references.
	tests := []any{
		&ParameterOrRef{Ref: &Reference{Ref: ""}},
		&ResponseOrRef{Ref: &Reference{Ref: ""}},
		&HeaderOrRef{Ref: &Reference{Ref: ""}},
		&MediaTypeOrRef{Ref: &Reference{Ref: ""}},
		&ExampleOrRef{Ref: &Reference{Ref: ""}},
	}
	for _, i := range tests {
		values := reflect.ValueOf(i).MethodByName("MarshalYAML").Call(nil)

		if len(values) != 2 {
			t.Errorf("expected MarshalYAML to return 2 args, got %d", len(values))
		}
		ret := values[0]

		if !ret.CanInterface() {
			t.Error("cannot get interface for returned type")
		}
		if _, ok := ret.Interface().(*Reference); !ok {
			t.Error("returned type is not a reference")
		}
	}
}

// TestYAMLMarshalingTypes tests that spec types
// that contains embedded types  are properly
// marshaled to YAML.
func TestYAMLMarshalingTypes(t *testing.T) {
	// tests for types.
	tests := map[any]any{
		&ParameterOrRef{Value: new(Parameter)}: &Parameter{},
		&ResponseOrRef{Value: new(Response)}:   &Response{},
		&HeaderOrRef{Value: new(Header)}:       &Header{},
		&MediaTypeOrRef{Value: new(MediaType)}: &MediaType{},
		&ExampleOrRef{Value: new(Example)}:     &Example{},
	}
	for i, e := range tests {
		values := reflect.ValueOf(i).MethodByName("MarshalYAML").Call(nil)

		if len(values) != 2 {
			t.Errorf("expected MarshalYAML to return 2 args, got %d", len(values))
		}
		ret := values[0]

		if ret.Type().Kind() != reflect.Interface {
			t.Error("cannot get underlying type of non interface")
		}
		uret := ret.Elem()

		if uret.Type() != reflect.TypeOf(e) {
			t.Errorf("expected type to be %s, got %s", reflect.TypeOf(e).String(), uret.Type().String())
		}
	}
}
