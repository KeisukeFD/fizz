package acceptance

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/loopfz/gadgeto/tonic"

	"github.com/wI2L/fizz"
	"github.com/wI2L/fizz/internal/acceptance/at"
	boxfix "github.com/wI2L/fizz/internal/acceptance/fixtures/naming/fixtures"
	xmodels "github.com/wI2L/fizz/internal/acceptance/fixtures/naming/x"
	ymodels "github.com/wI2L/fizz/internal/acceptance/fixtures/naming/y"
	"github.com/wI2L/fizz/internal/specvalidate"
	"github.com/wI2L/fizz/openapi"
)

// Batch E: naming and operation IDs.
//
// The plan names the test types eFoo, eA, ... The package rule prefixes
// every identifier with bE, so the components read AcceptanceBEFoo.

var bENameRE = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

type (
	bEFoo struct {
		A string `json:"a"`
	}
	bEA struct {
		A string `json:"a"`
	}
	bEB struct {
		B string `json:"b"`
	}
	// bETyper reports a name that is not a legal component key.
	bETyper struct {
		A string `json:"a"`
	}
	bEÜber struct {
		A string `json:"a"`
	}
)

func (bETyper) TypeName() string { return "my type/with spaces" }

func bEGetFoo(*gin.Context) (*bEFoo, error) { return nil, nil }

// bEOut returns a handler that produces a T.
func bEOut[T any]() gin.HandlerFunc {
	return tonic.Handler(func(*gin.Context) (*T, error) { return nil, nil }, 200)
}

func bEGetA(*gin.Context) (*bEA, error) { return nil, nil }

// bESchemas returns /components/schemas of the spec as a map.
func bESchemas(tb testing.TB, doc []byte) map[string]json.RawMessage {
	tb.Helper()
	var d struct {
		Components struct {
			Schemas map[string]json.RawMessage `json:"schemas"`
		} `json:"components"`
	}
	if err := json.Unmarshal(doc, &d); err != nil {
		tb.Fatalf("decode spec: %v", err)
	}
	return d.Components.Schemas
}

// bERespRef returns the $ref name of the 200 JSON response of GET path.
func bERespRef(tb testing.TB, doc []byte, path string) string {
	tb.Helper()
	var d map[string]any
	if err := json.Unmarshal(doc, &d); err != nil {
		tb.Fatalf("decode spec: %v", err)
	}
	cur := any(d)
	for _, k := range []string{"paths", path, "get", "responses", "200", "content", "application/json", "schema", "$ref"} {
		m, ok := cur.(map[string]any)
		if !ok {
			tb.Fatalf("path %q: no %q", path, k)
		}
		cur = m[k]
	}
	s, _ := cur.(string)
	return strings.TrimPrefix(s, "#/components/schemas/")
}

func TestBEDefaultComponentNameFormat(t *testing.T) {
	at.Pending(t, "P4.5", func(tb testing.TB) {
		f := fizz.New()
		rows := []struct {
			path string
			h    gin.HandlerFunc
			want string
		}{
			{"/foo", tonic.Handler(bEGetFoo, 200), "AcceptanceBEFoo"},
			{"/box-string", bEOut[boxfix.Box[string]](), "FixturesBox_string"},
			{"/box-item", bEOut[boxfix.Box[xmodels.Item]](), "FixturesBox_models.Item"},
			{"/box-list", bEOut[boxfix.Box[[]string]](), "FixturesBox_List_string"},
			{"/box-map", bEOut[boxfix.Box[map[string]int]](), "FixturesBox_Map_string_int"},
		}
		for _, r := range rows {
			f.GET(r.path, []fizz.OperationOption{fizz.ID("id" + strings.ReplaceAll(r.path, "/", "_"))}, r.h)
		}
		doc := at.Spec(tb, f)
		for _, r := range rows {
			if got := bERespRef(tb, doc, r.path); got != r.want {
				tb.Errorf("%s: component %q, want %q", r.path, got, r.want)
			}
			if !bENameRE.MatchString(r.want) {
				tb.Errorf("%s: expected name %q is not a valid key", r.path, r.want)
			}
		}
		// Box[*Item] renders like Box[Item] but is another type with
		// another schema (nullable field): checked on its own spec, as
		// both in one spec collide and the second is inlined.
		fp := fizz.New()
		fp.GET("/box-ptr", []fizz.OperationOption{fizz.ID("id_box_ptr")}, bEOut[boxfix.Box[*xmodels.Item]]())
		if got := bERespRef(tb, at.Spec(tb, fp), "/box-ptr"); got != "FixturesBox_models.Item" {
			tb.Errorf("/box-ptr: component %q, want %q", got, "FixturesBox_models.Item")
		}
		for name := range bESchemas(tb, doc) {
			if !bENameRE.MatchString(name) {
				tb.Errorf("component name %q does not match %s", name, bENameRE)
			}
		}
	})
}

func TestBEComponentNamesSanitized(t *testing.T) {
	at.Pending(t, "P4.5", func(tb testing.TB) {
		f := fizz.New()
		f.GET("/a", []fizz.OperationOption{fizz.ID("a")}, bEOut[boxfix.Box[string]]())
		f.GET("/b", []fizz.OperationOption{fizz.ID("b")}, bEOut[boxfix.Box[map[string][]xmodels.Item]]())
		f.GET("/c", []fizz.OperationOption{fizz.ID("c")}, bEOut[bETyper]())
		f.GET("/d", []fizz.OperationOption{fizz.ID("d")}, bEOut[bEÜber]())
		f.GET("/e", []fizz.OperationOption{fizz.ID("e")}, bEOut[boxfix.Box[ymodels.Item]]())
		doc, err := bEBuild(f)
		if err != nil {
			tb.Fatal(err)
		}
		if err := specvalidate.LintComponentNames(doc); err != nil {
			tb.Error(err)
		}
	})
}

// bEBuild builds the spec without the meta-schema gate of at.Spec, which
// would flag unrelated problems.
func bEBuild(f *fizz.Fizz) ([]byte, error) {
	f.Generator().SetInfo(&openapi.Info{Title: "t", Version: "0"})
	return json.Marshal(f.Generator().API())
}

func TestBEOverrideTypeNameCollision(t *testing.T) {
	at.Pending(t, "P4.5", func(tb testing.TB) {
		f := fizz.New()
		f.GET("/a", []fizz.OperationOption{fizz.ID("getA")}, tonic.Handler(bEGetA, 200))
		before := at.Spec(tb, f)
		at.Frag(tb, before, "/components/schemas/AcceptanceBEA/properties/a", `{"type":"string"}`)

		err := f.Generator().OverrideTypeName(reflect.TypeFor[bEB](), "AcceptanceBEA")
		if err == nil {
			tb.Error("OverrideTypeName onto an existing component name returned nil")
		}
		after := at.Spec(tb, f)
		at.Frag(tb, after, "/components/schemas/AcceptanceBEA/properties/a", `{"type":"string"}`)
		at.Absent(tb, after, "/components/schemas/AcceptanceBEA/properties/b")
	})
}

// --- operation IDs ---

type bESvc struct{}

func (*bESvc) List(*gin.Context) error { return nil }

func bEListPets(*gin.Context) error { return nil }

func bEGen[T any](*gin.Context) error { return nil }

func bERegister(f *fizz.Fizz) {
	f.GET("/closure", nil, tonic.Handler(func(*gin.Context) error { return nil }, 200))
}

// bEOpID returns the operationId of GET path, or "" when absent.
func bEOpID(tb testing.TB, doc []byte, path string) string {
	tb.Helper()
	var d struct {
		Paths map[string]struct {
			Get *struct {
				ID string `json:"operationId"`
			} `json:"get"`
		} `json:"paths"`
	}
	if err := json.Unmarshal(doc, &d); err != nil {
		tb.Fatalf("decode spec: %v", err)
	}
	if p, ok := d.Paths[path]; ok && p.Get != nil {
		return p.Get.ID
	}
	return ""
}

// bESafe runs fn and converts a panic to an error.
func bESafe(fn func()) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	fn()
	return nil
}

func TestBEDefaultOperationIDs(t *testing.T) {
	at.Pending(t, "P4.7", func(tb testing.TB) {
		f := fizz.New()
		svc := &bESvc{}
		regs := []struct {
			path string
			reg  func()
			want string
		}{
			{"/pets", func() { f.GET("/pets", nil, tonic.Handler(bEListPets, 200)) }, "bEListPets"},
			{"/svc", func() { f.GET("/svc", nil, tonic.Handler(svc.List, 200)) }, "List"},
			{"/gen", func() { f.GET("/gen", nil, tonic.Handler(bEGen[int], 200)) }, "bEGen"},
			{"/closure", func() { bERegister(f) }, "bERegister"},
			{"/x", func() { f.GET("/x", nil, tonic.Handler(xmodels.Get, 200)) }, "Get"},
			{"/y", func() { f.GET("/y", nil, tonic.Handler(ymodels.Get, 200)) }, "models.Get"},
			{"/a/:id", func() { f.GET("/a/:id", nil, tonic.Handler(xmodels.Get, 200)) }, "get_a_id"},
		}
		for _, r := range regs {
			if err := bESafe(r.reg); err != nil {
				tb.Errorf("register %s: %v", r.path, err)
			}
		}
		doc, err := bEBuild(f)
		if err != nil {
			tb.Fatal(err)
		}
		for _, r := range regs {
			p := strings.ReplaceAll(r.path, ":id", "{id}")
			if got := bEOpID(tb, doc, p); got != r.want {
				tb.Errorf("GET %s: operationId %q, want %q", p, got, r.want)
			}
		}
	})
}

func TestBEExplicitIDCollision(t *testing.T) {
	at.Pending(t, "P4.7", func(tb testing.TB) {
		f := fizz.New()
		f.GET("/one", []fizz.OperationOption{fizz.ID("dup")}, tonic.Handler(bEListPets, 200))
		regErr := bESafe(func() {
			f.GET("/two", []fizz.OperationOption{fizz.ID("dup")}, tonic.Handler(bEListPets, 200))
		})
		// Registration must not panic (the error is recorded), and the
		// recorded error names the ID.
		if regErr != nil {
			tb.Errorf("registering the duplicate panicked: %v", regErr)
		}
		found := false
		for _, e := range f.Errors() {
			if strings.Contains(e.Error(), "dup") {
				found = true
			}
		}
		if !found {
			tb.Errorf("no error mentions \"dup\": %v", f.Errors())
		}
		doc, err := bEBuild(f)
		if err != nil {
			tb.Fatal(err)
		}
		at.Absent(tb, doc, "/paths/~1two/get")
		if w := at.Do(f, http.MethodGet, "/two"); w.Code != http.StatusOK {
			tb.Errorf("GET /two: %d, want 200", w.Code)
		}
	})
}

// E03.
func TestSchemaNamerOption(t *testing.T) {
	at.Pending(t, "P4.5+P6.1", func(tb testing.TB) {
		f := fizz.New(fizz.WithSchemaNamer(func(t reflect.Type) string { return "X_" + t.Name() }))
		f.GET("/foo", []fizz.OperationOption{fizz.ID("foo")}, tonic.Handler(bEGetFoo, 200))
		doc := at.Spec(tb, f)
		if got := bERespRef(tb, doc, "/foo"); got != "X_bEFoo" {
			tb.Errorf("component %q, want %q", got, "X_bEFoo")
		}
	})
}
