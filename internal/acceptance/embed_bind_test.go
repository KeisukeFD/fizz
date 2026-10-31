package acceptance

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"slices"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/loopfz/gadgeto/tonic"

	"github.com/wI2L/fizz"
	"github.com/wI2L/fizz/bind"
	"github.com/wI2L/fizz/openapi"
)

type (
	eFilters struct {
		Filter map[string]string `query:"filter" style:"deepObject"`
		Sess   string            `cookie:"sess"`
	}
	// EFilters is exported so that it can be embedded by pointer.
	EFilters struct {
		Filter map[string]string `query:"filter" style:"deepObject"`
		Sess   string            `cookie:"sess"`
	}
	eUnexpVal struct{ eFilters }
	eUnexpPtr struct{ *eFilters }
	eExpVal   struct{ EFilters }
	eExpPtr   struct{ *EFilters }
	eTagged   struct {
		eFilters `query:"x" style:"deepObject"`
	}
	eNested    struct{ eUnexpVal }
	eSelfCycle struct {
		*eSelfCycle
		Sess string `cookie:"sess"`
	}
)

// eFields reads Filter and Sess of the struct pointed to by in, through the
// promoted fields; a nil embedded pointer reads as empty.
func eFields(in any) (filter map[string]string, sess string) {
	v := reflect.ValueOf(in).Elem()
	defer func() { _ = recover() }()
	if f := v.FieldByName("Filter"); f.IsValid() {
		filter, _ = f.Interface().(map[string]string)
	}
	if f := v.FieldByName("Sess"); f.IsValid() {
		sess, _ = f.Interface().(string)
	}
	return filter, sess
}

func eRequest() *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/x?filter[a]=b", nil)
	r.AddCookie(&http.Cookie{Name: "sess", Value: "s1"})
	return r
}

// eBoundSet reports which parameters bind fills in a T, using the bind
// runtime path directly.
func eBoundSet[T any](t *testing.T) []string {
	t.Helper()
	p, err := bind.PlanFor(reflect.TypeFor[T](), bBTags)
	if err != nil {
		t.Fatal(err)
	}
	dst := new(T)
	r := eRequest()
	if err := bind.Query(r, dst, p); err != nil {
		t.Fatal(err)
	}
	if err := bind.Cookies(r, dst, p); err != nil {
		t.Fatal(err)
	}
	return eFilled(dst)
}

func eFilled(in any) []string {
	var got []string
	filter, sess := eFields(in)
	if filter["a"] == "b" {
		got = append(got, "query:filter")
	}
	if sess == "s1" {
		got = append(got, "cookie:sess")
	}
	slices.Sort(got)
	return got
}

// eServedSet reports which parameters a fizz handler receives for T.
func eServedSet[T any](t *testing.T) []string {
	t.Helper()
	bGGinTestMode(t)
	f := fizz.New()
	var got []string
	f.GET("/x", bGOpts(fizz.ID("x")), tonic.Handler(func(c *gin.Context, in *T) error {
		got = eFilled(in)
		return nil
	}, 200))
	w := httptest.NewRecorder()
	f.ServeHTTP(w, eRequest())
	if w.Code != 200 {
		t.Fatalf("status %d: %s", w.Code, w.Body)
	}
	return got
}

// eDoc returns the parameters the generator documents for T, and the
// number of errors recorded on the operation.
func eDoc[T any](t *testing.T) (params []string, opErrs int) {
	t.Helper()
	bGGinTestMode(t)
	f := fizz.New()
	f.GET("/x", bGOpts(fizz.ID("x")), tonic.Handler(func(*gin.Context, *T) error { return nil }, 200))
	for _, err := range f.Generator().Errors() {
		var w *openapi.Warning
		if !errors.As(err, &w) {
			opErrs++
		}
	}
	for _, p := range f.Generator().API().Paths["/x"].GET.Parameters {
		params = append(params, p.Value.In+":"+p.Value.Name)
	}
	slices.Sort(params)
	return params, opErrs
}

// TestEmbeddedDriftGuard keeps the parameters the generator documents in
// agreement with those the binder fills, for every embedding shape.
func TestEmbeddedDriftGuard(t *testing.T) {
	both := []string{"cookie:sess", "query:filter"}
	rows := []struct {
		name   string
		doc    func(*testing.T) ([]string, int)
		bound  func(*testing.T) []string
		served func(*testing.T) []string
		errs   int
		want   []string
	}{
		{"unexported value", eDoc[eUnexpVal], eBoundSet[eUnexpVal], eServedSet[eUnexpVal], 0, both},
		{"exported value", eDoc[eExpVal], eBoundSet[eExpVal], eServedSet[eExpVal], 0, both},
		{"exported pointer", eDoc[eExpPtr], eBoundSet[eExpPtr], eServedSet[eExpPtr], 0, both},
		{"nested unexported", eDoc[eNested], eBoundSet[eNested], eServedSet[eNested], 0, both},
		{"embedded with query tag", eDoc[eTagged], eBoundSet[eTagged], eServedSet[eTagged], 0, both},
		{"unexported pointer", eDoc[eUnexpPtr], eBoundSet[eUnexpPtr], nil, 1, nil},
		{"pointer cycle", eDoc[eSelfCycle], eBoundSet[eSelfCycle], nil, 0, []string{"cookie:sess"}},
	}
	for _, r := range rows {
		t.Run(r.name, func(t *testing.T) {
			doc, errs := r.doc(t)
			bound := r.bound(t)
			if !slices.Equal(doc, r.want) {
				t.Errorf("documented %v, want %v", doc, r.want)
			}
			if !slices.Equal(bound, doc) {
				t.Errorf("bind fills %v, generator documents %v", bound, doc)
			}
			// tonic itself panics on a nil unexported embedded pointer, so
			// the unexported-pointer and cycle shapes have no served row.
			if r.served != nil {
				if served := r.served(t); !slices.Equal(served, doc) {
					t.Errorf("handler receives %v, generator documents %v", served, doc)
				}
			}
			if errs != r.errs {
				t.Errorf("operation errors = %d, want %d", errs, r.errs)
			}
		})
	}
}
