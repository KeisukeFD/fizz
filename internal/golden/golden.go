// Package golden compares generated output against golden files.
//
// Run a test binary with -update to rewrite the golden files.
package golden

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "rewrite golden files")

// Assert compares got with the golden file at path.
//
// JSON content is canonically indented (four spaces, trailing newline)
// and compared semantically, so object key order is irrelevant; a
// mismatch names the first differing JSON pointer (RFC 6901). Any
// other content is compared byte-for-byte. With -update the file is
// rewritten instead, unless it is JSON that is already semantically equal.
func Assert(t testing.TB, path string, got []byte) {
	t.Helper()

	gotVal, gotIsJSON := decode(got)
	if gotIsJSON {
		var buf bytes.Buffer
		if err := json.Indent(&buf, got, "", "    "); err != nil {
			t.Errorf("golden: indent %s: %v", path, err)
			return
		}
		buf.WriteByte('\n')
		got = buf.Bytes()
	}
	if *update {
		// Leave semantically identical JSON goldens untouched so that
		// -update does not churn key order.
		if old, err := os.ReadFile(path); err == nil && gotIsJSON {
			if oldVal, ok := decode(old); ok {
				if _, _, _, diff := firstDiff(oldVal, gotVal, ""); !diff {
					return
				}
			}
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Errorf("golden: %v", err)
			return
		}
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Errorf("golden: %v", err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Errorf("golden: %v (run with -update to create it)", err)
		return
	}
	if wantVal, ok := decode(want); ok && gotIsJSON {
		if ptr, w, g, diff := firstDiff(wantVal, gotVal, ""); diff {
			t.Errorf("golden: %s differs at JSON pointer %q\nwant: %s\ngot:  %s", path, ptr, w, g)
		}
		return
	}
	if !bytes.Equal(want, got) {
		t.Errorf("golden: %s differs\nwant:\n%s\ngot:\n%s", path, want, got)
	}
}

func decode(b []byte) (any, bool) {
	var v any
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	if err := dec.Decode(&v); err != nil || dec.More() {
		return nil, false
	}
	return v, true
}

// firstDiff returns the pointer of the first difference between a and b,
// visiting object keys in sorted order.
func firstDiff(a, b any, ptr string) (string, string, string, bool) {
	switch av := a.(type) {
	case map[string]any:
		if bv, ok := b.(map[string]any); ok {
			keys := make([]string, 0, len(av)+len(bv))
			for k := range av {
				keys = append(keys, k)
			}
			for k := range bv {
				if _, ok := av[k]; !ok {
					keys = append(keys, k)
				}
			}
			slices.Sort(keys)
			for _, k := range keys {
				x, xok := av[k]
				y, yok := bv[k]
				p := ptr + "/" + escape(k)
				if !xok || !yok {
					return p, show(x, xok), show(y, yok), true
				}
				if p, w, g, d := firstDiff(x, y, p); d {
					return p, w, g, true
				}
			}
			return "", "", "", false
		}
	case []any:
		if bv, ok := b.([]any); ok {
			for i := range max(len(av), len(bv)) {
				p := ptr + "/" + strconv.Itoa(i)
				if i >= len(av) || i >= len(bv) {
					var x, y any
					if i < len(av) {
						x = av[i]
					}
					if i < len(bv) {
						y = bv[i]
					}
					return p, show(x, i < len(av)), show(y, i < len(bv)), true
				}
				if p, w, g, d := firstDiff(av[i], bv[i], p); d {
					return p, w, g, true
				}
			}
			return "", "", "", false
		}
	default:
		if reflect.DeepEqual(a, b) {
			return "", "", "", false
		}
	}
	return ptr, show(a, true), show(b, true), true
}

func show(v any, present bool) string {
	if !present {
		return "<missing>"
	}
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprint(v)
	}
	return string(b)
}

// escape applies RFC 6901 escaping to a reference token.
func escape(s string) string {
	return strings.NewReplacer("~", "~0", "/", "~1").Replace(s)
}
