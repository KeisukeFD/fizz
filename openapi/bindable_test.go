package openapi

import (
	"net"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"
)

type (
	bdTextStruct struct{ v string }
	bdPlain      struct{ X string }

	bdMapQuery struct {
		M map[string]string `query:"m"`
	}
	bdStructQuery struct {
		S *bdPlain `query:"s"`
	}
	bdMapHeader struct {
		M map[string]string `header:"X-M"`
	}
	bdStructPath struct {
		S bdPlain `path:"s"`
	}
	bdSliceStruct struct {
		S []bdPlain `query:"s"`
	}
	bdNested struct {
		S [][]string `query:"s"`
	}
	bdDeep struct {
		M map[string]string `query:"m" style:"deepObject"`
	}
	bdOK struct {
		T  time.Time      `query:"t"`
		D  time.Duration  `query:"d"`
		TS []time.Time    `query:"ts"`
		IP net.IP         `header:"X-Ip"`
		TX bdTextStruct   `query:"tx"`
		PT *bdTextStruct  `query:"ptx"`
		SS []string       `query:"ss"`
		SP []*int         `query:"sp"`
		B  []byte         `query:"b"`
		A  [2]string      `query:"a"`
		P  string         `path:"p"`
		_  struct{}       `query:"-"`
		C  chan int       `query:"-"`
		R  *time.Location `query:"-"`
	}
)

func (bdTextStruct) MarshalText() ([]byte, error)  { return nil, nil }
func (*bdTextStruct) UnmarshalText(b []byte) error { return nil }

func TestNonDeepObjectMapOrStructParamIsError(t *testing.T) {
	tests := []struct {
		name string
		typ  reflect.Type
		want string
	}{
		{"map query", reflect.TypeFor[bdMapQuery](), "query"},
		{"struct query", reflect.TypeFor[bdStructQuery](), "query"},
		{"map header", reflect.TypeFor[bdMapHeader](), "header"},
		{"struct path", reflect.TypeFor[bdStructPath](), "path"},
		{"slice of struct", reflect.TypeFor[bdSliceStruct](), "query"},
		{"nested slice", reflect.TypeFor[bdNested](), "query"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g, op := build(t, tt.typ)
			if len(op.Parameters) != 0 {
				t.Errorf("params = %v, want none", paramKeys(op))
			}
			errs := g.Errors()
			if len(errs) != 1 || !strings.Contains(errs[0].Error(), tt.want+" parameter must be a scalar") {
				t.Errorf("errors = %q", errorsText(g))
			}
		})
	}
}

func TestDeepObjectMapParamIsAllowed(t *testing.T) {
	g, op := build(t, reflect.TypeFor[bdDeep]())
	if len(g.Errors()) != 0 {
		t.Fatalf("errors = %s", errorsText(g))
	}
	if got := paramKeys(op); !slices.Equal(got, []string{"query:m"}) {
		t.Errorf("params = %v", got)
	}
}

// Types that tonic binds (scalars, encoding.TextUnmarshaler, slices of
// them) stay allowed.
func TestBindableParamTypesStayAllowed(t *testing.T) {
	g, op := build(t, reflect.TypeFor[bdOK]())
	if len(g.Errors()) != 0 {
		t.Fatalf("errors = %s", errorsText(g))
	}
	if n := len(op.Parameters); n != 11 {
		t.Errorf("params = %v, want 11", paramKeys(op))
	}
}

type (
	bdSliceHeader struct {
		H []string `header:"X-H"`
	}
	bdArrayHeader struct {
		H [2]int `header:"X-H"`
	}
	bdSlicePath struct {
		P []string `path:"p"`
	}
	bdPtrSliceHeader struct {
		H *[]string `header:"X-H"`
	}
)

// A slice or array header or path parameter is an error (D29, P10.7e,
// P10.7l): tonic binds them from one value. []byte and net.IP are strings.
func TestSliceHeaderAndPathParamIsError(t *testing.T) {
	for name, typ := range map[string]reflect.Type{
		"slice header":   reflect.TypeFor[bdSliceHeader](),
		"array header":   reflect.TypeFor[bdArrayHeader](),
		"slice path":     reflect.TypeFor[bdSlicePath](),
		"pointer header": reflect.TypeFor[bdPtrSliceHeader](),
	} {
		t.Run(name, func(t *testing.T) {
			g, op := build(t, typ)
			if len(op.Parameters) != 0 {
				t.Errorf("params = %v, want none", paramKeys(op))
			}
			errs := g.Errors()
			if len(errs) != 1 || !strings.Contains(errs[0].Error(), singleValueMsg) {
				t.Errorf("errors = %q", errorsText(g))
			}
		})
	}
}

func TestSliceParameterOptionInHeaderOrPathIsError(t *testing.T) {
	for _, in := range []string{InHeader, InPath, InQuery} {
		t.Run(in, func(t *testing.T) {
			g := gen(t)
			op := &Operation{}
			arr := &Schema{Type: Types("array"), Items: &Schema{Type: Types("string")}}
			g.applyParameters(op, "/x/{p}", []*Parameter{{Name: "p", In: in, Schema: arr}})
			want := in != InQuery
			if got := len(g.Errors()) == 1 && strings.Contains(g.Errors()[0].Error(), singleValueMsg); got != want {
				t.Errorf("error recorded = %v, want %v: %q", got, want, errorsText(g))
			}
			if n := len(op.Parameters); (n == 0) != want {
				t.Errorf("params = %d, want dropped = %v", n, want)
			}
		})
	}
}

type (
	hdrNums struct {
		I   int      `header:"X-I"`
		U   uint8    `header:"X-U"`
		F   float32  `header:"X-F"`
		B   bool     `header:"X-B"`
		E   string   `header:"X-E" enum:"a,b"`
		P   *int     `header:"X-P"`
		S   string   `header:"X-S"`
		D   int      `header:"X-D" default:"1"`
		O   int      `header:"X-O,default=1"`
		R   int      `header:"X-R" validate:"required"`
		Q   int      `query:"q"`
		Raw []byte   `header:"X-Raw"`
		T   struct{} `header:"-"`
	}
)

func TestOptionalNonStringHeaderWarns(t *testing.T) {
	g, op := build(t, reflect.TypeFor[hdrNums]())
	var warned []string
	for _, e := range g.Errors() {
		w, ok := e.(*Warning)
		if !ok {
			t.Fatalf("unexpected error %v", e)
		}
		warned = append(warned, w.Msg)
	}
	var want []string
	for _, h := range []string{"X-I", "X-U", "X-F", "X-B", "X-E", "X-P"} {
		want = append(want, `absent optional header "`+h+`" is bound as "" by tonic and returns 400; add a default or make it a string`)
	}
	if !slices.Equal(warned, want) {
		t.Errorf("warnings = %q\nwant %q", warned, want)
	}
	if n := len(op.Parameters); n != 12 {
		t.Errorf("params = %v, want 12 (warnings drop none)", paramKeys(op))
	}
}
