package kitchensink

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/KeisukeFD/fizz"
	"github.com/KeisukeFD/fizz/openapi"
)

// resolve follows an RFC 6901 pointer in v.
func resolve(v any, ptr string) (any, bool) {
	if ptr == "" {
		return v, true
	}
	if !strings.HasPrefix(ptr, "/") {
		return nil, false
	}
	for tok := range strings.SplitSeq(ptr[1:], "/") {
		tok = strings.NewReplacer("~1", "/", "~0", "~").Replace(tok)
		switch c := v.(type) {
		case map[string]any:
			next, ok := c[tok]
			if !ok {
				return nil, false
			}
			v = next
		case []any:
			i, err := strconv.Atoi(tok)
			if err != nil || i < 0 || i >= len(c) {
				return nil, false
			}
			v = c[i]
		default:
			return nil, false
		}
	}
	return v, true
}

// AssertFragments reports, through tb.Errorf, every fragment that does not
// hold in doc.
func AssertFragments(tb testing.TB, doc []byte, frags []Fragment) {
	tb.Helper()
	var root any
	if err := json.Unmarshal(doc, &root); err != nil {
		tb.Errorf("invalid spec JSON: %v", err)
		return
	}
	for _, fr := range frags {
		got, ok := resolve(root, fr.Pointer)
		if fr.Absent {
			if ok {
				tb.Errorf("pointer %q must be absent, got %s", fr.Pointer, show(got))
			}
			continue
		}
		if !ok {
			tb.Errorf("pointer %q not found", fr.Pointer)
			continue
		}
		var want any
		if err := json.Unmarshal([]byte(fr.JSON), &want); err != nil {
			tb.Errorf("pointer %q: invalid expected JSON: %v", fr.Pointer, err)
			continue
		}
		// Reuse the golden comparison: it is semantic and names the
		// first differing sub-pointer.
		if d := diff(want, got); d != "" {
			tb.Errorf("pointer %q: %s\nwant: %s\ngot:  %s", fr.Pointer, d, show(want), show(got))
		}
	}
}

// checkWarnings reports every substring that no warning in errs contains.
// Only entries implementing IsWarning() bool and returning true count.
func checkWarnings(tb testing.TB, errs []error, subs []string) {
	tb.Helper()
	var warnings []error
	for _, e := range errs {
		var w interface{ IsWarning() bool }
		if errors.As(e, &w) && w.IsWarning() {
			warnings = append(warnings, e)
		}
	}
	for _, s := range subs {
		found := false
		for _, e := range warnings {
			if strings.Contains(e.Error(), s) {
				found = true
				break
			}
		}
		if !found {
			tb.Errorf("Errors() has no warning containing %q (got %v)", s, errs)
		}
	}
}

// checkErrors reports every substring missing from errs.
func checkErrors(tb testing.TB, errs []error, subs []string) {
	tb.Helper()
	for _, s := range subs {
		found := false
		for _, e := range errs {
			if strings.Contains(e.Error(), s) {
				found = true
				break
			}
		}
		if !found {
			tb.Errorf("Errors() has no entry containing %q (got %v)", s, errs)
		}
	}
}

func show(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprint(v)
	}
	return string(b)
}

// diff returns "" when a and b are semantically equal JSON values,
// otherwise a message naming the first differing sub-pointer.
func diff(a, b any) string {
	if p, ok := firstDiff(a, b, ""); ok {
		return fmt.Sprintf("differs at sub-pointer %q", p)
	}
	return ""
}

func firstDiff(a, b any, ptr string) (string, bool) {
	switch av := a.(type) {
	case map[string]any:
		if bv, ok := b.(map[string]any); ok {
			keys := slices.Sorted(maps.Keys(av))
			for k := range bv {
				if _, in := av[k]; !in {
					keys = append(keys, k)
				}
			}
			for _, k := range keys {
				p := ptr + "/" + strings.NewReplacer("~", "~0", "/", "~1").Replace(k)
				x, xok := av[k]
				y, yok := bv[k]
				if !xok || !yok {
					return p, true
				}
				if q, d := firstDiff(x, y, p); d {
					return q, true
				}
			}
			return "", false
		}
	case []any:
		if bv, ok := b.([]any); ok {
			for i := range max(len(av), len(bv)) {
				p := ptr + "/" + strconv.Itoa(i)
				if i >= len(av) || i >= len(bv) {
					return p, true
				}
				if q, d := firstDiff(av[i], bv[i], p); d {
					return q, true
				}
			}
			return "", false
		}
	default:
		if reflect.DeepEqual(a, b) {
			return "", false
		}
	}
	return ptr, true
}

// BuildSpec marshals the document of f. It reads the generator's API
// directly instead of serving OpenAPI() through httptest: the handler
// only wraps the same value in gin's renderer, so going through it
// would couple every case to gin and to the handler's future shape
// (P6) for no extra coverage.
func BuildSpec(f *fizz.Fizz) ([]byte, error) {
	f.Generator().SetInfo(&openapi.Info{Title: "kitchensink", Version: "0"})
	return json.Marshal(f.Generator().API())
}

// recorder collects failures instead of reporting them, so the runner
// can decide whether a failing case is a regression or a known bug.
// It also serves as the fake testing.TB in the harness's own tests.
type recorder struct {
	testing.TB
	msgs []string
}

func (r *recorder) Helper() {}
func (r *recorder) Errorf(format string, args ...any) {
	r.msgs = append(r.msgs, fmt.Sprintf(format, args...))
}
