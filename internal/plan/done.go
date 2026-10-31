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
	"P1.0":   true,
	"P1.1":   true,
	"P1.2":   true,
	"P1.3":   true,
	"P2.1a":  true,
	"P2.1b":  true,
	"P2.1c":  true,
	"P2.1d":  true,
	"P2.2":   true,
	"P2.3":   true,
	"P2.5":   true,
	"P2.6":   true,
	"P3.1a":  true,
	"P3.1b":  true,
	"P3.2":   true,
	"P3.3":   true,
	"P3.4":   true,
	"P3.5":   true,
	"P3.6":   true,
	"P4.1a":  true,
	"P4.1b":  true,
	"P4.1c":  true,
	"P4.2":   true,
	"P4.2c":  true,
	"P4.3":   true,
	"P4.4":   true,
	"P4.5":   true,
	"P4.6":   true,
	"P4.7":   true,
	"P4.8":   true,
	"P4.9":   true,
	"P5.1":   true,
	"P6.1":   true,
	"P6.2":   true,
	"P6.3":   true,
	"P6.4":   true,
	"P6.5":   true,
	"P6.6a":  true,
	"P6.6b":  true,
	"P6.6c":  true,
	"P7.1":   true,
	"P7.2a":  true,
	"P7.2b":  true,
	"P7.2c":  true,
	"P8.1":   true,
	"P9.3":   true,
	"P8.2":   true,
	"P8.3":   true,
	"P8.4":   true,
	"P9.1":   true,
	"P9.2":   true,
	"P9.4":   true,
	"P10.0":  true,
	"P10.1":  true,
	"P10.2":  true,
	"P10.2b": true,
	"P10.3":  true,
	"P10.3a": true,
	"P10.3b": true,
	"P10.3c": true,
	"P10.3d": true,
	"P10.3e": true,
	"P10.3f": true,
	"P10.4":  true,
	"P10.5a": true,
	"P10.5b": true,
	"P10.5c": true,
	"P10.6":  true,
	"P10.7j": true,
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
