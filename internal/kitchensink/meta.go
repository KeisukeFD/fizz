package kitchensink

import (
	"strings"

	"github.com/wI2L/fizz/internal/specvalidate"
)

// MetaProblems checks doc against the OAS 3.2 meta-schema and the three
// lints (stray null, nullable, component names). It returns one entry
// per failing check, empty when the document is clean.
//
// Run applies it to every otherwise-passing case once plan.Done["P2.6"].
func MetaProblems(doc []byte) []string {
	var problems []string
	for _, check := range []struct {
		name string
		fn   func([]byte) error
	}{
		{"validate", specvalidate.Validate},
		{"stray-null", specvalidate.LintNoStrayNull},
		{"nullable", specvalidate.LintNoNullable},
		{"component-names", specvalidate.LintComponentNames},
	} {
		if err := check.fn(doc); err != nil {
			problems = append(problems, check.name+": "+strings.ReplaceAll(err.Error(), "\n", "\n  "))
		}
	}
	return problems
}
