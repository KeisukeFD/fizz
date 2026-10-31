package kitchensink

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/wI2L/fizz"
	"github.com/wI2L/fizz/openapi"
)

// serveSpec returns the body served by the OpenAPI handler for ct.
func serveSpec(t *testing.T, f *fizz.Fizz, ct string) []byte {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/openapi", f.OpenAPI(&openapi.Info{Title: "kitchensink", Version: "0"}, ct))
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/openapi", nil)
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("%s: status %d: %s", ct, rec.Code, rec.Body)
	}
	return rec.Body.Bytes()
}

func TestDeterminism(t *testing.T) {
	const runs = 20
	for _, procs := range []int{1, 8} {
		t.Run("GOMAXPROCS="+string(rune('0'+procs)), func(t *testing.T) {
			prev := runtime.GOMAXPROCS(procs)
			t.Cleanup(func() { runtime.GOMAXPROCS(prev) })

			var wantJSON, wantYAML []byte
			for i := range runs {
				gotJSON, err := BuildSpec(NewWholeFizz())
				if err != nil {
					t.Fatal(err)
				}
				gotYAML := serveSpec(t, NewWholeFizz(), "yaml")
				if i == 0 {
					wantJSON, wantYAML = gotJSON, bytes.Clone(gotYAML)
					continue
				}
				if !bytes.Equal(gotJSON, wantJSON) {
					t.Fatalf("run %d: JSON differs from run 0", i)
				}
				if !bytes.Equal(gotYAML, wantYAML) {
					t.Fatalf("run %d: YAML differs from run 0", i)
				}
			}
		})
	}
}
