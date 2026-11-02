package conformance

import (
	"bytes"
	"io/fs"
	"os"
	"path"
	"sort"
	"strings"
	"testing"

	"github.com/KeisukeFD/fizz/internal/conformance/upgrade"
	"github.com/KeisukeFD/fizz/internal/specvalidate"
)

func buildCorpus(t *testing.T) *upgrade.Corpus {
	t.Helper()
	c, err := upgrade.Build(os.DirFS("oai"))
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// TestUpgradeGolden reruns the converter in memory over oai/orig and
// requires the committed oai/upgraded tree and the NOTICE section to be
// byte-identical. Regenerate with: go generate ./internal/conformance/...
func TestUpgradeGolden(t *testing.T) {
	c := buildCorpus(t)
	const hint = "run go generate ./internal/conformance/..."

	for name, want := range c.Files {
		got, err := os.ReadFile("oai/" + name)
		if err != nil {
			t.Errorf("%s: %v (%s)", name, err, hint)
			continue
		}
		if !bytes.Equal(got, want) {
			t.Errorf("%s differs from the converter output (%s)", name, hint)
		}
	}

	// No stale committed file.
	err := fs.WalkDir(os.DirFS("oai"), upgrade.UpgradedDir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if _, ok := c.Files[p]; !ok {
			t.Errorf("%s is committed but not produced by the converter (%s)", p, hint)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	notice, err := os.ReadFile("oai/NOTICE")
	if err != nil {
		t.Fatal(err)
	}
	wantNotice, err := upgrade.ApplyNotice(notice, c.Notice)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(notice, wantNotice) {
		t.Errorf("oai/NOTICE generated section is stale (%s)", hint)
	}
}

// TestCorpusCounts pins the shape of the corpus so a lost file is noticed.
func TestCorpusCounts(t *testing.T) {
	c := buildCorpus(t)
	count := map[string]int{}
	for _, e := range c.Entries {
		switch {
		case strings.HasPrefix(e.File, "upgraded/learn/"):
			count["learn"]++
		case strings.HasPrefix(e.File, "upgraded/oas/tests/schema/pass/"):
			count["pass"]++
		case strings.HasPrefix(e.File, "upgraded/oas/tests/schema/fail/"):
			count["fail"]++
		case strings.HasPrefix(e.File, "upgraded/spectext/"):
			count["spectext"]++
		}
	}
	want := map[string]int{"learn": 11, "pass": 15, "fail": 5}
	for k, v := range want {
		if count[k] != v {
			t.Errorf("%s files = %d, want %d", k, count[k], v)
		}
	}
	if total := count["spectext"] + len(c.Excluded); total != 127 {
		t.Errorf("spec text blocks = %d included + %d excluded, want 127 in total", count["spectext"], len(c.Excluded))
	}
}

// expectedFailures lists corpus documents that do not pass Validate after a
// faithful upgrade, with the exact validator output and why. It is empty
// while every document passes. An entry that starts passing fails the test.
var expectedFailures = map[string]struct{ message, reason string }{}

// TestCorpusValid checks that every upgraded learn example, every upgraded
// pass fixture and every included spec-text document passes Validate. Fail
// fixtures are deliberately not checked here.
func TestCorpusValid(t *testing.T) {
	c := buildCorpus(t)
	var names []string
	for name := range c.Files {
		switch {
		case strings.HasPrefix(name, "upgraded/learn/"),
			strings.HasPrefix(name, "upgraded/oas/tests/schema/pass/"),
			strings.HasPrefix(name, "upgraded/spectext/"):
			names = append(names, name)
		}
	}
	sort.Strings(names)
	if len(names) == 0 {
		t.Fatal("no corpus documents")
	}
	seen := map[string]bool{}
	for _, name := range names {
		t.Run(strings.TrimPrefix(strings.TrimSuffix(name, ".json"), "upgraded/"), func(t *testing.T) {
			err := specvalidate.Validate(c.Files[name])
			exp, expected := expectedFailures[name]
			seen[name] = true
			switch {
			case err == nil && expected:
				t.Errorf("listed in expectedFailures (%s) but passes: remove the entry", exp.reason)
			case err != nil && !expected:
				t.Errorf("%s does not validate: %v", path.Base(name), err)
			case err != nil && err.Error() != exp.message:
				t.Errorf("validator message changed\n got: %q\nwant: %q (%s)", err.Error(), exp.message, exp.reason)
			}
		})
	}
	for name := range expectedFailures {
		if !seen[name] {
			t.Errorf("expectedFailures entry %s matches no corpus document", name)
		}
	}
}
