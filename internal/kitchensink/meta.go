package kitchensink

import (
	"errors"
	"strings"

	"github.com/KeisukeFD/fizz/internal/specvalidate"
)

// MetaProblems checks doc against the OAS 3.2 meta-schema, the three
// lints (stray null, nullable, component names) and the semantic lints
// (specvalidate.LintSemantics). It returns one entry per failing check,
// empty when the document is clean.
//
// Run applies it to every case that registers without panicking.
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
		{"semantics", lintSemantics},
	} {
		if err := check.fn(doc); err != nil {
			problems = append(problems, check.name+": "+strings.ReplaceAll(err.Error(), "\n", "\n  "))
		}
	}
	return problems
}

// lintSemantics adapts specvalidate.LintSemantics to the error-returning
// check signature.
func lintSemantics(doc []byte) error {
	ps := specvalidate.LintSemantics(doc)
	if len(ps) == 0 {
		return nil
	}
	lines := make([]string, len(ps))
	for i, p := range ps {
		lines[i] = p.String()
	}
	return errors.New(strings.Join(lines, "\n"))
}
