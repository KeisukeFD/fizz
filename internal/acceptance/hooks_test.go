package acceptance

// Batch F (default build): generator hooks and the precedence table.
// See docs/plan/ACCEPTANCE.md, F23-F32. The precedence row that
// needs the option of P6.6c is in precedence_pending_test.go.
//
// Every package-level identifier is prefixed fH or starts with f, as
// sibling files share the package.

import (
	"errors"
	"net/http"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/loopfz/gadgeto/tonic"

	"github.com/wI2L/fizz"
	"github.com/wI2L/fizz/internal/acceptance/at"
	"github.com/wI2L/fizz/openapi"
)

// fQueryIn has one query parameter.
type fQueryIn struct {
	Q string `query:"q"`
}

const fAddrRef = "#/components/schemas/AcceptanceBFAddr"

// fPet is the component of the hook tests.
type fPet struct {
	Name   string `json:"name" validate:"required"`
	Secret string `json:"secret" validate:"required"`
	Addr   bFAddr `json:"addr"`
}

type fPetWrap struct {
	Pet fPet   `json:"pet"`
	ID  string `json:"id"`
}

type fEmpty struct{}

type fPathIn struct {
	ID string `path:"id"`
	Q  string `query:"q"`
}

// fHandler returns a tonic handler with input In and no output.
func fHandler[In any]() gin.HandlerFunc {
	return tonic.Handler(func(*gin.Context, *In) error { return nil }, 200)
}

// fGetPet returns a handler that outputs a fPet.
func fGetPet() gin.HandlerFunc {
	return tonic.Handler(func(*gin.Context, *fEmpty) (*fPet, error) { return nil, nil }, 200)
}

// fHooked returns a fizz with the hooks h.
func fHooked(h openapi.Hooks) *fizz.Fizz { return fizz.New(fizz.WithHooks(h)) }

// F23: the SchemaHook runs once per component, with the type and name.
func TestSchemaHookContext(t *testing.T) {
	at.Pending(t, "P6.4", func(tb testing.TB) {
		calls := map[string]int{}
		var typ reflect.Type
		f := fHooked(openapi.Hooks{Schema: func(c *openapi.SchemaContext, s *openapi.Schema) {
			calls[c.Name]++
			if c.Name == "AcceptanceFPet" {
				typ = c.Type
				s.Description = "hooked"
			}
		}})
		f.POST("/a", []fizz.OperationOption{fizz.ID("a")}, fHandler[fPet]())
		f.POST("/b", []fizz.OperationOption{fizz.ID("b")}, fHandler[fPetWrap]())
		f.GET("/c", []fizz.OperationOption{fizz.ID("c")}, fGetPet())

		doc := at.Spec(tb, f)
		if typ != reflect.TypeFor[fPet]() {
			tb.Errorf("ctx.Type = %v, want %v", typ, reflect.TypeFor[fPet]())
		}
		if calls["AcceptanceFPet"] != 1 || calls["AcceptanceBFAddr"] != 1 {
			tb.Errorf("calls = %v, want one per component", calls)
		}
		at.Frag(tb, doc, "/components/schemas/AcceptanceFPet/description", `"hooked"`)
	})
}

// F24: a skipped field is no property and no longer required.
func TestFieldHookSkip(t *testing.T) {
	at.Pending(t, "P6.4", func(tb testing.TB) {
		f := fHooked(openapi.Hooks{Field: func(c *openapi.FieldContext, _ *openapi.Schema) bool {
			return c.StructField.Name == "Secret"
		}})
		bFPost[fPet](f, "/b", "b")
		doc := at.Spec(tb, f)
		s := bFBodyPtr(tb, doc, "/b")
		at.Absent(tb, doc, s+"/properties/secret")
		at.Frag(tb, doc, s+"/required", `["name"]`)
	})
}

// F25: the field context; an annotation on a $ref leaves the component alone.
func TestFieldHookContext(t *testing.T) {
	at.Pending(t, "P6.4", func(tb testing.TB) {
		var parent reflect.Type
		var sf, name string
		f := fHooked(openapi.Hooks{Field: func(c *openapi.FieldContext, s *openapi.Schema) bool {
			if c.StructField.Name == "Addr" {
				parent, sf, name = c.Parent, c.StructField.Name, c.Name
				s.Description = "the addr"
			}
			return false
		}})
		bFPost[fPet](f, "/b", "b")
		doc := at.Spec(tb, f)
		if parent != reflect.TypeFor[fPet]() || sf != "Addr" || name != "addr" {
			tb.Errorf("context = %v, %q, %q", parent, sf, name)
		}
		at.Frag(tb, doc, bFBodyPtr(tb, doc, "/b")+"/properties/addr",
			`{"$ref":"`+fAddrRef+`","description":"the addr"}`)
		at.Absent(tb, doc, "/components/schemas/AcceptanceBFAddr/description")
	})
}

// F26: the parameter context carries the OpenAPI path.
func TestParameterHookContext(t *testing.T) {
	at.Pending(t, "P6.4", func(tb testing.TB) {
		got := map[string]openapi.ParameterContext{}
		f := fHooked(openapi.Hooks{Parameter: func(c *openapi.ParameterContext, _ *openapi.Parameter) bool {
			got[c.In] = *c
			return false
		}})
		f.GET("/pets/:id", []fizz.OperationOption{fizz.ID("getPet")}, fHandler[fPathIn]())
		at.Spec(tb, f)
		for in, field := range map[string]string{"path": "ID", "query": "Q"} {
			c := got[in]
			if c.OperationID != "getPet" || c.Method != "GET" || c.Path != "/pets/{id}" || c.StructField.Name != field {
				tb.Errorf("%s context = %+v", in, c)
			}
		}
	})
}

// F27: the OperationHook wins over the options; an error keeps the op.
func TestOperationHook(t *testing.T) {
	at.Pending(t, "P6.4", func(tb testing.TB) {
		f := fHooked(openapi.Hooks{Operation: func(c *openapi.OperationContext, op *openapi.Operation) error {
			op.Summary = "from hook"
			if c.OperationID == "y" {
				return errors.New("boom")
			}
			return nil
		}})
		f.GET("/x", []fizz.OperationOption{fizz.ID("x"), fizz.Summary("option")}, fHandler[fEmpty]())
		f.GET("/y", []fizz.OperationOption{fizz.ID("y"), fizz.Summary("option")}, fHandler[fEmpty]())
		doc := at.Spec(tb, f)
		at.Frag(tb, doc, "/paths/~1x/get/summary", `"from hook"`)
		at.Frag(tb, doc, "/paths/~1y/get/summary", `"option"`)
		bFWantErr(tb, f, "(y)", "boom")
	})
}

// fTagGroupHook adds an x-tagGroups entry and a tag to the document.
func fTagGroupHook(doc *openapi.OpenAPI) error {
	doc.XTagGroups = append(doc.XTagGroups, &openapi.XTagGroup{Name: "all", Tags: []string{"extra"}})
	doc.Tags = append(doc.Tags, &openapi.Tag{Name: "extra"})
	return nil
}

// F28: the hook edits a clone: served, never in Generator().API().
func TestDocumentHookOnClone(t *testing.T) {
	at.Pending(t, "P6.4", func(tb testing.TB) {
		f := fHooked(openapi.Hooks{Document: fTagGroupHook})
		f.Engine().GET("/spec.json", gin.WrapH(f.SpecHandler("json")))
		f.GET("/a", []fizz.OperationOption{fizz.ID("a")}, fHandler[fEmpty]())

		doc := at.Do(f, "GET", "/spec.json").Body.Bytes()
		at.Frag(tb, doc, "/x-tagGroups", `[{"name":"all","tags":["extra"]}]`)
		at.Frag(tb, doc, "/tags", `[{"name":"extra"}]`)
		api := f.Generator().API()
		if len(api.XTagGroups) != 0 || len(api.Tags) != 0 {
			tb.Errorf("API() holds the hook's changes: %v %v", api.XTagGroups, api.Tags)
		}
		// A second generation: the tag still appears once.
		f.GET("/b", []fizz.OperationOption{fizz.ID("b")}, fHandler[fEmpty]())
		doc = at.Do(f, "GET", "/spec.json").Body.Bytes()
		at.Frag(tb, doc, "/tags", `[{"name":"extra"}]`)
		at.Frag(tb, doc, "/x-tagGroups", `[{"name":"all","tags":["extra"]}]`)
	})
}

// F29: JSON and YAML share one run of the hook per generation.
func TestDocumentHookCachedPerGeneration(t *testing.T) {
	at.Pending(t, "P6.4", func(tb testing.TB) {
		var n atomic.Int64
		f := fHooked(openapi.Hooks{Document: func(*openapi.OpenAPI) error { n.Add(1); return nil }})
		f.Engine().GET("/spec.json", gin.WrapH(f.SpecHandler("json")))
		f.Engine().GET("/spec.yaml", gin.WrapH(f.SpecHandler("yaml")))
		f.GET("/a", []fizz.OperationOption{fizz.ID("a")}, fHandler[fEmpty]())

		for range 3 {
			at.Do(f, "GET", "/spec.json")
		}
		at.Do(f, "GET", "/spec.yaml")
		if n.Load() != 1 {
			tb.Errorf("hook ran %d times, want 1", n.Load())
		}
		f.GET("/b", []fizz.OperationOption{fizz.ID("b")}, fHandler[fEmpty]())
		at.Do(f, "GET", "/spec.json")
		at.Do(f, "GET", "/spec.yaml")
		if n.Load() != 2 {
			tb.Errorf("hook ran %d times, want 2", n.Load())
		}
	})
}

// F30: a failing hook is a 500 without a document, recorded once per
// generation.
func TestDocumentHookError(t *testing.T) {
	at.Pending(t, "P6.4", func(tb testing.TB) {
		f := fHooked(openapi.Hooks{Document: func(*openapi.OpenAPI) error { return errors.New("nope") }})
		f.Engine().GET("/spec.json", gin.WrapH(f.SpecHandler("json")))
		f.Engine().GET("/spec.yaml", gin.WrapH(f.SpecHandler("yaml")))
		f.GET("/a", []fizz.OperationOption{fizz.ID("a")}, fHandler[fEmpty]())

		count := func() int {
			n := 0
			for _, e := range bFErrs(f) {
				if strings.Contains(e, "DocumentHook") && strings.Contains(e, "nope") {
					n++
				}
			}
			return n
		}
		for _, p := range []string{"/spec.json", "/spec.json", "/spec.yaml"} {
			w := at.Do(f, "GET", p)
			if w.Code != http.StatusInternalServerError || strings.Contains(w.Body.String(), `"openapi"`) {
				tb.Errorf("GET %s = %d %q", p, w.Code, w.Body.String())
			}
		}
		if count() != 1 {
			tb.Errorf("recorded %d times, want 1: %q", count(), bFErrs(f))
		}
		if _, err := f.Document(); err == nil {
			tb.Error("Document() returned no error")
		}
		f.GET("/b", []fizz.OperationOption{fizz.ID("b")}, fHandler[fEmpty]())
		at.Do(f, "GET", "/spec.json")
		if count() != 2 {
			tb.Errorf("recorded %d times after a new generation, want 2", count())
		}
	})
}

// Types of the precedence table.

// fPrecProvider: the Provider replaces reflection.
type fPrecProvider struct {
	N string `json:"n"`
}

func (fPrecProvider) OpenAPISchema() *openapi.Schema {
	return &openapi.Schema{Type: openapi.Types("integer"), Title: "provider"}
}

// fPrecEnumer: the Enumer overrides the enum of the Provider.
type fPrecEnumer string

func (fPrecEnumer) OpenAPISchema() *openapi.Schema {
	return &openapi.Schema{Type: openapi.Types("string"), Enum: []any{"provider"}}
}
func (fPrecEnumer) Enum() []any { return []any{"enumer"} }

// fPrecCustomizer: the Customizer overrides the Enumer; the hook, the Customizer.
type fPrecCustomizer string

func (fPrecCustomizer) Enum() []any { return []any{"enumer"} }
func (fPrecCustomizer) CustomizeSchema(s *openapi.Schema) error {
	s.Enum = []any{"customizer"}
	s.Title = "customizer"
	return nil
}

type fPrecBody struct {
	P fPrecProvider   `json:"p"`
	E fPrecEnumer     `json:"e"`
	C fPrecCustomizer `json:"c"`
	// Desc has a description tag that the FieldHook overrides.
	Desc string `json:"desc" description:"tag"`
	// Color keeps the component enum but a tag narrows it here.
	Color bIColor `json:"color" enum:"red"`
}

// F32: one row per boundary of the precedence table (PLAN-v2 §1). Rows
// 7 and 10 are in precedence_pending_test.go.
func TestPrecedence(t *testing.T) {
	const comp = "/components/schemas/Acceptance"
	rows := []struct {
		name string
		run  func(tb testing.TB)
	}{
		{"1 reflection < Provider", func(tb testing.TB) {
			_, doc, _ := bFBody[fPrecBody](tb)
			at.Frag(tb, doc, comp+"FPrecProvider", `{"type":"integer","title":"provider"}`)
		}},
		{"2 Provider < Enumer", func(tb testing.TB) {
			_, doc, _ := bFBody[fPrecBody](tb)
			at.Frag(tb, doc, comp+"FPrecEnumer/enum", `["enumer"]`)
		}},
		{"3 Enumer < Customizer", func(tb testing.TB) {
			_, doc, _ := bFBody[fPrecBody](tb)
			at.Frag(tb, doc, comp+"FPrecCustomizer/enum", `["customizer"]`)
		}},
		{"4 Customizer < SchemaHook", func(tb testing.TB) {
			f := fHooked(openapi.Hooks{Schema: func(c *openapi.SchemaContext, s *openapi.Schema) {
				if c.Name == "AcceptanceFPrecCustomizer" {
					s.Title = "hook"
				}
			}})
			bFPost[fPrecBody](f, "/b", "b")
			doc := at.Spec(tb, f)
			at.Frag(tb, doc, comp+"FPrecCustomizer/title", `"hook"`)
			at.Frag(tb, doc, comp+"FPrecCustomizer/enum", `["customizer"]`)
		}},
		{"5 component < use-site tag", func(tb testing.TB) {
			_, doc, s := bFBody[fPrecBody](tb)
			at.Frag(tb, doc, s+"/properties/color", `{"$ref":"`+bIColorRef+`","enum":["red"]}`)
			at.Frag(tb, doc, "/components/schemas/AcceptanceBIColor/enum", `["red","green"]`)
		}},
		{"6 tag < FieldHook", func(tb testing.TB) {
			f := fHooked(openapi.Hooks{Field: func(c *openapi.FieldContext, s *openapi.Schema) bool {
				if c.Name == "desc" {
					s.Description = "hook"
				}
				return false
			}})
			bFPost[fPrecBody](f, "/b", "b")
			doc := at.Spec(tb, f)
			at.Frag(tb, doc, bFBodyPtr(tb, doc, "/b")+"/properties/desc/description", `"hook"`)
		}},
		{"7 ParameterHook < Parameter option", func(tb testing.TB) {
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
		}},
		{"8 options < OperationHook", func(tb testing.TB) {
			f := fHooked(openapi.Hooks{Operation: func(_ *openapi.OperationContext, op *openapi.Operation) error {
				op.Summary = "hook"
				return nil
			}})
			f.GET("/x", []fizz.OperationOption{fizz.ID("x"), fizz.Summary("option")}, fHandler[fEmpty]())
			at.Frag(tb, at.Spec(tb, f), "/paths/~1x/get/summary", `"hook"`)
		}},
		{"9 OperationHook < DocumentHook (served only)", func(tb testing.TB) {
			f := fHooked(openapi.Hooks{
				Operation: func(_ *openapi.OperationContext, op *openapi.Operation) error {
					op.Summary = "operation"
					return nil
				},
				Document: func(doc *openapi.OpenAPI) error {
					doc.Paths["/x"].GET.Summary = "document"
					return nil
				},
			})
			f.GET("/x", []fizz.OperationOption{fizz.ID("x")}, fHandler[fEmpty]())
			f.Engine().GET("/spec.json", gin.WrapH(f.SpecHandler("json")))
			at.Frag(tb, at.Do(f, "GET", "/spec.json").Body.Bytes(), "/paths/~1x/get/summary", `"document"`)
			if got := f.Generator().API().Paths["/x"].GET.Summary; got != "operation" {
				tb.Errorf("API() summary = %q, want the OperationHook's", got)
			}
		}},
	}
	for _, r := range rows {
		t.Run(r.name, func(t *testing.T) { at.Pending(t, "P6.4", r.run) })
	}
}
