// Package plan is the done ledger of docs/plan/PLAN-v2.md and the
// red-until-done test harness built on it.
//
// Every task PR adds its own ID to Done. From then on a test whose
// FixedBy names that task must pass, instead of being skipped as red.
package plan

import (
	"os"
	"strings"
)

// Done lists the plan tasks that are merged ("P2.6", "P4.1b", ...).
// A task PR adds its own ID here and nothing else; the harness then
// fails every skipped test that names the task in FixedBy.
var Done = map[string]bool{
	"P1.0":  true,
	"P1.1":  true,
	"P1.2":  true,
	"P1.3":  true,
	"P2.1a": true,
	"P2.1b": true,
	"P2.1c": true,
	"P2.1d": true,
	"P2.2":  true,
	"P2.3":  true,
	"P2.5":  true,
	"P2.6":  true,
	"P3.1a": true,
	"P3.1b": true,
	"P3.2":  true,
	"P3.3":  true,
}

// IsDone reports whether task is in the done ledger. A composite ID
// such as "P3.3+P4.8" is done only when every "+"-joined part is.
func IsDone(task string) bool {
	for part := range strings.SplitSeq(task, "+") {
		if !Done[part] {
			return false
		}
	}
	return true
}

// StaleFail reports whether stale skips (a test with FixedBy that
// already passes) are failures. CI sets FIZZ_STALE=fail.
func StaleFail() bool { return os.Getenv("FIZZ_STALE") == "fail" }
