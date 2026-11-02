package at_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/KeisukeFD/fizz/internal/acceptance/at"
	"github.com/KeisukeFD/fizz/internal/plan"
)

// fakeT stands in for the *testing.T that Pending reports to: Skip and
// Error are recorded instead of ending or failing the real test.
type fakeT struct {
	testing.TB
	skipped string
	errs    []string
}

func (f *fakeT) Helper()                      {}
func (f *fakeT) Skip(args ...any)             { f.skipped = fmtArgs(args) }
func (f *fakeT) Error(args ...any)            { f.errs = append(f.errs, fmtArgs(args)) }
func (f *fakeT) Errorf(s string, args ...any) { f.errs = append(f.errs, sprintf(s, args)) }

func fmtArgs(a []any) string           { return strings.TrimSpace(fmt.Sprintln(a...)) }
func sprintf(s string, a []any) string { return fmt.Sprintf(s, a...) }

// markDone puts task in the ledger for the duration of the test.
func markDone(t *testing.T, task string) {
	t.Helper()
	plan.Done[task] = true
	t.Cleanup(func() { delete(plan.Done, task) })
}

func run(t *testing.T, fixedBy string, fn func(testing.TB)) *fakeT {
	t.Helper()
	f := &fakeT{TB: t}
	at.Pending(f, fixedBy, fn)
	return f
}

func TestPendingSkipsFailing(t *testing.T) {
	f := run(t, "PX.1", func(tb testing.TB) { tb.Fatalf("boom %d", 7) })
	if !strings.Contains(f.skipped, "fixed by PX.1") || !strings.Contains(f.skipped, "boom 7") {
		t.Errorf("skip = %q, want it to name the task and the first failure", f.skipped)
	}
	if len(f.errs) != 0 {
		t.Errorf("unexpected errors: %v", f.errs)
	}
}

func TestPendingStale(t *testing.T) {
	pass := func(testing.TB) {}

	t.Run("skip", func(t *testing.T) {
		t.Setenv("FIZZ_STALE", "")
		f := run(t, "PX.1", pass)
		if !strings.Contains(f.skipped, "(passes now") || len(f.errs) != 0 {
			t.Errorf("skipped=%q errs=%v, want a stale skip", f.skipped, f.errs)
		}
	})
	t.Run("fail", func(t *testing.T) {
		t.Setenv("FIZZ_STALE", "fail")
		f := run(t, "PX.1", pass)
		if len(f.errs) != 1 || !strings.Contains(f.errs[0], "(passes now") || f.skipped != "" {
			t.Errorf("skipped=%q errs=%v, want one stale error and no skip", f.skipped, f.errs)
		}
	})
	t.Run("done and green", func(t *testing.T) {
		t.Setenv("FIZZ_STALE", "fail")
		markDone(t, "PX.1")
		f := run(t, "PX.1", pass)
		if f.skipped != "" || len(f.errs) != 0 {
			t.Errorf("skipped=%q errs=%v, want green", f.skipped, f.errs)
		}
	})
}

func TestPendingDoneLedger(t *testing.T) {
	markDone(t, "PX.1")
	f := run(t, "PX.1", func(tb testing.TB) { tb.Errorf("first"); tb.Fatalf("second") })
	if f.skipped != "" {
		t.Errorf("a done task must not skip, got %q", f.skipped)
	}
	if len(f.errs) != 1 || !strings.Contains(f.errs[0], "PX.1 is done, test red") ||
		!strings.Contains(f.errs[0], "first") || !strings.Contains(f.errs[0], "second") {
		t.Errorf("errs = %q, want one error naming the task with every failure", f.errs)
	}
}

func TestPendingRecoversPanic(t *testing.T) {
	f := run(t, "PX.1", func(testing.TB) { panic("kaboom") })
	if !strings.Contains(f.skipped, "panic: kaboom") {
		t.Errorf("skip = %q, want the recovered panic as the failure", f.skipped)
	}

	markDone(t, "PX.1")
	f = run(t, "PX.1", func(testing.TB) { var m map[string]int; m["x"] = 1 })
	if len(f.errs) != 1 || !strings.Contains(f.errs[0], "panic:") {
		t.Errorf("errs = %q, want a recorded panic failure", f.errs)
	}
}
