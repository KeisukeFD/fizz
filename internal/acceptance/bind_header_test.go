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

// Runtime follows the document (D29): the default tag of an absent header
// is applied (P10.7i), through plain HTTP.

type bHeaderIn struct {
	Lang  string  `header:"X-Lang" default:"en"`
	Limit int     `header:"X-Limit" default:"10"`
	Ratio float64 `header:"X-Ratio" default:"0.5"`
	On    bool    `header:"X-On" default:"true"`
	Ptr   *int    `header:"X-Ptr" default:"3"`
	Opt   string  `header:"X-Opt,default=opt"`
	Both  string  `header:"X-Both,default=option" default:"tag"`
	Plain string  `header:"X-Plain"`
	Empty string  `header:"X-Empty" default:""`
	Skip  string  `header:"-" default:"skipped"`
	Body  *string `json:"body"`
}

func TestHeaderDefault(t *testing.T) {
	bGGinTestMode(t)
	f := fizz.New()
	bBHandle(f, "GET", "/h", tonic.Handler(func(c *gin.Context, in *bHeaderIn) (*bHeaderIn, error) { return in, nil }, 200))
	rows := []struct {
		name   string
		header http.Header
		want   string
	}{
		{"all absent", nil,
			`{"Lang":"en","Limit":10,"Ratio":0.5,"On":true,"Ptr":3,"Opt":"opt","Both":"option","Plain":"","Empty":"","Skip":"","body":null}`},
		{"present", http.Header{"X-Lang": {"fr"}, "X-Limit": {"2"}, "X-Ratio": {"1.5"}, "X-On": {"false"}, "X-Ptr": {"9"}, "X-Both": {"x"}},
			`{"Lang":"fr","Limit":2,"Ratio":1.5,"On":false,"Ptr":9,"Opt":"opt","Both":"x","Plain":"","Empty":"","Skip":"","body":null}`},
		{"present but empty keeps the empty value", http.Header{"X-Lang": {""}},
			`{"Lang":"","Limit":10,"Ratio":0.5,"On":true,"Ptr":3,"Opt":"opt","Both":"option","Plain":"","Empty":"","Skip":"","body":null}`},
	}
	for _, r := range rows {
		t.Run(r.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/h", nil)
			for k, v := range r.header {
				req.Header[k] = v
			}
			before := req.Header.Clone()
			w := httptest.NewRecorder()
			f.ServeHTTP(w, req)
			bBOK(t, w, r.want)
			if !reflect.DeepEqual(req.Header, before) {
				t.Errorf("the caller's request header was modified: %v, was %v", req.Header, before)
			}
		})
	}
}

// TestHeaderDefaultDriftGuard keeps the rule that picks the headers that get
// a default in agreement with the document: an absent header is bound to
// the default that the generator documents, and to the zero value when it
// documents none.
func TestHeaderDefaultDriftGuard(t *testing.T) {
	bGGinTestMode(t)
	type (
		tag struct {
			H int `header:"X-H" default:"4" json:"X-H"`
		}
		option struct {
			H string `header:"X-H,default=opt" json:"X-H"`
		}
		both struct {
			H string `header:"X-H,default=opt" default:"tag" json:"X-H"`
		}
		none struct {
			H string `header:"X-H" json:"X-H"`
		}
		ptr struct {
			H *bool `header:"X-H" default:"true" json:"X-H"`
		}
	)
	rows := []struct {
		name string
		typ  reflect.Type
		app  func() *fizz.Fizz
	}{
		{"tag", reflect.TypeFor[tag](), func() *fizz.Fizz { return headerDriftApp[tag]() }},
		{"option", reflect.TypeFor[option](), func() *fizz.Fizz { return headerDriftApp[option]() }},
		{"option and tag", reflect.TypeFor[both](), func() *fizz.Fizz { return headerDriftApp[both]() }},
		{"none", reflect.TypeFor[none](), func() *fizz.Fizz { return headerDriftApp[none]() }},
		{"pointer", reflect.TypeFor[ptr](), func() *fizz.Fizz { return headerDriftApp[ptr]() }},
	}
	for _, r := range rows {
		t.Run(r.name, func(t *testing.T) {
			f := r.app()
			item := f.Generator().API().Paths["/x"]
			if item == nil || item.GET == nil || len(item.GET.Parameters) != 1 {
				t.Fatalf("GET /x has no single parameter: %v", f.Errors())
			}
			p := item.GET.Parameters[0].Value
			w := httptest.NewRecorder()
			f.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/x", nil))
			if p.Schema == nil || p.Schema.Default == nil {
				// Without a documented default tonic binds "", which only
				// a string accepts.
				if r.name == "none" {
					bBOK(t, w, `{"X-H":""}`)
				}
				return
			}
			got, _ := bBDecode(t, w).(map[string]any)
			if want := p.Schema.Default; !reflect.DeepEqual(normalize(got["X-H"]), normalize(want)) {
				t.Errorf("absent header bound %v, the document says the default is %v", got["X-H"], want)
			}
			// bind injects exactly when tonic would not apply it itself.
			plan, err := bind.PlanFor(r.typ, bind.Tags{Header: "header", Default: "default"})
			if err != nil {
				t.Fatal(err)
			}
			_, injects := bind.RewriteHeader(http.Header{}, plan)
			if want := r.name == "tag" || r.name == "pointer"; injects != want {
				t.Errorf("bind injects = %v, want %v", injects, want)
			}
		})
	}
}

func headerDriftApp[T any]() *fizz.Fizz {
	f := fizz.New()
	f.Handle("/x", "GET", []fizz.OperationOption{fizz.ID("x")}, tonic.Handler(func(c *gin.Context, in *T) (*T, error) { return in, nil }, 200))
	return f
}

// normalize turns the numbers of a documented default or of a decoded JSON
// value into float64, so that they compare.
func normalize(v any) any {
	switch n := v.(type) {
	case int:
		return float64(n)
	case int64:
		return float64(n)
	}
	return v
}
