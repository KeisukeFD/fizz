package openapi

import (
	"errors"
	"reflect"
	"slices"
	"testing"
)

func TestRequiredFromValidateTag(t *testing.T) {
	tests := []struct {
		tag      string
		required bool
		invalid  bool
	}{
		{"required", true, false},
		{"omitempty,required", false, false},
		{"required,omitempty", false, false},
		{"required_if=Foo bar", false, false},
		{"required_with=Foo", false, false},
		{"required_without=Foo", false, false},
		{"dive,required", false, false},
		{"required|eq=x", false, false},
		{"min=1,required", true, false},
		{"required,min=1", true, false},
		{"-", false, false},
		{"", false, false},
		{"required,,", false, true},
	}
	for _, tt := range tests {
		t.Run(tt.tag, func(t *testing.T) {
			st := reflect.StructOf([]reflect.StructField{{
				Name: "F", Type: reflect.TypeFor[[]string](),
				Tag: reflect.StructTag(`json:"f" validate:"` + tt.tag + `"`),
			}})
			g := gen(t)
			sor := g.newSchemaFromType(st)
			if sor == nil {
				t.Fatal("nil schema")
			}
			s := g.resolveSchema(sor)
			if got := slices.Contains(s.Required, "f"); got != tt.required {
				t.Errorf("required = %v, want %v (Required=%v)", got, tt.required, s.Required)
			}
			errs := g.Errors()
			if !tt.invalid {
				if len(errs) != 0 {
					t.Fatalf("errors = %v", errs)
				}
				return
			}
			var fe *FieldError
			if len(errs) != 1 || !errors.As(errs[0], &fe) {
				t.Fatalf("want exactly one FieldError, got %v", errs)
			}
			if fe.Name != "f" || fe.Field != "F" || fe.Parent != st {
				t.Errorf("incomplete FieldError: %+v", fe)
			}
		})
	}
}
