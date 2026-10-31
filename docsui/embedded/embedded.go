package embedded

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"

	"github.com/wI2L/fizz/docsui"
)

//go:embed assets
var assetsFS embed.FS

// assets is the content of the assets directory.
func assetsDir() fs.FS {
	sub, err := fs.Sub(assetsFS, "assets")
	if err != nil {
		panic(err)
	}
	return sub
}

// SwaggerUI returns a handler that serves a Swagger UI page for the
// OpenAPI document at specURL, and the assets it needs, with no request
// to a third-party host. The page is served at prefix and the assets at
// prefix/assets/, so mount the handler on both (for a net/http.ServeMux,
// on prefix and prefix+"/"). prefix is the URL path where the handler is
// mounted, such as "/docs".
//
// The assets are the files pinned by the docsui URL constants; the page
// carries the same Subresource Integrity values. A program that imports
// this package carries the assets only if it calls SwaggerUI or
// ScalarUI: the linker drops them otherwise.
func SwaggerUI(specURL, prefix string, opts ...docsui.Option) http.Handler {
	prefix = strings.TrimRight(prefix, "/")
	opts = append(opts[:len(opts):len(opts)],
		docsui.WithAssetURLs(
			prefix+"/assets/swagger-ui-bundle.js",
			prefix+"/assets/swagger-ui.css",
			docsui.SwaggerUIJSIntegrity, docsui.SwaggerUICSSIntegrity))
	return handler(prefix, docsui.SwaggerUI(specURL, opts...))
}

// ScalarUI returns a handler that serves a Scalar API Reference page
// for the OpenAPI document at specURL, and its script, with no request
// to a third-party host. The default fonts of Scalar, which it loads
// from a CDN, are disabled (withDefaultFonts is false whatever
// WithUIConfig says). See SwaggerUI for prefix and the mounting rules.
func ScalarUI(specURL, prefix string, opts ...docsui.Option) http.Handler {
	prefix = strings.TrimRight(prefix, "/")
	opts = append(opts[:len(opts):len(opts)],
		docsui.WithUIConfig(map[string]any{"withDefaultFonts": false}),
		docsui.WithAssetURLs(prefix+"/assets/standalone.js", "", docsui.ScalarJSIntegrity))
	return handler(prefix, docsui.ScalarUI(specURL, opts...))
}

// handler serves page at prefix and the embedded assets under
// prefix/assets/. Every other path is a 404.
func handler(prefix string, page http.Handler) http.Handler {
	files := http.FileServerFS(assetsDir())
	dir := prefix + "/assets/"
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		switch {
		case p == prefix || p == prefix+"/":
			page.ServeHTTP(w, r)
		case strings.HasPrefix(p, dir):
			name := p[len(dir):]
			if name == "" || strings.Contains(name, "/") || name == "." || name == ".." {
				http.NotFound(w, r)
				return
			}
			if r.Method != http.MethodGet && r.Method != http.MethodHead {
				w.Header().Set("Allow", "GET, HEAD")
				http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
				return
			}
			switch path.Ext(name) {
			case ".js":
				w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
			case ".css":
				w.Header().Set("Content-Type", "text/css; charset=utf-8")
			}
			r2 := r.Clone(r.Context())
			r2.URL.Path = "/" + name
			files.ServeHTTP(w, r2)
		default:
			http.NotFound(w, r)
		}
	})
}
