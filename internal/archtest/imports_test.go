// Package archtest holds the architecture acceptance tests of
// docs/plan/ACCEPTANCE.md, Batch A: the import rule and module invariants.
package archtest

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/wI2L/fizz/internal/acceptance/at"
)

// bARoot returns the repository root (the directory holding go.mod).
func bARoot(tb testing.TB) string {
	tb.Helper()
	dir, err := os.Getwd()
	if err != nil {
		tb.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			tb.Fatal("go.mod not found above the working directory")
		}
		dir = parent
	}
}

// bAGo runs the go tool at the repository root and returns stdout.
func bAGo(tb testing.TB, args ...string) string {
	tb.Helper()
	cmd := exec.Command("go", args...)
	cmd.Dir = bARoot(tb)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		tb.Fatalf("go %s: %v\n%s", strings.Join(args, " "), err, stderr.String())
	}
	return stdout.String()
}

// bAStrip removes the " [p.test]" variant suffix of a test package path.
func bAStrip(p string) string {
	if i := strings.Index(p, " ["); i >= 0 {
		return p[:i]
	}
	return p
}

// bAForbidden lists the import-path prefixes the router-agnostic packages
// must not reach.
var bAForbidden = []string{"github.com/gin-gonic/gin", "github.com/loopfz/gadgeto"}

// bAViolations runs `go list -deps` (plus -test when withTests) over
// patterns and returns one line per forbidden package reached, with the
// importer chain from a root package. Patterns that match no package are
// skipped.
func bAViolations(tb testing.TB, withTests bool, forbidden []string, patterns ...string) []string {
	tb.Helper()
	var out []string
	for _, pat := range patterns {
		dir := strings.TrimSuffix(strings.TrimSuffix(pat, "..."), "/")
		if _, err := os.Stat(filepath.Join(bARoot(tb), dir)); err != nil {
			continue // directory does not exist (yet)
		}
		args := []string{"list"}
		if withTests {
			args = append(args, "-test")
		}
		roots := strings.Fields(bAGo(tb, append(append([]string{}, args...), pat)...))
		if len(roots) == 0 {
			continue // no package under this pattern (yet)
		}
		args = append(args, "-deps", "-f", `{{.ImportPath}}|{{join .Imports " "}}|{{join .TestImports " "}}|{{join .XTestImports " "}}`, pat)
		graph := map[string][]string{}
		for _, line := range strings.Split(bAGo(tb, args...), "\n") {
			parts := strings.Split(line, "|")
			if len(parts) != 4 {
				continue
			}
			from := bAStrip(parts[0])
			for _, p := range parts[1:] {
				for _, imp := range strings.Fields(p) {
					graph[from] = append(graph[from], bAStrip(imp))
				}
			}
		}
		// BFS from the matched roots, remembering the parent of each node.
		parent := map[string]string{}
		var queue []string
		for _, r := range roots {
			r = bAStrip(r)
			if _, ok := parent[r]; !ok {
				parent[r] = ""
				queue = append(queue, r)
			}
		}
		for len(queue) > 0 {
			p := queue[0]
			queue = queue[1:]
			for _, imp := range graph[p] {
				if _, ok := parent[imp]; !ok {
					parent[imp] = p
					queue = append(queue, imp)
				}
			}
		}
		var found []string
		for p := range parent {
			for _, f := range forbidden {
				if p == f || strings.HasPrefix(p, f+"/") {
					found = append(found, p)
					break
				}
			}
		}
		sort.Strings(found)
		for _, p := range found {
			var chain []string
			for c := p; c != ""; c = parent[c] {
				chain = append([]string{c}, chain...)
			}
			out = append(out, pat+": "+strings.Join(chain, " -> "))
		}
	}
	return out
}

// A01: the router-agnostic packages never import gin or gadgeto.
func TestRouterAgnostic(t *testing.T) {
	at.Pending(t, "P1.3", func(tb testing.TB) {
		v := bAViolations(tb, true, bAForbidden, "./openapi/...", "./bind/...", "./docsui/...")
		for _, l := range v {
			tb.Errorf("forbidden import: %s", l)
		}
	})
}

// A02: the same checker flags the root package, so the rule is not vacuous.
func TestRouterAgnosticSelfCheck(t *testing.T) {
	v := bAViolations(t, true, bAForbidden, ".")
	var gin, gadgeto bool
	for _, l := range v {
		gin = gin || strings.Contains(l, "github.com/gin-gonic/gin")
		gadgeto = gadgeto || strings.Contains(l, "github.com/loopfz/gadgeto")
	}
	if !gin || !gadgeto {
		t.Fatalf("checker found no gin (%v) or gadgeto (%v) under the root package: %v", gin, gadgeto, v)
	}
	if !strings.Contains(strings.Join(v, "\n"), " -> ") {
		t.Fatalf("violations carry no importer chain: %v", v)
	}
}

// bANoDep fails tb for every package under prefix that ./openapi reaches.
func bANoDep(tb testing.TB, prefix string) {
	tb.Helper()
	for d := range strings.FieldsSeq(bAGo(tb, "list", "-deps", "./openapi")) {
		if d == prefix || strings.HasPrefix(d, prefix+"/") {
			tb.Errorf("./openapi depends on %s", d)
		}
	}
}

// A03: openapi does not depend on a third-party uuid library. gofrs goes
// with P1.2; google is reached through gadgeto/tonic until P1.3.
func TestNoUUIDDep(t *testing.T) {
	t.Run("gofrs", func(t *testing.T) {
		bANoDep(t, "github.com/gofrs/uuid")
	})
	t.Run("google", func(t *testing.T) {
		at.Pending(t, "P1.3", func(tb testing.TB) { bANoDep(tb, "github.com/google/uuid") })
	})
}

// A06: the embedded docs UI lives in the main module.
func TestDocsUIEmbeddedSameModule(t *testing.T) {
	root := bARoot(t)
	if st, err := os.Stat(filepath.Join(root, "docsui", "embedded")); err != nil || !st.IsDir() {
		t.Fatal("docsui/embedded does not exist")
	}
	err := filepath.WalkDir(filepath.Join(root, "docsui"), func(p string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() && d.Name() == "go.mod" {
			t.Errorf("nested module: %s", p)
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	mod := strings.TrimSpace(bAGo(t, "list", "-m"))
	got := strings.TrimSpace(bAGo(t, "list", "-f", "{{.Module.Path}}", "./docsui/embedded"))
	if got != mod {
		t.Fatalf("docsui/embedded module = %q, want %q", got, mod)
	}
}
