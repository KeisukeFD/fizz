package contract

import (
	"testing"
)

// na marks a combination that the spec table says is n/a (undefined).
const na = "\x00n/a"

// styleTable is the "Style Examples" table of OAS 3.2.1 (internal/conformance/
// oai/orig/oas/versions/3.2.1.md, Parameter Object > "Style Examples"), row by
// row. The parameter is named color; the values are undefined, "blue",
// ["blue", "black", "brown"] and {"R": 100, "G": 200, "B": 150}. An empty
// string is the table's _empty_.
//
// The table has one "form" row pair, for query. Its cookie rows are the
// "cookie" style, which has no percent-encoding.
var styleTable = []struct {
	style   string
	explode bool
	in      string // the location the row is run in (the table lists the style only)
	undef   string
	str     string
	arr     string
	obj     string
}{
	{"matrix", false, "path", "", ";color=blue", ";color=blue,black,brown", ";color=R,100,G,200,B,150"},
	{"matrix", true, "path", "", ";color=blue", ";color=blue;color=black;color=brown", ";R=100;G=200;B=150"},
	{"label", false, "path", "", ".blue", ".blue,black,brown", ".R,100,G,200,B,150"},
	{"label", true, "path", "", ".blue", ".blue.black.brown", ".R=100.G=200.B=150"},
	{"simple", false, "path", "", "blue", "blue,black,brown", "R,100,G,200,B,150"},
	{"simple", true, "path", "", "blue", "blue,black,brown", "R=100,G=200,B=150"},
	{"form", false, "query", "color=", "color=blue", "color=blue,black,brown", "color=R,100,G,200,B,150"},
	{"form", true, "query", "color=", "color=blue", "color=blue&color=black&color=brown", "R=100&G=200&B=150"},
	{"spaceDelimited", false, "query", na, na, "color=blue%20black%20brown", "color=R%20100%20G%20200%20B%20150"},
	{"spaceDelimited", true, "query", na, na, na, na},
	{"pipeDelimited", false, "query", na, na, "color=blue%7Cblack%7Cbrown", "color=R%7C100%7CG%7C200%7CB%7C150"},
	{"pipeDelimited", true, "query", na, na, na, na},
	{"deepObject", false, "query", na, na, na, "color%5BR%5D=100&color%5BG%5D=200&color%5BB%5D=150"},
	{"deepObject", true, "query", na, na, na, "color%5BR%5D=100&color%5BG%5D=200&color%5BB%5D=150"},
	{"cookie", false, "cookie", "color=", "color=blue", na, na},
	{"cookie", true, "cookie", "color=", "color=blue", "color=blue; color=black; color=brown", "R=100; G=200; B=150"},
}

func (pl Placement) in(in string) string {
	switch in {
	case "query":
		return pl.Query
	case "path":
		return pl.Path
	case "header":
		return pl.Header
	}
	return pl.Cookie
}

// TestSerializeStyleTable runs every cell of the spec's style table.
func TestSerializeStyleTable(t *testing.T) {
	values := []struct {
		name string
		v    any
		want func(i int) string
	}{
		{"undefined", nil, func(i int) string { return styleTable[i].undef }},
		{"string", "blue", func(i int) string { return styleTable[i].str }},
		{"array", []any{"blue", "black", "brown"}, func(i int) string { return styleTable[i].arr }},
		{"object", Object{{"R", float64(100)}, {"G", float64(200)}, {"B", float64(150)}}, func(i int) string { return styleTable[i].obj }},
	}
	for i, row := range styleTable {
		// deepObject has no explode in the table: both values must agree.
		p := Param{Name: "color", In: row.in, Style: row.style, Explode: &row.explode}
		for _, val := range values {
			t.Run(p.In+"/"+row.style+"/explode="+boolS(row.explode)+"/"+val.name, func(t *testing.T) {
				got, err := Serialize(p, val.v)
				want := val.want(i)
				if want == na {
					if err == nil {
						t.Fatalf("Serialize = %q, want an error (n/a in the table)", got.in(row.in))
					}
					return
				}
				if err != nil {
					t.Fatalf("Serialize: %v", err)
				}
				if s := got.in(row.in); s != want {
					t.Errorf("Serialize = %q, want %q", s, want)
				}
			})
		}
	}
}

// TestSerializeSimpleHeader: the simple rows of the table also hold for a
// header, where the header name is not part of the value.
func TestSerializeSimpleHeader(t *testing.T) {
	for _, row := range styleTable {
		if row.style != "simple" {
			continue
		}
		p := Param{Name: "color", In: "header", Style: "simple", Explode: &row.explode}
		for name, c := range map[string]struct {
			v    any
			want string
		}{
			"undefined": {nil, row.undef},
			"string":    {"blue", row.str},
			"array":     {[]any{"blue", "black", "brown"}, row.arr},
			"object":    {Object{{"R", 100.0}, {"G", 200.0}, {"B", 150.0}}, row.obj},
		} {
			got, err := Serialize(p, c.v)
			if err != nil || got.Header != c.want {
				t.Errorf("header explode=%v %s: got %q, %v; want %q", row.explode, name, got.Header, err, c.want)
			}
		}
	}
}

// TestSerializeDefaults: style and explode default as in 3.2.1.
func TestSerializeDefaults(t *testing.T) {
	arr := []any{"a", "b"}
	for _, c := range []struct {
		name string
		p    Param
		want Placement
	}{
		{"query: form, exploded", Param{Name: "n", In: "query"}, Placement{Query: "n=a&n=b"}},
		{"path: simple", Param{Name: "n", In: "path"}, Placement{Path: "a,b"}},
		{"header: simple", Param{Name: "n", In: "header"}, Placement{Header: "a,b"}},
		{"cookie: form, exploded", Param{Name: "n", In: "cookie"}, Placement{Cookie: "n=a; n=b"}},
	} {
		got, err := Serialize(c.p, arr)
		if err != nil || got != c.want {
			t.Errorf("%s: got %+v, %v; want %+v", c.name, got, err, c.want)
		}
	}
}

// TestSerializeEncoding: percent-encoding (3.2.1 "URL Percent-Encoding").
func TestSerializeEncoding(t *testing.T) {
	no, yes := false, true
	for _, c := range []struct {
		name string
		p    Param
		v    any
		want string
	}{
		{"query: all but unreserved", Param{Name: "q", In: "query"}, "a b&c=d/e?é~", "q=a%20b%26c%3Dd%2Fe%3F%C3%A9~"},
		{"query: name is encoded", Param{Name: "a b", In: "query"}, "x", "a%20b=x"},
		{"query: allowReserved keeps reserved", Param{Name: "q", In: "query", AllowReserved: true}, "a/b?c=d:e", "q=a/b?c=d:e"},
		{"query: allowReserved encodes space and %", Param{Name: "q", In: "query", AllowReserved: true}, "a b 50%", "q=a%20b%2050%25"},
		{"query: allowReserved keeps triples", Param{Name: "q", In: "query", AllowReserved: true}, "a%2Fb", "q=a%2Fb"},
		{"query: comma in an item is encoded", Param{Name: "q", In: "query", Explode: &no}, []any{"a,b", "c"}, "q=a%2Cb,c"},
		{"query: deepObject keys and values", Param{Name: "f", In: "query", Style: "deepObject"}, map[string]any{"a b": "c&d"}, "f%5Ba%20b%5D=c%26d"},
		{"query: map keys are sorted", Param{Name: "f", In: "query", Style: "form", Explode: &yes}, map[string]any{"b": "2", "a": "1"}, "a=1&b=2"},
		{"query: integer-valued number", Param{Name: "n", In: "query"}, float64(7), "n=7"},
		{"query: fractional number", Param{Name: "n", In: "query"}, 1.5, "n=1.5"},
		{"query: boolean", Param{Name: "n", In: "query"}, true, "n=true"},
		{"query: empty array is absent", Param{Name: "n", In: "query"}, []any{}, ""},
		{"path: encoded", Param{Name: "n", In: "path"}, "a/b c", "a%2Fb%20c"},
		{"path: allowReserved", Param{Name: "n", In: "path", AllowReserved: true}, "a:b", "a:b"},
		{"path: matrix empty value", Param{Name: "n", In: "path", Style: "matrix"}, "", ";n"},
		{"header: not encoded", Param{Name: "n", In: "header"}, "a b&c", "a b&c"},
		{"cookie: form encodes", Param{Name: "n", In: "cookie"}, "a b", "n=a%20b"},
		{"cookie: form allowReserved", Param{Name: "n", In: "cookie", AllowReserved: true}, "a;b", "n=a;b"},
		{"cookie: style cookie is raw", Param{Name: "n", In: "cookie", Style: "cookie"}, "a%20b", "n=a%20b"},
	} {
		t.Run(c.name, func(t *testing.T) {
			got, err := Serialize(c.p, c.v)
			if err != nil {
				t.Fatal(err)
			}
			if s := got.in(c.p.In); s != c.want {
				t.Errorf("got %q, want %q", s, c.want)
			}
		})
	}
}

func TestSerializeErrors(t *testing.T) {
	for name, p := range map[string]Param{
		"style not allowed in location": {Name: "n", In: "header", Style: "form"},
		"deepObject in path":            {Name: "n", In: "path", Style: "deepObject"},
	} {
		if _, err := Serialize(p, "x"); err == nil {
			t.Errorf("%s: no error", name)
		}
	}
	if _, err := Serialize(Param{Name: "n", In: "query", Style: "deepObject"}, map[string]any{"a": []any{"x"}}); err == nil {
		t.Error("nested array in deepObject: no error")
	}
}

func boolS(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
