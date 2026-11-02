package acceptance

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/KeisukeFD/fizz/internal/acceptance/at"
)

// Batch I (docs): I12 and I15. I10 and I11 live in docs_snippets_test.go with
// the snippet checker. Everything here reads files, so it compiles against
// today's tree. Each test is red until its doc task lands.

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

var bIVersionHeadRe = regexp.MustCompile(`(?m)^#+ .*\d+\.\d+\.\d+`)

// I12
func TestBIReleaseNotes(t *testing.T) {
	at.Pending(t, "P9.5", func(tb testing.TB) {
		doc := bIRead(tb, "CHANGELOG.md")
		i := strings.Index(doc, "v0.30.0")
		if i < 0 {
			tb.Fatal("CHANGELOG.md has no v0.30.0 section")
		}
		section := doc[i:]
		// The section ends at the next version heading.
		if j := bIVersionHeadRe.FindAllStringIndex(section, 2); len(j) == 2 {
			section = section[:j[1][0]]
		}
		bIContains(tb, "CHANGELOG.md v0.30.0", section,
			"Go 1.27", "1.26", "3.2.1", "MIGRATION", "bind hook", "oapi-codegen")
	})
}

var bIPinRe = regexp.MustCompile("\\|\\s*`([^`]+)@([^`]+)`\\s*\\|\\s*`([^`]+)`\\s*\\|\\s*`(sha384-[^`]+)`")

// I15. The pin table of docs/compat.md must match the docsui constants.
// docsui does not exist yet, so its sources are scanned as text.
func TestBIDocsCompatTable(t *testing.T) {
	at.Pending(t, "P8.4", func(tb testing.TB) {
		doc := bIRead(tb, "docs/compat.md")
		pins := bIPinRe.FindAllStringSubmatch(doc, -1)
		if len(pins) < 3 {
			tb.Fatalf("docs/compat.md pin table has %d rows, want >= 3", len(pins))
		}
		var src bytes.Buffer
		files, _ := filepath.Glob(filepath.Join(bIRoot, "docsui", "*.go"))
		for _, f := range files {
			if strings.HasSuffix(f, "_test.go") {
				continue
			}
			b, _ := os.ReadFile(f)
			src.Write(b)
		}
		if src.Len() == 0 {
			tb.Fatal("no docsui sources to compare the pins with")
		}
		for _, p := range pins {
			pkg, version, sum := p[1], p[2], p[4]
			if !bytes.Contains(src.Bytes(), []byte(sum)) {
				tb.Errorf("docsui has no constant for %s@%s (%s)", pkg, version, sum)
			}
			if !bytes.Contains(src.Bytes(), []byte(pkg+"@"+version)) {
				tb.Errorf("docsui does not reference %s@%s", pkg, version)
			}
		}
		bIContains(tb, "docs/compat.md", doc, "AllOfWrap", "11065")
	})
}
