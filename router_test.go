package fizz

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/loopfz/gadgeto/tonic"
)

// TestHandleDoesNotMutateHandlers tests that Handle leaves the caller's
// variadic slice untouched, so that it can be reused for several routes.
func TestHandleDoesNotMutateHandlers(t *testing.T) {
	f := New()

	var seen []string
	mw := func(c *gin.Context) { c.Next() }
	hs := []gin.HandlerFunc{mw, tonic.Handler(func(c *gin.Context) error {
		op, err := OperationFromContext(c)
		if err != nil {
			t.Errorf("no operation in context: %v", err)
			return nil
		}
		seen = append(seen, op.ID)
		return nil
	}, http.StatusNoContent)}

	before := make([]uintptr, len(hs))
	for i, h := range hs {
		before[i] = reflect.ValueOf(h).Pointer()
	}
	f.GET("/one", []OperationOption{ID("one")}, hs...)
	f.GET("/two", []OperationOption{ID("two")}, hs...)

	for i, h := range hs {
		if got := reflect.ValueOf(h).Pointer(); got != before[i] {
			t.Errorf("hs[%d] was mutated by Handle", i)
		}
	}
	for _, id := range []string{"one", "two"} {
		seen = nil
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/"+id, nil)
		f.ServeHTTP(rec, req)
		if len(seen) != 1 || seen[0] != id {
			t.Errorf("GET /%s: operations seen = %v, want [%s]", id, seen, id)
		}
	}
}
