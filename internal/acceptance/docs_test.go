package acceptance

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Helpers of the docs tests: they read documentation files from the
// repository root. The tests live in docs_snippets_test.go and
// docsui_test.go.

const bIRoot = "../.."

func bIRead(tb testing.TB, rel string) string {
	tb.Helper()
	b, err := os.ReadFile(filepath.Join(bIRoot, filepath.FromSlash(rel)))
	if err != nil {
		tb.Fatalf("read %s: %v", rel, err)
	}
	return string(b)
}

func bIContains(tb testing.TB, file, doc string, words ...string) {
	tb.Helper()
	for _, w := range words {
		if !strings.Contains(doc, w) {
			tb.Errorf("%s does not contain %q", file, w)
		}
	}
}

func bIAbsent(tb testing.TB, file, doc string, words ...string) {
	tb.Helper()
	for _, w := range words {
		if strings.Contains(doc, w) {
			tb.Errorf("%s must not contain %q", file, w)
		}
	}
}

var bIPinRe = regexp.MustCompile("\\|\\s*`([^`]+)@([^`]+)`\\s*\\|\\s*`([^`]+)`\\s*\\|\\s*`(sha384-[^`]+)`")
