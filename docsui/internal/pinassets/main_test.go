package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFindAssetsAndRewrite(t *testing.T) {
	dir := t.TempDir()
	src := `package docsui

const (
	FooJS = "https://example.com/foo.js"
	FooJSIntegrity = "sha384-old"
	BarCSS = "https://example.com/bar.css"
	BarCSSIntegrity = "sha384-bar"
	Other = "https://example.com/other"
)
`
	f := filepath.Join(dir, "x.go")
	if err := os.WriteFile(f, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	as, err := findAssets(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(as) != 2 || as[0].name != "BarCSS" || as[1].name != "FooJS" {
		t.Fatalf("assets = %+v", as)
	}
	if err := rewrite(as, map[string]string{"FooJS": "sha384-new", "BarCSS": "sha384-bar"}); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(f)
	if !strings.Contains(string(got), `"sha384-new"`) || !strings.Contains(string(got), `"sha384-bar"`) {
		t.Fatalf("rewritten:\n%s", got)
	}
}

func TestSRI(t *testing.T) {
	// sha384 of the empty input.
	const want = "sha384-OLBgp1GsljhM2TJ+sbHjaiH9txEUvgdDTAzHv2P24donTt6/529l+9Ua0vFImLlb"
	if got := sri(nil); got != want {
		t.Fatalf("sri(nil) = %s", got)
	}
}
