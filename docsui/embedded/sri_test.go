package embedded_test

import (
	"crypto/sha512"
	"encoding/base64"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/KeisukeFD/fizz/docsui"
)

// pinned maps every pinned asset URL to its SRI constant.
var pinned = map[string]string{
	docsui.SwaggerUIJS:  docsui.SwaggerUIJSIntegrity,
	docsui.SwaggerUICSS: docsui.SwaggerUICSSIntegrity,
	docsui.ScalarJS:     docsui.ScalarJSIntegrity,
}

// C20: the embedded copies match the docsui SRI constants.
func TestEmbeddedSRI(t *testing.T) {
	seen := map[string]bool{}
	for url, want := range pinned {
		name := path.Base(url)
		seen[name] = true
		b, err := os.ReadFile(filepath.Join("assets", name))
		if err != nil {
			t.Errorf("%s: %v (run go generate)", name, err)
			continue
		}
		h := sha512.Sum384(b)
		if got := "sha384-" + base64.StdEncoding.EncodeToString(h[:]); got != want {
			t.Errorf("%s: sha384 = %s, constant = %s", name, got, want)
		}
	}
	entries, err := os.ReadDir("assets")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if !seen[e.Name()] {
			t.Errorf("assets/%s has no pinned constant", e.Name())
		}
	}
}

// C21: the generate directive is present.
func TestPinAssetsDirective(t *testing.T) {
	b, err := os.ReadFile("gen.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "//go:generate go run ../internal/pinassets") {
		t.Fatal("gen.go lacks the pinassets go:generate directive")
	}
}
