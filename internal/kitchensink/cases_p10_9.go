package kitchensink

import (
	"github.com/loopfz/gadgeto/tonic"
)

// Cases found by the reproduction apps (PLAN-v2 P10.9x). Every
// identifier starts with R so sibling cases_*.go files cannot clash.

type RDescIn struct {
	Limit int32 `query:"limit" description:"how many"`
}

func init() {
	register(
		Case{
			// The description tag documents the parameter. It is also
			// copied into the schema, where the official documents
			// (and the OAS) do not have it.
			Name:     "params/description-tag-not-in-schema",
			Register: pRegister("GET", "/x", tonic.Handler(pHandler[RDescIn], 200)),
			Expect: []Fragment{
				{Pointer: "/paths/~1x/get/parameters/0/description", JSON: `"how many"`},
				{Pointer: "/paths/~1x/get/parameters/0/schema/description", Absent: true},
			},
		},
	)
}
