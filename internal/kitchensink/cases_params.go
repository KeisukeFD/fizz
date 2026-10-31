package kitchensink

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/loopfz/gadgeto/tonic"

	"github.com/wI2L/fizz"
)

// Parameter input types. The P prefix avoids clashes with sibling files.
type (
	PSkipIn struct {
		A string `query:"-"`
		B string `query:""`
		C string `query:"kept"`
	}
	PRequiredIn struct {
		Q string `query:"q,required"`
	}
	PDefaultIn struct {
		N int `query:"n,default=3"`
	}
	POmitemptyIn struct {
		Q string `query:"q,omitempty"`
		R string `query:"r"`
	}
	PWildcardIn struct {
		Path string `path:"path"`
	}
	PMissingIn struct {
		Other string `query:"other"`
	}
	PConflictIn struct {
		ID string `query:"id" path:"id"`
	}
	PExplodeIn struct {
		Tags []string `query:"tags" explode:"false"`
	}
	PDurationIn struct {
		D time.Duration `query:"d" default:"1000000000"`
	}
	PDefReqIn struct {
		N int `query:"n" default:"3" validate:"required"`
	}
	PDeleteIn struct {
		ID   string `path:"id"`
		Name string `json:"name"`
	}
	PDeepIn struct {
		Filter map[string]string `query:"filter" style:"deepObject"`
	}
	PCookieIn struct {
		Session string `cookie:"session"`
	}
)

func pHandler[T any](*gin.Context, *T) error { return nil }

func pRegister(method, path string, h gin.HandlerFunc) func(*fizz.Fizz) {
	return func(f *fizz.Fizz) {
		f.Handle(path, method, []fizz.OperationOption{fizz.ID("op")}, h)
	}
}

func init() {
	register(
		Case{
			Name:     "params/query-dash-and-empty-skipped",
			Register: pRegister("GET", "/x", tonic.Handler(pHandler[PSkipIn], 200)),
			Expect: []Fragment{
				{Pointer: "/paths/~1x/get/parameters/0/name", JSON: `"kept"`},
				{Pointer: "/paths/~1x/get/parameters/1", Absent: true},
			},
			FixedBy: "P4.2",
		},
		Case{
			Name:     "params/query-required",
			Register: pRegister("GET", "/x", tonic.Handler(pHandler[PRequiredIn], 200)),
			Expect: []Fragment{
				{Pointer: "/paths/~1x/get/parameters/0/name", JSON: `"q"`},
				{Pointer: "/paths/~1x/get/parameters/0/required", JSON: `true`},
			},
			FixedBy: "P4.2",
		},
		Case{
			Name:     "params/query-default-option",
			Register: pRegister("GET", "/x", tonic.Handler(pHandler[PDefaultIn], 200)),
			Expect: []Fragment{
				{Pointer: "/paths/~1x/get/parameters/0/schema/default", JSON: `3`},
			},
			FixedBy: "P4.2",
		},
		Case{
			Name:     "params/query-omitempty-is-error-not-panic",
			Register: pRegister("GET", "/x", tonic.Handler(pHandler[POmitemptyIn], 200)),
			Errors:   []string{"omitempty"},
			Expect: []Fragment{
				{Pointer: "/paths/~1x/get/parameters/0/name", JSON: `"r"`},
				{Pointer: "/paths/~1x/get/parameters/1", Absent: true},
			},
			FixedBy: "P4.2",
		},
		Case{
			Name:     "params/gin-wildcard-path",
			Register: pRegister("GET", "/files/*path", tonic.Handler(pHandler[PWildcardIn], 200)),
			Expect: []Fragment{
				{Pointer: "/paths/~1files~1{path}/get/parameters/0/name", JSON: `"path"`},
				{Pointer: "/paths/~1files~1{path}/get/parameters/0/in", JSON: `"path"`},
				{Pointer: "/paths/~1files~1{path}/get/parameters/0/required", JSON: `true`},
			},
			FixedBy: "P4.3",
		},
		Case{
			Name:     "params/path-param-without-input",
			Register: pRegister("GET", "/users/:id", tonic.Handler(pHandler[struct{}], 200)),
			Errors:   []string{"id"},
			Expect: []Fragment{
				{Pointer: "/paths/~1users~1{id}/get/parameters/0/name", JSON: `"id"`},
				{Pointer: "/paths/~1users~1{id}/get/parameters/0/in", JSON: `"path"`},
				{Pointer: "/paths/~1users~1{id}/get/parameters/0/required", JSON: `true`},
				{Pointer: "/paths/~1users~1{id}/get/parameters/0/schema/type", JSON: `"string"`},
			},
			FixedBy: "P4.3",
		},
		Case{
			Name:     "params/path-param-missing-from-input",
			Register: pRegister("GET", "/users/:id", tonic.Handler(pHandler[PMissingIn], 200)),
			Errors:   []string{"id"},
			Expect: []Fragment{
				{Pointer: "/paths/~1users~1{id}/get/parameters/0/name", JSON: `"id"`},
				{Pointer: "/paths/~1users~1{id}/get/parameters/0/required", JSON: `true`},
			},
			FixedBy: "P4.3",
		},
		Case{
			Name:     "params/conflicting-locations",
			Register: pRegister("GET", "/users/:id", tonic.Handler(pHandler[PConflictIn], 200)),
			Errors:   []string{"id"},
			FixedBy:  "P4.8",
		},
		Case{
			Name:     "params/query-explode-false",
			Register: pRegister("GET", "/x", tonic.Handler(pHandler[PExplodeIn], 200)),
			Expect: []Fragment{
				{Pointer: "/paths/~1x/get/parameters/0/explode", JSON: `false`},
			},
			FixedBy: "P4.2",
		},
		Case{
			Name:     "params/query-duration-default-ns-integer",
			Register: pRegister("GET", "/x", tonic.Handler(pHandler[PDurationIn], 200)),
			Expect: []Fragment{
				{Pointer: "/paths/~1x/get/parameters/0/schema/type", JSON: `"integer"`},
				{Pointer: "/paths/~1x/get/parameters/0/schema/default", JSON: `1000000000`},
			},
			FixedBy: "P3.3",
		},
		Case{
			Name:     "params/default-with-required-drops-required",
			Register: pRegister("GET", "/x", tonic.Handler(pHandler[PDefReqIn], 200)),
			Errors:   []string{"default"},
			Expect: []Fragment{
				{Pointer: "/paths/~1x/get/parameters/0/required", Absent: true},
				{Pointer: "/paths/~1x/get/parameters/0/schema/default", JSON: `3`},
			},
			FixedBy: "P3.6",
		},
		Case{
			// PDeleteIn has a path parameter, so its body is named after
			// the operation (P4.6): OpInput.
			Name:     "params/delete-body-documented",
			Register: pRegister("DELETE", "/items/:id", tonic.Handler(pHandler[PDeleteIn], 200)),
			Expect: []Fragment{
				{Pointer: "/paths/~1items~1{id}/delete/requestBody/content/application~1json/schema", JSON: `{"$ref":"#/components/schemas/OpInput"}`},
				{Pointer: "/components/schemas/OpInput/properties/name", JSON: `{"type":"string"}`},
			},
			FixedBy: "P4.6",
		},
		Case{
			Name:     "params/deep-object-placeholder",
			Register: pRegister("GET", "/x", tonic.Handler(pHandler[PDeepIn], 200)),
			Expect: []Fragment{
				{Pointer: "/paths/~1x/get/parameters/0/in", JSON: `"query"`},
				{Pointer: "/paths/~1x/get/parameters/0/style", JSON: `"deepObject"`},
				{Pointer: "/paths/~1x/get/parameters/0/explode", JSON: `true`},
				{Pointer: "/paths/~1x/get/parameters/0/schema/type", JSON: `"object"`},
			},
			FixedBy: "P7.1",
		},
		Case{
			Name:     "params/cookie-placeholder",
			Register: pRegister("GET", "/x", tonic.Handler(pHandler[PCookieIn], 200)),
			Expect: []Fragment{
				{Pointer: "/paths/~1x/get/parameters/0/name", JSON: `"session"`},
				{Pointer: "/paths/~1x/get/parameters/0/in", JSON: `"cookie"`},
			},
			FixedBy: "P4.2c",
		},
	)
}
