package fizz

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/KeisukeFD/fizz/docsui"
	"github.com/KeisukeFD/fizz/docsui/embedded"
)

type docsRenderer int

const (
	rendererSwagger docsRenderer = iota
	rendererScalar
)

func (r docsRenderer) String() string {
	if r == rendererScalar {
		return "Scalar"
	}
	return "Swagger UI"
}

// DocsUI selects the renderer of Docs and its options. Build it with
// SwaggerUI or ScalarUI. The zero value is Swagger UI.
type DocsUI struct {
	renderer docsRenderer
	// build returns the handler for specURL mounted at path; nil means
	// the CDN Swagger UI. Only the closures of the Embedded selectors
	// name package embedded, so the linker drops its assets from
	// programs that do not call them.
	build  func(path, specURL string) http.Handler
	assets bool // Docs also serves <path>/assets/*
}

// SwaggerUI selects Swagger UI as the renderer of Docs.
//
// Swagger UI renders OpenAPI 3.2 only partly. Known gaps: itemSchema
// is not rendered (swagger-ui#11065) and the tag summary is not shown;
// tag kind and parent are unverified (see docs/compat.md). Use ScalarUI
// for fuller 3.2 rendering. The choice is always honored: fizz never
// falls back to another renderer.
func SwaggerUI(opts ...docsui.Option) DocsUI {
	return DocsUI{renderer: rendererSwagger, build: func(_, specURL string) http.Handler {
		return docsui.SwaggerUI(specURL, opts...)
	}}
}

// ScalarUI selects Scalar as the renderer of Docs.
func ScalarUI(opts ...docsui.Option) DocsUI {
	return DocsUI{renderer: rendererScalar, build: func(_, specURL string) http.Handler {
		return docsui.ScalarUI(specURL, opts...)
	}}
}

// EmbeddedSwaggerUI selects Swagger UI as the renderer of Docs, with
// its assets served from the binary at <path>/assets/ instead of a
// CDN. The page makes no request to a third-party host.
//
// Only a program that calls EmbeddedSwaggerUI or EmbeddedScalarUI
// carries the assets: the linker drops them otherwise. The Swagger UI
// 3.2 gaps listed at SwaggerUI apply.
func EmbeddedSwaggerUI(opts ...docsui.Option) DocsUI {
	return DocsUI{renderer: rendererSwagger, assets: true, build: func(path, specURL string) http.Handler {
		return embedded.SwaggerUI(specURL, path, opts...)
	}}
}

// EmbeddedScalarUI selects Scalar as the renderer of Docs, with its
// script served from the binary at <path>/assets/ instead of a CDN.
// The page makes no request to a third-party host. See
// EmbeddedSwaggerUI for the binary size.
func EmbeddedScalarUI(opts ...docsui.Option) DocsUI {
	return DocsUI{renderer: rendererScalar, assets: true, build: func(path, specURL string) http.Handler {
		return embedded.ScalarUI(specURL, path, opts...)
	}}
}

// handler returns the net/http handler of u for specURL, mounted at path.
func (u DocsUI) handler(path, specURL string) http.Handler {
	if u.build == nil {
		return docsui.SwaggerUI(specURL)
	}
	return u.build(path, specURL)
}

// Docs registers a GET route at path that serves an interactive UI
// for the OpenAPI document served at specURL. The route is not part
// of the specification.
//
// Without ui the renderer is Swagger UI, the default. Pass SwaggerUI
// or ScalarUI to choose one and set its options; the choice is always
// honored. More than one ui is a mistake: it is recorded in Errors and
// the first one is used.
func (f *Fizz) Docs(path, specURL string, ui ...DocsUI) {
	var u DocsUI
	if len(ui) > 0 {
		u = ui[0]
	}
	if len(ui) > 1 {
		f.gen.RecordError(fmt.Errorf("docs %s: %d renderers given (%s and %s): using %s",
			path, len(ui), ui[0].renderer, ui[1].renderer, ui[0].renderer))
	}
	h := gin.WrapH(u.handler(path, specURL))
	f.engine.GET(path, h)
	if u.assets {
		f.engine.GET(strings.TrimRight(path, "/")+"/assets/*filepath", h)
	}
}
