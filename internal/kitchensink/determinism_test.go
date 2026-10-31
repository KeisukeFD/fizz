package kitchensink

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/loopfz/gadgeto/tonic"

	"github.com/wI2L/fizz"
	"github.com/wI2L/fizz/internal/kitchensink/generic"
	"github.com/wI2L/fizz/openapi"
)

// newLegacyFizz registers the same routes as TestLegacyGolden on a fresh
// instance. It is a copy because the golden test registers inline and
// must stay untouched. Operation IDs are fixed with fizz.ID.
func newLegacyFizz(t *testing.T) *fizz.Fizz {
	t.Helper()
	f := fizz.New()

	g := f.Group("/pets", "Pets", "Pet operations")
	g.GET("", []fizz.OperationOption{
		fizz.ID("listPets"), fizz.Summary("List pets"),
		fizz.Header("X-Rate-Limit", "Requests left", 0),
		fizz.ResponseWithExamples("400", "Bad request", nil, nil, map[string]any{
			"empty": generic.Page[legacyPet]{},
		}),
	}, tonic.Handler(legacyList, 200))
	g.GET("/:id", []fizz.OperationOption{
		fizz.ID("getPet"), fizz.Description("Fetch one pet."),
		fizz.Response("404", "Not found", nil, nil, nil),
	}, tonic.Handler(legacyGet, 200))
	g.POST("", []fizz.OperationOption{
		fizz.ID("createPet"), fizz.StatusDescription("Created"),
		fizz.Security(&openapi.SecurityRequirement{"api_key": []string{}}),
	}, tonic.Handler(legacyCreate, 201))
	g.DELETE("/:id", []fizz.OperationOption{
		fizz.ID("deletePet"), fizz.Deprecated(true),
	}, tonic.Handler(legacyDelete, 204))
	f.GET("/items", []fizz.OperationOption{fizz.ID("getItem")}, tonic.Handler(legacyItem, 200))

	gen := f.Generator()
	gen.SetServers([]*openapi.Server{{URL: "https://api.example.com/v1", Description: "production"}})
	gen.SetSecurityRequirement([]*openapi.SecurityRequirement{{"api_key": []string{}}})
	gen.SetSecuritySchemes(map[string]*openapi.SecuritySchemeOrRef{
		"api_key": {Value: &openapi.SecurityScheme{Type: "apiKey", Name: "api_key", In: "header"}},
	})
	if errs := f.Errors(); len(errs) != 0 {
		t.Fatalf("registration errors: %v", errs)
	}
	return f
}

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
				gotJSON, err := BuildSpec(newLegacyFizz(t))
				if err != nil {
					t.Fatal(err)
				}
				gotYAML := serveSpec(t, newLegacyFizz(t), "yaml")
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
