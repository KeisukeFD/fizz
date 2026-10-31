//go:build acceptance

package acceptance

// F32 rows (7) and (10): they need options that P6.6b and P6.6c add,
// which exist only as stubs of zz_stub_acceptance.go. The task that
// replaces the stubs moves its row into TestPrecedence (hooks_test.go).

import (
	"testing"

	"github.com/wI2L/fizz"
	"github.com/wI2L/fizz/internal/acceptance/at"
	"github.com/wI2L/fizz/openapi"
)

// F32 row 7: the ParameterHook < the Parameter option.
func TestPrecedenceParameterOption(t *testing.T) {
	at.Pending(t, "P6.6b", func(tb testing.TB) {
		f := fHooked(openapi.Hooks{Parameter: func(_ *openapi.ParameterContext, p *openapi.Parameter) bool {
			p.Description = "hook"
			return false
		}})
		f.GET("/x", []fizz.OperationOption{
			fizz.ID("x"),
			fizz.Parameter(&openapi.Parameter{Name: "q", In: "query", Description: "option"}),
		}, fHandler[fQueryIn]())
		doc := at.Spec(tb, f)
		at.Frag(tb, doc, bFParamPtr(tb, doc, "/x", "get", "q")+"/description", `"option"`)
	})
}

// F32 row 10: CustomizeOperation < the OperationHook.
func TestPrecedenceCustomizeOperation(t *testing.T) {
	at.Pending(t, "P6.6c", func(tb testing.TB) {
		ran := false
		f := fHooked(openapi.Hooks{Operation: func(_ *openapi.OperationContext, op *openapi.Operation) error {
			op.Summary = "hook"
			return nil
		}})
		f.GET("/x", []fizz.OperationOption{
			fizz.ID("x"),
			fizz.CustomizeOperation(func(op *openapi.Operation) error { ran = true; op.Summary = "customize"; return nil }),
		}, fHandler[fEmpty]())
		at.Frag(tb, at.Spec(tb, f), "/paths/~1x/get/summary", `"hook"`)
		if !ran {
			tb.Error("the CustomizeOperation function did not run")
		}
	})
}

type fQueryIn struct {
	Q string `query:"q"`
}
