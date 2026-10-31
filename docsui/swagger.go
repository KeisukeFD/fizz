package docsui

import "net/http"

// Pinned Swagger UI assets (docs/compat.md).
const (
	// SwaggerUIJS is the pinned Swagger UI bundle.
	SwaggerUIJS = "https://cdn.jsdelivr.net/npm/swagger-ui-dist@5.34.1/swagger-ui-bundle.js"
	// SwaggerUICSS is the pinned Swagger UI style sheet.
	SwaggerUICSS = "https://cdn.jsdelivr.net/npm/swagger-ui-dist@5.34.1/swagger-ui.css"
	// SwaggerUIJSIntegrity is the sha384 SRI value of SwaggerUIJS.
	SwaggerUIJSIntegrity = "sha384-XFYN47VzPiDY6qPtxYKg0nqUeW/upo20xf1j2SUE7gT/i35FJB4cuS/71Iegzk8Z"
	// SwaggerUICSSIntegrity is the sha384 SRI value of SwaggerUICSS.
	SwaggerUICSSIntegrity = "sha384-4c3vsE5f+trhFyHrFrLM1fFGUn531NIaC7dcqEWIJdoRQJ5CkJbSuSCwXLpaS1KF"
)

// SwaggerUI returns a handler that serves a Swagger UI page showing the
// OpenAPI document at specURL. The assets are loaded from jsDelivr with
// Subresource Integrity (see the SwaggerUIJS constants).
//
// The page sets validatorUrl to null whatever WithUIConfig says.
// It needs no router: mount it on any http.ServeMux.
//
// Swagger UI renders OpenAPI 3.2 only partly: itemSchema is not shown
// (swagger-ui#11065) and $ref siblings or tag kind and parent may be
// ignored. Use ScalarUI for fuller 3.2 rendering.
func SwaggerUI(specURL string, opts ...Option) http.Handler {
	c := newConfig(opts)
	m := uiConfig(c, specURL)
	m["dom_id"] = "#ui"
	m["validatorUrl"] = nil
	return render("swagger", c, m, SwaggerUIJS, SwaggerUICSS, SwaggerUIJSIntegrity, SwaggerUICSSIntegrity)
}
