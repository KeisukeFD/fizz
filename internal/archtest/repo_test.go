package archtest

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// A04: go.mod invariants.
func TestModuleInvariants(t *testing.T) {
	b, err := os.ReadFile(filepath.Join(bARoot(t), "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	mod := string(b)
	for name, re := range map[string]string{
		"module path":    `(?m)^module github\.com/KeisukeFD/fizz\s*$`,
		"go directive":   `(?m)^go 1\.27(\.\d+)?\s*$`,
		"gadgeto v0.9.0": `(?m)^\s*github\.com/loopfz/gadgeto v0\.9\.0\s*(//.*)?$`,
	} {
		if !regexp.MustCompile(re).MatchString(mod) {
			t.Errorf("go.mod: %s not found", name)
		}
	}
}

// A05: the lint tooling lives in tools/go.mod, not in the main module.
func TestToolsModuleIsolated(t *testing.T) {
	all := bAGo(t, "list", "-m", "all")
	for _, bad := range []string{"honnef.co/go/tools", "golang.org/x/vuln"} {
		if strings.Contains(all, bad) {
			t.Errorf("main module graph contains %s", bad)
		}
	}
}

// bASemver parses vX.Y.Z[-pre|+meta] into its numeric parts.
func bASemver(v string) [3]int {
	var out [3]int
	v = strings.TrimPrefix(v, "v")
	if i := strings.IndexAny(v, "-+"); i >= 0 {
		v = v[:i]
	}
	for i, p := range strings.SplitN(v, ".", 3) {
		out[i], _ = strconv.Atoi(p)
	}
	return out
}

// A10: yaml.v2 is gone and gin is new enough to not need it.
func TestYAMLDepSwap(t *testing.T) {
	for _, d := range strings.Fields(bAGo(t, "list", "-deps", "./...")) {
		if d == "gopkg.in/yaml.v2" {
			t.Errorf("non-test dependency on gopkg.in/yaml.v2")
		}
	}
	ver := strings.TrimSpace(bAGo(t, "list", "-m", "-f", "{{.Version}}", "github.com/gin-gonic/gin"))
	got, want := bASemver(ver), [3]int{1, 12, 0}
	for i := range got {
		if got[i] != want[i] {
			if got[i] < want[i] {
				t.Errorf("gin %s < v1.12.0", ver)
			}
			break
		}
	}

}
