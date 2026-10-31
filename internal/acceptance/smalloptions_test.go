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

	"github.com/wI2L/fizz"
	"github.com/wI2L/fizz/internal/acceptance/at"
	"github.com/wI2L/fizz/openapi"
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
	at.Pending(t, "P6.6c", func(tb testing.TB) {
		f := fizz.New()
		g := f.Group("/", "pets", "Pets")
		g.GET("/p", []fizz.OperationOption{fizz.ID("bSOGet"), fizz.Tags("admin", "", "pets")}, bSOHandler())
		doc := at.Spec(tb, f)
		at.Frag(tb, doc, bSOGet+"/tags", `["pets","admin"]`)
		at.Frag(tb, doc, "/tags", `[{"name":"pets","description":"Pets"}]`)
	})
}

// D10.
func TestExternalDocsOption(t *testing.T) {
	at.Pending(t, "P6.6c", func(tb testing.TB) {
		f := bSOAdd(fizz.ExternalDocs("https://x", "More"))
		at.Frag(tb, at.Spec(tb, f), bSOGet+"/externalDocs", `{"url":"https://x","description":"More"}`)

		f = bSOAdd(fizz.ExternalDocs("", "More"))
		at.Absent(tb, at.Spec(tb, f), bSOGet+"/externalDocs")
		if f.Err() == nil {
			tb.Error("an empty URL must be an error")
		}
	})
}

// D11.
func TestExtensionOption(t *testing.T) {
	at.Pending(t, "P6.6c", func(tb testing.TB) {
		f := bSOAdd(fizz.Extension("x-rate", 1), fizz.Extension("x-rate", 10))
		at.Frag(tb, at.Spec(tb, f), bSOGet+"/x-rate", `10`)
		if err := f.Err(); err != nil {
			tb.Errorf("unexpected error: %v", err)
		}

		f = bSOAdd(fizz.Extension("rate", 1))
		doc := at.Spec(tb, f)
		at.Absent(tb, doc, bSOGet+"/rate")
		if !errors.Is(f.Err(), openapi.ErrInvalidExtensionKey) {
			tb.Errorf("Err = %v, want ErrInvalidExtensionKey", f.Err())
		}

		f = bSOAdd(fizz.Extension("x-oai-a", 1))
		doc = at.Spec(tb, f)
		at.Absent(tb, doc, bSOGet+"/x-oai-a")
		if !errors.Is(f.Err(), openapi.ErrReservedExtension) {
			tb.Errorf("Err = %v, want ErrReservedExtension", f.Err())
		}
	})
}

// D12.
func TestCustomizeOperationDeferred(t *testing.T) {
	at.Pending(t, "P6.6c", func(tb testing.TB) {
		var seen string
		f := bSOAdd(
			fizz.CustomizeOperation(func(op *openapi.Operation) error {
				seen = op.Summary
				op.Summary = "c"
				return nil
			}),
			fizz.Summary("s"),
		)
		at.Frag(tb, at.Spec(tb, f), bSOGet+"/summary", `"c"`)
		if seen != "s" {
			tb.Errorf("the function saw Summary %q, want \"s\"", seen)
		}

		f = bSOAdd(
			fizz.CustomizeOperation(func(op *openapi.Operation) error {
				op.Summary = "lost"
				return errors.New("boom")
			}),
			fizz.Summary("s"),
		)
		at.Frag(tb, at.Spec(tb, f), bSOGet+"/summary", `"s"`)
		var oe *openapi.OperationError
		if !errors.As(f.Err(), &oe) || oe.ID != "bSOGet" || !strings.Contains(oe.Error(), "boom") {
			tb.Errorf("Err = %v, want an OperationError (bSOGet) wrapping boom", f.Err())
		}
	})
}

// D13.
func TestResponseWith(t *testing.T) {
	at.Pending(t, "P6.6c", func(tb testing.TB) {
		f := bSOAdd(fizz.ResponseWith("404", "Not found",
			fizz.WithModel(dErr{}),
			fizz.WithExamples(map[string]any{"a": map[string]any{"message": "nf"}}),
			fizz.WithHeaders(&openapi.ResponseHeader{Name: "X-Req", Model: ""}),
			fizz.WithContent("application/xml", dErr{}),
			fizz.WithResponseExtension("x-r", true),
		))
		doc := at.Spec(tb, f)
		const r = bSOGet + "/responses/404"
		at.Frag(tb, doc, r+"/description", `"Not found"`)
		at.Frag(tb, doc, r+"/content/application~1json/schema/$ref", `"#/components/schemas/AcceptanceDErr"`)
		at.Frag(tb, doc, r+"/content/application~1json/examples/a/value", `{"message":"nf"}`)
		at.Frag(tb, doc, r+"/content/application~1xml/schema/$ref", `"#/components/schemas/AcceptanceDErr"`)
		at.Frag(tb, doc, r+"/headers/X-Req/schema", `{"type":"string"}`)
		at.Frag(tb, doc, r+"/x-r", `true`)
		if err := f.Err(); err != nil {
			tb.Errorf("unexpected error: %v", err)
		}

		// A media type given twice and example with examples are errors.
		f = bSOAdd(fizz.ResponseWith("400", "bad",
			fizz.WithModel(dErr{}),
			fizz.WithContent("application/json", nil),
			fizz.WithExample("e"),
			fizz.WithExamples(map[string]any{"a": 1}),
		))
		doc = at.Spec(tb, f)
		at.Frag(tb, doc, bSOGet+"/responses/400/content/application~1json/schema/$ref", `"#/components/schemas/AcceptanceDErr"`)
		at.Frag(tb, doc, bSOGet+"/responses/400/content/application~1json/examples/a/value", `1`)
		at.Absent(tb, doc, bSOGet+"/responses/400/content/application~1json/example")
		if n := len(f.Errors()); n != 2 {
			tb.Errorf("%d errors, want 2: %v", n, f.Errors())
		}

		// A code that exists fails the operation.
		f = bSOAdd(fizz.ResponseWith("200", "again"))
		at.Absent(tb, at.Spec(tb, f), bSOGet)
		if f.Err() == nil {
			tb.Error("a duplicate code must be an error")
		}
	})
}

// D14: the positional Response is unchanged.
func TestResponsePositionalKept(t *testing.T) {
	f := bSOAdd(fizz.Response("404", "nf", nil, nil, nil))
	at.Frag(t, at.Spec(t, f), bSOGet+"/responses/404", `{"description":"nf"}`)
}

// D16.
func TestXCodeSampleTypedWins(t *testing.T) {
	at.Pending(t, "P6.6c", func(tb testing.TB) {
		f := bSOAdd(
			fizz.XCodeSample(&openapi.XCodeSample{Lang: "sh", Source: "curl"}),
			fizz.Extension("x-codeSamples", "other"),
		)
		at.Frag(tb, at.Spec(tb, f), bSOGet+"/x-codeSamples", `[{"lang":"sh","source":"curl"}]`)
		if f.Err() == nil {
			tb.Error("the extension on a typed key must record an error")
		}
	})
}
