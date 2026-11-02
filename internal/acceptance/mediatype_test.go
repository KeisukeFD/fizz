package acceptance

// Batch D (default build): the default media type option. See
// docs/plan/ACCEPTANCE.md, D15.
//
// Every package-level identifier is prefixed bD, as sibling files
// share the package.

import (
	"encoding/json"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/loopfz/gadgeto/tonic"

	"github.com/KeisukeFD/fizz"
	"github.com/KeisukeFD/fizz/internal/acceptance/at"
)

type bDIn struct {
	Name string `json:"name"`
}

type bDOut struct {
	ID string `json:"id"`
}

// bDContentKeys returns the keys of the content object at the given
// top-level path and operation location.
func bDContentKeys(tb testing.TB, doc []byte, get func(op map[string]any) any) []string {
	tb.Helper()
	var root map[string]any
	if err := json.Unmarshal(doc, &root); err != nil {
		tb.Fatal(err)
	}
	op := root["paths"].(map[string]any)["/p"].(map[string]any)["post"].(map[string]any)
	content, ok := get(op).(map[string]any)
	if !ok {
		tb.Fatalf("no content object in %s", doc)
	}
	var keys []string
	for k := range content {
		keys = append(keys, k)
	}
	return keys
}

// D15.
func TestWithDefaultMediaType(t *testing.T) {
	const mt = "application/vnd.api+json"
	gin.SetMode(gin.TestMode)
	f := fizz.New(fizz.WithDefaultMediaType(mt))
	f.POST("/p", []fizz.OperationOption{fizz.ID("bDPost")},
		tonic.Handler(func(*gin.Context, *bDIn) (*bDOut, error) { return nil, nil }, 201))
	doc := at.Spec(t, f)
	req := bDContentKeys(t, doc, func(op map[string]any) any {
		return op["requestBody"].(map[string]any)["content"]
	})
	resp := bDContentKeys(t, doc, func(op map[string]any) any {
		return op["responses"].(map[string]any)["201"].(map[string]any)["content"]
	})
	if len(req) != 1 || req[0] != mt {
		t.Errorf("requestBody content keys = %v, want [%s]", req, mt)
	}
	if len(resp) != 1 || resp[0] != mt {
		t.Errorf("response content keys = %v, want [%s]", resp, mt)
	}

}
