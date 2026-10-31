package acceptance

import (
	"math"
	"net/http"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/loopfz/gadgeto/tonic"

	"github.com/wI2L/fizz"
	"github.com/wI2L/fizz/internal/acceptance/at"
	"github.com/wI2L/fizz/openapi"
)

// Batch G (lifecycle, errors, context): the tests that compile against
// today's API. tonic.Handler writes a process-global map, so none of
// these tests runs in parallel.

type (
	bGEmpty struct{}

	bGOmitIn struct {
		Q string `query:"q,omitempty"`
	}
	bGConflictIn struct {
		ID string `query:"id" path:"id"`
	}
	bGChanBody struct {
		C chan int `json:"c"`
	}
	bGBadBoolIn struct {
		B bool `query:"b" default:"maybe"`
	}
	bGDurationIn struct {
		D time.Duration `query:"d" default:"1s"`
	}
	bGCookieStruct struct {
		A string `json:"a"`
	}
	bGCookieIn struct {
		S bGCookieStruct `cookie:"s"`
	}
	bGDeepStructIn struct {
		F map[string]struct{} `query:"f" style:"deepObject"`
	}
	bGDeepRequiredIn struct {
		F map[string]string `query:"f,required" style:"deepObject"`
	}
	bGErrIn struct {
		B bool `query:"b" default:"maybe"`
		N int  `query:"n" default:"3" validate:"required"`
	}
	bGOut struct {
		Value string `json:"value"`
	}
)

func bGHandler[T any](*gin.Context, *T) error { return nil }

func bGOpts(o ...fizz.OperationOption) []fizz.OperationOption { return o }

// bGGinTestMode sets gin to test mode for the duration of t.
func bGGinTestMode(t testing.TB) {
	t.Helper()
	prev := gin.Mode()
	gin.SetMode(gin.TestMode)
	t.Cleanup(func() { gin.SetMode(prev) })
}

// bGRow is one row of TestNoPanicsOnBadInput.
type bGRow struct {
	name    string
	fixedBy string // "" when the row passes today
	method  string
	path    string // the route, and the substring Errors() must contain
	// want overrides path as the substring Errors() must contain.
	want     string
	register func(f *fizz.Fizz)
	// extra runs after the generic checks.
	extra func(tb testing.TB, f *fizz.Fizz)
}

func bGRows() []bGRow {
	get := http.MethodGet
	return []bGRow{
		{
			name: "r01-omitempty", fixedBy: "P4.2+P4.8", method: get, path: "/r01",
			register: func(f *fizz.Fizz) {
				f.GET("/r01", bGOpts(fizz.ID("r01")), tonic.Handler(bGHandler[bGOmitIn], 200))
			},
		},
		{
			name: "r02-response-code-clash", fixedBy: "P4.8", method: get, path: "/r02",
			register: func(f *fizz.Fizz) {
				f.GET("/r02", bGOpts(fizz.ID("r02"), fizz.Response("200", "again", nil, nil, nil)),
					tonic.Handler(bGHandler[bGEmpty], 200))
			},
		},
		{
			name: "r03-conflicting-locations", fixedBy: "P4.8", method: get, path: "/r03/1",
			want: "/r03/",
			register: func(f *fizz.Fizz) {
				f.GET("/r03/:id", bGOpts(fizz.ID("r03")), tonic.Handler(bGHandler[bGConflictIn], 200))
			},
		},
		{
			name: "r04-duplicate-explicit-id", fixedBy: "P4.8", method: get, path: "/r04b",
			register: func(f *fizz.Fizz) {
				f.GET("/r04a", bGOpts(fizz.ID("r04")), tonic.Handler(bGHandler[bGEmpty], 200))
				f.GET("/r04b", bGOpts(fizz.ID("r04")), tonic.Handler(bGHandler[bGEmpty], 200))
			},
		},
		{
			name: "r05-two-tonic-handlers", fixedBy: "P4.8", method: get, path: "/r05",
			register: func(f *fizz.Fizz) {
				f.GET("/r05", bGOpts(fizz.ID("r05")),
					tonic.Handler(bGHandler[bGEmpty], 200), tonic.Handler(bGHandler[bGEmpty], 200))
			},
		},
		{
			name: "r06-chan-body-field", fixedBy: "P3.3+P4.8", method: http.MethodPost, path: "/r06",
			register: func(f *fizz.Fizz) {
				f.POST("/r06", bGOpts(fizz.ID("r06")), tonic.Handler(bGHandler[bGChanBody], 200))
			},
		},
		{
			name: "r07-nan-example", fixedBy: "P2.5", method: get, path: "/r07",
			register: func(f *fizz.Fizz) {
				f.GET("/r07", bGOpts(fizz.ID("r07"),
					fizz.ResponseWithExamples("400", "bad", new(bGOut), nil, map[string]any{"bad": math.NaN()})),
					tonic.Handler(bGHandler[bGEmpty], 200))
				f.GET("/r07-spec", nil, f.OpenAPI(&openapi.Info{Title: "t", Version: "0"}, "json"))
			},
			extra: func(tb testing.TB, f *fizz.Fizz) {
				if w := at.Do(f, get, "/r07-spec"); w.Code != http.StatusOK {
					tb.Errorf("serving the spec: status %d, want 200", w.Code)
				}
			},
		},
		{
			name: "r08-bool-default", fixedBy: "P3.6+P4.8", method: get, path: "/r08",
			register: func(f *fizz.Fizz) {
				f.GET("/r08", bGOpts(fizz.ID("r08")), tonic.Handler(bGHandler[bGBadBoolIn], 200))
			},
		},
		{
			name: "r09-duration-default", fixedBy: "P3.6+P4.8", method: get, path: "/r09",
			register: func(f *fizz.Fizz) {
				f.GET("/r09", bGOpts(fizz.ID("r09")), tonic.Handler(bGHandler[bGDurationIn], 200))
			},
		},
		{
			name: "r10-cookie-on-struct", fixedBy: "P4.2c+P4.8", method: get, path: "/r10",
			register: func(f *fizz.Fizz) {
				f.GET("/r10", bGOpts(fizz.ID("r10")), tonic.Handler(bGHandler[bGCookieIn], 200))
			},
		},
		{
			name: "r11-deepobject-struct-value", fixedBy: "P7.1", method: get, path: "/r11",
			register: func(f *fizz.Fizz) {
				f.GET("/r11", bGOpts(fizz.ID("r11")), tonic.Handler(bGHandler[bGDeepStructIn], 200))
			},
		},
		{
			name: "r12-deepobject-required-option", fixedBy: "P7.1", method: get, path: "/r12",
			register: func(f *fizz.Fizz) {
				f.GET("/r12", bGOpts(fizz.ID("r12")), tonic.Handler(bGHandler[bGDeepRequiredIn], 200))
			},
		},
		{
			// The error names the scheme: it has no route of its own.
			name: "r13-empty-security-scheme", fixedBy: "P4.8", method: get, path: "/r13", want: "r13scheme",
			register: func(f *fizz.Fizz) {
				f.GET("/r13", bGOpts(fizz.ID("r13")), tonic.Handler(bGHandler[bGEmpty], 200))
				f.Generator().SetSecuritySchemes(map[string]*openapi.SecuritySchemeOrRef{"r13scheme": {}})
			},
			extra: func(tb testing.TB, f *fizz.Fizz) { at.Spec(tb, f) },
		},
	}
}

// TestNoPanicsOnBadInput registers one bad route per row on a single
// fizz. Each row must not panic, keep its gin route (anything but 404)
// and record an error that names the route.
func TestNoPanicsOnBadInput(t *testing.T) {
	bGGinTestMode(t)
	f := fizz.New()
	for _, r := range bGRows() {
		t.Run(r.name, func(t *testing.T) {
			// Pending with an empty task turns failures and panics into
			// plain failures: that is the unwrapped form for passing rows.
			at.Pending(t, r.fixedBy, func(tb testing.TB) {
				tb.Helper()
				r.register(f) // a panic fails the row
				if w := at.Do(f, r.method, r.path); w.Code == http.StatusNotFound {
					tb.Errorf("%s %s: gin route not registered (404)", r.method, r.path)
				}
				want := r.want
				if want == "" {
					want = r.path
				}
				if !slices.ContainsFunc(f.Errors(), func(err error) bool {
					return strings.Contains(err.Error(), want)
				}) {
					tb.Errorf("no Errors() entry contains %q; errors: %v", want, f.Errors())
				}
				if r.extra != nil {
					r.extra(tb, f)
				}
			})
		})
	}
}

// bGErrorsFizz returns a fizz with at least one recorded error.
func bGErrorsFizz(t testing.TB) *fizz.Fizz {
	t.Helper()
	f := fizz.New()
	f.GET("/g04", bGOpts(fizz.ID("g04")), tonic.Handler(bGHandler[bGErrIn], 200))
	if len(f.Errors()) < 1 {
		t.Fatalf("fixture records %d errors, want at least 1", len(f.Errors()))
	}
	return f
}

func TestErrorsReturnsCopy(t *testing.T) {
	at.Pending(t, "P4.8", func(tb testing.TB) {
		f := bGErrorsFizz(tb)
		first := f.Errors()
		want := make([]string, len(first))
		for i, e := range first {
			want[i] = e.Error()
		}
		first[0] = nil
		got := f.Errors()
		if len(got) != len(want) {
			tb.Fatalf("Errors() length changed: got %d, want %d", len(got), len(want))
		}
		for i, e := range got {
			if e == nil || e.Error() != want[i] {
				tb.Fatalf("Errors()[%d] changed after the caller mutated a previous result", i)
			}
		}
	})
}

func TestFailedOpLeavesNoTrace(t *testing.T) {
	at.Pending(t, "P4.8", func(tb testing.TB) {
		bGGinTestMode(tb)
		f := fizz.New()
		f.GET("/clash", bGOpts(fizz.ID("x"), fizz.Response("200", "again", nil, nil, nil)),
			tonic.Handler(bGHandler[bGEmpty], 200))
		at.Absent(tb, at.Spec(tb, f), "/paths/~1clash")

		f.GET("/second", bGOpts(fizz.ID("x")), tonic.Handler(bGHandler[bGEmpty], 200))
		for _, err := range f.Errors() {
			if s := err.Error(); strings.Contains(s, "/second") || strings.Contains(s, "already used") {
				tb.Errorf("reusing ID x after a failed operation recorded: %v", err)
			}
		}
		at.Frag(tb, at.Spec(tb, f), "/paths/~1second/get/operationId", `"x"`)
	})
}

// bGSpecFizz serves the JSON spec on /openapi.json.
func bGSpecFizz() *fizz.Fizz {
	f := fizz.New()
	f.GET("/openapi.json", nil, f.OpenAPI(&openapi.Info{Title: "g", Version: "0"}, "json"))
	return f
}

func TestSpecCacheInvalidation(t *testing.T) {
	bGGinTestMode(t)
	f := bGSpecFizz()
	f.GET("/first", bGOpts(fizz.ID("first")), tonic.Handler(bGHandler[bGEmpty], 200))

	a := at.Do(f, http.MethodGet, "/openapi.json")
	b := at.Do(f, http.MethodGet, "/openapi.json")
	if a.Code != http.StatusOK || a.Body.String() != b.Body.String() {
		t.Fatalf("two GETs differ (status %d, %d)", a.Code, b.Code)
	}
	f.GET("/added", bGOpts(fizz.ID("added")), tonic.Handler(bGHandler[bGEmpty], 200))
	c := at.Do(f, http.MethodGet, "/openapi.json")
	at.Frag(t, c.Body.Bytes(), "/paths/~1added/get/operationId", `"added"`)
	at.Frag(t, c.Body.Bytes(), "/paths/~1first/get/operationId", `"first"`)
}

func TestAPIReturnsDeepClone(t *testing.T) {
	at.Pending(t, "P5.1", func(tb testing.TB) {
		bGGinTestMode(tb)
		f := bGSpecFizz()
		f.Generator().SetInfo(&openapi.Info{Title: "orig", Version: "0"})
		f.GET("/x", bGOpts(fizz.ID("x")), tonic.Handler(bGHandler[bGEmpty], 200))
		before := at.Do(f, http.MethodGet, "/openapi.json").Body.String()

		a := f.Generator().API()
		a.Paths["/x"] = nil
		a.Info.Title = "mutated"

		if got := f.Generator().API(); got.Paths["/x"] == nil || got.Info.Title != "orig" {
			tb.Errorf("next API() sees the caller's mutation: path nil=%t, title=%q",
				got.Paths["/x"] == nil, got.Info.Title)
		}
		if after := at.Do(f, http.MethodGet, "/openapi.json").Body.String(); after != before {
			tb.Errorf("served document changed after mutating API():\nbefore %s\nafter  %s", before, after)
		}
	})
}

// bGOpRecord is what a handler observed about its operation.
type bGOpRecord struct {
	ctxOp, ctxOp2, ginOp *openapi.Operation
	ctxErr, ginErr       error
	keysHoldOperation    bool
}

// bGOpHandler returns a handler that inspects its operation and, when
// mutate is set, edits the first copy it gets.
func bGOpHandler(rec *bGOpRecord, mutate bool) gin.HandlerFunc {
	return tonic.Handler(func(c *gin.Context, _ *bGEmpty) error {
		ctx := c.Request.Context()
		rec.ctxOp, rec.ctxErr = fizz.OperationFromContext(ctx)
		rec.ctxOp2, _ = fizz.OperationFromContext(ctx)
		rec.ginOp, rec.ginErr = fizz.OperationFromContext(c)
		for _, v := range c.Keys {
			if _, ok := v.(*openapi.Operation); ok {
				rec.keysHoldOperation = true
			}
		}
		if mutate && rec.ctxOp != nil {
			rec.ctxOp.Summary = "mutated"
			rec.ctxOp.Tags = append(rec.ctxOp.Tags, "mutated")
		}
		return nil
	}, 200)
}

func TestOperationInRequestContext(t *testing.T) {
	at.Pending(t, "P5.1", func(tb testing.TB) {
		bGGinTestMode(tb)
		var rec bGOpRecord
		f := fizz.New()
		f.GET("/ctx", bGOpts(fizz.ID("getCtx")), bGOpHandler(&rec, false))
		if w := at.Do(f, http.MethodGet, "/ctx"); w.Code != http.StatusOK {
			tb.Fatalf("status %d: %s", w.Code, w.Body)
		}
		if rec.ctxErr != nil || rec.ctxOp == nil {
			tb.Fatalf("operation not in the request context: %v", rec.ctxErr)
		}
		if rec.ctxOp.ID != "getCtx" {
			tb.Errorf("operation ID %q, want getCtx", rec.ctxOp.ID)
		}
		if rec.keysHoldOperation {
			tb.Error("c.Keys holds a *openapi.Operation; it must travel in the request context")
		}
	})
}

func TestOperationFromContextIsClone(t *testing.T) {
	at.Pending(t, "P5.1", func(tb testing.TB) {
		bGGinTestMode(tb)
		var rec bGOpRecord
		f := bGSpecFizz()
		bGNoContextFallback(f.Engine())
		f.GET("/clone", bGOpts(fizz.ID("getClone"), fizz.Summary("orig")), bGOpHandler(&rec, true))
		before := at.Do(f, http.MethodGet, "/openapi.json").Body.String()
		if w := at.Do(f, http.MethodGet, "/clone"); w.Code != http.StatusOK {
			tb.Fatalf("status %d: %s", w.Code, w.Body)
		}
		if rec.ctxOp == nil || rec.ctxOp2 == nil {
			tb.Fatalf("operation not in the request context: %v", rec.ctxErr)
		}
		if rec.ctxOp == rec.ctxOp2 {
			tb.Error("two calls returned the same pointer, want distinct clones")
		}
		if after := at.Do(f, http.MethodGet, "/openapi.json").Body.String(); after != before {
			tb.Errorf("spec changed after the handler mutated its operation:\nbefore %s\nafter  %s", before, after)
		}
		if rec.ginErr != nil || rec.ginOp == nil {
			tb.Errorf("OperationFromContext(*gin.Context) with ContextWithFallback=false: %v", rec.ginErr)
		}
	})
}

// bGNoContextFallback clears Engine.ContextWithFallback. gin v1.7.7
// has no such field (it is false by construction); later versions do,
// hence the reflection.
func bGNoContextFallback(e *gin.Engine) {
	if v := reflect.ValueOf(e).Elem().FieldByName("ContextWithFallback"); v.IsValid() && v.CanSet() {
		v.SetBool(false)
	}
}
