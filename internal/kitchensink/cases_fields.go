package kitchensink

import (
	"github.com/gin-gonic/gin"
	"github.com/loopfz/gadgeto/tonic"

	"github.com/KeisukeFD/fizz"
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
		},
		Case{
			Name:     "fields/json-string-option-on-int",
			Register: fieldsResp[FStringOpt]("x"),
			Expect: []Fragment{{
				Pointer: "/components/schemas/KitchensinkFStringOpt/properties/n",
				JSON:    `{"type":"string"}`,
			}},
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
		},
		Case{
			Name:     "fields/embedded-cycle-response",
			Register: fieldsResp[FCycA]("x"),
			Expect:   []Fragment{{Pointer: "/components/schemas/KitchensinkFCycA/type", JSON: `"object"`}},
		},
		Case{
			// Same cycle through the param walker (generator.go:523).
			Name:     "fields/embedded-cycle-request",
			Register: fieldsReq[FCycA]("cycReq"),
			// No JSON members at all, so no request body; the point is that the walk ends.
			Expect: []Fragment{{Pointer: "/paths/~1cycReq/post/requestBody", Absent: true}},
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
		},
		Case{
			Name:     "fields/shared-component-annotations-at-use-site",
			Register: fieldsResp[FHolder]("x"),
			Expect:   holderProps,
		},
		Case{
			Name:     "fields/shared-component-order-holder-first",
			Register: fieldsBoth(fieldsResp[FHolder]("h"), fieldsResp[FShared]("s")),
			Expect:   holderProps,
		},
		Case{
			Name:     "fields/shared-component-order-holder-last",
			Register: fieldsBoth(fieldsResp[FShared]("s"), fieldsResp[FHolder]("h")),
			Expect:   holderProps,
		},
		Case{
			Name:     "fields/nullable-ptr-struct",
			Register: fieldsResp[FNullable]("x"),
			Expect: []Fragment{{
				Pointer: "/components/schemas/KitchensinkFNullable/properties/p",
				JSON:    `{"anyOf":[{"$ref":"` + fSharedRef + `"},{"type":"null"}]}`,
			}},
		},
		Case{
			Name:     "fields/nullable-ptr-slice",
			Register: fieldsResp[FNullable]("x"),
			Expect: []Fragment{{
				Pointer: "/components/schemas/KitchensinkFNullable/properties/s",
				JSON:    `{"type":["array","null"],"items":{"type":"string"}}`,
			}},
		},
		Case{
			Name:     "fields/nullable-ptr-map",
			Register: fieldsResp[FNullable]("x"),
			Expect: []Fragment{{
				Pointer: "/components/schemas/KitchensinkFNullable/properties/m",
				JSON:    `{"type":["object","null"],"additionalProperties":{"type":"string"}}`,
			}},
		},
		Case{
			Name:     "fields/embedded-non-struct",
			Register: fieldsResp[FEmbedStr]("x"),
			Expect: []Fragment{{
				Pointer: "/components/schemas/KitchensinkFEmbedStr/properties",
				JSON:    `{"FStr":{"type":"string"},"y":{"type":"integer","format":"int32"}}`,
			}},
		},
	)
}

// New struct tags (P6.2): one case per tag.
type (
	FTagTitle struct {
		Name string `json:"name" title:"Pet name"`
	}
	FTagPattern struct {
		Slug string `json:"slug" pattern:"^[a-z]+$"`
		Bad  string `json:"bad" pattern:"[a-"`
	}
	FTagMultipleOf struct {
		Price float64 `json:"price" multipleOf:"0.01"`
		Zero  float64 `json:"zero" multipleOf:"0"`
		Bad   float64 `json:"bad" multipleOf:"x"`
	}
	FTagRW struct {
		RO   string `json:"ro" readOnly:"true"`
		WO   string `json:"wo" writeOnly:"true"`
		Both string `json:"both" readOnly:"true" writeOnly:"true"`
	}
	FTagNullable struct {
		P *string  `json:"p" nullable:"false"`
		S string   `json:"s" nullable:"true"`
		R *FShared `json:"r" nullable:"false"`
		N FShared  `json:"n" nullable:"true"`
	}
	FTagClosed struct {
		Anon struct {
			X int `json:"x"`
		} `json:"anon" additionalProperties:"false"`
		Named FShared `json:"named" additionalProperties:"false"`
		N     int     `json:"n" additionalProperties:"false"`
	}
	FTagInline struct {
		Home  FShared `json:"home" inline:"true"`
		Other FShared `json:"other"`
	}
	FTagStyleQuery struct {
		IDs []string `query:"ids" style:"pipeDelimited"`
		M   string   `query:"m" style:"matrix"`
	}
	FTagReservedQuery struct {
		U string `query:"u" allowReserved:"true"`
		H string `header:"X-H" allowReserved:"true"`
	}
)

// fieldsQuery registers GET /id (operation id) with T as the input.
func fieldsQuery[T any](id string) func(*fizz.Fizz) {
	return func(f *fizz.Fizz) {
		f.GET("/"+id, []fizz.OperationOption{fizz.ID(id)}, tonic.Handler(func(*gin.Context, *T) error { return nil }, 200))
	}
}

func init() {
	register(
		Case{
			Name:     "fields/tag-title",
			Register: fieldsReq[FTagTitle]("tagTitle"),
			Expect:   []Fragment{{Pointer: "/components/schemas/TagTitleInput/properties/name/title", JSON: `"Pet name"`}},
		},
		Case{
			Name:     "fields/tag-pattern",
			Register: fieldsReq[FTagPattern]("tagPattern"),
			Expect: []Fragment{
				{Pointer: "/components/schemas/TagPatternInput/properties/slug/pattern", JSON: `"^[a-z]+$"`},
				{Pointer: "/components/schemas/TagPatternInput/properties/bad/pattern", Absent: true},
			},
			Errors: []string{"pattern"},
		},
		Case{
			Name:     "fields/tag-multipleof",
			Register: fieldsReq[FTagMultipleOf]("tagMultipleOf"),
			Expect: []Fragment{
				{Pointer: "/components/schemas/TagMultipleOfInput/properties/price/multipleOf", JSON: `0.01`},
				{Pointer: "/components/schemas/TagMultipleOfInput/properties/zero/multipleOf", Absent: true},
				{Pointer: "/components/schemas/TagMultipleOfInput/properties/bad/multipleOf", Absent: true},
			},
			Errors: []string{"multipleOf"},
		},
		Case{
			Name:     "fields/tag-readonly-writeonly",
			Register: fieldsReq[FTagRW]("tagRW"),
			Expect: []Fragment{
				{Pointer: "/components/schemas/TagRWInput/properties/ro/readOnly", JSON: `true`},
				{Pointer: "/components/schemas/TagRWInput/properties/wo/writeOnly", JSON: `true`},
				{Pointer: "/components/schemas/TagRWInput/properties/both", JSON: `{"type":"string"}`},
			},
			Warnings: []string{"both"},
		},
		Case{
			Name:     "fields/tag-nullable",
			Register: fieldsReq[FTagNullable]("tagNullable"),
			Expect: []Fragment{
				{Pointer: "/components/schemas/TagNullableInput/properties/p", JSON: `{"type":"string"}`},
				{Pointer: "/components/schemas/TagNullableInput/properties/s", JSON: `{"type":["string","null"]}`},
				{Pointer: "/components/schemas/TagNullableInput/properties/r", JSON: `{"$ref":"` + fSharedRef + `"}`},
				{Pointer: "/components/schemas/TagNullableInput/properties/n", JSON: `{"anyOf":[{"$ref":"` + fSharedRef + `"},{"type":"null"}]}`},
			},
		},
		Case{
			Name:     "fields/tag-additionalproperties-false",
			Register: fieldsReq[FTagClosed]("tagClosed"),
			Expect: []Fragment{
				{Pointer: "/components/schemas/TagClosedInput/properties/anon/additionalProperties", JSON: `false`},
				{Pointer: "/components/schemas/TagClosedInput/properties/named", JSON: `{"$ref":"` + fSharedRef + `","unevaluatedProperties":false}`},
				{Pointer: "/components/schemas/TagClosedInput/properties/n", JSON: `{"type":"integer","format":"int64"}`},
			},
			Errors: []string{"additionalProperties"},
		},
		Case{
			Name:     "fields/tag-inline",
			Register: fieldsReq[FTagInline]("tagInline"),
			Expect: []Fragment{
				{Pointer: "/components/schemas/TagInlineInput/properties/home/$ref", Absent: true},
				{Pointer: "/components/schemas/TagInlineInput/properties/home/type", JSON: `"object"`},
				{Pointer: "/components/schemas/TagInlineInput/properties/other", JSON: `{"$ref":"` + fSharedRef + `"}`},
				{Pointer: "/components/schemas/KitchensinkFShared/type", JSON: `"object"`},
			},
		},
		Case{
			Name:     "params/tag-style",
			Register: fieldsQuery[FTagStyleQuery]("tagStyle"),
			Expect: []Fragment{
				{Pointer: "/paths/~1tagStyle/get/parameters/0/style", JSON: `"pipeDelimited"`},
				{Pointer: "/paths/~1tagStyle/get/parameters/0/explode", JSON: `false`},
				{Pointer: "/paths/~1tagStyle/get/parameters/1/style", Absent: true},
			},
			Errors: []string{"style"},
		},
		Case{
			Name:     "params/tag-allowreserved",
			Register: fieldsQuery[FTagReservedQuery]("tagReserved"),
			Expect: []Fragment{
				{Pointer: "/paths/~1tagReserved/get/parameters/0/allowReserved", JSON: `true`},
				{Pointer: "/paths/~1tagReserved/get/parameters/1/allowReserved", Absent: true},
			},
			Errors: []string{"allowReserved"},
		},
	)
}
