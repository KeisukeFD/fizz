package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// server serves every pinned URL with a body derived from the request path.
func server(t *testing.T, suffix string) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("body of " + r.URL.Path + suffix))
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

func TestFetchAll(t *testing.T) {
	blobs, err := fetchAll(http.DefaultClient, server(t, ""))
	if err != nil {
		t.Fatal(err)
	}
	if want := 11 + 20 + 1 + 2; len(blobs) != want {
		t.Fatalf("got %d blobs, want %d", len(blobs), want)
	}
	for i := 1; i < len(blobs); i++ {
		if blobs[i-1].entry.Path >= blobs[i].entry.Path {
			t.Fatalf("not sorted at %s", blobs[i].entry.Path)
		}
	}
	paths := map[string]entry{}
	for _, b := range blobs {
		paths[b.entry.Path] = b.entry
	}
	e, ok := paths["orig/learn/examples/v3.0/petstore.yaml"]
	if !ok || e.Commit != learnCommit || e.License != "CC-BY-4.0" || len(e.SHA256) != 64 {
		t.Fatalf("petstore entry = %+v", e)
	}
	if e := paths["orig/oas/versions/3.2.1.md"]; e.Commit != oasSpec {
		t.Fatalf("spec entry = %+v", e)
	}
	if e := paths["LICENSE-oas-Apache-2.0"]; e.SourcePath != "LICENSE" || e.Commit != oasTests {
		t.Fatalf("license entry = %+v", e)
	}
}

func TestUpdateThenVerify(t *testing.T) {
	out := t.TempDir()
	blobs, err := fetchAll(http.DefaultClient, server(t, ""))
	if err != nil {
		t.Fatal(err)
	}
	m := manifestOf(blobs)
	if err := write(out, blobs, m); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(out, "orig/oas/tests/schema/pass/mega.yaml"))
	if err != nil || !strings.Contains(string(got), "mega.yaml") {
		t.Fatalf("mega.yaml = %q, %v", got, err)
	}
	if err := verify(out, m); err != nil {
		t.Fatalf("verify after update: %v", err)
	}

	// Upstream content changed: verify must fail and name the file.
	changed, err := fetchAll(http.DefaultClient, server(t, " changed"))
	if err != nil {
		t.Fatal(err)
	}
	err = verify(out, manifestOf(changed))
	if err == nil || !strings.Contains(err.Error(), "orig/learn/examples/v3.0/petstore.yaml") {
		t.Fatalf("verify on changed content: %v", err)
	}

	// A manifest entry that is no longer fetched fails too.
	if err := verify(out, m[1:]); err == nil || !strings.Contains(err.Error(), "no longer fetched") {
		t.Fatalf("verify on shrunk list: %v", err)
	}
}

func TestUpdateRemovesStaleFiles(t *testing.T) {
	out := t.TempDir()
	stale := filepath.Join(out, "orig/learn/old.yaml")
	if err := os.MkdirAll(filepath.Dir(stale), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stale, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	blobs, err := fetchAll(http.DefaultClient, server(t, ""))
	if err != nil {
		t.Fatal(err)
	}
	if err := write(out, blobs, manifestOf(blobs)); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Fatalf("stale file survived: %v", err)
	}
}

func TestDownloadErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "empty") {
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()
	if _, err := download(srv.Client(), srv.URL+"/missing"); err == nil || !strings.Contains(err.Error(), "404") {
		t.Fatalf("404: %v", err)
	}
	if _, err := download(srv.Client(), srv.URL+"/empty"); err == nil || !strings.Contains(err.Error(), "empty body") {
		t.Fatalf("empty: %v", err)
	}
}
