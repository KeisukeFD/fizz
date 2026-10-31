package acceptance

// Batch F (default build): the component stage interfaces SchemaProvider,
// SchemaCustomizer and Enumer. See docs/plan/ACCEPTANCE.md, F15-F22.
//
// Every package-level identifier is prefixed bI, as sibling files share
// the package.

import (
	"errors"
	"testing"

	"github.com/wI2L/fizz/internal/acceptance/at"
	"github.com/wI2L/fizz/openapi"
)

const bIColorRef = "#/components/schemas/AcceptanceBIColor"

// bIColor is a named string with an Enumer.
type bIColor string

func (bIColor) Enum() []any { return []any{"red", "green"} }

type bIColorBody struct {
	C bIColor `json:"c"`
	T bIColor `json:"t" enum:"red"`
}

// F15: a Provider replaces reflection, on either receiver.
type bIMoney struct {
	Amount int `json:"amount"`
}

func (bIMoney) OpenAPISchema() *openapi.Schema {
	return &openapi.Schema{Type: openapi.Types("string"), Pattern: `^\d+\.\d{2}$`}
}

type bIMoneyP struct {
	Amount int `json:"amount"`
}

func (*bIMoneyP) OpenAPISchema() *openapi.Schema {
	return &openapi.Schema{Type: openapi.Types("string"), Pattern: `^\d+\.\d{2}$`}
}

type bIMoneyBody struct {
	M bIMoney `json:"m"`
}

type bIMoneyPBody struct {
	M bIMoneyP `json:"m"`
}

func TestSchemaProviderReplaces(t *testing.T) {
	at.Pending(t, "P6.3", func(tb testing.TB) {
		want := `{"type":"string","pattern":"^\\d+\\.\\d{2}$"}`
		_, doc, _ := bFBody[bIMoneyBody](tb)
		at.Frag(tb, doc, "/components/schemas/AcceptanceBIMoney", want)
		_, doc, _ = bFBody[bIMoneyPBody](tb)
		at.Frag(tb, doc, "/components/schemas/AcceptanceBIMoneyP", want)
	})
}

// F16: a nil schema falls back to reflection; an error is recorded.
type bINilProvider struct {
	N string `json:"n"`
}

func (bINilProvider) OpenAPISchema() *openapi.Schema { return nil }

type bINilBody struct {
	V bINilProvider `json:"v"`
}

func TestSchemaProviderNilFallsBack(t *testing.T) {
	at.Pending(t, "P6.3", func(tb testing.TB) {
		f, doc, _ := bFBody[bINilBody](tb)
		s := "/components/schemas/AcceptanceBINilProvider"
		at.Frag(tb, doc, s+"/properties/n", `{"type":"string"}`)
		at.Frag(tb, doc, s+"/type", `"object"`)
		bFWantErr(tb, f, "OpenAPISchema returned nil")
	})
}

// F17: a Customizer edits the component; an error discards its changes.
type bICustom struct {
	N string `json:"n"`
}

func (bICustom) CustomizeSchema(s *openapi.Schema) error {
	s.Description = "c"
	return nil
}

type bICustomFail struct {
	N string `json:"n"`
}

func (bICustomFail) CustomizeSchema(s *openapi.Schema) error {
	s.Description = "lost"
	s.Properties["n"].Description = "lost"
	return errors.New("boom")
}

type bICustomBody struct {
	V bICustom `json:"v"`
}

type bICustomFailBody struct {
	V bICustomFail `json:"v"`
}

func TestSchemaCustomizer(t *testing.T) {
	at.Pending(t, "P6.3", func(tb testing.TB) {
		_, doc, _ := bFBody[bICustomBody](tb)
		at.Frag(tb, doc, "/components/schemas/AcceptanceBICustom/description", `"c"`)

		f, doc, _ := bFBody[bICustomFailBody](tb)
		s := "/components/schemas/AcceptanceBICustomFail"
		at.Absent(tb, doc, s+"/description")
		at.Absent(tb, doc, s+"/properties/n/description")
		bFWantErr(tb, f, "bICustomFail", "boom")
	})
}

// F18.
func TestEnumerComponent(t *testing.T) {
	at.Pending(t, "P6.3", func(tb testing.TB) {
		_, doc, s := bFBody[bIColorBody](tb)
		at.Frag(tb, doc, "/components/schemas/AcceptanceBIColor", `{"type":"string","enum":["red","green"]}`)
		at.Frag(tb, doc, s+"/properties/c", `{"$ref":"`+bIColorRef+`"}`)
	})
}

// F19: the field tag wins at the use site; the component keeps its enum.
func TestEnumerVsFieldTag(t *testing.T) {
	at.Pending(t, "P6.3", func(tb testing.TB) {
		_, doc, s := bFBody[bIColorBody](tb)
		at.Frag(tb, doc, s+"/properties/t", `{"$ref":"`+bIColorRef+`","enum":["red"]}`)
		at.Frag(tb, doc, "/components/schemas/AcceptanceBIColor/enum", `["red","green"]`)
	})
}

// F20: the three interfaces declared on the pointer receiver.
type bIPtrAll struct {
	N string `json:"n"`
}

func (*bIPtrAll) CustomizeSchema(s *openapi.Schema) error { s.Title = "ptr"; return nil }

type bIPtrEnum string

func (*bIPtrEnum) Enum() []any { return []any{"a", "b"} }

type bIPtrBody struct {
	P bIPtrAll  `json:"p"`
	M bIMoneyP  `json:"m"`
	E bIPtrEnum `json:"e"`
}

func TestInterfacesPointerReceivers(t *testing.T) {
	at.Pending(t, "P6.3", func(tb testing.TB) {
		_, doc, s := bFBody[bIPtrBody](tb)
		at.Frag(tb, doc, "/components/schemas/AcceptanceBIPtrAll/title", `"ptr"`)
		at.Frag(tb, doc, "/components/schemas/AcceptanceBIMoneyP/pattern", `"^\\d+\\.\\d{2}$"`)
		at.Frag(tb, doc, "/components/schemas/AcceptanceBIPtrEnum", `{"type":"string","enum":["a","b"]}`)
		at.Frag(tb, doc, s+"/properties/e", `{"$ref":"#/components/schemas/AcceptanceBIPtrEnum"}`)
	})
}

// F22: a value of the wrong kind records an error and omits the enum.
type bIBadEnum string

func (bIBadEnum) Enum() []any { return []any{"a", 1} }

type bIBadEnumBody struct {
	B bIBadEnum `json:"b"`
}

func TestEnumerTypeMismatch(t *testing.T) {
	at.Pending(t, "P6.3", func(tb testing.TB) {
		f, doc, _ := bFBody[bIBadEnumBody](tb)
		at.Frag(tb, doc, "/components/schemas/AcceptanceBIBadEnum", `{"type":"string"}`)
		bFWantErr(tb, f, "bIBadEnum", "Enum()")
	})
}
