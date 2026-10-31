package bind_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/wI2L/fizz/bind"
)

var rwTags = bind.Tags{Query: "query", Cookie: "cookie", Style: "style", Explode: "explode", Default: "default"}

type listIn struct {
	Space []string `query:"space" style:"spaceDelimited"`
	Pipe  []int    `query:"pipe" style:"pipeDelimited"`
	// Documented exploded: tonic's repeated form, left alone.
	Rep   []string `query:"rep" style:"pipeDelimited" explode:"true"`
	Plain []string `query:"plain"`
	// Unexploded for tonic too: one comma-separated value.
	Comma [2]string `query:"comma" style:"spaceDelimited" explode:"false"`
	Ptr   *[]string `query:"ptr" style:"pipeDelimited"`
	Bytes []byte    `query:"bytes" style:"pipeDelimited"` // not an array
	Str   string    `query:"str" style:"pipeDelimited"`   // not an array
}

func TestRewriteQuery(t *testing.T) {
	p, err := bind.PlanFor(reflect.TypeFor[listIn](), rwTags)
	if err != nil {
		t.Fatal(err)
	}
	rows := []struct {
		name, in, want string
		wantErr        string
	}{
		{"space %20", "space=a%20b%20c", "space=a&space=b&space=c", ""},
		{"space plus", "space=a+b", "space=a&space=b", ""},
		{"space encoded element", "space=a%2Cb%20c", "space=a%2Cb&space=c", ""},
		{"pipe", "pipe=1|2|3", "pipe=1&pipe=2&pipe=3", ""},
		{"pipe encoded", "pipe=1%7C2", "pipe=1&pipe=2", ""},
		{"single element", "pipe=7", "pipe=7", ""},
		{"empty value", "space=&x=1", "x=1", ""},
		{"empty elements dropped", "pipe=1||2|", "pipe=1&pipe=2", ""},
		{"keeps the other pairs", "a=1&space=x%20y&b=%20z", "a=1&space=x&space=y&b=%20z", ""},
		{"encoded name", "sp%61ce=a%20b", "space=a&space=b", ""},
		{"exploded style is untouched", "rep=a|b", "rep=a|b", ""},
		{"plain is untouched", "plain=a%20b", "plain=a%20b", ""},
		{"not an array is untouched", "str=a|b&bytes=a|b", "str=a|b&bytes=a|b", ""},
		{"no planned key", "x=1", "x=1", ""},
		{"explode false joins with commas", "comma=a%20b", "comma=a%2Cb", ""},
		{"explode false rejects a comma", "comma=a,b%20c", "", `"comma"`},
		{"pointer to slice", "ptr=a|b", "ptr=a&ptr=b", ""},
		{"bad escape is left to tonic", "space=%zz", "space=%zz", ""},
	}
	for _, r := range rows {
		t.Run(r.name, func(t *testing.T) {
			got, changed, err := bind.RewriteQuery(r.in, p)
			if r.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), r.wantErr) {
					t.Fatalf("err = %v, want one containing %q", err, r.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != r.want {
				t.Errorf("RewriteQuery(%q) = %q, want %q", r.in, got, r.want)
			}
			if changed != (got != r.in) {
				t.Errorf("changed = %v for %q -> %q", changed, r.in, got)
			}
		})
	}
}

func TestRewriteQueryNoPlan(t *testing.T) {
	if got, changed, err := bind.RewriteQuery("space=a%20b", nil); got != "space=a%20b" || changed || err != nil {
		t.Errorf("nil plan: %q %v %v", got, changed, err)
	}
	type plain struct {
		Q []string `query:"q"`
	}
	p, err := bind.PlanFor(reflect.TypeFor[plain](), rwTags)
	if err != nil {
		t.Fatal(err)
	}
	if !p.Empty() {
		t.Error("a plain query array must not be planned")
	}
}
