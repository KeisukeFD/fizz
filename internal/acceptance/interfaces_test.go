package acceptance

// Batch F (default build): the component stage interfaces SchemaProvider,
// SchemaCustomizer and Enumer. See docs/plan/ACCEPTANCE.md, F15-F22.
//
// Every package-level identifier is prefixed bI, as sibling files share
// the package.

import (
	"errors"
	"testing"

	"github.com/KeisukeFD/fizz/internal/acceptance/at"
	"github.com/KeisukeFD/fizz/openapi"
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
	want := `{"type":"string","pattern":"^\\d+\\.\\d{2}$"}`
	_, doc, _ := bFBody[bIMoneyBody](t)
	at.Frag(t, doc, "/components/schemas/AcceptanceBIMoney", want)
	_, doc, _ = bFBody[bIMoneyPBody](t)
	at.Frag(t, doc, "/components/schemas/AcceptanceBIMoneyP", want)

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
	f, doc, _ := bFBody[bINilBody](t)
	s := "/components/schemas/AcceptanceBINilProvider"
	at.Frag(t, doc, s+"/properties/n", `{"type":"string"}`)
	at.Frag(t, doc, s+"/type", `"object"`)
	bFWantErr(t, f, "OpenAPISchema returned nil")

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
	_, doc, _ := bFBody[bICustomBody](t)
	at.Frag(t, doc, "/components/schemas/AcceptanceBICustom/description", `"c"`)

	f, doc, _ := bFBody[bICustomFailBody](t)
	s := "/components/schemas/AcceptanceBICustomFail"
	at.Absent(t, doc, s+"/description")
	at.Absent(t, doc, s+"/properties/n/description")
	bFWantErr(t, f, "bICustomFail", "boom")

}

// F18.
func TestEnumerComponent(t *testing.T) {
	_, doc, s := bFBody[bIColorBody](t)
	at.Frag(t, doc, "/components/schemas/AcceptanceBIColor", `{"type":"string","enum":["red","green"]}`)
	at.Frag(t, doc, s+"/properties/c", `{"$ref":"`+bIColorRef+`"}`)

}

// F19: the field tag wins at the use site; the component keeps its enum.
func TestEnumerVsFieldTag(t *testing.T) {
	_, doc, s := bFBody[bIColorBody](t)
	at.Frag(t, doc, s+"/properties/t", `{"$ref":"`+bIColorRef+`","enum":["red"]}`)
	at.Frag(t, doc, "/components/schemas/AcceptanceBIColor/enum", `["red","green"]`)

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
	_, doc, s := bFBody[bIPtrBody](t)
	at.Frag(t, doc, "/components/schemas/AcceptanceBIPtrAll/title", `"ptr"`)
	at.Frag(t, doc, "/components/schemas/AcceptanceBIMoneyP/pattern", `"^\\d+\\.\\d{2}$"`)
	at.Frag(t, doc, "/components/schemas/AcceptanceBIPtrEnum", `{"type":"string","enum":["a","b"]}`)
	at.Frag(t, doc, s+"/properties/e", `{"$ref":"#/components/schemas/AcceptanceBIPtrEnum"}`)

}

// F22: a value of the wrong kind records an error and omits the enum.
type bIBadEnum string

func (bIBadEnum) Enum() []any { return []any{"a", 1} }

type bIBadEnumBody struct {
	B bIBadEnum `json:"b"`
}

func TestEnumerTypeMismatch(t *testing.T) {
	f, doc, _ := bFBody[bIBadEnumBody](t)
	at.Frag(t, doc, "/components/schemas/AcceptanceBIBadEnum", `{"type":"string"}`)
	bFWantErr(t, f, "bIBadEnum", "Enum()")

}
