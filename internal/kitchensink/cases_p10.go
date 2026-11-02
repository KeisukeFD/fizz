package kitchensink

import (
	"github.com/loopfz/gadgeto/tonic"

	"github.com/KeisukeFD/fizz"
	"github.com/KeisukeFD/fizz/openapi"
)

// Review findings that the semantic lint (specvalidate.LintSemantics, part
// of MetaProblems) catches in fizz output.
type (
	PSemQueryIn struct {
		Q string `query:"q"`
	}
	PSemAuthIn struct {
		Auth string `header:"Authorization"`
	}
	PSliceHeaderIn struct {
		H []string `header:"X-List"`
	}
	PSlicePathIn struct {
		P []string `path:"p"`
	}
)

func init() {
	register(
		Case{
			// "Parameter Objects MUST include either a content field or a
			// schema field, but not both": an option's Content must replace
			// the inferred Schema.
			Name: "params/option-content-replaces-inferred-schema",
			Register: func(f *fizz.Fizz) {
				f.Handle("/x", "GET", []fizz.OperationOption{
					fizz.ID("op"),
					fizz.Parameter(&openapi.Parameter{
						Name: "q", In: "query",
						Content: map[string]*openapi.MediaType{"application/json": {}},
					}),
				}, tonic.Handler(pHandler[PSemQueryIn], 200))
			},
			Expect: []Fragment{
				{Pointer: "/paths/~1x/get/parameters/0/schema", Absent: true},
				{Pointer: "/paths/~1x/get/parameters/0/content/application~1json", JSON: `{}`},
			},
		},
		Case{
			// Parameter Object, in: only query, querystring, header, path
			// and cookie exist. "body" is an operation error and the
			// parameter is dropped.
			Name: "params/option-invalid-in-is-error",
			Register: func(f *fizz.Fizz) {
				f.Handle("/x", "GET", []fizz.OperationOption{
					fizz.ID("op"),
					fizz.Parameter(&openapi.Parameter{
						Name: "b", In: "body",
						Schema: &openapi.Schema{Type: openapi.Types("string")},
					}),
				}, tonic.Handler(pHandler[struct{}], 200))
			},
			Errors: []string{"body"},
			Expect: []Fragment{
				{Pointer: "/paths/~1x/get/parameters", Absent: true},
			},
		},
		Case{
			// Paths Object: templated paths that differ only in variable
			// names MUST NOT both exist. gin accepts both routes (the
			// methods differ), so the second is an operation error and
			// is not documented under a new key.
			Name: "paths/equivalent-templates-is-error",
			Register: func(f *fizz.Fizz) {
				f.Handle("/x/:a", "GET", []fizz.OperationOption{fizz.ID("getX")}, tonic.Handler(pHandler[struct{}], 200))
				f.Handle("/x/:c", "POST", []fizz.OperationOption{fizz.ID("postX")}, tonic.Handler(pHandler[struct{}], 200))
			},
			Errors: []string{"POST /x/{c}", "GET /x/{a}"},
			Expect: []Fragment{
				{Pointer: "/paths/~1x~1{a}/get/operationId", JSON: `"getX"`},
				{Pointer: "/paths/~1x~1{c}", Absent: true},
			},
		},
		Case{
			// Parameter Object, name: a header parameter named
			// Authorization SHALL be ignored, so fizz warns (security
			// schemes are the way) and does not emit it.
			Name:     "params/header-authorization-reserved",
			Register: pRegister("GET", "/x", tonic.Handler(pHandler[PSemAuthIn], 200)),
			Warnings: []string{"Authorization"},
			Expect: []Fragment{
				{Pointer: "/paths/~1x/get/parameters", Absent: true},
			},
		},
		Case{
			// tonic binds a header from one string and does not split it
			// (D29): a slice header is an operation error and is not
			// documented.
			Name:     "params/header-slice-is-error",
			Register: pRegister("GET", "/x", tonic.Handler(pHandler[PSliceHeaderIn], 200)),
			Errors:   []string{"tonic binds header/path parameters from a single value; use a query parameter or a string field"},
			Expect: []Fragment{
				{Pointer: "/paths/~1x/get/parameters", Absent: true},
			},
		},
		Case{
			// The same for a path parameter (D29): the parameter is dropped
			// and, as the path declares it, synthesized as a string.
			Name:     "params/path-slice-is-error",
			Register: pRegister("GET", "/x/:p", tonic.Handler(pHandler[PSlicePathIn], 200)),
			Errors:   []string{"tonic binds header/path parameters from a single value; use a query parameter or a string field"},
			Warnings: []string{"synthesized as a required string"},
			Expect: []Fragment{
				{Pointer: "/paths/~1x~1{p}/get/parameters/0/schema/type", JSON: `"string"`},
			},
		},
	)
}
