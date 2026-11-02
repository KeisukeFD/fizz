package acceptance

// ADR 0001 section 4, "Binary size of fizz.Embedded*UI": a binary that
// imports fizz but does not call Embedded*UI carries no docs assets.

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

const bEmbSrc = `package main

import "github.com/KeisukeFD/fizz"

func main() {
	f := fizz.New()
	%s
	_ = f
}
`

func TestEmbeddedAssetsLinkedOnlyWhenUsed(t *testing.T) {
	if testing.Short() {
		t.Skip("builds four programs")
	}
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	asset, err := os.ReadFile(filepath.Join(root, "docsui", "embedded", "assets", "swagger-ui-bundle.js"))
	if err != nil {
		t.Fatal(err)
	}
	marker := asset[3000:3060]

	// The probes are built with -overlay: their sources live outside the
	// repository, mapped onto package paths inside the module, so they
	// resolve fizz and its dependencies without network and the tree
	// that other tests walk is never written to.
	tmp := t.TempDir()

	for _, tc := range []struct {
		name string
		stmt string
		want bool
	}{
		{"new-only", ``, false},
		{"docs-default", `f.Docs("/docs", "/openapi.json")`, false},
		{"docs-scalar", `f.Docs("/docs", "/openapi.json", fizz.ScalarUI())`, false},
		{"docs-embedded", `f.Docs("/docs", "/openapi.json", fizz.EmbeddedSwaggerUI())`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			src := filepath.Join(tmp, tc.name+".go")
			code := bytes.Replace([]byte(bEmbSrc), []byte("%s"), []byte(tc.stmt), 1)
			if err := os.WriteFile(src, code, 0o644); err != nil {
				t.Fatal(err)
			}
			pkg := filepath.Join("internal", "sizeprobe", tc.name)
			overlay, err := json.Marshal(map[string]any{
				"Replace": map[string]string{filepath.Join(root, pkg, "main.go"): src},
			})
			if err != nil {
				t.Fatal(err)
			}
			ov := filepath.Join(tmp, tc.name+".json")
			if err := os.WriteFile(ov, overlay, 0o644); err != nil {
				t.Fatal(err)
			}
			bin := filepath.Join(tmp, tc.name+".bin")
			cmd := exec.Command("go", "build", "-overlay", ov, "-o", bin, "./"+filepath.ToSlash(pkg))
			cmd.Dir = root
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("go build: %v\n%s", err, out)
			}
			b, err := os.ReadFile(bin)
			if err != nil {
				t.Fatal(err)
			}
			if got := bytes.Contains(b, marker); got != tc.want {
				t.Errorf("binary of %q contains the assets = %v, want %v (size %d)", tc.stmt, got, tc.want, len(b))
			}
		})
	}
}
