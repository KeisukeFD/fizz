package acceptance

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/loopfz/gadgeto/tonic"

	"github.com/wI2L/fizz"
	"github.com/wI2L/fizz/bind"
)

// Runtime follows the document (D26): spaceDelimited and pipeDelimited
// query arrays (P10.7j), through plain HTTP.

type bDelimIn struct {
	Space []string `query:"space" style:"spaceDelimited"`
	Pipe  []int    `query:"pipe" style:"pipeDelimited"`
	Rep   []string `query:"rep" style:"pipeDelimited" explode:"true"`
	Plain []string `query:"plain"`
	Page  int      `query:"page"`
}

type bDelimCommaIn struct {
	Pipe []string `query:"pipe" style:"pipeDelimited" explode:"false"`
}

func bDelimApp() *fizz.Fizz {
	f := fizz.New()
	bBHandle(f, "GET", "/delim", tonic.Handler(func(c *gin.Context, in *bDelimIn) (*bDelimIn, error) { return in, nil }, 200))
	bBHandle(f, "GET", "/delimcomma", tonic.Handler(func(c *gin.Context, in *bDelimCommaIn) (*bDelimCommaIn, error) { return in, nil }, 200))
	return f
}

func TestDelimitedQueryArrays(t *testing.T) {
	bGGinTestMode(t)
	app := bDelimApp()
	rows := []struct {
		name, target, want string
		status             int
	}{
		{"space %20", "/delim?space=a%20b%20c", `{"Space":["a","b","c"],"Pipe":null,"Rep":null,"Plain":null,"Page":0}`, 200},
		{"space plus", "/delim?space=a+b", `{"Space":["a","b"],"Pipe":null,"Rep":null,"Plain":null,"Page":0}`, 200},
		{"pipe ints", "/delim?pipe=1|2|3&page=4", `{"Space":null,"Pipe":[1,2,3],"Rep":null,"Plain":null,"Page":4}`, 200},
		{"pipe encoded", "/delim?pipe=1%7C2", `{"Space":null,"Pipe":[1,2],"Rep":null,"Plain":null,"Page":0}`, 200},
		{"one element", "/delim?space=a", `{"Space":["a"],"Pipe":null,"Rep":null,"Plain":null,"Page":0}`, 200},
		{"absent", "/delim", `{"Space":null,"Pipe":null,"Rep":null,"Plain":null,"Page":0}`, 200},
		{"empty", "/delim?space=", `{"Space":null,"Pipe":null,"Rep":null,"Plain":null,"Page":0}`, 200},
		{"exploded keeps the repeated form", "/delim?rep=a&rep=b", `{"Space":null,"Pipe":null,"Rep":["a","b"],"Plain":null,"Page":0}`, 200},
		{"plain array is untouched", "/delim?plain=a%20b|c", `{"Space":null,"Pipe":null,"Rep":null,"Plain":["a b|c"],"Page":0}`, 200},
		{"bad element type", "/delim?pipe=1|x", "", 400},
		{"explode false: joined for tonic", "/delimcomma?pipe=a|b", `{"Pipe":["a","b"]}`, 200},
		{"explode false: comma in an element", "/delimcomma?pipe=a,b|c", "", 400},
	}
	for _, r := range rows {
		t.Run(r.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			app.ServeHTTP(w, httptest.NewRequest(http.MethodGet, r.target, nil))
			if r.status == 200 {
				bBOK(t, w, r.want)
			} else {
				bB400(t, w)
			}
		})
	}
}

// TestDelimitedDriftGuard keeps the rule that picks the arrays to split in
// agreement with the document: bind plans an array exactly when the
// generator emits a spaceDelimited or pipeDelimited style with
// explode=false for it.
func TestDelimitedDriftGuard(t *testing.T) {
	bGGinTestMode(t)
	type (
		unset struct {
			A []string `query:"a" style:"spaceDelimited"`
		}
		explFalse struct {
			A []string `query:"a" style:"pipeDelimited" explode:"false"`
		}
		explTrue struct {
			A []string `query:"a" style:"pipeDelimited" explode:"true"`
		}
		explBad struct {
			A []string `query:"a" style:"pipeDelimited" explode:"maybe"`
		}
		array struct {
			A [2]int `query:"a" style:"spaceDelimited"`
		}
		ptr struct {
			A *[]int `query:"a" style:"spaceDelimited"`
		}
		form struct {
			A []string `query:"a" explode:"false"`
		}
	)
	rows := []struct {
		name string
		typ  reflect.Type
		gen  func() *fizz.Fizz
	}{
		{"unset", reflect.TypeFor[unset](), func() *fizz.Fizz { return driftApp(func(*gin.Context, *unset) error { return nil }) }},
		{"explode false", reflect.TypeFor[explFalse](), func() *fizz.Fizz { return driftApp(func(*gin.Context, *explFalse) error { return nil }) }},
		{"explode true", reflect.TypeFor[explTrue](), func() *fizz.Fizz { return driftApp(func(*gin.Context, *explTrue) error { return nil }) }},
		{"explode invalid", reflect.TypeFor[explBad](), func() *fizz.Fizz { return driftApp(func(*gin.Context, *explBad) error { return nil }) }},
		{"array", reflect.TypeFor[array](), func() *fizz.Fizz { return driftApp(func(*gin.Context, *array) error { return nil }) }},
		{"pointer to slice", reflect.TypeFor[ptr](), func() *fizz.Fizz { return driftApp(func(*gin.Context, *ptr) error { return nil }) }},
		{"form", reflect.TypeFor[form](), func() *fizz.Fizz { return driftApp(func(*gin.Context, *form) error { return nil }) }},
	}
	for _, r := range rows {
		t.Run(r.name, func(t *testing.T) {
			p, err := bind.PlanFor(r.typ, bBDelimTags)
			if err != nil {
				t.Fatal(err)
			}
			_, changed, _ := bind.RewriteQuery("a=1%202|3", p)
			doc := documentedDelimited(t, r.gen())
			if changed != doc {
				t.Errorf("bind splits = %v, document says delimited and unexploded = %v", changed, doc)
			}
		})
	}
}

var bBDelimTags = bind.Tags{Query: "query", Path: "path", Cookie: "cookie", Style: "style", Explode: "explode", Default: "default"}

func driftApp[T any](h func(*gin.Context, *T) error) *fizz.Fizz {
	f := fizz.New()
	f.Handle("/x", "GET", []fizz.OperationOption{fizz.ID("x")}, tonic.Handler(h, 200))
	return f
}

// documentedDelimited reports whether the only parameter of GET /x is a
// spaceDelimited or pipeDelimited one with explode=false.
func documentedDelimited(t *testing.T, f *fizz.Fizz) bool {
	t.Helper()
	item := f.Generator().API().Paths["/x"]
	if item == nil || item.GET == nil || len(item.GET.Parameters) != 1 {
		t.Fatalf("GET /x has no single parameter: %v", f.Errors())
	}
	p := item.GET.Parameters[0].Value
	if p == nil {
		t.Fatal("parameter is a reference")
	}
	delimited := p.Style == "spaceDelimited" || p.Style == "pipeDelimited"
	return delimited && p.Explode != nil && !*p.Explode
}
