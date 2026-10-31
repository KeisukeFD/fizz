package golden

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	flag.Parse()
	// Tests exercise the compare path; never rewrite files here.
	*update = false
	os.Exit(m.Run())
}

// recorder captures failures instead of failing the real test.
type recorder struct {
	testing.TB
	msgs []string
}

func (r *recorder) Helper() {}
func (r *recorder) Errorf(format string, args ...any) {
	r.msgs = append(r.msgs, fmt.Sprintf(format, args...))
}

func write(t *testing.T, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "g.json")
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestAssertReportsPointer(t *testing.T) {
	p := write(t, `{"paths":{"/x":{"get":{"summary":"a"}}}}`)
	r := &recorder{TB: t}
	Assert(r, p, []byte(`{"paths":{"/x":{"get":{"summary":"b"}}}}`))
	if len(r.msgs) != 1 || !strings.Contains(r.msgs[0], "/paths/~1x/get") {
		t.Fatalf("message does not name pointer: %q", r.msgs)
	}
}

func TestAssertEqualJSONIgnoresFormatting(t *testing.T) {
	p := write(t, "{\n  \"a\": [1, 2]\n}\n")
	r := &recorder{TB: t}
	Assert(r, p, []byte(`{"a":[1,2]}`))
	if len(r.msgs) != 0 {
		t.Fatalf("unexpected failure: %q", r.msgs)
	}
}

func TestAssertArrayAndMissingKey(t *testing.T) {
	p := write(t, `{"a":[1,2],"b":1}`)
	r := &recorder{TB: t}
	Assert(r, p, []byte(`{"a":[1,3]}`))
	if len(r.msgs) != 1 || !strings.Contains(r.msgs[0], "/a/1") {
		t.Fatalf("got %q", r.msgs)
	}
}

func TestAssertRawContent(t *testing.T) {
	p := filepath.Join(t.TempDir(), "g.yaml")
	os.WriteFile(p, []byte("a: 1\n"), 0o644)
	r := &recorder{TB: t}
	Assert(r, p, []byte("a: 2\n"))
	if len(r.msgs) != 1 {
		t.Fatalf("expected mismatch, got %q", r.msgs)
	}
	r = &recorder{TB: t}
	Assert(r, p, []byte("a: 1\n"))
	if len(r.msgs) != 0 {
		t.Fatalf("unexpected: %q", r.msgs)
	}
}

func TestEscape(t *testing.T) {
	if got := escape("a/b~c"); got != "a~1b~0c" {
		t.Fatal(got)
	}
}
