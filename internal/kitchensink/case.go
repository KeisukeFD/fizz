// Package kitchensink is a catalogue of small registration cases, each
// carrying hand-written expectations of the OAS 3.2 target output.
//
// Cases for bugs that are not fixed yet set FixedBy; the runner skips
// them (after registering, so crashes stay visible) and reports a skip
// count. Domain files (cases_types.go, cases_fields.go, ...) add cases
// from an init function through register.
package kitchensink

import (
	"fmt"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/wI2L/fizz"
	"github.com/wI2L/fizz/internal/golden"
	"github.com/wI2L/fizz/internal/plan"
)

// Fragment is a hand-written expectation on the generated document.
type Fragment struct {
	// Pointer is an RFC 6901 JSON pointer; "" designates the whole document.
	Pointer string
	// JSON is the expected value at Pointer, compared semantically.
	// Ignored when Absent is set.
	JSON string
	// Absent asserts that Pointer does not resolve to anything.
	Absent bool
}

// Case is one kitchen-sink scenario.
type Case struct {
	Name string
	// New, when set, builds the *fizz.Fizz the case registers on
	// (to pass options). The default is fizz.New().
	New func() *fizz.Fizz
	// Register declares routes on a fresh *fizz.Fizz.
	Register func(f *fizz.Fizz)
	// Expect lists fragments of the OAS 3.2 target output.
	Expect []Fragment
	// Errors lists substrings that f.Errors() must contain (one error
	// each may match). Use for bugs recorded rather than raised.
	Errors []string
	// Warnings lists substrings that must each be contained in an
	// f.Errors() entry that implements interface{ IsWarning() bool }
	// and reports true. Other entries never match.
	Warnings []string
	// Panics, when non-empty, requires Register to panic with a message
	// containing it. By default a panic is a failure.
	Panics string
	// FixedBy names the plan task that fixes the case (for example
	// "P3.1a"). Such cases are skipped once registered.
	FixedBy string
	// Crashes marks cases that cannot be recovered (stack overflow).
	// They are skipped before Register runs; FixedBy must be set too.
	Crashes bool
}

// cases is the registry run by TestKitchenSink.
var cases []Case

// register adds cases to the registry. Call it from init in a
// cases_*.go file so no shared code needs editing.
func register(cs ...Case) {
	cases = append(cases, cs...)
}

// newFizz returns the *fizz.Fizz a case registers on.
func (c Case) newFizz() *fizz.Fizz {
	if c.New != nil {
		return c.New()
	}
	return fizz.New()
}

// caseGoldenDir holds one golden document per passing case.
const caseGoldenDir = "testdata/cases"

// caseGoldenPath returns the golden file of the case named name: the
// name with "/" replaced by "__".
func caseGoldenPath(name string) string {
	return filepath.Join(caseGoldenDir, strings.ReplaceAll(name, "/", "__")+".json")
}

// assertCaseGolden compares the whole document of a passing case with
// its golden file. Cases that expect a panic produce no document and
// have no golden; skipped cases have none either.
func assertCaseGolden(t *testing.T, c Case, doc []byte) {
	t.Helper()
	if doc == nil {
		return
	}
	golden.Assert(t, caseGoldenPath(c.Name), doc)
}

// Run executes cs, one subtest each, on a fresh fizz.New() per case
// (or c.New()).
//
// Every passing case also asserts its whole document against
// testdata/cases/<name>.json ("/" in the name becomes "__"); run the
// tests with -update to create or refresh these goldens.
//
// A panic in Register is a failure unless the case sets Panics. Cases
// with FixedBy still register and are checked, but are then skipped
// with t.Skip("fixed by ..."), so a crash or a now-passing case is
// visible in the skip message. Cases with Crashes are skipped before
// Register runs. A skip summary is logged on t (visible with -v).
func Run(t *testing.T, cs []Case) {
	t.Helper()
	var passed, skipped, stale int
	byTask := map[string]int{}

	for _, c := range cs {
		t.Run(c.Name, func(t *testing.T) {
			if c.Crashes {
				if c.FixedBy == "" {
					t.Fatal("kitchensink: Crashes requires FixedBy")
				}
				if plan.IsDone(c.FixedBy) {
					t.Fatal("task " + c.FixedBy + " is done, case still marked Crashes")
				}
				skipped++
				byTask[c.FixedBy]++
				t.Skip("fixed by " + c.FixedBy + " (crashes, not run)")
			}
			doc, problems := checkCase(c)
			if c.FixedBy == "" {
				for _, p := range problems {
					t.Error(p)
				}
				if len(problems) == 0 {
					passed++
					assertCaseGolden(t, c, doc)
				}
				return
			}
			if len(problems) > 0 && plan.IsDone(c.FixedBy) {
				t.Error("task " + c.FixedBy + " is done, case red:\n" + strings.Join(problems, "\n"))
				return
			}
			skipped++
			byTask[c.FixedBy]++
			reason := "fixed by " + c.FixedBy
			if len(problems) == 0 {
				if plan.IsDone(c.FixedBy) {
					skipped--
					byTask[c.FixedBy]--
					passed++
					assertCaseGolden(t, c, doc)
					return
				}
				stale++
				if plan.StaleFail() {
					t.Error("fixed by " + c.FixedBy + " (case passes now: drop FixedBy)")
					return
				}
				reason += " (case passes now: drop FixedBy)"
			} else {
				reason += ": " + problems[0]
			}
			t.Skip(reason)
		})
	}

	tasks := make([]string, 0, len(byTask))
	for task, n := range byTask {
		tasks = append(tasks, fmt.Sprintf("%s=%d", task, n))
	}
	slices.Sort(tasks)
	t.Logf("kitchensink: %d cases, %d passed, %d skipped [%s], %d skipped but passing",
		len(cs), passed, skipped, strings.Join(tasks, " "), stale)
}

// runCase registers and checks one case, returning its problems.
func runCase(c Case) []string {
	_, problems := checkCase(c)
	return problems
}

// checkCase registers and checks one case. It returns the generated
// document (nil when the case expects a panic) and the problems found.
func checkCase(c Case) (doc []byte, problems []string) {
	f := c.newFizz()
	panicMsg, panicked := safeRegister(c, f)
	switch {
	case c.Panics != "":
		if !panicked {
			return nil, []string{"Register must panic with " + strconv.Quote(c.Panics)}
		}
		if !strings.Contains(panicMsg, c.Panics) {
			return nil, []string{fmt.Sprintf("panic %q does not contain %q", panicMsg, c.Panics)}
		}
		return nil, nil
	case panicked:
		return nil, []string{"Register panicked: " + panicMsg}
	}
	rec := &recorder{}
	checkErrors(rec, f.Errors(), c.Errors)
	checkWarnings(rec, f.Errors(), c.Warnings)
	doc, err := BuildSpec(f)
	if err != nil {
		return nil, append(rec.msgs, "build spec: "+err.Error())
	}
	AssertFragments(rec, doc, c.Expect)
	if plan.IsDone("P2.6") {
		rec.msgs = append(rec.msgs, MetaProblems(doc)...)
	}
	return doc, rec.msgs
}

func safeRegister(c Case, f *fizz.Fizz) (msg string, panicked bool) {
	defer func() {
		if r := recover(); r != nil {
			msg, panicked = fmt.Sprint(r), true
		}
	}()
	c.Register(f)
	return "", false
}
