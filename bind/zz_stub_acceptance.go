//go:build acceptance

// Stub declarations for symbols whose tasks are still open (ADR 0001 §5, §6).
// Each declaration is deleted by the task named in its FixedBy line.

package bind

import (
	"errors"
	"net/http"
)

// Cookies is a stub.
// FixedBy: P7.2c
func Cookies(r *http.Request, dst any, p *Plan) error {
	return errors.New("stub: P7.2c")
}
