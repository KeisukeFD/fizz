//go:build acceptance

package acceptance

// F32 row (10): it needs an option that P6.6c adds, which exists only
// as a stub of zz_stub_acceptance.go. The task that replaces the stub
// moves the row into TestPrecedence (hooks_test.go).

import (
	"testing"

	"github.com/wI2L/fizz"
	"github.com/wI2L/fizz/internal/acceptance/at"
	"github.com/wI2L/fizz/openapi"
)

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
