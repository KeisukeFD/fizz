package bind_test

import (
	"net/http"
	"reflect"
	"testing"

	"github.com/wI2L/fizz/bind"
)

type headerIn struct {
	Lang   string   `header:"X-Lang" default:"en"`
	N      *int     `header:"x-n" default:"3"`
	Opt    string   `header:"X-Opt,default=o"` // tonic applies it
	NoDef  string   `header:"X-None"`
	Empty  string   `header:"X-Empty" default:""`
	List   []string `header:"X-List" default:"a,b"` // not a single value
	Skip   string   `header:"-" default:"x"`
	Unnamd string   `header:"" default:"x"`
	Q      string   `query:"q" default:"x"`
}

func TestRewriteHeader(t *testing.T) {
	p, err := bind.PlanFor(reflect.TypeFor[headerIn](), rwTags2)
	if err != nil {
		t.Fatal(err)
	}
	rows := []struct {
		name string
		in   http.Header
		want http.Header
		ok   bool
	}{
		{"absent", http.Header{}, http.Header{"X-Lang": {"en"}, "X-N": {"3"}}, true},
		{"nil header", nil, http.Header{"X-Lang": {"en"}, "X-N": {"3"}}, true},
		{"one present", http.Header{"X-Lang": {"fr"}}, http.Header{"X-Lang": {"fr"}, "X-N": {"3"}}, true},
		{"present but empty", http.Header{"X-Lang": {""}, "X-N": {"1"}}, http.Header{"X-Lang": {""}, "X-N": {"1"}}, false},
		{"others are kept", http.Header{"X-Other": {"v"}}, http.Header{"X-Other": {"v"}, "X-Lang": {"en"}, "X-N": {"3"}}, true},
	}
	for _, r := range rows {
		t.Run(r.name, func(t *testing.T) {
			var before http.Header
			if r.in != nil {
				before = r.in.Clone()
			}
			got, ok := bind.RewriteHeader(r.in, p)
			if ok != r.ok || !reflect.DeepEqual(got, r.want) {
				t.Errorf("RewriteHeader = %v, %v; want %v, %v", got, ok, r.want, r.ok)
			}
			if !reflect.DeepEqual(r.in, before) {
				t.Errorf("the input header was modified: %v, was %v", r.in, before)
			}
		})
	}
	if got, ok := bind.RewriteHeader(http.Header{}, nil); ok || len(got) != 0 {
		t.Errorf("nil plan changed the header: %v, %v", got, ok)
	}
}

func TestHeaderPlanIsNotEmpty(t *testing.T) {
	type in struct {
		H string `header:"X-H" default:"d"`
	}
	type none struct {
		H string `header:"X-H"`
	}
	for _, c := range []struct {
		typ   reflect.Type
		tags  bind.Tags
		empty bool
	}{
		{reflect.TypeFor[in](), rwTags2, false},
		{reflect.TypeFor[none](), rwTags2, true},
		{reflect.TypeFor[in](), rwTags, true}, // Header tag disabled
	} {
		p, err := bind.PlanFor(c.typ, c.tags)
		if err != nil {
			t.Fatal(err)
		}
		if p.Empty() != c.empty {
			t.Errorf("%s: Empty() = %v, want %v", c.typ, p.Empty(), c.empty)
		}
	}
}

var rwTags2 = bind.Tags{Query: "query", Path: "path", Header: "header", Cookie: "cookie", Style: "style", Explode: "explode", Default: "default"}
