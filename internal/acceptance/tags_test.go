package acceptance

// Batch F (default build): new struct tags, the legacy interfaces and
// the use-site rule. See docs/plan/ACCEPTANCE.md, Batch F.
//
// Every package-level identifier is prefixed bF/BF, as sibling files
// share the package.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/loopfz/gadgeto/tonic"

	"github.com/wI2L/fizz"
	"github.com/wI2L/fizz/internal/acceptance/at"
	"github.com/wI2L/fizz/openapi"
)

const bFAddrRef = "#/components/schemas/AcceptanceBFAddr"

// bFAddr is the shared named struct that becomes a component.
type bFAddr struct {
	City string `json:"city"`
}

// bFPost registers POST path with T as the JSON body.
func bFPost[T any](f *fizz.Fizz, path, id string) {
	f.POST(path, []fizz.OperationOption{fizz.ID(id)},
		tonic.Handler(func(*gin.Context, *T) error { return nil }, 200))
}

// bFGet registers GET path with T as the input (query/header/path).
func bFGet[T any](f *fizz.Fizz, path, id string) {
	f.GET(path, []fizz.OperationOption{fizz.ID(id)},
		tonic.Handler(func(*gin.Context, *T) error { return nil }, 200))
}

// bFBody builds a fizz with T as POST /b body and returns its document
// and the JSON pointer of the body schema component.
func bFBody[T any](tb testing.TB) (f *fizz.Fizz, doc []byte, schema string) {
	tb.Helper()
	f = fizz.New()
	bFPost[T](f, "/b", "b")
	doc = at.Spec(tb, f)
	return f, doc, bFBodyPtr(tb, doc, "/b")
}

// bFDecode unmarshals doc into a generic value.
func bFDecode(tb testing.TB, doc []byte) map[string]any {
	tb.Helper()
	var m map[string]any
	if err := json.Unmarshal(doc, &m); err != nil {
		tb.Fatalf("decode doc: %v", err)
	}
	return m
}

// bFDig walks keys through nested objects; it returns nil when absent.
func bFDig(v any, keys ...string) any {
	for _, k := range keys {
		m, ok := v.(map[string]any)
		if !ok {
			return nil
		}
		v = m[k]
	}
	return v
}

// bFBodyPtr resolves the JSON pointer of the component that the JSON
// request body of POST path references.
func bFBodyPtr(tb testing.TB, doc []byte, path string) string {
	tb.Helper()
	ref, _ := bFDig(bFDecode(tb, doc), "paths", path, "post", "requestBody",
		"content", "application/json", "schema", "$ref").(string)
	name, ok := strings.CutPrefix(ref, "#/components/schemas/")
	if !ok {
		tb.Fatalf("POST %s: body schema is not a component $ref: %q", path, ref)
	}
	return "/components/schemas/" + name
}

// bFParamPtr returns the JSON pointer of the named parameter of the op.
func bFParamPtr(tb testing.TB, doc []byte, path, method, name string) string {
	tb.Helper()
	params, _ := bFDig(bFDecode(tb, doc), "paths", path, method, "parameters").([]any)
	for i, p := range params {
		if bFDig(p, "name") == name {
			return fmt.Sprintf("/paths/%s/%s/parameters/%d",
				strings.ReplaceAll(path, "/", "~1"), method, i)
		}
	}
	tb.Fatalf("%s %s: parameter %q not found", method, path, name)
	return ""
}

// bFErrs returns the recorded errors of f as strings.
func bFErrs(f *fizz.Fizz) []string {
	var out []string
	for _, err := range f.Errors() {
		out = append(out, err.Error())
	}
	return out
}

// bFWantErr fails tb unless an error of f contains every substring.
func bFWantErr(tb testing.TB, f *fizz.Fizz, subs ...string) {
	tb.Helper()
	for _, e := range bFErrs(f) {
		ok := true
		for _, s := range subs {
			ok = ok && strings.Contains(e, s)
		}
		if ok {
			return
		}
	}
	tb.Errorf("no error contains %q; errors: %q", subs, bFErrs(f))
}

// bFComponent returns the canonical JSON of a component schema.
func bFComponent(tb testing.TB, doc []byte, name string) string {
	tb.Helper()
	c := bFDig(bFDecode(tb, doc), "components", "schemas", name)
	if c == nil {
		tb.Fatalf("component %q not found", name)
	}
	b, err := json.Marshal(c)
	if err != nil {
		tb.Fatal(err)
	}
	return string(b)
}

// F01.
type bFTitleBody struct {
	Name string `json:"name" title:"Pet name"`
}

func TestTagTitle(t *testing.T) {
	at.Pending(t, "P6.2", func(tb testing.TB) {
		_, doc, s := bFBody[bFTitleBody](tb)
		at.Frag(tb, doc, s+"/properties/name/title", `"Pet name"`)
	})
}

// F02.
type bFPatternBody struct {
	Slug string `json:"slug" pattern:"^[a-z]+$"`
}

func TestTagPattern(t *testing.T) {
	at.Pending(t, "P6.2", func(tb testing.TB) {
		_, doc, s := bFBody[bFPatternBody](tb)
		at.Frag(tb, doc, s+"/properties/slug/pattern", `"^[a-z]+$"`)
	})
}

// F03.
type bFMultipleOfBody struct {
	Price float64 `json:"price" multipleOf:"0.01"`
	Zero  float64 `json:"zero" multipleOf:"0"`
	Bad   float64 `json:"bad" multipleOf:"x"`
}

func TestTagMultipleOf(t *testing.T) {
	at.Pending(t, "P6.2", func(tb testing.TB) {
		f, doc, s := bFBody[bFMultipleOfBody](tb)
		at.Frag(tb, doc, s+"/properties/price/multipleOf", `0.01`)
		if !bytes.Contains(doc, []byte(`"multipleOf":0.01`)) {
			tb.Errorf(`raw document does not contain "multipleOf":0.01`)
		}
		at.Absent(tb, doc, s+"/properties/zero/multipleOf")
		at.Absent(tb, doc, s+"/properties/bad/multipleOf")
		n := 0
		for _, e := range bFErrs(f) {
			if strings.Contains(e, "multipleOf") {
				n++
			}
		}
		if n != 2 {
			tb.Errorf("want 2 errors mentioning multipleOf (0 and x), got %d: %q", n, bFErrs(f))
		}
	})
}

// F04.
type bFRWBody struct {
	RO    string `json:"ro" readOnly:"true"`
	WO    string `json:"wo" writeOnly:"true"`
	Maybe string `json:"maybe" readOnly:"maybe"`
}

type bFRWBothBody struct {
	Both string `json:"both" readOnly:"true" writeOnly:"true"`
}

func TestTagReadWriteOnly(t *testing.T) {
	at.Pending(t, "P6.2", func(tb testing.TB) {
		f, doc, s := bFBody[bFRWBody](tb)
		at.Frag(tb, doc, s+"/properties/ro/readOnly", `true`)
		at.Frag(tb, doc, s+"/properties/wo/writeOnly", `true`)
		at.Absent(tb, doc, s+"/properties/maybe/readOnly")
		bFWantErr(tb, f, "readOnly")

		f2, _, _ := bFBody[bFRWBothBody](tb)
		warned := false
		for _, err := range f2.Errors() {
			if w, ok := err.(interface{ IsWarning() bool }); ok && w.IsWarning() {
				warned = true
			}
		}
		if !warned {
			tb.Errorf("readOnly and writeOnly both true: want a warning, got %q", bFErrs(f2))
		}
	})
}

// F05.
type bFNullBody struct {
	P *string `json:"p" nullable:"false"`
	S string  `json:"s" nullable:"true"`
	R *bFAddr `json:"r" nullable:"false"`
}

func TestTagNullable(t *testing.T) {
	at.Pending(t, "P6.2", func(tb testing.TB) {
		_, doc, s := bFBody[bFNullBody](tb)
		at.Frag(tb, doc, s+"/properties/p", `{"type":"string"}`)
		at.Frag(tb, doc, s+"/properties/s", `{"type":["string","null"]}`)
		at.Frag(tb, doc, s+"/properties/r", `{"$ref":"`+bFAddrRef+`"}`)
	})
}

// F06.
type bFAPBody struct {
	Anon struct {
		X int `json:"x"`
	} `json:"anon" additionalProperties:"false"`
	Named bFAddr `json:"named" additionalProperties:"false"`
	N     int    `json:"n" additionalProperties:"false"`
}

func TestTagAdditionalPropertiesFalse(t *testing.T) {
	at.Pending(t, "P6.2", func(tb testing.TB) {
		f, doc, s := bFBody[bFAPBody](tb)
		at.Frag(tb, doc, s+"/properties/anon/additionalProperties", `false`)
		// A sibling additionalProperties:false next to $ref would reject
		// every property (ACCEPTANCE.md O11): unevaluatedProperties.
		at.Frag(tb, doc, s+"/properties/named",
			`{"$ref":"`+bFAddrRef+`","unevaluatedProperties":false}`)
		at.Absent(tb, doc, s+"/properties/named/additionalProperties")
		at.Absent(tb, doc, s+"/properties/n/additionalProperties")
		at.Absent(tb, doc, s+"/properties/n/unevaluatedProperties")
		bFWantErr(tb, f, "additionalProperties")
	})
}

// F07.
type bFInlineBody struct {
	Home  bFAddr `json:"home" inline:"true"`
	Other bFAddr `json:"other"`
}

func TestTagInline(t *testing.T) {
	at.Pending(t, "P6.2", func(tb testing.TB) {
		_, doc, s := bFBody[bFInlineBody](tb)
		at.Frag(tb, doc, s+"/properties/home/properties/city", `{"type":"string"}`)
		at.Absent(tb, doc, s+"/properties/home/$ref")
		at.Frag(tb, doc, s+"/properties/other", `{"$ref":"`+bFAddrRef+`"}`)
		at.Frag(tb, doc, "/components/schemas/AcceptanceBFAddr/properties/city", `{"type":"string"}`)
	})
}

// F08.
type bFStyleQuery struct {
	IDs []string `query:"ids" style:"pipeDelimited"`
	M   string   `query:"m" style:"matrix"`
}

func TestParamTagStyle(t *testing.T) {
	at.Pending(t, "P6.2", func(tb testing.TB) {
		f := fizz.New()
		bFGet[bFStyleQuery](f, "/q", "q")
		doc := at.Spec(tb, f)
		at.Frag(tb, doc, bFParamPtr(tb, doc, "/q", "get", "ids")+"/style", `"pipeDelimited"`)
		at.Absent(tb, doc, bFParamPtr(tb, doc, "/q", "get", "m")+"/style")
		bFWantErr(tb, f, "style")
	})
}

// F09.
type bFReservedQuery struct {
	U string `query:"u" allowReserved:"true"`
	H string `header:"X-H" allowReserved:"true"`
}

func TestParamTagAllowReserved(t *testing.T) {
	at.Pending(t, "P6.2", func(tb testing.TB) {
		f := fizz.New()
		bFGet[bFReservedQuery](f, "/q", "q")
		doc := at.Spec(tb, f)
		at.Frag(tb, doc, bFParamPtr(tb, doc, "/q", "get", "u")+"/allowReserved", `true`)
		at.Absent(tb, doc, bFParamPtr(tb, doc, "/q", "get", "X-H")+"/allowReserved")
		bFWantErr(tb, f, "allowReserved")
	})
}

// F10: a use-site annotation lands on the use site, never on the
// shared component.
type bFUseSiteBody struct {
	A bFAddr `json:"a" title:"T"`
	B bFAddr `json:"b"`
}

func TestTagsUseSiteOnly(t *testing.T) {
	at.Pending(t, "P6.2", func(tb testing.TB) {
		_, doc, s := bFBody[bFUseSiteBody](tb)
		at.Frag(tb, doc, s+"/properties/a", `{"$ref":"`+bFAddrRef+`","title":"T"}`)
		at.Frag(tb, doc, s+"/properties/b", `{"$ref":"`+bFAddrRef+`"}`)
		at.Absent(tb, doc, "/components/schemas/AcceptanceBFAddr/title")
	})
}

// F14. The renamed-binder-tag rule is B1: only fizz-owned tags are
// configurable.
func TestRenamedBinderTagRejected(t *testing.T) {
	at.Pending(t, "P6.1", func(tb testing.TB) {
		_, err := openapi.NewGenerator(&openapi.SpecGenConfig{
			ValidatorTag:      tonic.ValidationTag,
			PathLocationTag:   tonic.PathTag,
			QueryLocationTag:  "q",
			HeaderLocationTag: tonic.HeaderTag,
			EnumTag:           tonic.EnumTag,
			DefaultTag:        tonic.DefaultTag,
		})
		if err == nil {
			tb.Fatal("QueryLocationTag \"q\": want an error")
		}
		if !strings.Contains(err.Error(), "QueryLocationTag") {
			tb.Errorf("error does not name QueryLocationTag: %v", err)
		}
		if _, err := openapi.NewGenerator(&openapi.SpecGenConfig{
			ValidatorTag:      tonic.ValidationTag,
			PathLocationTag:   tonic.PathTag,
			QueryLocationTag:  tonic.QueryTag,
			HeaderLocationTag: tonic.HeaderTag,
			EnumTag:           tonic.EnumTag,
			DefaultTag:        tonic.DefaultTag,
		}); err != nil {
			tb.Errorf("all-tonic config: %v", err)
		}
	})
}

// F21: the pre-v0.30 interfaces keep working (regression, passes today).
type (
	bFTyper  struct{ V string }
	bFDType  struct{ V string }
	bFNullT  string
	bFExampl struct{ V string }
)

func (bFTyper) TypeName() string { return "BFCustomName" }

func (bFDType) Type() string   { return "string" }
func (bFDType) Format() string { return "bf-custom" }

func (bFNullT) Nullable() bool { return true }

func (*bFExampl) ParseExample(s string) (any, error) { return "parsed:" + s, nil }

type bFLegacyBody struct {
	T bFTyper  `json:"t"`
	D bFDType  `json:"d"`
	N bFNullT  `json:"n"`
	E bFExampl `json:"e" example:"zz"`
}

func TestLegacyInterfacesKept(t *testing.T) {
	_, doc, s := bFBody[bFLegacyBody](t)

	// Typer names the component.
	at.Frag(t, doc, s+"/properties/t", `{"$ref":"#/components/schemas/BFCustomName"}`)
	at.Frag(t, doc, "/components/schemas/BFCustomName/type", `"object"`)
	// DataType gives an inline type and format.
	at.Frag(t, doc, s+"/properties/d", `{"type":"string","format":"bf-custom"}`)
	// Nullable: OAS 3.0 "nullable":true, or a type union with null.
	n := bFDig(bFDecode(t, doc), "components", "schemas", strings.TrimPrefix(s, "/components/schemas/"), "properties", "n")
	typ, _ := bFDig(n, "type").([]any)
	if bFDig(n, "nullable") != true && !(len(typ) == 2 && typ[1] == "null") {
		t.Errorf("Nullable() not honored: %v", n)
	}
	// Exampler parses the example tag.
	if !bytes.Contains(doc, []byte(`"example":"parsed:zz"`)) {
		t.Errorf("Exampler result not in the document")
	}
}

// F33: the component never carries use-site annotations, in either
// registration order.
type (
	bFPlainUse struct {
		P bFAddr `json:"p"`
	}
	bFAnnotatedUse struct {
		A bFAddr  `json:"a" title:"T" description:"d" deprecated:"true"`
		B *bFAddr `json:"b" nullable:"true" description:"nullable d"`
	}
)

func TestComponentNeverMutatedByUseSite(t *testing.T) {
	at.Pending(t, "P3.4", func(tb testing.TB) {
		build := func(annotatedFirst bool, both bool) string {
			f := fizz.New()
			switch {
			case !both:
				bFPost[bFPlainUse](f, "/plain", "plain")
			case annotatedFirst:
				bFPost[bFAnnotatedUse](f, "/anno", "anno")
				bFPost[bFPlainUse](f, "/plain", "plain")
			default:
				bFPost[bFPlainUse](f, "/plain", "plain")
				bFPost[bFAnnotatedUse](f, "/anno", "anno")
			}
			return bFComponent(tb, at.Spec(tb, f), "AcceptanceBFAddr")
		}
		base := build(false, false)
		for name, got := range map[string]string{
			"plain then annotated": build(false, true),
			"annotated then plain": build(true, true),
		} {
			if got != base {
				tb.Errorf("%s: component changed\n got: %s\nwant: %s", name, got, base)
			}
		}
	})
}
