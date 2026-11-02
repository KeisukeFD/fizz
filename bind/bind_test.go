package bind_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/KeisukeFD/fizz/bind"
)

var tags = bind.Tags{Query: "query", Cookie: "cookie", Style: "style", Default: "default"}

type sIn struct {
	Name string   `json:"name"`
	Age  int      `json:"age"`
	Tags []string `json:"tags"`
	Raw  string   // no json tag: matched by Go name
}

type mapIn struct {
	Page   int               `query:"page"`
	Filter map[string]string `query:"filter" style:"deepObject"`
}
type intMapIn struct {
	Filter map[string]int `query:"filter" style:"deepObject"`
}
type sliceMapIn struct {
	Filter map[string][]int `query:"filter" style:"deepObject"`
}
type structIn struct {
	Page   int  `query:"page"`
	Filter *sIn `query:"filter" style:"deepObject"`
}
type valStructIn struct {
	Filter sIn `query:"filter,omitempty" style:"deepObject"`
}
type noDeepIn struct {
	A map[string]string `query:"a"`
}

func req(rawQuery string) *http.Request {
	return httptest.NewRequest(http.MethodGet, "/?"+rawQuery, nil)
}

func decode[T any](t *testing.T, rawQuery string, opts ...bind.QueryOption) (*T, error) {
	t.Helper()
	p, err := bind.PlanFor(reflect.TypeFor[T](), tags)
	if err != nil {
		t.Fatalf("PlanFor: %v", err)
	}
	dst := new(T)
	return dst, bind.Query(req(rawQuery), dst, p, opts...)
}

func TestQueryDeepObject(t *testing.T) {
	t.Run("dotted key", func(t *testing.T) {
		got, err := decode[mapIn](t, "filter[attr.sss]=v")
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got.Filter, map[string]string{"attr.sss": "v"}) {
			t.Fatalf("got %v", got.Filter)
		}
	})
	t.Run("map with other params untouched", func(t *testing.T) {
		got, err := decode[mapIn](t, "page=2&filter[name]=x&filter[age]=3")
		if err != nil {
			t.Fatal(err)
		}
		want := map[string]string{"name": "x", "age": "3"}
		if !reflect.DeepEqual(got.Filter, want) || got.Page != 0 {
			t.Fatalf("got %+v", got)
		}
	})
	t.Run("encoded brackets", func(t *testing.T) {
		got, err := decode[mapIn](t, "filter%5Bname%5D=x")
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got.Filter, map[string]string{"name": "x"}) {
			t.Fatalf("got %v", got.Filter)
		}
	})
	t.Run("int map", func(t *testing.T) {
		got, err := decode[intMapIn](t, "filter[a]=3")
		if err != nil || got.Filter["a"] != 3 {
			t.Fatalf("got %v, %v", got.Filter, err)
		}
	})
	t.Run("struct json names", func(t *testing.T) {
		got, err := decode[structIn](t, "filter[name]=x&filter[age]=3&filter[tags]=a&filter[tags]=b&filter[Raw]=r&filter[zzz]=1")
		if err != nil {
			t.Fatal(err)
		}
		want := &sIn{Name: "x", Age: 3, Tags: []string{"a", "b"}, Raw: "r"}
		if !reflect.DeepEqual(got.Filter, want) {
			t.Fatalf("got %+v", got.Filter)
		}
	})
	t.Run("value struct", func(t *testing.T) {
		got, err := decode[valStructIn](t, "filter[name]=x")
		if err != nil || got.Filter.Name != "x" {
			t.Fatalf("got %+v, %v", got.Filter, err)
		}
	})
	t.Run("case sensitive struct key ignored", func(t *testing.T) {
		got, err := decode[structIn](t, "filter[NAME]=x")
		if err != nil {
			t.Fatal(err)
		}
		if got.Filter == nil || got.Filter.Name != "" {
			t.Fatalf("got %+v", got.Filter)
		}
	})
	t.Run("no keys nil map and nil pointer", func(t *testing.T) {
		m, err := decode[mapIn](t, "page=1&filterx[a]=1")
		if err != nil || m.Filter != nil {
			t.Fatalf("map: %#v, %v", m.Filter, err)
		}
		s, err := decode[structIn](t, "")
		if err != nil || s.Filter != nil {
			t.Fatalf("struct: %#v, %v", s.Filter, err)
		}
	})
	t.Run("repeated key into slice", func(t *testing.T) {
		got, err := decode[sliceMapIn](t, "filter[t]=1&filter[t]=2")
		if err != nil || !reflect.DeepEqual(got.Filter["t"], []int{1, 2}) {
			t.Fatalf("got %v, %v", got.Filter, err)
		}
	})
	t.Run("only planned fields overwritten", func(t *testing.T) {
		p, _ := bind.PlanFor(reflect.TypeFor[mapIn](), tags)
		dst := &mapIn{Page: 7}
		if err := bind.Query(req("filter[a]=b"), dst, p); err != nil || dst.Page != 7 {
			t.Fatalf("%+v, %v", dst, err)
		}
	})

	errCases := []struct {
		name  string
		query string
		in    func(*testing.T, string) error
		want  string // must appear in the message
		is    error
	}{
		{"type error", "filter[age]=x", func(t *testing.T, q string) error { _, e := decode[intMapIn](t, q); return e }, "filter[age]", nil},
		{"struct type error", "filter[age]=x", func(t *testing.T, q string) error { _, e := decode[structIn](t, q); return e }, "filter[age]", nil},
		{"nested", "filter[a][b]=1", func(t *testing.T, q string) error { _, e := decode[mapIn](t, q); return e }, "filter[a][b]", bind.ErrNestedKey},
		{"nested encoded", "filter%5Ba%5D%5Bb%5D=1", func(t *testing.T, q string) error { _, e := decode[mapIn](t, q); return e }, "filter[a][b]", bind.ErrNestedKey},
		{"repeated scalar", "filter[a]=1&filter[a]=2", func(t *testing.T, q string) error { _, e := decode[mapIn](t, q); return e }, "filter[a]", nil},
		{"empty key", "filter[]=1", func(t *testing.T, q string) error { _, e := decode[mapIn](t, q); return e }, "filter[]", nil},
		{"unclosed", "filter[a=1", func(t *testing.T, q string) error { _, e := decode[mapIn](t, q); return e }, "filter[a", nil},
		{"bare bracket", "filter[=1", func(t *testing.T, q string) error { _, e := decode[mapIn](t, q); return e }, "filter[", nil},
		{"bare name", "filter=x", func(t *testing.T, q string) error { _, e := decode[mapIn](t, q); return e }, "filter", nil},
		{"trailing junk", "filter[a]b=1", func(t *testing.T, q string) error { _, e := decode[mapIn](t, q); return e }, "filter[a]b", nil},
		{"bad query string", "filter[a]=%zz", func(t *testing.T, q string) error { _, e := decode[mapIn](t, q); return e }, "", nil},
	}
	for _, c := range errCases {
		t.Run(c.name, func(t *testing.T) {
			err := c.in(t, c.query)
			if err == nil {
				t.Fatal("want error")
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Fatalf("error %q lacks %q", err, c.want)
			}
			if c.is != nil && !errors.Is(err, c.is) {
				t.Fatalf("error %v is not %v", err, c.is)
			}
		})
	}
}

func TestQueryMaxKeys(t *testing.T) {
	_, err := decode[mapIn](t, "filter[a]=1&filter[b]=2&filter[c]=3", bind.WithMaxKeys(2))
	if !errors.Is(err, bind.ErrTooManyKeys) {
		t.Fatalf("got %v", err)
	}
	if _, err := decode[mapIn](t, "filter[a]=1&filter[b]=2", bind.WithMaxKeys(2)); err != nil {
		t.Fatal(err)
	}
	if _, err := decode[mapIn](t, "filter[a]=1&filter[b]=2&filter[c]=3"); err != nil {
		t.Fatalf("no cap by default: %v", err)
	}
}

func TestQueryBadDst(t *testing.T) {
	p, _ := bind.PlanFor(reflect.TypeFor[mapIn](), tags)
	for _, dst := range []any{nil, mapIn{}, (*mapIn)(nil), new(intMapIn)} {
		if err := bind.Query(req(""), dst, p); err == nil {
			t.Errorf("dst %T: want error", dst)
		}
	}
	if err := bind.Query(req(""), new(mapIn), nil); err == nil {
		t.Error("nil plan: want error")
	}
}

func TestQueryNoDeepObjectFields(t *testing.T) {
	p, err := bind.PlanFor(reflect.TypeFor[noDeepIn](), tags)
	if err != nil {
		t.Fatal(err)
	}
	dst := new(noDeepIn)
	if err := bind.Query(req("a[x]=1"), dst, p); err != nil || dst.A != nil {
		t.Fatalf("%+v, %v", dst, err)
	}
}

func TestPlanForErrors(t *testing.T) {
	type nestedVal struct {
		F map[string]struct{ X int } `query:"f" style:"deepObject"`
	}
	type mapOfMap struct {
		F map[string]map[string]string `query:"f" style:"deepObject"`
	}
	type badMember struct {
		F struct{ M map[string]string } `query:"f" style:"deepObject"`
	}
	type badKind struct {
		F []string `query:"f" style:"deepObject"`
	}
	type badKey struct {
		F map[int]string `query:"f" style:"deepObject"`
	}
	for name, typ := range map[string]reflect.Type{
		"non-struct":     reflect.TypeFor[int](),
		"ptr non-struct": reflect.TypeFor[*[]string](),
		"struct value":   reflect.TypeFor[nestedVal](),
		"map of map":     reflect.TypeFor[mapOfMap](),
		"bad member":     reflect.TypeFor[badMember](),
		"slice field":    reflect.TypeFor[badKind](),
		"int key":        reflect.TypeFor[badKey](),
	} {
		t.Run(name, func(t *testing.T) {
			p, err := bind.PlanFor(typ, tags)
			if err == nil || p != nil {
				t.Fatalf("got %v, %v", p, err)
			}
			if name != "non-struct" && name != "ptr non-struct" && !strings.Contains(err.Error(), "F") {
				t.Fatalf("error %q does not name the field", err)
			}
		})
	}
	if _, err := bind.PlanFor(reflect.TypeFor[*mapIn](), tags); err != nil {
		t.Fatalf("pointer to struct: %v", err)
	}
}

func TestPlanForCachedAndConcurrent(t *testing.T) {
	first, err := bind.PlanFor(reflect.TypeFor[mapIn](), tags)
	if err != nil {
		t.Fatal(err)
	}
	if again, _ := bind.PlanFor(reflect.TypeFor[mapIn](), tags); again != first {
		t.Fatal("PlanFor not cached")
	}
	if other, _ := bind.PlanFor(reflect.TypeFor[mapIn](), bind.Tags{Query: "q", Style: "style"}); other == first {
		t.Fatal("different tags must give a different plan")
	}

	type fresh struct {
		F map[string]string `query:"f" style:"deepObject"`
	}
	var wg sync.WaitGroup
	plans := make([]*bind.Plan, 50)
	for i := range plans {
		wg.Go(func() {
			p, err := bind.PlanFor(reflect.TypeFor[fresh](), tags)
			if err != nil {
				t.Error(err)
				return
			}
			dst := new(fresh)
			if err := bind.Query(req("f[a]=1"), dst, p); err != nil || dst.F["a"] != "1" {
				t.Errorf("%v %v", dst.F, err)
			}
			plans[i] = p
		})
	}
	wg.Wait()
	for _, p := range plans {
		if p != plans[0] {
			t.Fatal("concurrent PlanFor returned different plans")
		}
	}
}
