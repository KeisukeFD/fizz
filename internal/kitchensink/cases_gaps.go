package kitchensink

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/loopfz/gadgeto/tonic"

	"github.com/wI2L/fizz"
	"github.com/wI2L/fizz/internal/kitchensink/a/models"
	bmodels "github.com/wI2L/fizz/internal/kitchensink/b/models"
)

// Gap cases (Batch K). Every identifier starts with K so sibling
// cases_*.go files cannot clash.

type (
	KAny struct {
		V any `json:"v"`
	}
	KQueryIn struct {
		Q string `query:"q"`
	}
	KDurStrIn struct {
		D time.Duration `query:"d" default:"1s"`
	}

	KNe struct {
		S string `json:"s" validate:"ne=a"`
	}
	KKeys struct {
		M map[string]string `json:"m" validate:"dive,keys,min=2,endkeys"`
	}
	KUnique struct {
		L []string `json:"l" validate:"unique"`
	}
	KAlpha struct {
		A string `json:"a" validate:"alpha"`
		B string `json:"b" validate:"alphanum"`
		C string `json:"c" validate:"numeric"`
		D string `json:"d" validate:"hexadecimal"`
	}
	KFormats struct {
		URL      string `json:"url" validate:"url"`
		URI      string `json:"uri" validate:"uri"`
		UUID     string `json:"uuid" validate:"uuid"`
		UUID4    string `json:"uuid4" validate:"uuid4"`
		IPv4     string `json:"ipv4" validate:"ipv4"`
		IPv6     string `json:"ipv6" validate:"ipv6"`
		Hostname string `json:"hostname" validate:"hostname"`
		Email    string `json:"email" validate:"email"`
	}
	KDatetime struct {
		T string `json:"t" validate:"datetime=2006"`
	}
	KLen struct {
		S string `json:"s" validate:"len=3"`
	}
	KDurBound struct {
		D time.Duration `json:"d" validate:"min=1s"`
	}
	KOrUnmapped struct {
		S string `json:"s" validate:"email|customrule"`
	}
	KDecimal struct {
		F float64 `json:"f" validate:"gte=0.5,lte=1.5"`
	}

	KReqOmit struct {
		A string `json:"a,omitempty" validate:"required"`
	}
	KReqIf struct {
		A string `json:"a"`
		B string `json:"b" validate:"required_if=A x"`
	}
	KReqDive struct {
		L []string `json:"l" validate:"dive,required"`
	}

	KPet struct {
		Name string `json:"name"`
	}
	KPetEmbedIn struct {
		KPet
	}
	KReqBody struct {
		Name string `json:"name" validate:"required"`
	}
	KJSONIn struct {
		Name string `json:"name"`
	}

	KCookieStructIn struct {
		Session struct{ A string } `cookie:"session"`
	}

	KDeepFilter struct {
		Name string `json:"name"`
		Age  int32  `json:"age"`
	}
	KDeepStructIn struct {
		F KDeepFilter `query:"f" style:"deepObject"`
	}
	KDeepPtrMapIn struct {
		F *map[string]string `query:"f" style:"deepObject"`
	}
	KDeepNestedIn struct {
		F map[string]struct{ A string } `query:"f" style:"deepObject"`
	}
	KDeepRequiredIn struct {
		F map[string]string `query:"f,required" style:"deepObject"`
	}
	KDeepDefaultIn struct {
		F map[string]string `query:"f" style:"deepObject" default:"x"`
	}
	KDeepValidateIn struct {
		F map[string]string `query:"f" style:"deepObject" validate:"required"`
	}
	KDeepSliceIn struct {
		F map[string][]string `query:"f" style:"deepObject"`
	}
	KPathsIn struct {
		ID   string `path:"id"`
		Path string `path:"path"`
	}
	KWildIn struct {
		Path string `path:"path"`
	}
)

func kOut[T any](*gin.Context) (*T, error) { return new(T), nil }

// kGet registers GET /x returning T, so the component is named after T.
func kGet[T any](f *fizz.Fizz) {
	f.GET("/x", nil, tonic.Handler(kOut[T], 200))
}

// kProp returns the pointer of property p of the component for type c.
func kProp(c, p string) string {
	return "/components/schemas/Kitchensink" + c + "/properties/" + p
}

func kPostBody[T any](*gin.Context, *T) error { return nil }

func kRegister(f *fizz.Fizz) {
	f.GET("/c", nil, tonic.Handler(func(*gin.Context) error { return nil }, 200))
}

func kTwoHandlers(f *fizz.Fizz, id, path string) {
	f.GET(path, opts(fizz.ID(id)),
		tonic.Handler(func(*gin.Context) error { return nil }, 200),
		tonic.Handler(func(*gin.Context) error { return nil }, 200))
}

const kPrm = "/paths/~1x/get/parameters/0"

func init() {
	register(
		Case{
			Name:     "types/interface-any",
			Register: kGet[KAny],
			Expect:   []Fragment{{Pointer: kProp("KAny", "v"), JSON: `{}`}},
			FixedBy:  "P2.6",
		},
		Case{
			Name:     "params/no-allow-empty-value",
			Register: pRegister("GET", "/x", tonic.Handler(pHandler[KQueryIn], 200)),
			Expect:   []Fragment{{Pointer: kPrm + "/allowEmptyValue", Absent: true}},
		},
		Case{
			Name:     "params/explode-absent-by-default",
			Register: pRegister("GET", "/x", tonic.Handler(pHandler[KQueryIn], 200)),
			Expect:   []Fragment{{Pointer: kPrm + "/explode", Absent: true}},
		},
		Case{
			// "1s" is not an integer: an error, not a panic (D16).
			Name:     "params/query-duration-default-string-error",
			Register: pRegister("GET", "/x", tonic.Handler(pHandler[KDurStrIn], 200)),
			Errors:   []string{"default"},
			FixedBy:  "P3.6",
		},
		Case{
			Name:     "validate/ne-not-const",
			Register: kGet[KNe],
			Expect:   []Fragment{{Pointer: kProp("KNe", "s") + "/not", JSON: `{"const":"a"}`}},
			FixedBy:  "P4.1b",
		},
		Case{
			Name:     "validate/keys-property-names",
			Register: kGet[KKeys],
			Expect:   []Fragment{{Pointer: kProp("KKeys", "m") + "/propertyNames", JSON: `{"minLength":2}`}},
			FixedBy:  "P4.1b",
		},
		Case{
			Name:     "validate/unique-items",
			Register: kGet[KUnique],
			Expect:   []Fragment{{Pointer: kProp("KUnique", "l") + "/uniqueItems", JSON: `true`}},
			FixedBy:  "P4.1b",
		},
		Case{
			Name:     "validate/alpha-patterns",
			Register: kGet[KAlpha],
			Expect: []Fragment{
				{Pointer: kProp("KAlpha", "a") + "/pattern", JSON: `"^[a-zA-Z]+$"`},
				{Pointer: kProp("KAlpha", "b") + "/pattern", JSON: `"^[a-zA-Z0-9]+$"`},
				{Pointer: kProp("KAlpha", "c") + "/pattern", JSON: `"^[-+]?[0-9]+(?:\\.[0-9]+)?$"`},
				{Pointer: kProp("KAlpha", "d") + "/pattern", JSON: `"^[0-9a-fA-F]+$"`},
			},
			FixedBy: "P4.1b",
		},
		Case{
			Name:     "validate/formats",
			Register: kGet[KFormats],
			Expect: []Fragment{
				{Pointer: kProp("KFormats", "url") + "/format", JSON: `"uri"`},
				{Pointer: kProp("KFormats", "uri") + "/format", JSON: `"uri"`},
				{Pointer: kProp("KFormats", "uuid") + "/format", JSON: `"uuid"`},
				{Pointer: kProp("KFormats", "uuid4") + "/format", JSON: `"uuid"`},
				{Pointer: kProp("KFormats", "ipv4") + "/format", JSON: `"ipv4"`},
				{Pointer: kProp("KFormats", "ipv6") + "/format", JSON: `"ipv6"`},
				{Pointer: kProp("KFormats", "hostname") + "/format", JSON: `"hostname"`},
				{Pointer: kProp("KFormats", "email") + "/format", JSON: `"email"`},
			},
			FixedBy: "P4.1b",
		},
		Case{
			// datetime=<layout> has no OpenAPI format: ignored, no error.
			Name:     "validate/datetime-not-mapped",
			Register: kGet[KDatetime],
			Expect:   []Fragment{{Pointer: kProp("KDatetime", "t") + "/format", Absent: true}},
		},
		Case{
			Name:     "validate/string-len-both-bounds",
			Register: kGet[KLen],
			Expect: []Fragment{
				{Pointer: kProp("KLen", "s") + "/minLength", JSON: `3`},
				{Pointer: kProp("KLen", "s") + "/maxLength", JSON: `3`},
			},
		},
		Case{
			Name:     "validate/duration-bounds-ns",
			Register: kGet[KDurBound],
			Expect:   []Fragment{{Pointer: kProp("KDurBound", "d") + "/minimum", JSON: `1000000000`}},
			FixedBy:  "P4.1b",
		},
		Case{
			Name:     "validate/or-group-dropped-if-unmapped",
			Register: kGet[KOrUnmapped],
			Expect:   []Fragment{{Pointer: kProp("KOrUnmapped", "s") + "/anyOf", Absent: true}},
		},
		Case{
			Name:     "validate/gte-lte-inclusive-decimal",
			Register: kGet[KDecimal],
			Expect: []Fragment{
				{Pointer: kProp("KDecimal", "f") + "/minimum", JSON: `0.5`},
				{Pointer: kProp("KDecimal", "f") + "/maximum", JSON: `1.5`},
			},
			FixedBy: "P4.1b",
		},
		Case{
			Name:     "required/omitempty-required",
			Register: kGet[KReqOmit],
			Expect:   []Fragment{{Pointer: "/components/schemas/KitchensinkKReqOmit/required", JSON: `["a"]`}},
		},
		Case{
			Name:     "required/required-if-not-required",
			Register: kGet[KReqIf],
			Expect:   []Fragment{{Pointer: "/components/schemas/KitchensinkKReqIf/required", Absent: true}},
		},
		Case{
			Name:     "required/dive-required-not-required",
			Register: kGet[KReqDive],
			Expect:   []Fragment{{Pointer: "/components/schemas/KitchensinkKReqDive/required", Absent: true}},
		},
		Case{
			// The examples merge into the handler's default response.
			Name: "ops/default-response-examples-merge",
			Register: func(f *fizz.Fizz) {
				f.GET("/x", opts(fizz.DefaultResponseExamples(map[string]any{"a": "b"})), tonic.Handler(kOut[KPet], 200))
			},
			Expect: []Fragment{
				{Pointer: "/paths/~1x/get/responses/200/content/" + ctJSON + "/schema/$ref", JSON: `"#/components/schemas/KitchensinkKPet"`},
				{Pointer: "/paths/~1x/get/responses/200/content/" + ctJSON + "/examples/a/value", JSON: `"b"`},
			},
		},
		Case{
			Name: "ops/body-single-struct-direct-ref",
			Register: func(f *fizz.Fizz) {
				f.POST("/pets", opts(fizz.ID("createPet")), tonic.Handler(kPostBody[KPetEmbedIn], 201))
			},
			Expect: []Fragment{
				{Pointer: "/paths/~1pets/post/requestBody/content/" + ctJSON + "/schema", JSON: `{"$ref":"#/components/schemas/KitchensinkKPet"}`},
				{Pointer: schemas + "CreatePetInput", Absent: true},
			},
			FixedBy: "P4.6",
		},
		Case{
			Name: "ops/body-required-if-any-field-required",
			Register: func(f *fizz.Fizz) {
				f.POST("/a", opts(fizz.ID("postA")), tonic.Handler(kPostBody[KReqBody], 201))
				f.POST("/b", opts(fizz.ID("postB")), tonic.Handler(kPostBody[KPet], 201))
			},
			Expect: []Fragment{
				{Pointer: "/paths/~1a/post/requestBody/required", JSON: `true`},
				{Pointer: "/paths/~1b/post/requestBody/required", Absent: true},
			},
			FixedBy: "P4.6",
		},
		Case{
			Name:     "params/get-body-not-documented",
			Register: pRegister("GET", "/x", tonic.Handler(pHandler[KJSONIn], 200)),
			Expect:   []Fragment{{Pointer: "/paths/~1x/get/requestBody", Absent: true}},
		},
		Case{
			Name:     "params/head-body-not-documented",
			Register: pRegister("HEAD", "/x", tonic.Handler(pHandler[KJSONIn], 200)),
			Expect:   []Fragment{{Pointer: "/paths/~1x/head/requestBody", Absent: true}},
		},
		Case{
			Name:     "ops/default-id-closure",
			Register: kRegister,
			Expect:   []Fragment{{Pointer: "/paths/~1c/get/operationId", JSON: `"kRegister"`}},
			FixedBy:  "P4.7",
		},
		Case{
			Name: "ops/default-id-collision-package-prefix",
			Register: func(f *fizz.Fizz) {
				f.GET("/a", nil, tonic.Handler(models.Get, 200))
				f.GET("/b", nil, tonic.Handler(bmodels.Get, 200))
			},
			Expect: []Fragment{
				{Pointer: "/paths/~1a/get/operationId", JSON: `"Get"`},
				{Pointer: "/paths/~1b/get/operationId", JSON: `"models.Get"`},
			},
			FixedBy: "P4.7",
		},
		Case{
			// The same handler registered a third time clashes on Get and
			// on models.Get: the method and path name it.
			Name: "ops/default-id-collision-method-path",
			Register: func(f *fizz.Fizz) {
				f.GET("/a", nil, tonic.Handler(models.Get, 200))
				f.GET("/b", nil, tonic.Handler(bmodels.Get, 200))
				f.GET("/c/:id", nil, tonic.Handler(models.Get, 200))
			},
			Expect: []Fragment{
				{Pointer: "/paths/~1a/get/operationId", JSON: `"Get"`},
				{Pointer: "/paths/~1b/get/operationId", JSON: `"models.Get"`},
				{Pointer: "/paths/~1c~1{id}/get/operationId", JSON: `"get_c_id"`},
			},
			FixedBy: "P4.7",
		},
		Case{
			Name: "ops/explicit-id-collision-error",
			Register: func(f *fizz.Fizz) {
				f.GET("/a", opts(fizz.ID("dup")), tonic.Handler(oPlain, 200))
				f.GET("/b", opts(fizz.ID("dup")), tonic.Handler(oPlain, 200))
			},
			Expect: []Fragment{
				{Pointer: "/paths/~1a/get/operationId", JSON: `"dup"`},
				{Pointer: "/paths/~1b", Absent: true},
			},
			Errors:  []string{"dup"},
			FixedBy: "P4.7",
		},
		Case{
			Name:     "ops/multiple-tonic-handlers-error",
			Register: func(f *fizz.Fizz) { kTwoHandlers(f, "two", "/m") },
			Errors:   []string{"/m"},
			FixedBy:  "P4.8",
		},
		Case{
			// A failed operation leaves its ID free.
			Name: "ops/failed-op-id-reusable",
			Register: func(f *fizz.Fizz) {
				kTwoHandlers(f, "reuse", "/bad")
				f.GET("/good", opts(fizz.ID("reuse")), tonic.Handler(oPlain, 200))
			},
			Expect: []Fragment{
				{Pointer: "/paths/~1bad", Absent: true},
				{Pointer: "/paths/~1good/get/operationId", JSON: `"reuse"`},
			},
			Errors:  []string{"/bad"},
			FixedBy: "P4.8",
		},
		Case{
			Name:     "params/cookie-non-scalar-error",
			Register: pRegister("GET", "/x", tonic.Handler(pHandler[KCookieStructIn], 200)),
			Expect:   []Fragment{{Pointer: kPrm, Absent: true}},
			Errors:   []string{"cookie"},
			FixedBy:  "P4.2c",
		},
		Case{
			Name:     "params/deep-object-struct",
			Register: pRegister("GET", "/x", tonic.Handler(pHandler[KDeepStructIn], 200)),
			Expect: []Fragment{
				{Pointer: kPrm + "/style", JSON: `"deepObject"`},
				{Pointer: kPrm + "/schema/properties/name", JSON: `{"type":"string"}`},
				{Pointer: kPrm + "/schema/properties/age", JSON: `{"type":"integer","format":"int32"}`},
			},
			FixedBy: "P7.1",
		},
		Case{
			Name:     "params/deep-object-ptr-map",
			Register: pRegister("GET", "/x", tonic.Handler(pHandler[KDeepPtrMapIn], 200)),
			Expect: []Fragment{
				{Pointer: kPrm + "/style", JSON: `"deepObject"`},
				{Pointer: kPrm + "/schema", JSON: `{"type":"object","additionalProperties":{"type":"string"}}`},
			},
			FixedBy: "P7.1",
		},
		Case{
			Name:     "params/deep-object-nested-value-error",
			Register: pRegister("GET", "/x", tonic.Handler(pHandler[KDeepNestedIn], 200)),
			Errors:   []string{"deepObject"},
			FixedBy:  "P7.1",
		},
		Case{
			Name:     "params/deep-object-required-option-error",
			Register: pRegister("GET", "/x", tonic.Handler(pHandler[KDeepRequiredIn], 200)),
			Errors:   []string{"required"},
			FixedBy:  "P7.1",
		},
		Case{
			Name:     "params/deep-object-default-tag-error",
			Register: pRegister("GET", "/x", tonic.Handler(pHandler[KDeepDefaultIn], 200)),
			Errors:   []string{"default"},
		},
		Case{
			Name:     "params/deep-object-validate-required",
			Register: pRegister("GET", "/x", tonic.Handler(pHandler[KDeepValidateIn], 200)),
			Expect:   []Fragment{{Pointer: kPrm + "/required", JSON: `true`}},
		},
		Case{
			Name:     "params/deep-object-slice-values",
			Register: pRegister("GET", "/x", tonic.Handler(pHandler[KDeepSliceIn], 200)),
			Expect: []Fragment{{
				Pointer: kPrm + "/schema/additionalProperties",
				JSON:    `{"type":"array","items":{"type":"string"}}`,
			}},
		},
		Case{
			// The exact wording is the target text for P4.3.
			Name:     "params/wildcard-description-notes-slash",
			Register: pRegister("GET", "/f/*path", tonic.Handler(pHandler[KWildIn], 200)),
			Expect: []Fragment{{
				Pointer: "/paths/~1f~1{path}/get/parameters/0/description",
				JSON:    `"The value may contain '/'."`,
			}},
			FixedBy: "P4.3",
		},
		Case{
			Name:     "params/gin-path-translation-multi",
			Register: pRegister("GET", "/u/:id/f/*path", tonic.Handler(pHandler[KPathsIn], 200)),
			Expect: []Fragment{
				{Pointer: "/paths/~1u~1{id}~1f~1{path}/get/operationId", JSON: `"op"`},
			},
			FixedBy: "P4.3",
		},
		Case{
			// Plain gin handlers stay undocumented (D11).
			Name: "ops/plain-gin-handler-undocumented",
			Register: func(f *fizz.Fizz) {
				f.GET("/plain", nil, func(*gin.Context) {})
			},
			Expect: []Fragment{{Pointer: "/paths/~1plain", Absent: true}},
		},
	)
}
