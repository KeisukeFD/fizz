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
