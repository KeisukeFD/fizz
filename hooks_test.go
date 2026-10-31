package fizz

import (
	"errors"
	"slices"
	"testing"

	"github.com/wI2L/fizz/openapi"
)

func TestWithHooksComposesInOrder(t *testing.T) {
	var log []string
	mark := func(s string) { log = append(log, s) }
	var cfg config
	WithHooks(openapi.Hooks{
		Schema: func(*openapi.SchemaContext, *openapi.Schema) { mark("s1") },
		Field:  func(*openapi.FieldContext, *openapi.Schema) bool { mark("f1"); return true },
		Parameter: func(*openapi.ParameterContext, *openapi.Parameter) bool {
			mark("p1")
			return false
		},
		Operation: func(*openapi.OperationContext, *openapi.Operation) error { mark("o1"); return errors.New("o1") },
		Document:  func(*openapi.OpenAPI) error { mark("d1"); return nil },
	})(&cfg)
	WithHooks(openapi.Hooks{ // nil members are skipped
		Schema:    func(*openapi.SchemaContext, *openapi.Schema) { mark("s2") },
		Field:     func(*openapi.FieldContext, *openapi.Schema) bool { mark("f2"); return false },
		Parameter: func(*openapi.ParameterContext, *openapi.Parameter) bool { mark("p2"); return true },
		Operation: func(*openapi.OperationContext, *openapi.Operation) error { mark("o2"); return nil },
	})(&cfg)
	WithHooks(openapi.Hooks{Document: func(*openapi.OpenAPI) error { mark("d2"); return errors.New("d2") }})(&cfg)

	h := cfg.hooks
	h.Schema(nil, nil)
	if got := h.Field(nil, nil); !got {
		t.Error("Field: the first skip must win")
	}
	if got := h.Parameter(nil, nil); !got {
		t.Error("Parameter: the second skip must win")
	}
	if err := h.Operation(nil, nil); err == nil || err.Error() != "o1" {
		t.Errorf("Operation = %v, want the first error", err)
	}
	if err := h.Document(nil); err == nil || err.Error() != "d2" {
		t.Errorf("Document = %v, want d2", err)
	}
	want := []string{"s1", "s2", "f1", "p1", "p2", "o1", "d1", "d2"}
	if !slices.Equal(log, want) {
		t.Errorf("calls = %v, want %v", log, want)
	}
}
