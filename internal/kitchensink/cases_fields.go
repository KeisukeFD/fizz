package kitchensink

import (
	"github.com/gin-gonic/gin"
	"github.com/loopfz/gadgeto/tonic"

	"github.com/wI2L/fizz"
)

// Field cases (P0.4c). Test types are prefixed F to stay clear of the
// sibling cases_*.go files, which share this package.

// fieldsResp registers GET /id (operation id) returning T.
func fieldsResp[T any](id string) func(*fizz.Fizz) {
	return func(f *fizz.Fizz) {
		f.GET("/"+id, []fizz.OperationOption{fizz.ID(id)}, tonic.Handler(func(*gin.Context, *struct{}) (*T, error) { return new(T), nil }, 200))
	}
}

// fieldsReq registers POST /id (operation id) with T as JSON body.
func fieldsReq[T any](id string) func(*fizz.Fizz) {
	return func(f *fizz.Fizz) {
		f.POST("/"+id, []fizz.OperationOption{fizz.ID(id)}, tonic.Handler(func(*gin.Context, *T) error { return nil }, 200))
	}
}

// fieldsBoth runs fn1 then fn2.
func fieldsBoth(fn1, fn2 func(*fizz.Fizz)) func(*fizz.Fizz) {
	return func(f *fizz.Fizz) { fn1(f); fn2(f) }
}

// Dominance.
type (
	FDomInner struct {
		Name  string `json:"name"`
		Extra string `json:"extra"`
	}
	FDomOuter struct {
		FDomInner
		Name int32 `json:"name"`
	}

	FTagA struct {
		Foo string // untagged: json name "Foo"
	}
	FTagB struct {
		Bar int32 `json:"Foo"`
	}
	FTagWins struct {
		FTagA
		FTagB
	}

	FAmbA struct {
		Dup string // untagged: json name "Dup"
	}
	FAmbB struct {
		Dup int32 // same name at the same depth
	}
	FAmbiguous struct {
		FAmbA
		FAmbB
		Keep bool `json:"keep"`
	}
)

// Skips and tag options.
type (
	FDash struct {
		Skipped string `json:"-"`
		Keep    string `json:"keep"`
	}
	FDashComma struct {
		Dash string `json:"-,"`
	}
	FStringOpt struct {
		N int32 `json:"n,string"`
	}
)

// Embedding.
type (
	FShared struct {
		V string `json:"v"`
	}
	FTaggedEmb struct {
		FShared `json:"shared"`
		Z       int32 `json:"z"`
	}

	// Cycle not through the root: A{B}, B{*C}, C{*B}.
	FCycA struct{ FCycB }
	FCycB struct{ *FCycC }
	FCycC struct{ *FCycB }

	FStr      string
	FEmbedStr struct {
		FStr
		Y int32 `json:"y"`
	}
)

// Shared component and nullability.
type (
	FHolder struct {
		A FShared `json:"a" description:"first"`
		B FShared `json:"b" description:"second" deprecated:"true"`
	}
	FNullable struct {
		P *FShared           `json:"p"`
		S *[]string          `json:"s"`
		M *map[string]string `json:"m"`
	}
)

const fSharedRef = "#/components/schemas/KitchensinkFShared"

func init() {
	holderProps := []Fragment{
		{Pointer: "/components/schemas/KitchensinkFHolder/properties/a", JSON: `{"$ref":"` + fSharedRef + `","description":"first"}`},
		{Pointer: "/components/schemas/KitchensinkFHolder/properties/b", JSON: `{"$ref":"` + fSharedRef + `","description":"second","deprecated":true}`},
		{Pointer: "/components/schemas/KitchensinkFShared", JSON: `{"type":"object","properties":{"v":{"type":"string"}}}`},
	}
	taggedEmbProps := []Fragment{{
		Pointer: "/components/schemas/KitchensinkFTaggedEmb/properties",
		JSON:    `{"shared":{"$ref":"` + fSharedRef + `"},"z":{"type":"integer","format":"int32"}}`,
	}}

	register(
		Case{
			Name:     "fields/dominance-shallowest-wins",
			Register: fieldsResp[FDomOuter]("x"),
			Expect: []Fragment{{
				Pointer: "/components/schemas/KitchensinkFDomOuter/properties",
				JSON:    `{"name":{"type":"integer","format":"int32"},"extra":{"type":"string"}}`,
			}},
		},
		Case{
			Name:     "fields/dominance-tagged-beats-untagged",
			Register: fieldsResp[FTagWins]("x"),
			Expect: []Fragment{{
				Pointer: "/components/schemas/KitchensinkFTagWins/properties",
				JSON:    `{"Foo":{"type":"integer","format":"int32"}}`,
			}},
		},
		Case{
			Name:     "fields/dominance-ambiguous-dropped",
			Register: fieldsResp[FAmbiguous]("x"),
			Expect: []Fragment{{
				Pointer: "/components/schemas/KitchensinkFAmbiguous/properties",
				JSON:    `{"keep":{"type":"boolean"}}`,
			}},
			FixedBy: "P3.1a",
		},
		Case{
			Name:     "fields/json-dash-skipped",
			Register: fieldsResp[FDash]("x"),
			Expect: []Fragment{{
				Pointer: "/components/schemas/KitchensinkFDash/properties",
				JSON:    `{"keep":{"type":"string"}}`,
			}},
		},
		Case{
			Name:     "fields/json-dash-comma-named-dash",
			Register: fieldsResp[FDashComma]("x"),
			Expect: []Fragment{{
				Pointer: "/components/schemas/KitchensinkFDashComma/properties",
				JSON:    `{"-":{"type":"string"}}`,
			}},
			FixedBy: "P3.1a",
		},
		Case{
			Name:     "fields/json-string-option-on-int",
			Register: fieldsResp[FStringOpt]("x"),
			Expect: []Fragment{{
				Pointer: "/components/schemas/KitchensinkFStringOpt/properties/n",
				JSON:    `{"type":"string"}`,
			}},
			FixedBy: "P3.1a",
		},
		Case{
			Name:     "fields/tagged-embedded-struct-response",
			Register: fieldsResp[FTaggedEmb]("x"),
			Expect:   taggedEmbProps,
		},
		Case{
			Name:     "fields/tagged-embedded-struct-request",
			Register: fieldsReq[FTaggedEmb]("tagReq"),
			Expect:   []Fragment{{Pointer: "/components/schemas/TagReqInput/properties", JSON: taggedEmbProps[0].JSON}},
			FixedBy:  "P3.1b",
		},
		Case{
			// Stack-overflows today (generator.go:1032): never run.
			Name:     "fields/embedded-cycle-response",
			Register: fieldsResp[FCycA]("x"),
			Expect:   []Fragment{{Pointer: "/components/schemas/KitchensinkFCycA/type", JSON: `"object"`}},
			FixedBy:  "P3.1a",
			Crashes:  true,
		},
		Case{
			// Same cycle through the param walker (generator.go:523).
			Name:     "fields/embedded-cycle-request",
			Register: fieldsReq[FCycA]("cycReq"),
			Expect:   []Fragment{{Pointer: "/components/schemas/CycReqInput/type", JSON: `"object"`}},
			FixedBy:  "P3.1b",
			Crashes:  true,
		},
		Case{
			// The "-" field leaks as property "" (generator.go:623-647).
			// The empty token ends the pointer with a slash.
			Name:     "fields/json-dash-no-empty-property-request",
			Register: fieldsReq[FDash]("dashReq"),
			Expect: []Fragment{
				{Pointer: "/components/schemas/DashReqInput/properties/", Absent: true},
				{Pointer: "/components/schemas/DashReqInput/properties", JSON: `{"keep":{"type":"string"}}`},
			},
			FixedBy: "P3.1b",
		},
		Case{
			Name:     "fields/shared-component-annotations-at-use-site",
			Register: fieldsResp[FHolder]("x"),
			Expect:   holderProps,
			FixedBy:  "P3.4",
		},
		Case{
			Name:     "fields/shared-component-order-holder-first",
			Register: fieldsBoth(fieldsResp[FHolder]("h"), fieldsResp[FShared]("s")),
			Expect:   holderProps,
			FixedBy:  "P3.4",
		},
		Case{
			Name:     "fields/shared-component-order-holder-last",
			Register: fieldsBoth(fieldsResp[FShared]("s"), fieldsResp[FHolder]("h")),
			Expect:   holderProps,
			FixedBy:  "P3.4",
		},
		Case{
			Name:     "fields/nullable-ptr-struct",
			Register: fieldsResp[FNullable]("x"),
			Expect: []Fragment{{
				Pointer: "/components/schemas/KitchensinkFNullable/properties/p",
				JSON:    `{"anyOf":[{"$ref":"` + fSharedRef + `"},{"type":"null"}]}`,
			}},
			FixedBy: "P3.5",
		},
		Case{
			Name:     "fields/nullable-ptr-slice",
			Register: fieldsResp[FNullable]("x"),
			Expect: []Fragment{{
				Pointer: "/components/schemas/KitchensinkFNullable/properties/s",
				JSON:    `{"type":["array","null"],"items":{"type":"string"}}`,
			}},
			FixedBy: "P3.5",
		},
		Case{
			Name:     "fields/nullable-ptr-map",
			Register: fieldsResp[FNullable]("x"),
			Expect: []Fragment{{
				Pointer: "/components/schemas/KitchensinkFNullable/properties/m",
				JSON:    `{"type":["object","null"],"additionalProperties":{"type":"string"}}`,
			}},
			FixedBy: "P3.5",
		},
		Case{
			Name:     "fields/embedded-non-struct",
			Register: fieldsResp[FEmbedStr]("x"),
			Expect: []Fragment{{
				Pointer: "/components/schemas/KitchensinkFEmbedStr/properties",
				JSON:    `{"FStr":{"type":"string"},"y":{"type":"integer","format":"int32"}}`,
			}},
			FixedBy: "P3.1a",
		},
	)
}
