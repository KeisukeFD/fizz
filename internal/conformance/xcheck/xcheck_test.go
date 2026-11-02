// Package xcheck cross-checks the fizz OAS 3.2.1 corpus and goldens with a
// second, independent validator, so that fizz's own specvalidate and the
// embedded meta-schema are not the only judges (plan task P10.8).
//
// # Tool choice
//
// The tool is github.com/pb33f/libopenapi-validator v0.16.2
// (schema_validation.ValidateOpenAPIDocument), which validates a document
// against the OAS 3.2.1 meta-schema. The runner-up, kin-openapi, was not
// needed: libopenapi-validator accepts every valid corpus document and
// rejects all five upgraded fail fixtures (see TestCrossCheckTool).
//
// # Disagreements
//
// A disagreement is a document on which the tool and the expectation
// (corpus: pass or fail directory, goldens: valid) differ. They are listed
// in the disagreements table below, each with the tool's message and a
// verdict on who is right. The table is empty: there is none. A document
// that stops disagreeing fails the test, so the table cannot go stale.
//
// # Messages for the fail fixtures
//
// The tool rejects each fail fixture for the same reason as the official
// test suite (checked in TestCrossCheckTool): the message must contain the
// listed fragment. This is agreement, not a disagreement.
//
// The module is separate (own go.mod) so the library go.mod and go.sum stay
// unchanged. It runs in the non-blocking xcheck CI job.
package xcheck

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi-validator/schema_validation"

	"github.com/KeisukeFD/fizz/internal/conformance/upgrade"
)

// disagreement records a document on which the tool and the expected
// verdict differ.
type disagreement struct {
	// message is a fragment of the tool's output.
	message string
	// verdict says who is right (fizz specvalidate, the official
	// meta-schema, or the tool) and why.
	verdict string
}

// corpusDisagreements is keyed by the corpus file name relative to
// internal/conformance/oai (slash-separated). It holds the valid corpus
// documents the tool rejects and the fail fixtures it accepts.
var corpusDisagreements = map[string]disagreement{}

// goldenDisagreements is keyed by the golden path from the module root
// (slash-separated). It holds the goldens the tool rejects while fizz's
// own MetaProblems accepts them, when the tool is the one at fault. A
// golden the tool rejects because of a fizz bug is not listed here: it gets
// a P10.7x/P10.9x line in the plan and a red case in internal/kitchensink.
var goldenDisagreements = map[string]disagreement{}

// failFragments is the part of the tool's message that must appear for each
// upgraded fail fixture (file name relative to oai/upgraded/oas/tests/schema/fail).
var failFragments = map[string]string{
	"invalid_schema_types.json": "want boolean or object",
	"no_containers.json":        "missing property 'paths'",
	"server_enum_empty.json":    "minItems: got 0, want 1",
	"servers.json":              "got object, want array",
	"unknown_container.json":    "false schema",
}

// validate reports the tool's verdict on a document (JSON or YAML): nil
// when valid, else an error with the joined messages.
func validate(src []byte) error {
	doc, err := libopenapi.NewDocument(src)
	if err != nil {
		return err
	}
	ok, errs := schema_validation.ValidateOpenAPIDocument(doc)
	if ok {
		return nil
	}
	var msgs []string
	for _, e := range errs {
		msgs = append(msgs, e.Message+": "+e.Reason)
		for _, sv := range e.SchemaValidationErrors {
			msgs = append(msgs, sv.FieldPath+": "+sv.Reason)
		}
	}
	return errors.New(strings.Join(msgs, "; "))
}

func moduleRoot(t testing.TB) string {
	t.Helper()
	root, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Fatalf("module root: %v", err)
	}
	return root
}

func corpus(t testing.TB) *upgrade.Corpus {
	t.Helper()
	c, err := upgrade.Build(os.DirFS("../oai"))
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// check compares the tool's verdict with the expectation, through the
// disagreements table.
func check(t *testing.T, name string, valid bool, err error, table map[string]disagreement, seen map[string]bool) {
	t.Helper()
	seen[name] = true
	d, listed := table[name]
	switch {
	case valid && err == nil, !valid && err != nil:
		if listed {
			t.Errorf("listed as a disagreement (%s) but the tool agrees: remove the entry", d.verdict)
		}
	case !listed:
		if valid {
			t.Errorf("tool rejects a valid document: %v", err)
		} else {
			t.Errorf("tool accepts an invalid document")
		}
	case err != nil && !strings.Contains(err.Error(), d.message):
		t.Errorf("tool message changed\n got: %v\nwant fragment: %q (%s)", err, d.message, d.verdict)
	}
}

func staleEntries(t *testing.T, table map[string]disagreement, seen map[string]bool) {
	t.Helper()
	for name := range table {
		if !seen[name] {
			t.Errorf("disagreements entry %s matches no document", name)
		}
	}
}

// TestCrossCheckTool verifies the tool itself: it must reject each
// upgraded fail fixture, for the reason the official suite expects.
func TestCrossCheckTool(t *testing.T) {
	const dir = "upgraded/oas/tests/schema/fail/"
	c := corpus(t)
	seen := map[string]bool{}
	for name, src := range c.Files {
		base, ok := strings.CutPrefix(name, dir)
		if !ok {
			continue
		}
		seen[base] = true
		t.Run(base, func(t *testing.T) {
			frag, ok := failFragments[base]
			if !ok {
				t.Fatalf("no expected fragment for fail fixture %s", base)
			}
			err := validate(src)
			if err == nil {
				t.Fatal("tool accepts a fail fixture")
			}
			if !strings.Contains(err.Error(), frag) {
				t.Errorf("message %q does not contain %q", err, frag)
			}
		})
	}
	for base := range failFragments {
		if !seen[base] {
			t.Errorf("failFragments entry %s matches no fail fixture", base)
		}
	}
	if len(seen) != 5 {
		t.Errorf("fail fixtures = %d, want 5", len(seen))
	}
}

// TestCrossCheckCorpus runs the tool over the upgraded learn examples, the
// upgraded pass fixtures and the included spec-text documents (all valid),
// and over the fail fixtures (all invalid).
func TestCrossCheckCorpus(t *testing.T) {
	c := corpus(t)
	var names []string
	for name := range c.Files {
		if strings.HasPrefix(name, "upgraded/") && strings.HasSuffix(name, ".json") &&
			name != "upgraded/CHANGES.json" && name != "upgraded/EXCLUDED.json" {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	seen := map[string]bool{}
	var valid, invalid int
	for _, name := range names {
		isValid := !strings.HasPrefix(name, "upgraded/oas/tests/schema/fail/")
		if isValid {
			valid++
		} else {
			invalid++
		}
		t.Run(strings.TrimSuffix(strings.TrimPrefix(name, "upgraded/"), ".json"), func(t *testing.T) {
			check(t, name, isValid, validate(c.Files[name]), corpusDisagreements, seen)
		})
	}
	staleEntries(t, corpusDisagreements, seen)
	if valid == 0 || invalid != 5 {
		t.Fatalf("corpus walk: %d valid, %d invalid documents", valid, invalid)
	}
	t.Logf("cross-checked %d valid and %d invalid corpus documents", valid, invalid)
}

// skipDirs are never walked, as in internal/conformance/goldens_test.go.
// The nested modules are skipped too: a go.mod marks a directory that is
// not part of the root module.
var skipDirs = map[string]bool{
	".git": true, ".claude": true, "vendor": true, "node_modules": true,
	"internal/conformance/oai": true,
}

// TestCrossCheckGoldens runs the tool over every whole-document golden under
// a testdata directory, and over every single-schema golden wrapped into a
// minimal 3.2.1 document (same discovery and wrapping as
// internal/conformance/goldens_test.go).
func TestCrossCheckGoldens(t *testing.T) {
	root := moduleRoot(t)
	var docs, schemas int
	seen := map[string]bool{}

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
			if rel != "." {
				if _, err := os.Stat(filepath.Join(path, "go.mod")); err == nil {
					return filepath.SkipDir // a nested module
				}
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
			wrapped := map[string]any{
				"openapi":    upgrade.TargetVersion,
				"info":       map[string]any{"title": "schema golden", "version": "1.0.0"},
				"components": map[string]any{"schemas": map[string]any{name: v}},
			}
			upgrade.StubDanglingRefs(wrapped)
			doc, err = json.Marshal(wrapped)
		default:
			return nil // not a document: goldens_test.go gates these
		}
		if err != nil {
			t.Errorf("%s: %v", rel, err)
			return nil
		}
		t.Run(rel, func(t *testing.T) {
			check(t, rel, true, validate(doc), goldenDisagreements, seen)
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	staleEntries(t, goldenDisagreements, seen)
	if docs == 0 || schemas == 0 {
		t.Fatalf("empty walk: %d documents, %d schema goldens", docs, schemas)
	}
	t.Logf("cross-checked %d documents and %d schema goldens", docs, schemas)
}
