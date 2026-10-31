//go:build acceptance

// Stub declarations for symbols whose tasks are still open (ADR 0001 §5, §6).
// Each declaration is deleted by the task named in its FixedBy line.

package embedded

import (
	"github.com/wI2L/fizz/docsui"
	"net/http"
)

// SwaggerUI is a stub.
// FixedBy: P8.3
func SwaggerUI(specURL, prefix string, opts ...docsui.Option) http.Handler {
	return http.NotFoundHandler()
}

// ScalarUI is a stub.
// FixedBy: P8.3
func ScalarUI(specURL, prefix string, opts ...docsui.Option) http.Handler {
	return http.NotFoundHandler()
}
