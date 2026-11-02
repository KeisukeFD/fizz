package kitchensink

import (
	"encoding/json"
	"math/big"
	"net"
	"net/netip"
	"net/url"
	"reflect"
	"time"
	"uuid"

	"github.com/gin-gonic/gin"
	gofrs "github.com/gofrs/uuid"
	google "github.com/google/uuid"
	"github.com/loopfz/gadgeto/tonic"

	"github.com/KeisukeFD/fizz"
)

// Test types for the type-mapping cases. All are prefixed T to stay clear
// of the sibling cases_*.go files.

type TInts struct {
	I   int    `json:"i"`
	I32 int32  `json:"i32"`
	U   uint   `json:"u"`
	U32 uint32 `json:"u32"`
	U64 uint64 `json:"u64"`
}

type TDur struct {
	D time.Duration `json:"d"`
}

type TStd struct {
	U url.URL `json:"u"`
}

type TBig struct {
	I big.Int   `json:"i"`
	F big.Float `json:"f"`
	R big.Rat   `json:"r"`
}

type TIP struct {
	IP   net.IP     `json:"ip"`
	Addr netip.Addr `json:"addr"`
}

type TText struct{ v string }

func (TText) MarshalText() ([]byte, error) { return nil, nil }

type TJSON struct{ v int }

func (TJSON) MarshalJSON() ([]byte, error) { return nil, nil }

type TBoth struct{ v string }

func (TBoth) MarshalText() ([]byte, error) { return nil, nil }
func (TBoth) MarshalJSON() ([]byte, error) { return nil, nil }

type TTextHolder struct {
	V TText `json:"v"`
}

type TJSONHolder struct {
	V TJSON `json:"v"`
}

type TBothHolder struct {
	V TBoth `json:"v"`
}

type TUnsupported struct {
	OK string     `json:"ok"`
	Ch chan int   `json:"ch"`
	Fn func()     `json:"fn"`
	Cx complex128 `json:"cx"`
	Er error      `json:"er"`
}

type TKeyText struct{ v string }

func (TKeyText) MarshalText() ([]byte, error) { return nil, nil }

// TKeyBoth is a string kind that also implements TextMarshaler: the
// string kind wins, as in encoding/json.
type TKeyBoth string

func (TKeyBoth) MarshalText() ([]byte, error) { return nil, nil }

// TKeyInt is an integer kind that also implements TextMarshaler: the
// TextMarshaler wins over the integer kind, as in encoding/json.
type TKeyInt int

func (TKeyInt) MarshalText() ([]byte, error) { return nil, nil }

type TMapInt struct {
	M map[int]string `json:"m"`
}

type TMapKeys struct {
	Text map[TKeyText]string `json:"text"`
	Both map[TKeyBoth]string `json:"both"`
	Int  map[TKeyInt]string  `json:"int"`
}

// TPtrDT implements DataType on the pointer receiver only.
type TPtrDT struct{ v int }

func (*TPtrDT) Type() string   { return "string" }
func (*TPtrDT) Format() string { return "tptr" }

// TPtrNull implements Nullable on the pointer receiver only.
type TPtrNull string

func (*TPtrNull) Nullable() bool { return true }

type TPtrRecvHolder struct {
	D TPtrDT   `json:"d"`
	N TPtrNull `json:"n"`
}

type TRaw struct {
	Raw   json.RawMessage `json:"raw"`
	Bytes []byte          `json:"bytes"`
	Arr   [3]string       `json:"arr"`
}

// TElemDT has a value-receiver DataType: elements must honour it.
type TElemDT struct{ v string }

func (TElemDT) Type() string   { return "string" }
func (TElemDT) Format() string { return "telem" }

// TElemOv is given a schema through Generator.OverrideDataType.
type TElemOv struct{ A string }

type TElemOvs struct {
	One  TElemOv            `json:"one"`
	List []TElemOv          `json:"list"`
	Map  map[string]TElemOv `json:"map"`
}

// TUUIDs holds uuid values as direct (non-slice) fields.
type TUUIDs struct {
	Std    uuid.UUID   `json:"std"`
	Google google.UUID `json:"google"`
	Gofrs  gofrs.UUID  `json:"gofrs"`
}

type TElems struct {
	Times  []time.Time        `json:"times"`
	Std    []uuid.UUID        `json:"std"`
	Google []google.UUID      `json:"google"`
	Gofrs  []gofrs.UUID       `json:"gofrs"`
	Blobs  [][]byte           `json:"blobs"`
	DT     []TElemDT          `json:"dt"`
	Map    map[string]TElemDT `json:"map"`
}

type TSelf struct {
	Name string  `json:"name"`
	Next *TSelf  `json:"next"`
	Kids []TSelf `json:"kids"`
}

type TL []TL

type TLHolder struct {
	L TL `json:"l"`
}

type TM map[string]TM

type TMHolder struct {
	M TM `json:"m"`
}

func tout[T any](*gin.Context, *struct{}) (*T, error) { return new(T), nil }

func tget[T any](f *fizz.Fizz) {
	f.GET("/t", []fizz.OperationOption{fizz.ID("t")}, tonic.Handler(tout[T], 200))
}

const tp = "/components/schemas/Kitchensink"

// tprop returns the pointer of property prop of the schema of the test
// type name (component names carry the package prefix today).
func tprop(name, prop string) string { return tp + name + "/properties/" + prop }

func tcase(name string, reg func(*fizz.Fizz), fixedBy string, expect ...Fragment) Case {
	return Case{Name: "types/" + name, Register: reg, FixedBy: fixedBy, Expect: expect}
}

func init() {
	ints := tget[TInts]
	register(
		tcase("int-kinds", ints, "P3.3",
			Fragment{Pointer: tprop("TInts", "i"), JSON: `{"type":"integer","format":"int64"}`},
			Fragment{Pointer: tprop("TInts", "i32"), JSON: `{"type":"integer","format":"int32"}`},
			Fragment{Pointer: tprop("TInts", "u"), JSON: `{"type":"integer","format":"uint64","minimum":0}`},
			Fragment{Pointer: tprop("TInts", "u32"), JSON: `{"type":"integer","format":"uint32","minimum":0}`},
			Fragment{Pointer: tprop("TInts", "u64"), JSON: `{"type":"integer","format":"uint64","minimum":0}`},
		),
		tcase("duration", tget[TDur], "P3.3",
			Fragment{Pointer: tprop("TDur", "d"), JSON: `{"type":"integer","format":"int64"}`},
		),
		tcase("url-url", tget[TStd], "P3.3",
			Fragment{Pointer: tprop("TStd", "u"), JSON: `{"$ref":"#/components/schemas/UrlURL"}`},
			Fragment{Pointer: "/components/schemas/UrlURL/type", JSON: `"object"`},
		),
		tcase("big-int", tget[TBig], "P3.3",
			Fragment{Pointer: tprop("TBig", "i"), JSON: `{"type":"integer"}`},
		),
		tcase("big-float-rat", tget[TBig], "P3.3",
			Fragment{Pointer: tprop("TBig", "f"), JSON: `{"type":"string"}`},
			Fragment{Pointer: tprop("TBig", "r"), JSON: `{"type":"string"}`},
		),
		tcase("net-ip", tget[TIP], "",
			Fragment{Pointer: tprop("TIP", "ip/type"), JSON: `"string"`},
		),
		tcase("netip-addr", tget[TIP], "P3.3",
			Fragment{Pointer: tprop("TIP", "addr/type"), JSON: `"string"`},
			Fragment{Pointer: tprop("TIP", "addr/$ref"), Absent: true},
		),
		tcase("marshaler/text-only", tget[TTextHolder], "P3.3",
			Fragment{Pointer: tprop("TTextHolder", "v"), JSON: `{"type":"string"}`},
		),
		// D14: kind-derived schema (a struct here) plus a warning.
		withErrors(tcase("marshaler/json-only", tget[TJSONHolder], "P3.3",
			Fragment{Pointer: tprop("TJSONHolder", "v/$ref"), JSON: `"#/components/schemas/KitchensinkTJSON"`},
			Fragment{Pointer: tp + "TJSON/type", JSON: `"object"`},
		), "SchemaProvider"),
		tcase("marshaler/both", tget[TBothHolder], "P3.3",
			Fragment{Pointer: tprop("TBothHolder", "v"), JSON: `{"type":"string"}`},
		),
		// Each unsupported property is omitted and an error is recorded;
		// a null schema must never be emitted.
		withErrors(tcase("unsupported-kinds", tget[TUnsupported], "",
			Fragment{Pointer: tprop("TUnsupported", "ok"), JSON: `{"type":"string"}`},
			Fragment{Pointer: tprop("TUnsupported", "ch"), Absent: true},
			Fragment{Pointer: tprop("TUnsupported", "fn"), Absent: true},
			Fragment{Pointer: tprop("TUnsupported", "cx"), Absent: true},
			Fragment{Pointer: tprop("TUnsupported", "er"), Absent: true},
		), "chan int", "func()", "complex128", "error"),
		tcase("map-int-key", tget[TMapInt], "P3.3",
			Fragment{Pointer: tprop("TMapInt", "m"), JSON: `{"type":"object","additionalProperties":{"type":"string"},"propertyNames":{"pattern":"^-?[0-9]+$"}}`},
		),
		// Map-key order follows encoding/json: string kind, then
		// TextMarshaler, then integer kinds.
		tcase("map-key/textmarshaler", tget[TMapKeys], "P3.3",
			Fragment{Pointer: tprop("TMapKeys", "text"), JSON: `{"type":"object","additionalProperties":{"type":"string"}}`},
		),
		tcase("map-key/string-kind-beats-textmarshaler", tget[TMapKeys], "",
			Fragment{Pointer: tprop("TMapKeys", "both"), JSON: `{"type":"object","additionalProperties":{"type":"string"}}`},
			Fragment{Pointer: tprop("TMapKeys", "both/propertyNames"), Absent: true},
		),
		tcase("map-key/textmarshaler-beats-int-kind", tget[TMapKeys], "P3.3",
			Fragment{Pointer: tprop("TMapKeys", "int"), JSON: `{"type":"object","additionalProperties":{"type":"string"}}`},
		),
		tcase("ptr-receiver/datatype", tget[TPtrRecvHolder], "P3.3",
			Fragment{Pointer: tprop("TPtrRecvHolder", "d"), JSON: `{"type":"string","format":"tptr"}`},
		),
		tcase("ptr-receiver/nullable", tget[TPtrRecvHolder], "P3.5",
			Fragment{Pointer: tprop("TPtrRecvHolder", "n"), JSON: `{"type":["string","null"]}`},
		),
		tcase("raw-message", tget[TRaw], "P3.3",
			Fragment{Pointer: tprop("TRaw", "raw"), JSON: `{}`},
		),
		tcase("byte-slice", tget[TRaw], "P2.6",
			Fragment{Pointer: tprop("TRaw", "bytes"), JSON: `{"type":"string","contentEncoding":"base64"}`},
		),
		tcase("array", tget[TRaw], "",
			Fragment{Pointer: tprop("TRaw", "arr"), JSON: `{"type":"array","items":{"type":"string"},"minItems":3,"maxItems":3}`},
		),
		tcase("elem/time", tget[TElems], "P3.2",
			Fragment{Pointer: tprop("TElems", "times"), JSON: `{"type":"array","items":{"type":"string","format":"date-time"}}`},
		),
		tcase("uuid/direct", tget[TUUIDs], "P1.2",
			Fragment{Pointer: tprop("TUUIDs", "std"), JSON: `{"type":"string","format":"uuid"}`},
			Fragment{Pointer: tprop("TUUIDs", "google"), JSON: `{"type":"string","format":"uuid"}`},
			Fragment{Pointer: tprop("TUUIDs", "gofrs"), JSON: `{"type":"string","format":"uuid"}`},
		),
		tcase("elem/uuid-stdlib", tget[TElems], "P3.2",
			Fragment{Pointer: tprop("TElems", "std"), JSON: `{"type":"array","items":{"type":"string","format":"uuid"}}`},
		),
		tcase("elem/uuid-google", tget[TElems], "P3.2",
			Fragment{Pointer: tprop("TElems", "google"), JSON: `{"type":"array","items":{"type":"string","format":"uuid"}}`},
		),
		tcase("elem/uuid-gofrs", tget[TElems], "P3.2",
			Fragment{Pointer: tprop("TElems", "gofrs"), JSON: `{"type":"array","items":{"type":"string","format":"uuid"}}`},
		),
		tcase("elem/byte-slices", tget[TElems], "P3.2",
			Fragment{Pointer: tprop("TElems", "blobs"), JSON: `{"type":"array","items":{"type":"string","contentEncoding":"base64"}}`},
		),
		tcase("elem/datatype", tget[TElems], "P3.2",
			Fragment{Pointer: tprop("TElems", "dt"), JSON: `{"type":"array","items":{"type":"string","format":"telem"}}`},
			Fragment{Pointer: tprop("TElems", "map"), JSON: `{"type":"object","additionalProperties":{"type":"string","format":"telem"}}`},
		),
		tcase("elem/override-datatype", overrideElems, "P3.2",
			Fragment{Pointer: tprop("TElemOvs", "one"), JSON: `{"type":"string","format":"tov"}`},
			Fragment{Pointer: tprop("TElemOvs", "list"), JSON: `{"type":"array","items":{"type":"string","format":"tov"}}`},
			Fragment{Pointer: tprop("TElemOvs", "map"), JSON: `{"type":"object","additionalProperties":{"type":"string","format":"tov"}}`},
		),
		tcase("cycle/struct-self", tget[TSelf], "",
			Fragment{Pointer: tprop("TSelf", "name"), JSON: `{"type":"string"}`},
			Fragment{Pointer: tprop("TSelf", "kids"), JSON: `{"type":"array","items":{"$ref":"#/components/schemas/KitchensinkTSelf"}}`},
		),
		tcase("cycle/slice-of-self", tget[TLHolder], "P3.2",
			Fragment{Pointer: tprop("TLHolder", "l"), JSON: `{"$ref":"#/components/schemas/KitchensinkTL"}`},
			Fragment{Pointer: tp + "TL", JSON: `{"type":"array","items":{"$ref":"#/components/schemas/KitchensinkTL"}}`},
		),
		tcase("cycle/map-of-self", tget[TMHolder], "P3.2",
			Fragment{Pointer: tprop("TMHolder", "m"), JSON: `{"$ref":"#/components/schemas/KitchensinkTM"}`},
			Fragment{Pointer: tp + "TM", JSON: `{"type":"object","additionalProperties":{"$ref":"#/components/schemas/KitchensinkTM"}}`},
		),
	)
}

func overrideElems(f *fizz.Fizz) {
	if err := f.Generator().OverrideDataType(reflect.TypeFor[TElemOv](), "string", "tov"); err != nil {
		panic(err)
	}
	tget[TElemOvs](f)
}

func withErrors(c Case, errs ...string) Case {
	c.Errors = errs
	return c
}

func crashes(c Case) Case {
	c.Crashes = true
	return c
}
