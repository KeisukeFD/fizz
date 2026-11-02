package acceptance

// Batch C (embedded docs UI): C17-C19. See docs/plan/ACCEPTANCE.md.

import (
	"crypto/sha512"
	"encoding/base64"
	"net/http"
	"strings"
	"testing"

	"github.com/KeisukeFD/fizz"
	"github.com/KeisukeFD/fizz/docsui"
	"github.com/KeisukeFD/fizz/internal/acceptance/at"
)

// C17. The embedded Scalar page disables the default fonts, which Scalar
// would otherwise load from a CDN.
func TestEmbeddedScalarNoFonts(t *testing.T) {
	_, body := bCDocs(t, fizz.EmbeddedScalarUI())
	if v, ok := bCConfig(t, body)["withDefaultFonts"]; !ok || v != false {
		t.Errorf("withDefaultFonts = %v, want false", v)
	}
	_, body = bCDocs(t, fizz.EmbeddedScalarUI(docsui.WithUIConfig(map[string]any{"withDefaultFonts": true})))
	if v := bCConfig(t, body)["withDefaultFonts"]; v != false {
		t.Errorf("withDefaultFonts with user override = %v, want false", v)
	}

}

// C18. Neither embedded page references an external URL.
func TestEmbeddedNoExternalURLs(t *testing.T) {
	for name, ui := range map[string]fizz.DocsUI{
		"swagger": fizz.EmbeddedSwaggerUI(),
		"scalar":  fizz.EmbeddedScalarUI(),
	} {
		_, body := bCDocs(t, ui)
		n := 0
		for _, e := range at.HTML([]byte(body)) {
			for _, k := range []string{"src", "href"} {
				v, ok := e.Attr[k]
				if !ok {
					continue
				}
				n++
				if strings.HasPrefix(v, "http://") || strings.HasPrefix(v, "https://") || strings.HasPrefix(v, "//") {
					t.Errorf("%s: %s %s=%q is external", name, e.Tag, k, v)
				}
			}
		}
		if n == 0 {
			t.Errorf("%s: no src or href found", name)
		}
	}

}

// C19. The assets are served with the pinned hash.
func TestEmbeddedAssetsServed(t *testing.T) {
	f, _ := bCDocs(t, fizz.EmbeddedSwaggerUI())
	w := at.Do(f, http.MethodGet, "/docs/assets/swagger-ui-bundle.js")
	if w.Code != http.StatusOK {
		t.Fatalf("asset = %d, want 200", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "text/javascript; charset=utf-8" {
		t.Errorf("Content-Type = %q", ct)
	}
	h := sha512.Sum384(w.Body.Bytes())
	if got := "sha384-" + base64.StdEncoding.EncodeToString(h[:]); got != docsui.SwaggerUIJSIntegrity {
		t.Errorf("sha384 = %s, want %s", got, docsui.SwaggerUIJSIntegrity)
	}
	if w := at.Do(f, http.MethodGet, "/docs/assets/nope.js"); w.Code != http.StatusNotFound {
		t.Errorf("nope.js = %d, want 404", w.Code)
	}
	if w := at.Do(f, http.MethodGet, "/docs/assets/../embedded.go"); w.Code == http.StatusOK {
		t.Errorf("traversal = %d, want not 200", w.Code)
	}

}
