package acceptance

// Batch H (default build): manual components and document setters.
// See docs/plan/ACCEPTANCE.md, Batch H, rows H05-H11.

import (
	"errors"
	"reflect"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/loopfz/gadgeto/tonic"

	"github.com/KeisukeFD/fizz"
	"github.com/KeisukeFD/fizz/internal/acceptance/at"
	"github.com/KeisukeFD/fizz/openapi"
)

type hPet struct {
	Name string `json:"name"`
}

// H05: Generator().SetExtension lands on the root; the last write wins.
func TestGeneratorSetExtension(t *testing.T) {
	f := fizz.New()
	if err := f.Generator().SetExtension("x-api-id", "a"); err != nil {
		t.Fatal(err)
	}
	at.Frag(t, at.Spec(t, f), "/x-api-id", `"a"`)
	if err := f.Generator().SetExtension("x-api-id", "b"); err != nil {
		t.Fatal(err)
	}
	at.Frag(t, at.Spec(t, f), "/x-api-id", `"b"`)

}

// H06: RegisterSchema adds a component; a second registration of the
// name is a CollisionError and the first is kept.
func TestRegisterSchema(t *testing.T) {
	f := fizz.New()
	g := f.Generator()
	if err := g.RegisterSchema("Money", &openapi.Schema{Type: openapi.Types("number")}); err != nil {
		t.Fatal(err)
	}
	err := g.RegisterSchema("Money", &openapi.Schema{Type: openapi.Types("string")})
	var ce *openapi.CollisionError
	if !errors.As(err, &ce) {
		t.Fatalf("err = %v, want *CollisionError", err)
	}
	at.Frag(t, at.Spec(t, f), "/components/schemas/Money", `{"type":"number"}`)

}

// H07: RegisterType emits the component with no route; a route later
// refers to the same schema.
func TestRegisterType(t *testing.T) {
	f := fizz.New()
	if err := f.Generator().RegisterType(reflect.TypeFor[hPet]()); err != nil {
		t.Fatal(err)
	}
	at.Frag(t, at.Spec(t, f), "/components/schemas/AcceptanceHPet/properties/name", `{"type":"string"}`)

	f.GET("/pets", []fizz.OperationOption{fizz.ID("getPet")},
		tonic.Handler(func(*gin.Context) (*hPet, error) { return nil, nil }, 200))
	doc := at.Spec(t, f)
	at.Frag(t, doc, "/paths/~1pets/get/responses/200/content/application~1json/schema/$ref",
		`"#/components/schemas/AcceptanceHPet"`)
	if err := f.Err(); err != nil {
		t.Fatalf("Err = %v", err)
	}

}

// H08: the other component kinds, each in its own map; duplicates fail.
func TestRegisterOtherComponents(t *testing.T) {
	f := fizz.New()
	g := f.Generator()
	for _, err := range []error{
		g.RegisterParameter("Limit", &openapi.Parameter{Name: "limit", In: "query", Schema: &openapi.Schema{Type: openapi.Types("integer")}}),
		g.RegisterResponse("NotFound", &openapi.Response{Description: "nf"}),
		g.RegisterExample("Ex", &openapi.Example{Summary: "s"}),
		g.RegisterRequestBody("Body", &openapi.RequestBody{Description: "b", Content: map[string]*openapi.MediaType{"application/json": {}}}),
		g.RegisterHeader("Rate", &openapi.Header{Description: "h", Schema: &openapi.Schema{Type: openapi.Types("string")}}),
	} {
		if err != nil {
			t.Fatal(err)
		}
	}
	doc := at.Spec(t, f)
	at.Frag(t, doc, "/components/parameters/Limit/name", `"limit"`)
	at.Frag(t, doc, "/components/responses/NotFound/description", `"nf"`)
	at.Frag(t, doc, "/components/examples/Ex/summary", `"s"`)
	at.Frag(t, doc, "/components/requestBodies/Body/description", `"b"`)
	at.Frag(t, doc, "/components/headers/Rate/description", `"h"`)

	if g.RegisterParameter("Limit", &openapi.Parameter{Name: "x", In: "query"}) == nil ||
		g.RegisterResponse("NotFound", &openapi.Response{}) == nil ||
		g.RegisterExample("Ex", &openapi.Example{}) == nil ||
		g.RegisterRequestBody("Body", &openapi.RequestBody{}) == nil ||
		g.RegisterHeader("Rate", &openapi.Header{}) == nil {
		t.Error("a duplicate registration returned nil")
	}

}

// H09: SetTag fields and SetTagGroups.
func TestSetTagAndGroups(t *testing.T) {
	f := fizz.New()
	g := f.Generator()
	g.SetTag(&openapi.Tag{Name: "pets", Summary: "Pets", Kind: "nav", Parent: "root"})
	g.SetTagGroups([]*openapi.XTagGroup{{Name: "all", Tags: []string{"pets"}}})
	doc := at.Spec(t, f)
	at.Frag(t, doc, "/tags/0", `{"name":"pets","summary":"Pets","parent":"root","kind":"nav"}`)
	at.Frag(t, doc, "/x-tagGroups", `[{"name":"all","tags":["pets"]}]`)

}

// H10: SetSelf, SetJSONSchemaDialect and SetExternalDocs.
func TestSetSelfAndDialect(t *testing.T) {
	f := fizz.New()
	g := f.Generator()
	g.SetSelf("https://example.com/openapi.json")
	g.SetJSONSchemaDialect("https://spec.openapis.org/oas/3.2/dialect/2026-02-26")
	g.SetExternalDocs(&openapi.ExternalDocs{URL: "https://example.com/docs"})
	doc := at.Spec(t, f)
	at.Frag(t, doc, "/$self", `"https://example.com/openapi.json"`)
	at.Frag(t, doc, "/jsonSchemaDialect", `"https://spec.openapis.org/oas/3.2/dialect/2026-02-26"`)
	at.Frag(t, doc, "/externalDocs/url", `"https://example.com/docs"`)

}

// H11: an explicit SetTag after Group wins, without a warning.
func TestSetTagAfterGroupOverrides(t *testing.T) {
	f := fizz.New()
	f.Group("/g", "grp", "A")
	f.Generator().SetTag(&openapi.Tag{Name: "grp", Description: "B"})
	at.Frag(t, at.Spec(t, f), "/tags/0/description", `"B"`)
	if errs := f.Errors(); len(errs) != 0 {
		t.Fatalf("Errors = %v", errs)
	}

}
