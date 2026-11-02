package acceptance

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/KeisukeFD/fizz/bind"
	"github.com/KeisukeFD/fizz/internal/acceptance/at"
)

// Batch B rows of P7.2a (ACCEPTANCE.md B31-B34): the bind package API on a
// plain *http.Request, with no gin involved.

var bBTags = bind.Tags{Query: "query", Cookie: "cookie", Style: "style", Default: "default"}

func TestBBPlanForCachedAndConcurrent(t *testing.T) {
	at.Pending(t, "P7.2a", func(tb testing.TB) {
		typ := reflect.TypeOf(bBMixIn{})
		first, err := bind.PlanFor(typ, bBTags)
		if err != nil {
			tb.Fatal(err)
		}
		if again, _ := bind.PlanFor(typ, bBTags); again != first {
			tb.Fatal("PlanFor twice returned different pointers")
		}
		plans := make([]*bind.Plan, 50)
		var wg sync.WaitGroup
		for i := range plans {
			wg.Go(func() { plans[i], _ = bind.PlanFor(typ, bBTags) })
		}
		wg.Wait()
		for _, p := range plans {
			if p != first {
				tb.Fatal("concurrent PlanFor returned a different pointer")
			}
		}
	})
}

func TestBBPlanForRejectsUnsupported(t *testing.T) {
	type structValue struct {
		F map[string]struct{ X int } `query:"f" style:"deepObject"`
	}
	type mapOfMap struct {
		F map[string]map[string]string `query:"f" style:"deepObject"`
	}
	for name, typ := range map[string]reflect.Type{
		"map of struct": reflect.TypeOf(structValue{}),
		"map of map":    reflect.TypeOf(mapOfMap{}),
	} {
		at.Pending(t, "P7.2a", func(tb testing.TB) {
			if _, err := bind.PlanFor(typ, bBTags); err == nil {
				tb.Fatalf("%s: want error", name)
			}
		})
	}
}

func TestBBQueryDecodeTable(t *testing.T) {
	do := func(tb testing.TB, dst any, query string, opts ...bind.QueryOption) error {
		tb.Helper()
		p, err := bind.PlanFor(reflect.TypeOf(dst).Elem(), bBTags)
		if err != nil {
			tb.Fatal(err)
		}
		return bind.Query(httptest.NewRequest("GET", "/?"+query, nil), dst, p, opts...)
	}
	rows := []struct {
		name  string
		query string
		dst   any
		want  any    // expected *dst, or nil when an error is expected
		errIs string // substring of the error
	}{
		{"map", "filter[name]=x&filter[age]=3", &bBMapIn{}, &bBMapIn{Filter: map[string]string{"name": "x", "age": "3"}}, ""},
		{"dotted", "filter[a.b]=x&filter[attr.sss]=v", &bBMapIn{}, &bBMapIn{Filter: map[string]string{"a.b": "x", "attr.sss": "v"}}, ""},
		{"encoded", "filter%5Bname%5D=x", &bBMapIn{}, &bBMapIn{Filter: map[string]string{"name": "x"}}, ""},
		{"nested", "filter[a][b]=x", &bBMapIn{}, nil, "filter[a][b]"},
		{"nil map", "", &bBMapIn{}, &bBMapIn{}, ""},
		{"int", "filter[a]=3", &bBIntIn{}, &bBIntIn{Filter: map[string]int{"a": 3}}, ""},
		{"int error", "filter[a]=x", &bBIntIn{}, nil, "filter[a]"},
		{"repeated slice", "filter[t]=a&filter[t]=b", &bBSliceIn{}, &bBSliceIn{Filter: map[string][]string{"t": {"a", "b"}}}, ""},
		{"repeated scalar", "filter[a]=1&filter[a]=2", &bBMapIn{}, nil, "filter[a]"},
		{"struct", "filter[name]=x&filter[age]=3&filter[tags]=a&filter[zzz]=1", &bBStructIn{},
			&bBStructIn{Filter: &bBS{Name: "x", Age: 3, Tags: []string{"a"}}}, ""},
		{"struct type error", "filter[age]=x", &bBStructIn{}, nil, "filter[age]"},
		{"struct nil", "", &bBStructIn{}, &bBStructIn{}, ""},
		{"struct case sensitive", "filter[NAME]=x", &bBStructIn{}, &bBStructIn{Filter: &bBS{}}, ""},
		{"empty key", "filter[]=x", &bBMapIn{}, nil, "filter[]"},
		{"unclosed", "filter[a=x", &bBMapIn{}, nil, "filter[a"},
		{"bare", "filter=x", &bBMapIn{}, nil, "filter"},
		{"other prefix ignored", "filterx[a]=1", &bBMapIn{}, &bBMapIn{}, ""},
	}
	for _, r := range rows {
		t.Run(r.name, func(t *testing.T) {
			at.Pending(t, "P7.2a", func(tb testing.TB) {
				err := do(tb, r.dst, r.query)
				if r.want == nil {
					if err == nil || !strings.Contains(err.Error(), r.errIs) {
						tb.Fatalf("want error containing %q, got %v", r.errIs, err)
					}
					return
				}
				if err != nil {
					tb.Fatal(err)
				}
				if !reflect.DeepEqual(r.dst, r.want) {
					tb.Fatalf("got %+v, want %+v", r.dst, r.want)
				}
			})
		})
	}
	t.Run("nested is ErrNestedKey", func(t *testing.T) {
		at.Pending(t, "P7.2a", func(tb testing.TB) {
			if err := do(tb, &bBMapIn{}, "filter[a][b]=x"); !errors.Is(err, bind.ErrNestedKey) {
				tb.Fatalf("got %v", err)
			}
		})
	})
}

func TestBBQueryMaxKeys(t *testing.T) {
	at.Pending(t, "P7.2a", func(tb testing.TB) {
		p, err := bind.PlanFor(reflect.TypeOf(bBMapIn{}), bBTags)
		if err != nil {
			tb.Fatal(err)
		}
		r := httptest.NewRequest("GET", "/?filter[a]=1&filter[b]=2&filter[c]=3", nil)
		err = bind.Query(r, &bBMapIn{}, p, bind.WithMaxKeys(2))
		if !errors.Is(err, bind.ErrTooManyKeys) {
			tb.Fatalf("got %v", err)
		}
	})
}

// B35: bind.Cookies on a plain *http.Request, with the rows of B19-B24.
func TestBBCookiesDecodeTable(t *testing.T) {
	type ptrIn struct {
		N *int    `cookie:"n"`
		S *string `cookie:"s" default:"d"`
	}
	type badDefault struct {
		N int `cookie:"n" default:"x"`
	}
	type optsIn struct {
		Lang string `cookie:"lang,default=es"`
		Both string `cookie:"both,default=opt" default:"tag"`
	}
	type optReq struct {
		T string `cookie:"t,required"`
	}
	type optBadDefault struct {
		N int `cookie:"n,default=x"`
	}
	type optReqDefault struct {
		T string `cookie:"t,required,default=d"`
	}
	rows := []struct {
		name    string
		cookies []http.Cookie
		dst     any
		want    any    // expected *dst, or nil when an error is expected
		errIs   string // substring of the error
	}{
		{"present", []http.Cookie{{Name: "session", Value: "abc"}, {Name: "n", Value: "5"}}, &bBCookieIn{},
			&bBCookieIn{Session: "abc", Lang: "en", N: 5}, ""},
		{"default when absent", nil, &bBCookieIn{}, &bBCookieIn{Lang: "en"}, ""},
		{"overrides default", []http.Cookie{{Name: "lang", Value: "fr"}}, &bBCookieIn{}, &bBCookieIn{Lang: "fr"}, ""},
		{"absent required is left to validation", nil, &bBCookieReq{}, &bBCookieReq{}, ""},
		{"bad type", []http.Cookie{{Name: "n", Value: "abc"}}, &bBCookieIn{}, nil, `cookie "n"`},
		{"empty keeps empty", []http.Cookie{{Name: "lang", Value: ""}}, &bBCookieIn{}, &bBCookieIn{}, ""},
		{"empty to int is an error", []http.Cookie{{Name: "n", Value: ""}}, &bBCookieIn{}, nil, `cookie "n"`},
		{"raw value", []http.Cookie{{Name: "session", Value: "a%20b"}}, &bBCookieIn{}, &bBCookieIn{Session: "a%20b", Lang: "en"}, ""},
		{"pointers", []http.Cookie{{Name: "n", Value: "7"}}, &ptrIn{}, &ptrIn{N: new(7), S: new("d")}, ""},
		{"pointer absent stays nil", nil, &ptrIn{}, &ptrIn{S: new("d")}, ""},
		{"bad default", nil, &badDefault{}, nil, `cookie "n"`},
		{"option default when absent, wins over tag", nil, &optsIn{}, &optsIn{Lang: "es", Both: "opt"}, ""},
		{"option default not applied when present", []http.Cookie{{Name: "lang", Value: "fr"}, {Name: "both", Value: "x"}}, &optsIn{}, &optsIn{Lang: "fr", Both: "x"}, ""},
		{"option default not applied when empty", []http.Cookie{{Name: "lang", Value: ""}, {Name: "both", Value: ""}}, &optsIn{}, &optsIn{}, ""},
		{"bad option default", nil, &optBadDefault{}, nil, `cookie "n"`},
		{"required absent", nil, &optReq{}, nil, `cookie "t"`},
		{"required empty is missing, as in tonic", []http.Cookie{{Name: "t", Value: ""}}, &optReq{}, nil, `cookie "t"`},
		{"required present", []http.Cookie{{Name: "t", Value: "v"}}, &optReq{}, &optReq{T: "v"}, ""},
		{"required with default absent", nil, &optReqDefault{}, &optReqDefault{T: "d"}, ""},
	}
	for _, r := range rows {
		t.Run(r.name, func(t *testing.T) {
			at.Pending(t, "P7.2c", func(tb testing.TB) {
				p, err := bind.PlanFor(reflect.TypeOf(r.dst).Elem(), bBTags)
				if err != nil {
					tb.Fatal(err)
				}
				req := httptest.NewRequest("GET", "/", nil)
				for _, c := range r.cookies {
					req.AddCookie(&c)
				}
				err = bind.Cookies(req, r.dst, p)
				if r.want == nil {
					if err == nil || !strings.Contains(err.Error(), r.errIs) {
						tb.Fatalf("want error containing %q, got %v", r.errIs, err)
					}
					return
				}
				if err != nil {
					tb.Fatal(err)
				}
				if !reflect.DeepEqual(r.dst, r.want) {
					tb.Fatalf("got %+v, want %+v", r.dst, r.want)
				}
			})
		})
	}
}
