package bind_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/KeisukeFD/fizz/bind"
)

type pathIn struct {
	L  string   `path:"l" style:"label"`
	M  int      `path:"m" style:"matrix"`
	P  *string  `path:"p" style:"label"`
	S  string   `path:"s"`                 // simple: untouched
	LA []string `path:"la" style:"label"`  // tonic reads one value: not planned
	MO struct{} `path:"mo" style:"matrix"` // not a scalar
	MX string   `path:"mx,omitempty" style:"matrix"`
}

func TestRewritePath(t *testing.T) {
	p, err := bind.PlanFor(reflect.TypeFor[pathIn](), rwTags)
	if err != nil {
		t.Fatal(err)
	}
	rows := []struct {
		name, param, in string
		want            string
		planned         bool
		wantErr         string
	}{
		{"label", "l", ".blue", "blue", true, ""},
		{"label empty", "l", ".", "", true, ""},
		{"label keeps inner dots", "l", ".a.b", "a.b", true, ""},
		{"label pointer", "p", ".x", "x", true, ""},
		{"label without the dot", "l", "blue", "", false, `"l"`},
		{"matrix", "m", ";m=5", "5", true, ""},
		{"matrix empty", "m", ";m", "", true, ""},
		{"matrix empty with equal", "m", ";m=", "", true, ""},
		{"matrix keeps equal signs", "mx", ";mx=a=b", "a=b", true, ""},
		{"matrix wrong name", "m", ";n=5", "", false, `"m"`},
		{"matrix longer name", "m", ";mm=5", "", false, `"m"`},
		{"matrix without prefix", "m", "5", "", false, `"m"`},
		{"simple is untouched", "s", ".x", ".x", false, ""},
		{"slice is untouched", "la", ".a,b", ".a,b", false, ""},
		{"struct is untouched", "mo", ";mo=1", ";mo=1", false, ""},
		{"unknown is untouched", "z", ".x", ".x", false, ""},
	}
	for _, r := range rows {
		t.Run(r.name, func(t *testing.T) {
			got, ok, err := bind.RewritePath(r.param, r.in, p)
			if r.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), r.wantErr) {
					t.Fatalf("err = %v, want one containing %q", err, r.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != r.want || ok != r.planned {
				t.Errorf("RewritePath(%q, %q) = %q, %v; want %q, %v", r.param, r.in, got, ok, r.want, r.planned)
			}
		})
	}
	if got, ok, err := bind.RewritePath("l", ".x", nil); got != ".x" || ok || err != nil {
		t.Errorf("nil plan: %q %v %v", got, ok, err)
	}
}
