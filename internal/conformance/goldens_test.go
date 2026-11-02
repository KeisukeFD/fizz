package conformance

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/KeisukeFD/fizz/internal/conformance/upgrade"
	"github.com/KeisukeFD/fizz/internal/kitchensink"
	"github.com/KeisukeFD/fizz/internal/plan"
)

// nonDocumentGoldens lists testdata JSON/YAML files (slash-separated, from
// the module root) that are neither a whole OAS document nor a schema
// golden, each with a one-word reason. A new golden that is none of the
// three fails TestAllGoldensValid, so it cannot slip through ungated.
var nonDocumentGoldens = map[string]string{}

// goldenFixedBy lists goldens that currently fail MetaProblems, keyed by
// slash-separated path from the module root, with the task that fixes them.
// The golden is red until that task is in plan.Done, then it must pass.
var goldenFixedBy = map[string]string{}

// skipDirs are never walked. internal/conformance/oai is covered by the
// corpus tests.
var skipDirs = map[string]bool{
	".git": true, ".claude": true, "vendor": true, "node_modules": true,
	"internal/conformance/oai": true,
}

func moduleRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found")
		}
		dir = parent
	}
}

// TestAllGoldensValid runs MetaProblems on every golden OAS document under a
// testdata directory, and on every single-schema golden wrapped into a
// minimal 3.2.1 document.
func TestAllGoldensValid(t *testing.T) {
	root := moduleRoot(t)
	var docs, schemas int
	seenNonDoc := map[string]bool{}

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			if skipDirs[d.Name()] || skipDirs[rel] {
				return filepath.SkipDir
			}
			return nil
		}
		ext := filepath.Ext(rel)
		if ext != ".json" && ext != ".yaml" && ext != ".yml" {
			return nil
		}
		if !strings.HasPrefix(rel, "testdata/") && !strings.Contains(rel, "/testdata/") {
			return nil
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var v any
		if ext == ".json" {
			if err := json.Unmarshal(src, &v); err != nil {
				t.Errorf("%s: invalid JSON: %v", rel, err)
				return nil
			}
		} else if v, err = upgrade.DecodeYAML(src); err != nil {
			t.Errorf("%s: %v", rel, err)
			return nil
		}
		m, _ := v.(map[string]any)

		var doc []byte
		switch _, isDoc := m["openapi"]; {
		case isDoc:
			docs++
			doc, err = json.Marshal(v)
		case strings.Contains(rel, "testdata/schemas/"):
			schemas++
			name := strings.TrimSuffix(filepath.Base(rel), ext)
			// A schema golden refers to schemas of its own test; the wrapper
			// stubs them so the references resolve.
			wrapped := map[string]any{
				"openapi":    upgrade.TargetVersion,
				"info":       map[string]any{"title": "schema golden", "version": "1.0.0"},
				"components": map[string]any{"schemas": map[string]any{name: v}},
			}
			upgrade.StubDanglingRefs(wrapped)
			doc, err = json.Marshal(wrapped)
		default:
			if _, ok := nonDocumentGoldens[rel]; !ok {
				t.Errorf("%s: neither an OAS document, a schema golden, nor listed in nonDocumentGoldens", rel)
			}
			seenNonDoc[rel] = true
			return nil
		}
		if err != nil {
			t.Errorf("%s: %v", rel, err)
			return nil
		}

		t.Run(rel, func(t *testing.T) {
			plan.Pending(t, goldenFixedBy[rel], func(tb testing.TB) {
				tb.Helper()
				for _, p := range kitchensink.MetaProblems(doc) {
					tb.Error(p)
				}
			})
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	for rel := range nonDocumentGoldens {
		if !seenNonDoc[rel] {
			t.Errorf("nonDocumentGoldens entry %s matches no file", rel)
		}
	}
	if docs == 0 || schemas == 0 {
		t.Fatalf("empty walk: %d documents, %d schema goldens", docs, schemas)
	}
	t.Logf("checked %d documents and %d schema goldens", docs, schemas)
}
