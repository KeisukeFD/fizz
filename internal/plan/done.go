// Package plan is the done ledger of docs/plan/PLAN-v2.md and the
// red-until-done test harness built on it.
//
// Every task PR adds its own ID to Done. From then on a test whose
// FixedBy names that task must pass, instead of being skipped as red.
package plan

import "os"

// Done lists the plan tasks that are merged ("P2.6", "P4.1b", ...).
// A task PR adds its own ID here and nothing else; the harness then
// fails every skipped test that names the task in FixedBy.
var Done = map[string]bool{}

// IsDone reports whether task is in the done ledger.
func IsDone(task string) bool { return Done[task] }

// StaleFail reports whether stale skips (a test with FixedBy that
// already passes) are failures. CI sets FIZZ_STALE=fail.
func StaleFail() bool { return os.Getenv("FIZZ_STALE") == "fail" }
