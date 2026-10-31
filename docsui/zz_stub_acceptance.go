//go:build acceptance

// Stub declarations for symbols whose tasks are still open (ADR 0001 §5, §6).
// Each declaration is deleted by the task named in its FixedBy line.

package docsui

import (
	"net/http"
)

// Option is a stub type.
// FixedBy: P8.1
type Option func(*config)

// config is a stub type.
// FixedBy: P8.1
type config struct{}

// WithTitle is a stub.
// FixedBy: P8.1
func WithTitle(string) Option {
	return func(*config) {}
}

// WithUIConfig is a stub.
// FixedBy: P8.1
func WithUIConfig(map[string]any) Option {
	return func(*config) {}
}

// WithCSPNonce is a stub.
// FixedBy: P8.1
func WithCSPNonce(string) Option {
	return func(*config) {}
}

// WithAssetURLs is a stub.
// FixedBy: P8.1
func WithAssetURLs(js, css string, integrity ...string) Option {
	return func(*config) {}
}

// SwaggerUI is a stub.
// FixedBy: P8.1
func SwaggerUI(specURL string, opts ...Option) http.Handler {
	return http.NotFoundHandler()
}

// ScalarUI is a stub.
// FixedBy: P8.1
func ScalarUI(specURL string, opts ...Option) http.Handler {
	return http.NotFoundHandler()
}

// SwaggerUIJS is a stub.
// FixedBy: P8.1
const SwaggerUIJS = ""

// SwaggerUICSS is a stub.
// FixedBy: P8.1
const SwaggerUICSS = ""

// SwaggerUIJSIntegrity is a stub.
// FixedBy: P8.1
const SwaggerUIJSIntegrity = ""

// SwaggerUICSSIntegrity is a stub.
// FixedBy: P8.1
const SwaggerUICSSIntegrity = ""

// ScalarJS is a stub.
// FixedBy: P8.1
const ScalarJS = ""

// ScalarJSIntegrity is a stub.
// FixedBy: P8.1
const ScalarJSIntegrity = ""
