// Package at is the acceptance-test harness: the red-until-done helper
// Pending and small helpers to drive a fizz under test and assert on
// what it serves.
//
// Acceptance tests wrap their assertions in Pending, naming the plan
// task that makes them pass. See docs/plan/ACCEPTANCE.md.
package at

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
	"golang.org/x/net/html"

	"github.com/KeisukeFD/fizz"
	"github.com/KeisukeFD/fizz/internal/kitchensink"
	"github.com/KeisukeFD/fizz/internal/plan"
)

// Pending runs fn, which the task fixedBy has yet to make pass.
//
//   - fn fails, task not done: t is skipped ("fixed by X: <first failure>").
//   - fn fails, task in plan.Done: t fails ("task done, test red").
//   - fn passes, task not done: t is skipped ("passes now: drop FixedBy"),
//     or fails when FIZZ_STALE=fail.
//   - fn passes, task done: green.
//
// Inside fn, Fatal aborts fn and a panic is recorded as a failure.
// A data race cannot be recovered; gate it with plan.IsDone.
func Pending(t testing.TB, fixedBy string, fn func(tb testing.TB)) {
	t.Helper()
	plan.Pending(t, fixedBy, fn)
}

// ReqOpt customizes the request built by Do.
type ReqOpt func(*http.Request)

// WithBody sets the request body and its Content-Type.
func WithBody(contentType, body string) ReqOpt {
	return func(r *http.Request) {
		r.Body = io.NopCloser(strings.NewReader(body))
		r.ContentLength = int64(len(body))
		r.GetBody = func() (io.ReadCloser, error) { return io.NopCloser(strings.NewReader(body)), nil }
		r.Header.Set("Content-Type", contentType)
	}
}

// WithCookie adds a cookie with the raw value (no encoding).
func WithCookie(name, value string) ReqOpt {
	return func(r *http.Request) { r.AddCookie(&http.Cookie{Name: name, Value: value}) }
}

// WithHeader adds a request header.
func WithHeader(key, value string) ReqOpt {
	return func(r *http.Request) { r.Header.Add(key, value) }
}

// Do serves one request on h and returns the recorded response.
// target may carry a raw query string, which is sent as written.
func Do(h http.Handler, method, target string, opts ...ReqOpt) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, target, nil)
	for _, o := range opts {
		o(r)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

// Spec returns the JSON document of f (Info set to a placeholder). It
// fails tb when the document cannot be built and, once plan.Done["P2.6"],
// when it breaks the meta-schema or a lint.
func Spec(tb testing.TB, f *fizz.Fizz) []byte {
	tb.Helper()
	doc, err := kitchensink.BuildSpec(f)
	if err != nil {
		tb.Fatalf("build spec: %v", err)
	}
	if plan.IsDone("P2.6") {
		for _, p := range kitchensink.MetaProblems(doc) {
			tb.Error(p)
		}
	}
	return doc
}

// Frag asserts that the value at pointer (RFC 6901) in doc equals the
// JSON text want, semantically.
func Frag(tb testing.TB, doc []byte, pointer, want string) {
	tb.Helper()
	kitchensink.AssertFragments(tb, doc, []kitchensink.Fragment{{Pointer: pointer, JSON: want}})
}

// Absent asserts that pointer does not resolve in doc.
func Absent(tb testing.TB, doc []byte, pointer string) {
	tb.Helper()
	kitchensink.AssertFragments(tb, doc, []kitchensink.Fragment{{Pointer: pointer, Absent: true}})
}

// YAMLToAny decodes YAML with yaml.v3 and normalizes the result through
// JSON, so it deep-equals a json.Unmarshal into any (numbers float64).
// An unquoted timestamp therefore decodes as a time string, not as the
// original text.
func YAMLToAny(tb testing.TB, b []byte) any {
	tb.Helper()
	var v any
	if err := yaml.Unmarshal(b, &v); err != nil {
		tb.Fatalf("decode yaml: %v", err)
	}
	j, err := json.Marshal(v)
	if err != nil {
		tb.Fatalf("yaml value is not JSON-compatible: %v", err)
	}
	var out any
	if err := json.Unmarshal(j, &out); err != nil {
		tb.Fatalf("re-decode: %v", err)
	}
	return out
}

// FromJSON unmarshals s into a T. It panics on invalid JSON (a bug in
// the test), which Pending records as a failure. It compiles against any
// shape of T, which is why acceptance tests build models this way.
func FromJSON[T any](s string) T {
	var v T
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		panic(fmt.Sprintf("at.FromJSON[%T]: %v", v, err))
	}
	return v
}

// Elem is an HTML element, with the text of its subtree.
type Elem struct {
	Tag  string
	Attr map[string]string
	Text string
}

// HTML parses body and returns every element in document order.
func HTML(body []byte) []Elem {
	doc, err := html.Parse(strings.NewReader(string(body)))
	if err != nil {
		panic(fmt.Sprintf("at.HTML: %v", err))
	}
	var out []Elem
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			e := Elem{Tag: n.Data, Attr: map[string]string{}, Text: text(n)}
			for _, a := range n.Attr {
				e.Attr[a.Key] = a.Val
			}
			out = append(out, e)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return out
}

// Tags returns the elements of es with the given tag name.
func Tags(es []Elem, tag string) []Elem {
	var out []Elem
	for _, e := range es {
		if e.Tag == tag {
			out = append(out, e)
		}
	}
	return out
}

func text(n *html.Node) string {
	var sb strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			sb.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return sb.String()
}
