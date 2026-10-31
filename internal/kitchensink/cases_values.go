package kitchensink

import (
	"math"
	"net/http"
	"net/http/httptest"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/loopfz/gadgeto/tonic"

	"github.com/wI2L/fizz"
	"github.com/wI2L/fizz/openapi"
)

// Value and validation cases (P0.4e). Types are prefixed V to avoid
// clashing with sibling case files in this package.

func vHandler[T any](*gin.Context) (*T, error) { return new(T), nil }

// vPost registers GET /x returning T, so the component is named after T
// rather than after the operation.
func vPost[T any]() func(f *fizz.Fizz) {
	return func(f *fizz.Fizz) {
		f.GET("/x", nil, tonic.Handler(vHandler[T], 200))
	}
}

// vComp returns the pointer of the component generated for type c.
// Today components are named "<package><Type>"; adjust the prefix here
// if the naming scheme (P4.5) changes.
func vComp(c string) string {
	return "/components/schemas/Kitchensink" + c
}

// vProp returns the pointer of property p of the component for type c.
func vProp(c, p string) string {
	return vComp(c) + "/properties/" + p
}

type VPtrDefault struct {
	N *int `json:"n" default:"10"`
}

type VDurationBody struct {
	D time.Duration `json:"d" default:"1s" enum:"1s,2s"`
}

type VEnumTrim struct {
	S string `json:"s" enum:"a, b"`
}

type VBadBool struct {
	B bool `json:"b" default:"yes"`
}

type VDefaultRequired struct {
	N int `json:"n" default:"3" validate:"required"`
}

type VFloat32Default struct {
	F float32 `json:"f" default:"0.1"`
}

type VExample struct {
	S string `json:"s" example:"hello"`
	N int    `json:"n" example:"7"`
}

type VMinZero struct {
	I int     `json:"i" validate:"min=0"`
	F float64 `json:"f" validate:"min=1.5"`
}

type VExclusive struct {
	F float64 `json:"f" validate:"gt=1,lt=2"`
}

type VDive struct {
	L []int `json:"l" validate:"dive,min=2"`
}

type VOrGroup struct {
	I int `json:"i" validate:"min=1|max=5"`
}

type VEmail struct {
	E string `json:"e" validate:"required,email,max=20"`
}

type VOneof struct {
	S string `json:"s" validate:"oneof=a b"`
}

type VConst struct {
	S string `json:"s" validate:"len=3"`
	I int    `json:"i" validate:"eq=5"`
	T string `json:"t" validate:"eq=x"`
}

type VGating struct {
	D time.Duration `json:"d" validate:"min=1s,max=1m"`
	B []byte        `json:"b" validate:"min=1,max=4"`
}

type VNaNExample struct {
	N int `json:"n"`
}

func init() {
	register(
		Case{
			Name:     "values/default-pointer-field",
			Register: vPost[VPtrDefault](),
			Expect: []Fragment{
				{Pointer: vProp("VPtrDefault", "n") + "/default", JSON: `10`},
			},
			FixedBy: "P3.6",
		},
		Case{
			Name:     "values/duration-default-enum-body",
			Register: vPost[VDurationBody](),
			Expect: []Fragment{
				{Pointer: vProp("VDurationBody", "d") + "/default", JSON: `1000000000`},
				{Pointer: vProp("VDurationBody", "d") + "/enum", JSON: `[1000000000,2000000000]`},
			},
		},
		Case{
			Name:     "values/enum-trimmed",
			Register: vPost[VEnumTrim](),
			Expect: []Fragment{
				{Pointer: vProp("VEnumTrim", "s") + "/enum", JSON: `["a","b"]`},
			},
			FixedBy: "P3.6",
		},
		Case{
			Name:     "values/invalid-bool-default",
			Register: vPost[VBadBool](),
			Errors:   []string{"default"},
			Expect: []Fragment{
				{Pointer: vProp("VBadBool", "b") + "/default", Absent: true},
			},
			FixedBy: "P3.6",
		},
		Case{
			// D15: body fields never get tonic defaults, so required is
			// kept, default dropped, warning recorded.
			Name:     "values/default-required-body",
			Register: vPost[VDefaultRequired](),
			Errors:   []string{"default"},
			Expect: []Fragment{
				{Pointer: vProp("VDefaultRequired", "n") + "/default", Absent: true},
				{Pointer: vComp("VDefaultRequired") + "/required", JSON: `["n"]`},
			},
		},
		Case{
			Name:     "values/float32-default-literal",
			Register: vPost[VFloat32Default](),
			Expect: []Fragment{
				{Pointer: vProp("VFloat32Default", "f") + "/default", JSON: `0.1`},
			},
			FixedBy: "P3.6",
		},
		Case{
			Name: "values/nan-example-recorded",
			Register: func(f *fizz.Fizz) {
				f.GET("/x", []fizz.OperationOption{
					fizz.ResponseWithExamples("400", "bad", new(VNaNExample), nil,
						map[string]any{"bad": math.NaN()}),
				}, tonic.Handler(vHandler[VNaNExample], 200))
				// Serving the spec must not panic and must answer 200.
				f.GET("/spec", nil, f.OpenAPI(&openapi.Info{Title: "t", Version: "0"}, "json"))
				w := httptest.NewRecorder()
				f.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/spec", nil))
				if w.Code != http.StatusOK {
					panic("serving the spec answered " + strconv.Itoa(w.Code))
				}
			},
			Errors: []string{"example"},
			Expect: []Fragment{
				{Pointer: "/paths/~1x/get/responses/400/content/application~1json/examples/bad", Absent: true},
			},
			FixedBy: "P2.5",
		},
		Case{
			Name: "values/default-response-examples-no-output",
			Register: func(f *fizz.Fizz) {
				f.GET("/x", []fizz.OperationOption{
					fizz.DefaultResponseExamples(map[string]any{"a": "b"}),
				}, tonic.Handler(func(*gin.Context) error { return nil }, 204))
			},
			// The examples merge into the handler's default (204) response.
			Expect: []Fragment{
				{Pointer: "/paths/~1x/get/responses/204/content/application~1json/schema", Absent: true},
				{Pointer: "/paths/~1x/get/responses/204/content/application~1json/examples/a/value", JSON: `"b"`},
			},
			FixedBy: "P4.4",
		},
		Case{
			Name: "values/empty-examples-map",
			Register: func(f *fizz.Fizz) {
				f.GET("/x", []fizz.OperationOption{
					fizz.ResponseWithExamples("400", "bad", nil, nil, map[string]any{}),
				}, tonic.Handler(func(*gin.Context) error { return nil }, 204))
			},
			Expect: []Fragment{
				{Pointer: "/paths/~1x/get/responses/400/content", Absent: true},
			},
			FixedBy: "P4.4",
		},
		Case{
			Name:     "values/example-tag-to-examples",
			Register: vPost[VExample](),
			Expect: []Fragment{
				{Pointer: vProp("VExample", "s") + "/examples", JSON: `["hello"]`},
				{Pointer: vProp("VExample", "n") + "/examples", JSON: `[7]`},
				{Pointer: vProp("VExample", "s") + "/example", Absent: true},
			},
			FixedBy: "P2.2",
		},
		Case{
			Name:     "validate/min-zero-and-decimal",
			Register: vPost[VMinZero](),
			Expect: []Fragment{
				{Pointer: vProp("VMinZero", "i") + "/minimum", JSON: `0`},
				{Pointer: vProp("VMinZero", "f") + "/minimum", JSON: `1.5`},
			},
			FixedBy: "P4.1b",
		},
		Case{
			Name:     "validate/float-exclusive-bounds",
			Register: vPost[VExclusive](),
			Expect: []Fragment{
				{Pointer: vProp("VExclusive", "f") + "/exclusiveMinimum", JSON: `1`},
				{Pointer: vProp("VExclusive", "f") + "/exclusiveMaximum", JSON: `2`},
				{Pointer: vProp("VExclusive", "f") + "/minimum", Absent: true},
				{Pointer: vProp("VExclusive", "f") + "/maximum", Absent: true},
			},
			FixedBy: "P4.1b",
		},
		Case{
			Name:     "validate/dive-min-on-items",
			Register: vPost[VDive](),
			Expect: []Fragment{
				{Pointer: vProp("VDive", "l") + "/items/minimum", JSON: `2`},
				{Pointer: vProp("VDive", "l") + "/minItems", Absent: true},
			},
			FixedBy: "P4.1b",
		},
		Case{
			// An OR group is anyOf or dropped, never ANDed.
			Name:     "validate/or-group-anyof",
			Register: vPost[VOrGroup](),
			Expect: []Fragment{
				{Pointer: vProp("VOrGroup", "i") + "/anyOf", JSON: `[{"minimum":1},{"maximum":5}]`},
				{Pointer: vProp("VOrGroup", "i") + "/minimum", Absent: true},
				{Pointer: vProp("VOrGroup", "i") + "/maximum", Absent: true},
			},
			FixedBy: "P4.1b",
		},
		Case{
			Name:     "validate/email-keeps-maxlength",
			Register: vPost[VEmail](),
			Expect: []Fragment{
				{Pointer: vProp("VEmail", "e") + "/format", JSON: `"email"`},
				{Pointer: vProp("VEmail", "e") + "/maxLength", JSON: `20`},
				{Pointer: vComp("VEmail") + "/required", JSON: `["e"]`},
			},
			FixedBy: "P4.1b",
		},
		Case{
			Name:     "validate/oneof-enum",
			Register: vPost[VOneof](),
			Expect: []Fragment{
				{Pointer: vProp("VOneof", "s") + "/enum", JSON: `["a","b"]`},
			},
			FixedBy: "P4.1b",
		},
		Case{
			Name:     "validate/len-eq-const",
			Register: vPost[VConst](),
			Expect: []Fragment{
				{Pointer: vProp("VConst", "s") + "/minLength", JSON: `3`},
				{Pointer: vProp("VConst", "s") + "/maxLength", JSON: `3`},
				{Pointer: vProp("VConst", "i") + "/const", JSON: `5`},
				{Pointer: vProp("VConst", "t") + "/const", JSON: `"x"`},
			},
			FixedBy: "P4.1b",
		},
		Case{
			// Duration bounds are parsed into ns; []byte gets no length rules.
			Name:     "validate/duration-and-bytes-gating",
			Register: vPost[VGating](),
			Expect: []Fragment{
				{Pointer: vProp("VGating", "d") + "/minimum", JSON: `1000000000`},
				{Pointer: vProp("VGating", "d") + "/maximum", JSON: `60000000000`},
				{Pointer: vProp("VGating", "b") + "/minLength", Absent: true},
				{Pointer: vProp("VGating", "b") + "/maxLength", Absent: true},
				{Pointer: vProp("VGating", "b") + "/minItems", Absent: true},
				{Pointer: vProp("VGating", "b") + "/maxItems", Absent: true},
			},
			FixedBy: "P4.1b",
		},
	)
}
