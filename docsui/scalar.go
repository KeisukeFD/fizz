package docsui

import "net/http"

// Pinned Scalar API Reference asset (docs/compat.md).
const (
	// ScalarJS is the pinned Scalar standalone bundle.
	ScalarJS = "https://cdn.jsdelivr.net/npm/@scalar/api-reference@1.76.4/dist/browser/standalone.js"
	// ScalarJSIntegrity is the sha384 SRI value of ScalarJS.
	ScalarJSIntegrity = "sha384-hwB9WroDTpZaI38gr+u9CXA6ksz5gUYgD2GxXazKl/agT1L2GWUiRFWGJYNiFu4Q"
)

// ScalarUI returns a handler that serves a Scalar API Reference page
// showing the OpenAPI document at specURL. The asset is loaded from
// jsDelivr with Subresource Integrity (see ScalarJS). The page calls
// Scalar.createApiReference with WithUIConfig merged into its
// configuration. It needs no router.
func ScalarUI(specURL string, opts ...Option) http.Handler {
	c := newConfig(opts)
	return render("scalar", c, uiConfig(c, specURL), ScalarJS, "", ScalarJSIntegrity, "")
}
