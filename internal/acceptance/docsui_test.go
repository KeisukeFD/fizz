package acceptance

// Batch C (default build): the docs UI renderers and f.Docs. See
// docs/plan/ACCEPTANCE.md, Batch C.
//
// Every package-level identifier is prefixed bC, as sibling files
// share the package.

import (
	"encoding/json"
	"go/ast"
	"go/doc"
	"go/parser"
	"go/token"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/wI2L/fizz"
	"github.com/wI2L/fizz/docsui"
	"github.com/wI2L/fizz/internal/acceptance/at"
	"github.com/wI2L/fizz/internal/golden"
)

var bCSRIRe = regexp.MustCompile(`^sha384-[A-Za-z0-9+/]{64}$`)

// bCDocs mounts ui at /docs on a new fizz and returns the body served.
func bCDocs(tb testing.TB, ui ...fizz.DocsUI) (f *fizz.Fizz, body string) {
	tb.Helper()
	f = fizz.New()
	f.Docs("/docs", "/openapi.json", ui...)
	w := at.Do(f, http.MethodGet, "/docs")
	if w.Code != http.StatusOK {
		tb.Fatalf("GET /docs = %d, want 200", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
		tb.Fatalf("Content-Type = %q", ct)
	}
	return f, w.Body.String()
}

// bCServe returns the body of h for GET /.
func bCServe(tb testing.TB, h http.Handler) string {
	tb.Helper()
	w := at.Do(h, http.MethodGet, "/")
	if w.Code != http.StatusOK {
		tb.Fatalf("GET = %d, want 200", w.Code)
	}
	return w.Body.String()
}

// bCScripts returns the elements with a src attribute.
func bCScripts(body string) []at.Elem {
	var out []at.Elem
	for _, e := range at.Tags(at.HTML([]byte(body)), "script") {
		if _, ok := e.Attr["src"]; ok {
			out = append(out, e)
		}
	}
	return out
}

// bCConfig decodes the fizz-docs-config script of body.
func bCConfig(tb testing.TB, body string) map[string]any {
	tb.Helper()
	for _, e := range at.Tags(at.HTML([]byte(body)), "script") {
		if e.Attr["id"] != "fizz-docs-config" {
			continue
		}
		if e.Attr["type"] != "application/json" {
			tb.Fatalf("config script type = %q", e.Attr["type"])
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(e.Text), &m); err != nil {
			tb.Fatalf("config JSON: %v\n%s", err, e.Text)
		}
		return m
	}
	tb.Fatal("no script#fizz-docs-config")
	return nil
}

func bCNoSubstr(tb testing.TB, body string, subs ...string) {
	tb.Helper()
	low := strings.ToLower(body)
	for _, s := range subs {
		if strings.Contains(low, s) {
			tb.Errorf("body contains %q:\n%s", s, body)
		}
	}
}

// golden HTML of both renderers.
func TestBCGoldenHTML(t *testing.T) {
	for _, tc := range []struct {
		name string
		h    http.Handler
	}{
		{"swagger", docsui.SwaggerUI("/openapi.json", docsui.WithTitle("Pets"))},
		{"scalar", docsui.ScalarUI("/openapi.json", docsui.WithTitle("Pets"))},
	} {
		t.Run(tc.name, func(t *testing.T) {
			at.Pending(t, "P8.1", func(tb testing.TB) {
				golden.Assert(tb, filepath.Join("..", "..", "docsui", "testdata", tc.name+".html"), []byte(bCServe(tb, tc.h)))
			})
		})
	}
}

// C01. The default renderer is Swagger UI.
func TestBCDocsDefaultIsSwaggerUI(t *testing.T) {
	at.Pending(t, "P8.1", func(tb testing.TB) {
		_, body := bCDocs(tb)
		scripts := bCScripts(body)
		if len(scripts) != 1 || scripts[0].Attr["src"] != docsui.SwaggerUIJS {
			tb.Errorf("script src = %v, want only %s", scripts, docsui.SwaggerUIJS)
		}
		var css []string
		for _, e := range at.Tags(at.HTML([]byte(body)), "link") {
			if e.Attr["rel"] == "stylesheet" {
				css = append(css, e.Attr["href"])
			}
		}
		if len(css) != 1 || css[0] != docsui.SwaggerUICSS {
			tb.Errorf("stylesheets = %v, want only %s", css, docsui.SwaggerUICSS)
		}
		bCNoSubstr(tb, body, "scalar")
	})
}

// C02. An explicit SwaggerUI never serves Scalar.
func TestBCDocsExplicitSwaggerNeverScalar(t *testing.T) {
	at.Pending(t, "P8.1", func(tb testing.TB) {
		for name, ui := range map[string]fizz.DocsUI{
			"plain":  fizz.SwaggerUI(),
			"title":  fizz.SwaggerUI(docsui.WithTitle("T")),
			"config": fizz.SwaggerUI(docsui.WithUIConfig(map[string]any{"deepLinking": true})),
			"assets": fizz.SwaggerUI(docsui.WithAssetURLs("/a.js", "/a.css")),
		} {
			_, body := bCDocs(tb, ui)
			bCNoSubstr(tb, body, "scalar")
			if !strings.Contains(body, "SwaggerUIBundle") {
				tb.Errorf("%s: no SwaggerUIBundle call", name)
			}
		}
	})
}

// C03. An explicit ScalarUI never serves Swagger UI.
func TestBCDocsExplicitScalarNeverSwagger(t *testing.T) {
	at.Pending(t, "P8.1", func(tb testing.TB) {
		_, body := bCDocs(tb, fizz.ScalarUI())
		scripts := bCScripts(body)
		if len(scripts) != 1 || scripts[0].Attr["src"] != docsui.ScalarJS {
			tb.Errorf("script src = %v, want only %s", scripts, docsui.ScalarJS)
		}
		bCNoSubstr(tb, body, "swagger-ui")
	})
}

// C04. Two renderers are an error; the first is served.
func TestBCDocsRendererConflict(t *testing.T) {
	at.Pending(t, "P8.1", func(tb testing.TB) {
		f, body := bCDocs(tb, fizz.SwaggerUI(), fizz.ScalarUI())
		bCNoSubstr(tb, body, "scalar")
		var found bool
		for _, err := range f.Errors() {
			if strings.Contains(err.Error(), "Swagger UI") && strings.Contains(err.Error(), "Scalar") {
				found = true
			}
		}
		if !found {
			tb.Errorf("no error naming both renderers in %v", f.Errors())
		}
	})
}

// C05. Every external asset carries SRI, crossorigin and a referrer policy.
func TestBCSRIAttributesMatchConstants(t *testing.T) {
	at.Pending(t, "P8.1", func(tb testing.TB) {
		for name, tc := range map[string]struct {
			ui   fizz.DocsUI
			want map[string]string // URL -> integrity
		}{
			"swagger": {fizz.SwaggerUI(), map[string]string{
				docsui.SwaggerUIJS:  docsui.SwaggerUIJSIntegrity,
				docsui.SwaggerUICSS: docsui.SwaggerUICSSIntegrity,
			}},
			"scalar": {fizz.ScalarUI(), map[string]string{docsui.ScalarJS: docsui.ScalarJSIntegrity}},
		} {
			_, body := bCDocs(tb, tc.ui)
			els := at.HTML([]byte(body))
			seen := 0
			for _, e := range append(at.Tags(els, "script"), at.Tags(els, "link")...) {
				url := e.Attr["src"]
				if url == "" {
					url = e.Attr["href"]
				}
				if url == "" {
					continue
				}
				seen++
				want, ok := tc.want[url]
				switch {
				case !ok:
					tb.Errorf("%s: unexpected asset %s", name, url)
				case e.Attr["integrity"] != want || !bCSRIRe.MatchString(want):
					tb.Errorf("%s: %s integrity = %q, want %q", name, url, e.Attr["integrity"], want)
				}
				if e.Attr["crossorigin"] != "anonymous" || e.Attr["referrerpolicy"] != "no-referrer" {
					tb.Errorf("%s: %s crossorigin=%q referrerpolicy=%q", name, url, e.Attr["crossorigin"], e.Attr["referrerpolicy"])
				}
			}
			if seen != len(tc.want) {
				tb.Errorf("%s: %d external assets, want %d", name, seen, len(tc.want))
			}
		}
	})
}

// C06. The pin table of docs/compat.md equals the docsui constants.
func TestBCSRIConstantsMatchCompatDoc(t *testing.T) {
	at.Pending(t, "P8.1", func(tb testing.TB) {
		b, err := os.ReadFile(filepath.Join("..", "..", "docs", "compat.md"))
		if err != nil {
			tb.Fatal(err)
		}
		pins := bIPinRe.FindAllStringSubmatch(string(b), -1)
		if len(pins) != 3 {
			tb.Fatalf("pin table has %d rows, want 3", len(pins))
		}
		type asset struct{ url, sum string }
		want := map[asset]bool{
			{docsui.SwaggerUIJS, docsui.SwaggerUIJSIntegrity}:   true,
			{docsui.SwaggerUICSS, docsui.SwaggerUICSSIntegrity}: true,
			{docsui.ScalarJS, docsui.ScalarJSIntegrity}:         true,
		}
		for _, p := range pins {
			a := asset{"https://cdn.jsdelivr.net/npm/" + p[1] + "@" + p[2] + "/" + p[3], p[4]}
			if !want[a] {
				tb.Errorf("pin %s@%s %s (%s) has no matching constants", p[1], p[2], p[3], p[4])
			}
			delete(want, a)
		}
		for a := range want {
			tb.Errorf("constant %s (%s) is not in the pin table", a.url, a.sum)
		}
	})
}

// C07. validatorUrl is null in Swagger UI, whatever the user config says.
func TestBCSwaggerValidatorURLNull(t *testing.T) {
	at.Pending(t, "P8.1", func(tb testing.TB) {
		for _, ui := range []fizz.DocsUI{
			fizz.SwaggerUI(),
			fizz.SwaggerUI(docsui.WithUIConfig(map[string]any{"validatorUrl": "https://validator.swagger.io"})),
		} {
			_, body := bCDocs(tb, ui)
			v, ok := bCConfig(tb, body)["validatorUrl"]
			if !ok || v != nil {
				tb.Errorf("validatorUrl = %v (present %v), want null", v, ok)
			}
		}
	})
}

// C08. The user config is merged and cannot break out of the script.
func TestBCDocsUIConfigMerged(t *testing.T) {
	at.Pending(t, "P8.1", func(tb testing.TB) {
		cfg := map[string]any{"deepLinking": true, "x": "</script><b>"}
		for name, ui := range map[string]fizz.DocsUI{
			"swagger": fizz.SwaggerUI(docsui.WithUIConfig(cfg)),
			"scalar":  fizz.ScalarUI(docsui.WithUIConfig(cfg)),
		} {
			_, body := bCDocs(tb, ui)
			m := bCConfig(tb, body)
			if m["deepLinking"] != true || m["x"] != "</script><b>" {
				tb.Errorf("%s: config = %v", name, m)
			}
			// config + 1 bootstrap + 1 asset script.
			if n := strings.Count(body, "</script>"); n != 3 {
				tb.Errorf("%s: %d </script> tags, want 3", name, n)
			}
		}
	})
}

// C09. The title is HTML-escaped.
func TestBCDocsTitleEscaped(t *testing.T) {
	at.Pending(t, "P8.1", func(tb testing.TB) {
		for _, ui := range []fizz.DocsUI{
			fizz.SwaggerUI(docsui.WithTitle("A & <b>")),
			fizz.ScalarUI(docsui.WithTitle("A & <b>")),
		} {
			_, body := bCDocs(tb, ui)
			if !strings.Contains(body, "<title>A &amp; &lt;b&gt;</title>") {
				tb.Errorf("title not escaped:\n%s", body)
			}
		}
	})
}

// C10. The spec URL only appears JSON-escaped, inside the config.
func TestBCDocsSpecURLEscaped(t *testing.T) {
	at.Pending(t, "P8.1", func(tb testing.TB) {
		const spec = `/o.json?x="y'</script>`
		for name, h := range map[string]http.Handler{
			"swagger": docsui.SwaggerUI(spec),
			"scalar":  docsui.ScalarUI(spec),
		} {
			body := bCServe(tb, h)
			if strings.Contains(body, spec) || strings.Contains(body, "</script>\"") {
				tb.Errorf("%s: raw spec URL in body:\n%s", name, body)
			}
			if got := bCConfig(tb, body)["url"]; got != spec {
				tb.Errorf("%s: config url = %q, want %q", name, got, spec)
			}
			if n := strings.Count(body, "</script>"); n != 3 {
				tb.Errorf("%s: %d </script> tags, want 3", name, n)
			}
		}
	})
}

// C11. WithAssetURLs drops SRI unless integrity values are given.
func TestBCWithAssetURLsSRI(t *testing.T) {
	at.Pending(t, "P8.1", func(tb testing.TB) {
		body := bCServe(tb, docsui.SwaggerUI("/s", docsui.WithAssetURLs("/js", "/css")))
		if strings.Contains(body, "integrity") {
			tb.Errorf("integrity without values:\n%s", body)
		}
		body = bCServe(tb, docsui.SwaggerUI("/s", docsui.WithAssetURLs("/js", "/css", "sha384-JS", "sha384-CSS")))
		els := at.HTML([]byte(body))
		if got := at.Tags(els, "link")[0]; got.Attr["href"] != "/css" || got.Attr["integrity"] != "sha384-CSS" {
			tb.Errorf("link = %v", got.Attr)
		}
		if got := bCScripts(body)[0]; got.Attr["src"] != "/js" || got.Attr["integrity"] != "sha384-JS" {
			tb.Errorf("script = %v", got.Attr)
		}
		// Scalar ignores an empty css.
		body = bCServe(tb, docsui.ScalarUI("/s", docsui.WithAssetURLs("/js", "")))
		if strings.Contains(body, "<link") || strings.Contains(body, "integrity") {
			tb.Errorf("scalar override:\n%s", body)
		}
	})
}

// C12. The nonce is on every script and there are no inline handlers.
func TestBCCSPNonce(t *testing.T) {
	at.Pending(t, "P8.1", func(tb testing.TB) {
		for name, h := range map[string]http.Handler{
			"swagger": docsui.SwaggerUI("/s", docsui.WithCSPNonce("n0nce")),
			"scalar":  docsui.ScalarUI("/s", docsui.WithCSPNonce("n0nce")),
		} {
			els := at.HTML([]byte(bCServe(tb, h)))
			scripts := at.Tags(els, "script")
			if len(scripts) != 3 {
				tb.Errorf("%s: %d scripts, want 3", name, len(scripts))
			}
			for _, e := range scripts {
				if e.Attr["nonce"] != "n0nce" {
					tb.Errorf("%s: script %v lacks the nonce", name, e.Attr)
				}
			}
			for _, e := range els {
				for k := range e.Attr {
					if strings.HasPrefix(k, "on") {
						tb.Errorf("%s: <%s> has inline handler %s", name, e.Tag, k)
					}
				}
			}
		}
	})
}

// C13. The docs route is not in the spec and is not an error.
func TestBCDocsRouteUndocumented(t *testing.T) {
	at.Pending(t, "P8.1", func(tb testing.TB) {
		f, _ := bCDocs(tb)
		var doc struct {
			Paths map[string]any `json:"paths"`
		}
		if err := json.Unmarshal(at.Spec(tb, f), &doc); err != nil {
			tb.Fatal(err)
		}
		if _, ok := doc.Paths["/docs"]; ok {
			tb.Errorf("/docs is in the spec: %v", doc.Paths)
		}
		if errs := f.Errors(); len(errs) != 0 {
			tb.Errorf("Errors() = %v", errs)
		}
	})
}

// C14. The handlers need no router.
func TestBCDocsHandlersOnNetHTTP(t *testing.T) {
	at.Pending(t, "P8.1", func(tb testing.TB) {
		mux := http.NewServeMux()
		mux.Handle("/d", docsui.SwaggerUI("/s.json"))
		mux.Handle("/e", docsui.ScalarUI("/s.json"))
		for _, p := range []string{"/d", "/e"} {
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, p, nil))
			if w.Code != http.StatusOK || w.Header().Get("Content-Type") != "text/html; charset=utf-8" {
				tb.Errorf("%s: %d %q", p, w.Code, w.Header().Get("Content-Type"))
			}
		}
	})
}

// C16. The godoc names the default and the Swagger UI gaps.
func TestBCDocsGodoc(t *testing.T) {
	at.Pending(t, "P8.1", func(tb testing.TB) {
		fset := token.NewFileSet()
		pkgs, err := parser.ParseDir(fset, filepath.Join("..", ".."), func(fi os.FileInfo) bool {
			return !strings.HasSuffix(fi.Name(), "_test.go") && !strings.HasPrefix(fi.Name(), "zz_")
		}, parser.ParseComments)
		if err != nil {
			tb.Fatal(err)
		}
		files := pkgs["fizz"]
		if files == nil {
			tb.Fatal("package fizz not found")
		}
		p, err := doc.NewFromFiles(fset, bCFiles(files.Files), "github.com/wI2L/fizz")
		if err != nil {
			tb.Fatal(err)
		}
		docs := map[string]string{}
		for _, f := range p.Funcs {
			docs[f.Name] = f.Doc
		}
		for _, ty := range p.Types {
			for _, f := range ty.Funcs {
				docs[f.Name] = f.Doc
			}
			for _, m := range ty.Methods {
				docs[ty.Name+"."+m.Name] = m.Doc
			}
		}
		for name, subs := range map[string][]string{
			"Fizz.Docs": {"Swagger UI", "default"},
			"SwaggerUI": {"ScalarUI", "itemSchema", "11065"},
		} {
			for _, s := range subs {
				if !strings.Contains(docs[name], s) {
					tb.Errorf("doc of %s lacks %q:\n%s", name, s, docs[name])
				}
			}
		}
	})
}

func bCFiles(m map[string]*ast.File) []*ast.File {
	out := make([]*ast.File, 0, len(m))
	for _, f := range m {
		out = append(out, f)
	}
	return out
}
