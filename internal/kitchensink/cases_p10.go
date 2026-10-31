package kitchensink

import (
	"github.com/loopfz/gadgeto/tonic"

	"github.com/wI2L/fizz"
	"github.com/wI2L/fizz/openapi"
)

// Review findings that the semantic lint (specvalidate.LintSemantics, part
// of MetaProblems) catches in fizz output. Each is red until its P10.7 task.
type (
	PSemQueryIn struct {
		Q string `query:"q"`
	}
	PSemAuthIn struct {
		Auth string `header:"Authorization"`
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
			FixedBy: "P10.7a",
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
			FixedBy: "P10.7a",
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
			FixedBy: "P10.7f",
		},
	)
}
