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
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/wI2L/fizz"
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
	// Register declares routes on a fresh *fizz.Fizz.
	Register func(f *fizz.Fizz)
	// Expect lists fragments of the OAS 3.2 target output.
	Expect []Fragment
	// Errors lists substrings that f.Errors() must contain (one error
	// each may match). Use for bugs recorded rather than raised.
	Errors []string
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

// Run executes cs, one subtest each, on a fresh fizz.New() per case.
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
				skipped++
				byTask[c.FixedBy]++
				t.Skip("fixed by " + c.FixedBy + " (crashes, not run)")
			}
			problems := runCase(c)
			if c.FixedBy == "" {
				for _, p := range problems {
					t.Error(p)
				}
				if len(problems) == 0 {
					passed++
				}
				return
			}
			skipped++
			byTask[c.FixedBy]++
			reason := "fixed by " + c.FixedBy
			if len(problems) == 0 {
				stale++
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
func runCase(c Case) (problems []string) {
	f := fizz.New()
	panicMsg, panicked := safeRegister(c, f)
	switch {
	case c.Panics != "":
		if !panicked {
			return []string{"Register must panic with " + strconv.Quote(c.Panics)}
		}
		if !strings.Contains(panicMsg, c.Panics) {
			return []string{fmt.Sprintf("panic %q does not contain %q", panicMsg, c.Panics)}
		}
		return nil
	case panicked:
		return []string{"Register panicked: " + panicMsg}
	}
	rec := &recorder{}
	checkErrors(rec, f.Errors(), c.Errors)
	doc, err := buildSpec(f)
	if err != nil {
		return append(rec.msgs, "build spec: "+err.Error())
	}
	checkFragments(rec, doc, c.Expect)
	return rec.msgs
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
