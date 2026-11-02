package acceptance

// Batch D: the small operation options (Tags, ExternalDocs, Extension,
// CustomizeOperation, ResponseWith). See docs/plan/ACCEPTANCE.md,
// D09-D14 and D16.
//
// Every package-level identifier is prefixed bSO, or is a dX type, as
// sibling files share the package.

import (
	"errors"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/loopfz/gadgeto/tonic"

	"github.com/KeisukeFD/fizz"
	"github.com/KeisukeFD/fizz/internal/acceptance/at"
	"github.com/KeisukeFD/fizz/openapi"
)

type dErr struct {
	Message string `json:"message"`
}

const bSOGet = "/paths/~1p/get"

func bSOHandler() gin.HandlerFunc {
	return tonic.Handler(func(*gin.Context) error { return nil }, 200)
}

// bSOAdd registers GET /p with the options, in a fresh fizz.
func bSOAdd(opts ...fizz.OperationOption) *fizz.Fizz {
	f := fizz.New()
	f.GET("/p", append([]fizz.OperationOption{fizz.ID("bSOGet")}, opts...), bSOHandler())
	return f
}

// D09.
func TestTagsOption(t *testing.T) {
	f := fizz.New()
	g := f.Group("/", "pets", "Pets")
	g.GET("/p", []fizz.OperationOption{fizz.ID("bSOGet"), fizz.Tags("admin", "", "pets")}, bSOHandler())
	doc := at.Spec(t, f)
	at.Frag(t, doc, bSOGet+"/tags", `["pets","admin"]`)
	at.Frag(t, doc, "/tags", `[{"name":"pets","description":"Pets"}]`)

}

// D10.
func TestExternalDocsOption(t *testing.T) {
	f := bSOAdd(fizz.ExternalDocs("https://x", "More"))
	at.Frag(t, at.Spec(t, f), bSOGet+"/externalDocs", `{"url":"https://x","description":"More"}`)

	f = bSOAdd(fizz.ExternalDocs("", "More"))
	at.Absent(t, at.Spec(t, f), bSOGet+"/externalDocs")
	if f.Err() == nil {
		t.Error("an empty URL must be an error")
	}

}

// D11.
func TestExtensionOption(t *testing.T) {
	f := bSOAdd(fizz.Extension("x-rate", 1), fizz.Extension("x-rate", 10))
	at.Frag(t, at.Spec(t, f), bSOGet+"/x-rate", `10`)
	if err := f.Err(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	f = bSOAdd(fizz.Extension("rate", 1))
	doc := at.Spec(t, f)
	at.Absent(t, doc, bSOGet+"/rate")
	if !errors.Is(f.Err(), openapi.ErrInvalidExtensionKey) {
		t.Errorf("Err = %v, want ErrInvalidExtensionKey", f.Err())
	}

	f = bSOAdd(fizz.Extension("x-oai-a", 1))
	doc = at.Spec(t, f)
	at.Absent(t, doc, bSOGet+"/x-oai-a")
	if !errors.Is(f.Err(), openapi.ErrReservedExtension) {
		t.Errorf("Err = %v, want ErrReservedExtension", f.Err())
	}

}

// D12.
func TestCustomizeOperationDeferred(t *testing.T) {
	var seen string
	f := bSOAdd(
		fizz.CustomizeOperation(func(op *openapi.Operation) error {
			seen = op.Summary
			op.Summary = "c"
			return nil
		}),
		fizz.Summary("s"),
	)
	at.Frag(t, at.Spec(t, f), bSOGet+"/summary", `"c"`)
	if seen != "s" {
		t.Errorf("the function saw Summary %q, want \"s\"", seen)
	}

	f = bSOAdd(
		fizz.CustomizeOperation(func(op *openapi.Operation) error {
			op.Summary = "lost"
			return errors.New("boom")
		}),
		fizz.Summary("s"),
	)
	at.Frag(t, at.Spec(t, f), bSOGet+"/summary", `"s"`)
	var oe *openapi.OperationError
	if !errors.As(f.Err(), &oe) || oe.ID != "bSOGet" || !strings.Contains(oe.Error(), "boom") {
		t.Errorf("Err = %v, want an OperationError (bSOGet) wrapping boom", f.Err())
	}

}

// D13.
func TestResponseWith(t *testing.T) {
	f := bSOAdd(fizz.ResponseWith("404", "Not found",
		fizz.WithModel(dErr{}),
		fizz.WithExamples(map[string]any{"a": map[string]any{"message": "nf"}}),
		fizz.WithHeaders(&openapi.ResponseHeader{Name: "X-Req", Model: ""}),
		fizz.WithContent("application/xml", dErr{}),
		fizz.WithResponseExtension("x-r", true),
	))
	doc := at.Spec(t, f)
	const r = bSOGet + "/responses/404"
	at.Frag(t, doc, r+"/description", `"Not found"`)
	at.Frag(t, doc, r+"/content/application~1json/schema/$ref", `"#/components/schemas/AcceptanceDErr"`)
	at.Frag(t, doc, r+"/content/application~1json/examples/a/value", `{"message":"nf"}`)
	at.Frag(t, doc, r+"/content/application~1xml/schema/$ref", `"#/components/schemas/AcceptanceDErr"`)
	at.Frag(t, doc, r+"/headers/X-Req/schema", `{"type":"string"}`)
	at.Frag(t, doc, r+"/x-r", `true`)
	if err := f.Err(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// A media type given twice and example with examples are errors.
	f = bSOAdd(fizz.ResponseWith("400", "bad",
		fizz.WithModel(dErr{}),
		fizz.WithContent("application/json", nil),
		fizz.WithExample("e"),
		fizz.WithExamples(map[string]any{"a": 1}),
	))
	doc = at.Spec(t, f)
	at.Frag(t, doc, bSOGet+"/responses/400/content/application~1json/schema/$ref", `"#/components/schemas/AcceptanceDErr"`)
	at.Frag(t, doc, bSOGet+"/responses/400/content/application~1json/examples/a/value", `1`)
	at.Absent(t, doc, bSOGet+"/responses/400/content/application~1json/example")
	if n := len(f.Errors()); n != 2 {
		t.Errorf("%d errors, want 2: %v", n, f.Errors())
	}

	// A code that exists fails the operation.
	f = bSOAdd(fizz.ResponseWith("200", "again"))
	at.Absent(t, at.Spec(t, f), bSOGet)
	if f.Err() == nil {
		t.Error("a duplicate code must be an error")
	}

}

// D14: the positional Response is unchanged.
func TestResponsePositionalKept(t *testing.T) {
	f := bSOAdd(fizz.Response("404", "nf", nil, nil, nil))
	at.Frag(t, at.Spec(t, f), bSOGet+"/responses/404", `{"description":"nf"}`)
}

// D16.
func TestXCodeSampleTypedWins(t *testing.T) {
	f := bSOAdd(
		fizz.XCodeSample(&openapi.XCodeSample{Lang: "sh", Source: "curl"}),
		fizz.Extension("x-codeSamples", "other"),
	)
	at.Frag(t, at.Spec(t, f), bSOGet+"/x-codeSamples", `[{"lang":"sh","source":"curl"}]`)
	if f.Err() == nil {
		t.Error("the extension on a typed key must record an error")
	}

}
