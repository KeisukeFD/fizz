package plan

import (
	"fmt"
	"strings"
	"sync"
	"testing"
)

// abort is the sentinel panicked by the recorder's Fatal/FailNow/Skip
// methods to unwind fn, the way runtime.Goexit does for a real T.
type abort struct{ skipped bool }

// recorder is the testing.TB handed to a Pending fn. Failures are
// collected instead of reported, so Pending decides what they mean.
// Everything else (Cleanup, TempDir, Setenv, Context, ...) delegates to
// the embedded real TB.
type recorder struct {
	testing.TB
	mu      sync.Mutex
	msgs    []string
	skipMsg string
}

func (r *recorder) record(msg string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.msgs = append(r.msgs, msg)
}

func (r *recorder) Helper()                   {}
func (r *recorder) Error(args ...any)         { r.record(fmt.Sprint(args...)) }
func (r *recorder) Errorf(f string, a ...any) { r.record(fmt.Sprintf(f, a...)) }
func (r *recorder) Fail()                     { r.record("Fail called") }
func (r *recorder) Failed() bool              { r.mu.Lock(); defer r.mu.Unlock(); return len(r.msgs) > 0 }
func (r *recorder) Fatal(args ...any)         { r.Error(args...); panic(abort{}) }
func (r *recorder) Fatalf(f string, a ...any) { r.Errorf(f, a...); panic(abort{}) }
func (r *recorder) FailNow()                  { r.Fail(); panic(abort{}) }
func (r *recorder) Skip(args ...any)          { r.skipMsg = fmt.Sprint(args...); panic(abort{skipped: true}) }
func (r *recorder) Skipf(f string, a ...any)  { r.Skip(fmt.Sprintf(f, a...)) }
func (r *recorder) SkipNow()                  { r.Skip("SkipNow called") }
func (r *recorder) Skipped() bool             { return false }

// run executes fn and returns what it recorded. A panic that is not
// the recorder's own sentinel is recorded as a failure, so a crashing
// test stays red instead of killing the process.
func (r *recorder) run(fn func(testing.TB)) {
	defer func() {
		switch p := recover().(type) {
		case nil:
		case abort:
		default:
			r.record(fmt.Sprintf("panic: %v", p))
		}
	}()
	fn(r)
}

// Pending runs fn, a test body that the task fixedBy has yet to make
// pass, and turns the outcome into t's result:
//
//   - fn fails, fixedBy not in Done: t.Skip("fixed by X: <first failure>").
//   - fn fails, fixedBy in Done: t.Error("task done, test red: ..."), plus every failure.
//   - fn passes, fixedBy not in Done: t.Skip("... (passes now: drop FixedBy)"),
//     or t.Error when FIZZ_STALE=fail.
//   - fn passes, fixedBy in Done: nothing; the test is green.
//
// fn receives a recorder in which Fatal and friends abort fn through a
// recovered panic; a panic raised by fn itself is recorded as a
// failure. fixedBy == "" makes the failures plain test failures.
//
// Pending cannot contain a data race: the detector fails the test
// regardless. Gate the racy part on IsDone instead.
func Pending(t testing.TB, fixedBy string, fn func(tb testing.TB)) {
	t.Helper()
	r := &recorder{TB: t}
	r.run(fn)

	if r.skipMsg != "" && len(r.msgs) == 0 {
		if fixedBy == "" {
			t.Skip(r.skipMsg)
		}
		t.Skip("fixed by " + fixedBy + ": fn skipped: " + r.skipMsg)
		return
	}
	failed := len(r.msgs) > 0
	switch {
	case fixedBy == "":
		for _, m := range r.msgs {
			t.Error(m)
		}
	case failed && IsDone(fixedBy):
		t.Error("task " + fixedBy + " is done, test red:\n" + strings.Join(r.msgs, "\n"))
	case failed:
		t.Skip("fixed by " + fixedBy + ": " + r.msgs[0])
	case IsDone(fixedBy):
		// green, as it must be
	case StaleFail():
		t.Error("fixed by " + fixedBy + " (passes now: drop FixedBy)")
	default:
		t.Skip("fixed by " + fixedBy + " (passes now: drop FixedBy)")
	}
}
