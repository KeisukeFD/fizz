//go:build acceptance

// Stub declarations for symbols whose tasks are still open (ADR 0001 §5, §6).
// Each declaration is deleted by the task named in its FixedBy line.

package fizz

import (
	"github.com/loopfz/gadgeto/tonic"
)

// BindHook is a stub.
// FixedBy: P7.2b
func BindHook(next tonic.BindHook) tonic.BindHook {
	return nil
}

// InstallBindHook is a stub.
// FixedBy: P7.2b
func InstallBindHook() {
}
