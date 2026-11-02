package repro_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/KeisukeFD/fizz/internal/conformance/repro"
	"github.com/KeisukeFD/fizz/internal/plan"
)

const (
	want = `{"a":1,"b":{"c":[1,2],"d":"x"},"e":true}`
	got  = `{"a":2,"b":{"c":[1],"z":0},"f":1}`
)

func problems(t *testing.T, got, want string, allow []repro.Allow) string {
	t.Helper()
	return strings.Join(repro.Check([]byte(got), []byte(want), allow), "\n")
}

func TestDiffs(t *testing.T) {
	d, err := repro.Diffs([]byte(got), []byte(want))
	if err != nil {
		t.Fatal(err)
	}
	var lines []string
	for _, x := range d {
		lines = append(lines, string(x.Kind)+" "+x.Pointer)
	}
	exp := "changed /a|missing /b/c/1|missing /b/d|extra /b/z|missing /e|extra /f"
	if s := strings.Join(lines, "|"); s != exp {
		t.Fatalf("diffs = %s\nwant    %s", s, exp)
	}
}

func TestUnmatchedDiffFails(t *testing.T) {
	p := problems(t, got, want, nil)
	if !strings.Contains(p, "6 difference(s) not in the allowlist") || !strings.Contains(p, `"pointer": "/b/d"`) {
		t.Fatalf("no failure or snippet:\n%s", p)
	}
	if p := problems(t, want, want, nil); p != "" {
		t.Fatalf("identical documents: %s", p)
	}
}

func TestUnusedEntryFails(t *testing.T) {
	allow := []repro.Allow{{Pointer: "/nope", Kind: repro.Missing, Category: repro.Naming, Reason: "x"}}
	if p := problems(t, want, want, allow); !strings.Contains(p, "matches no difference") {
		t.Fatalf("unused entry accepted: %q", p)
	}
	// The kind is part of the match.
	allow = []repro.Allow{{Pointer: "/e", Kind: repro.Extra, Category: repro.Naming, Reason: "x"}}
	p := problems(t, `{}`, `{"e":1}`, allow)
	if !strings.Contains(p, "matches no difference") || !strings.Contains(p, "1 difference(s)") {
		t.Fatalf("wrong kind accepted: %q", p)
	}
}

func TestSubtreeMatch(t *testing.T) {
	allow := []repro.Allow{
		{Pointer: "/b/**", Kind: repro.Any, Category: repro.Naming, Reason: "whole b"},
		{Pointer: "/a", Kind: repro.Changed, Category: repro.Naming, Reason: "a"},
		{Pointer: "/e", Kind: repro.Missing, Category: repro.Naming, Reason: "e"},
		{Pointer: "/f", Kind: repro.Extra, Category: repro.Naming, Reason: "f"},
	}
	if p := problems(t, got, want, allow); p != "" {
		t.Fatalf("subtree entry did not cover /b: %s", p)
	}
	// "/b/**" does not cover a sibling that merely shares the prefix.
	allow = []repro.Allow{{Pointer: "/b/**", Kind: repro.Any, Category: repro.Naming, Reason: "r"}}
	if p := problems(t, `{"bb":1}`, `{}`, allow); !strings.Contains(p, "/bb") {
		t.Fatalf("/b/** matched /bb: %s", p)
	}
	// Any needs a subtree pointer.
	allow = []repro.Allow{{Pointer: "/a", Kind: repro.Any, Category: repro.Naming, Reason: "r"}}
	if p := problems(t, got, want, allow); !strings.Contains(p, "needs a pointer ending in /**") {
		t.Fatalf("kind any on an exact pointer accepted: %s", p)
	}
}

// A limitation whose task is done is stale: the fix landed, so the entry
// must go or be reclassified.
func TestStaleLimitationFails(t *testing.T) {
	old := plan.Done
	t.Cleanup(func() { plan.Done = old })
	plan.Done = map[string]bool{"P10.9a": true}

	lim := func(task string) repro.Allow {
		return repro.Allow{Pointer: "/a", Kind: repro.Changed, Category: repro.Limitation, Task: task, Reason: "r"}
	}
	if err := lim("P10.9a").Validate(); err == nil || !strings.Contains(err.Error(), "stale: task P10.9a done; remove or reclassify") {
		t.Fatalf("done task accepted: %v", err)
	}
	if err := lim("P10.9b").Validate(); err != nil {
		t.Fatalf("pending task rejected: %v", err)
	}
	if p := problems(t, `{"a":1}`, `{"a":2}`, []repro.Allow{lim("P10.9a")}); !strings.Contains(p, "stale") {
		t.Fatalf("Check did not report the stale entry: %q", p)
	}
}

func TestEntryRules(t *testing.T) {
	ok := repro.Allow{Pointer: "/a", Kind: repro.Changed, Category: repro.Naming, Reason: "r"}
	tests := []struct {
		name string
		mod  func(a *repro.Allow)
		want string // "" = valid
	}{
		{"valid", func(*repro.Allow) {}, ""},
		{"no reason", func(a *repro.Allow) { a.Reason = " " }, "reason is required"},
		{"limitation without task", func(a *repro.Allow) { a.Category = repro.Limitation }, "needs a plan task ID"},
		{"limitation bad task", func(a *repro.Allow) { a.Category, a.Task = repro.Limitation, "later" }, "needs a plan task ID"},
		{"limitation with task", func(a *repro.Allow) { a.Category, a.Task = repro.Limitation, "P10.9a" }, ""},
		{"task on naming", func(a *repro.Allow) { a.Task = "P10.9a" }, "only for a limitation"},
		{"upstream without proof", func(a *repro.Allow) { a.Category = repro.Upstream }, "link"},
		{"upstream with url", func(a *repro.Allow) {
			a.Category, a.Reason = repro.Upstream, "see https://example.com/issue/1"
		}, ""},
		{"upstream with quote", func(a *repro.Allow) {
			a.Category, a.Reason = repro.Upstream, `the example says "oops"`
		}, ""},
		{"bad category", func(a *repro.Allow) { a.Category = "whim" }, "category"},
		{"bad kind", func(a *repro.Allow) { a.Kind = "gone" }, "kind"},
		{"bad pointer", func(a *repro.Allow) { a.Pointer = "a" }, "does not start with /"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := ok
			tt.mod(&a)
			err := a.Validate()
			switch {
			case tt.want == "" && err != nil:
				t.Fatalf("unexpected error: %v", err)
			case tt.want != "" && (err == nil || !strings.Contains(err.Error(), tt.want)):
				t.Fatalf("error = %v, want one containing %q", err, tt.want)
			}
		})
	}
	// An invalid entry fails the comparison even when it matches.
	bad := []repro.Allow{{Pointer: "/a", Kind: repro.Changed, Category: repro.Limitation, Reason: "r"}}
	if p := problems(t, `{"a":1}`, `{"a":2}`, bad); !strings.Contains(p, "needs a plan task ID") {
		t.Fatalf("limitation without task accepted: %q", p)
	}
}

func TestLoadAllow(t *testing.T) {
	path := filepath.Join(t.TempDir(), "allow.json")
	body := `[{"pointer":"/a","kind":"changed","category":"limitation","reason":"r","task":"P10.9a"}]`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	a := repro.LoadAllow(t, path)
	if len(a) != 1 || a[0].Task != "P10.9a" || a[0].Category != repro.Limitation {
		t.Fatalf("loaded %+v", a)
	}
}
