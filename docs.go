package fizz

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/wI2L/fizz/docsui"
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
	opts     []docsui.Option
}

// SwaggerUI selects Swagger UI as the renderer of Docs.
//
// Swagger UI renders OpenAPI 3.2 only partly. Known gaps: itemSchema
// is not rendered (swagger-ui#11065), and $ref siblings such as a
// description next to a $ref, and tag kind and parent, may be
// ignored. Use ScalarUI for fuller 3.2 rendering. The choice is
// always honored: fizz never falls back to another renderer.
func SwaggerUI(opts ...docsui.Option) DocsUI {
	return DocsUI{renderer: rendererSwagger, opts: opts}
}

// ScalarUI selects Scalar as the renderer of Docs.
func ScalarUI(opts ...docsui.Option) DocsUI {
	return DocsUI{renderer: rendererScalar, opts: opts}
}

// handler returns the net/http handler of u for specURL.
func (u DocsUI) handler(specURL string) http.Handler {
	if u.renderer == rendererScalar {
		return docsui.ScalarUI(specURL, u.opts...)
	}
	return docsui.SwaggerUI(specURL, u.opts...)
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
	f.engine.GET(path, gin.WrapH(u.handler(specURL)))
}
