// Package repro is the harness of the reproduction apps (docs/plan/PLAN-v2.md,
// P10.5, decision D23). Each app under this directory rebuilds an official
// OpenAPI example with the public fizz API; the harness compares the spec it
// serves with the upgraded official document.
//
// A difference is a JSON pointer plus a kind (missing, extra, changed). It
// must be covered by an entry of the example's allow.json, which carries a
// category and a reason. A difference without an entry fails, and so does an
// entry that covers no difference, so the list stays tight.
package repro

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/KeisukeFD/fizz"
	"github.com/KeisukeFD/fizz/internal/kitchensink"
	"github.com/KeisukeFD/fizz/internal/plan"
)

// Kind tells how a difference relates the served document (got) to the
// official one (want).
type Kind string

const (
	// Missing: the official document has the value, fizz does not.
	Missing Kind = "missing"
	// Extra: fizz emits a value the official document does not have.
	Extra Kind = "extra"
	// Changed: both have a value at the pointer, and they differ.
	Changed Kind = "changed"
	// Any matches every kind. Only a subtree entry (pointer ending in
	// "/**") may use it.
	Any Kind = "any"
)

// Category is why a difference is accepted.
type Category string

const (
	// Naming is something fizz controls on purpose (ADR 0001), with no
	// override: component names, operation ID format, ...
	Naming Category = "naming"
	// Upstream is a defect of the official example. The reason links it
	// (an http URL) or quotes it (between quotation marks).
	Upstream Category = "upstream"
	// Limitation is something fizz cannot express. It names the plan task
	// (Task, "P10.9a") that tracks it.
	Limitation Category = "limitation"
)

// Allow accepts the differences at Pointer of the given Kind.
//
// Pointer is an RFC 6901 pointer matched exactly, or a prefix followed by
// "/**" that matches the pointer itself and everything under it.
type Allow struct {
	Pointer  string   `json:"pointer"`
	Kind     Kind     `json:"kind"`
	Category Category `json:"category"`
	Reason   string   `json:"reason"`
	Task     string   `json:"task,omitempty"`
}

// Diff is one difference between two documents.
type Diff struct {
	Pointer string
	Kind    Kind
	// Got and Want are the JSON of each side, "" when absent.
	Got, Want string
}

func (d Diff) String() string {
	switch d.Kind {
	case Missing:
		return fmt.Sprintf("missing %s (want %s)", ptrOrRoot(d.Pointer), clip(d.Want))
	case Extra:
		return fmt.Sprintf("extra   %s (got %s)", ptrOrRoot(d.Pointer), clip(d.Got))
	}
	return fmt.Sprintf("changed %s: got %s, want %s", ptrOrRoot(d.Pointer), clip(d.Got), clip(d.Want))
}

func ptrOrRoot(p string) string {
	if p == "" {
		return "(root)"
	}
	return p
}

func clip(s string) string {
	if len(s) > 120 {
		return s[:117] + "..."
	}
	return s
}

// Diffs returns the differences between the JSON documents got and want,
// sorted by pointer. A value that exists on one side only is reported once,
// at its own pointer, without descending. Arrays are compared by index.
func Diffs(got, want []byte) ([]Diff, error) {
	g, err := decode(got)
	if err != nil {
		return nil, fmt.Errorf("got: %w", err)
	}
	w, err := decode(want)
	if err != nil {
		return nil, fmt.Errorf("want: %w", err)
	}
	var out []Diff
	walk(&out, "", g, w)
	slices.SortStableFunc(out, func(a, b Diff) int { return strings.Compare(a.Pointer, b.Pointer) })
	return out, nil
}

func decode(b []byte) (any, error) {
	var v any
	dec := json.NewDecoder(bytes.NewReader(b))
	if err := dec.Decode(&v); err != nil {
		return nil, err
	}
	if dec.More() {
		return nil, fmt.Errorf("trailing data after the JSON document")
	}
	return v, nil
}

func walk(out *[]Diff, ptr string, got, want any) {
	switch w := want.(type) {
	case map[string]any:
		if g, ok := got.(map[string]any); ok {
			keys := make(map[string]struct{}, len(g)+len(w))
			for k := range g {
				keys[k] = struct{}{}
			}
			for k := range w {
				keys[k] = struct{}{}
			}
			for k := range keys {
				p := ptr + "/" + escape(k)
				gv, gok := g[k]
				wv, wok := w[k]
				switch {
				case gok && wok:
					walk(out, p, gv, wv)
				case wok:
					*out = append(*out, Diff{Pointer: p, Kind: Missing, Want: show(wv)})
				default:
					*out = append(*out, Diff{Pointer: p, Kind: Extra, Got: show(gv)})
				}
			}
			return
		}
	case []any:
		if g, ok := got.([]any); ok {
			for i := range max(len(g), len(w)) {
				p := ptr + "/" + strconv.Itoa(i)
				switch {
				case i >= len(g):
					*out = append(*out, Diff{Pointer: p, Kind: Missing, Want: show(w[i])})
				case i >= len(w):
					*out = append(*out, Diff{Pointer: p, Kind: Extra, Got: show(g[i])})
				default:
					walk(out, p, g[i], w[i])
				}
			}
			return
		}
	default:
		if reflect.DeepEqual(got, want) {
			return
		}
	}
	*out = append(*out, Diff{Pointer: ptr, Kind: Changed, Got: show(got), Want: show(want)})
}

func show(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprint(v)
	}
	return string(b)
}

func escape(s string) string {
	return strings.NewReplacer("~", "~0", "/", "~1").Replace(s)
}

// Matches reports whether the entry covers d.
func (a Allow) Matches(d Diff) bool {
	if a.Kind != Any && a.Kind != d.Kind {
		return false
	}
	if prefix, ok := strings.CutSuffix(a.Pointer, "/**"); ok {
		return d.Pointer == prefix || strings.HasPrefix(d.Pointer, prefix+"/")
	}
	return a.Pointer == d.Pointer
}

var taskRE = regexp.MustCompile(`^P\d+\.\d+[a-z]?$`)

// Validate reports what is wrong with the entry itself.
func (a Allow) Validate() error {
	switch a.Kind {
	case Missing, Extra, Changed:
	case Any:
		if !strings.HasSuffix(a.Pointer, "/**") {
			return fmt.Errorf("kind %q needs a pointer ending in /**", Any)
		}
	default:
		return fmt.Errorf("kind %q is not missing, extra or changed", a.Kind)
	}
	if a.Pointer != "" && !strings.HasPrefix(a.Pointer, "/") {
		return fmt.Errorf("pointer %q does not start with /", a.Pointer)
	}
	if strings.TrimSpace(a.Reason) == "" {
		return fmt.Errorf("a reason is required")
	}
	switch a.Category {
	case Naming:
	case Upstream:
		if !strings.Contains(a.Reason, "http://") && !strings.Contains(a.Reason, "https://") &&
			!strings.ContainsAny(a.Reason, "\"“") {
			return fmt.Errorf("an upstream reason must link (http URL) or quote the official example")
		}
	case Limitation:
		if !taskRE.MatchString(a.Task) {
			return fmt.Errorf("a limitation needs a plan task ID (P10.9a, P10.7c, ...), got %q", a.Task)
		}
		if plan.IsDone(a.Task) {
			return fmt.Errorf("stale: task %s done; remove or reclassify", a.Task)
		}
	default:
		return fmt.Errorf("category %q is not naming, upstream or limitation", a.Category)
	}
	if a.Category != Limitation && a.Task != "" {
		return fmt.Errorf("task is only for a limitation")
	}
	return nil
}

// Check compares got with want under the allowlist and returns the
// problems found: invalid entries, unmatched differences (followed by a
// ready-to-paste allowlist snippet) and entries that match nothing. It
// returns nil when the documents agree up to the allowlist.
func Check(got, want []byte, allow []Allow) []string {
	var problems []string
	for i, a := range allow {
		if err := a.Validate(); err != nil {
			problems = append(problems, fmt.Sprintf("allow[%d] (%s %s): %v", i, a.Kind, a.Pointer, err))
		}
	}
	diffs, err := Diffs(got, want)
	if err != nil {
		return append(problems, err.Error())
	}
	used := make([]bool, len(allow))
	var unmatched []Diff
	for _, d := range diffs {
		matched := false
		for i, a := range allow {
			if a.Matches(d) {
				used[i], matched = true, true
			}
		}
		if !matched {
			unmatched = append(unmatched, d)
		}
	}
	for i, a := range allow {
		if !used[i] {
			problems = append(problems, fmt.Sprintf("allow[%d] (%s %s) matches no difference: remove it", i, a.Kind, a.Pointer))
		}
	}
	if len(unmatched) > 0 {
		var sb strings.Builder
		fmt.Fprintf(&sb, "%d difference(s) not in the allowlist:\n", len(unmatched))
		snippet := make([]Allow, 0, len(unmatched))
		for _, d := range unmatched {
			fmt.Fprintf(&sb, "  %s\n", d)
			snippet = append(snippet, Allow{Pointer: d.Pointer, Kind: d.Kind, Category: "naming|upstream|limitation", Reason: "TODO"})
		}
		b, _ := json.MarshalIndent(snippet, "", "  ")
		sb.WriteString("allowlist snippet to complete and paste into allow.json:\n")
		sb.Write(b)
		problems = append(problems, sb.String())
	}
	return problems
}

// Compare fails tb for each problem Check finds.
func Compare(tb testing.TB, got, want []byte, allow []Allow) {
	tb.Helper()
	for _, p := range Check(got, want, allow) {
		tb.Error(p)
	}
}

// LoadAllow reads an allowlist file (a JSON array of Allow). Unknown
// fields are an error. A missing file is a failure of tb.
func LoadAllow(tb testing.TB, path string) []Allow {
	tb.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		tb.Fatalf("allowlist: %v", err)
	}
	var allow []Allow
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&allow); err != nil {
		tb.Fatalf("allowlist %s: %v", path, err)
	}
	return allow
}

// Official returns the upgraded official document of the given path under
// oai/upgraded/learn/examples, e.g. "v3.0/petstore.json".
func Official(tb testing.TB, rel string) []byte {
	tb.Helper()
	return OfficialAt(tb, "learn/examples/"+rel)
}

// OfficialAt returns the upgraded official document of the given path
// under oai/upgraded, e.g. "spectext/base-uri-within-content-1.json".
func OfficialAt(tb testing.TB, rel string) []byte {
	tb.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		tb.Fatal("cannot locate the repro package")
	}
	p := filepath.Join(filepath.Dir(file), "..", "oai", "upgraded", filepath.FromSlash(rel))
	b, err := os.ReadFile(p)
	if err != nil {
		tb.Fatalf("official document: %v", err)
	}
	return b
}

// Serve serves the spec of f over HTTP and returns the JSON it fetched.
// The app must have set its Info (Generator().SetInfo). Serve fails tb
// when the generator recorded an error (warnings are logged), when the
// response is not 200, or when the document is not valid OAS 3.2.1.
func Serve(tb testing.TB, f *fizz.Fizz) []byte {
	tb.Helper()
	if err := f.Err(); err != nil {
		tb.Fatalf("fizz errors: %v", err)
	}
	for _, e := range f.Errors() {
		tb.Logf("fizz warning: %v", e)
	}
	srv := httptest.NewServer(f.SpecHandler("json"))
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/openapi.json")
	if err != nil {
		tb.Fatalf("GET /openapi.json: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		tb.Fatalf("read spec: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		tb.Fatalf("GET /openapi.json = %d: %s", resp.StatusCode, body)
	}
	for _, p := range kitchensink.MetaProblems(body) {
		tb.Errorf("served spec is invalid: %s", p)
	}
	return body
}

// Run is the whole check of an example: it serves f, loads the official
// document official (see Official) and the allowlist at allowPath, and
// compares them.
func Run(tb testing.TB, f *fizz.Fizz, official, allowPath string) {
	tb.Helper()
	got := Serve(tb, f)
	Compare(tb, got, Official(tb, official), LoadAllow(tb, allowPath))
}

// RunAt is Run for a document under oai/upgraded (see OfficialAt).
func RunAt(tb testing.TB, f *fizz.Fizz, official, allowPath string) {
	tb.Helper()
	got := Serve(tb, f)
	Compare(tb, got, OfficialAt(tb, official), LoadAllow(tb, allowPath))
}
