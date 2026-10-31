package acceptance

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/loopfz/gadgeto/tonic"

	"github.com/wI2L/fizz"
	"github.com/wI2L/fizz/internal/acceptance/at"
)

// Batch B (ACCEPTANCE.md §3): runtime binding of deepObject query
// parameters and cookies, through plain HTTP against a fizz engine.
//
// Every handler echoes its input as the 200 JSON body. Fields carry no
// json tag, so the echo keys are the Go field names. The default tonic
// error hook answers 400 {"error": ...}.

type bBMapIn struct {
	Filter map[string]string `query:"filter" style:"deepObject"`
}
type bBReqIn struct {
	Filter map[string]string `query:"filter" style:"deepObject" validate:"required"`
}
type bBIntIn struct {
	Filter map[string]int `query:"filter" style:"deepObject"`
}
type bBSliceIn struct {
	Filter map[string][]string `query:"filter" style:"deepObject"`
}
type bBS struct {
	Name string   `json:"name"`
	Age  int      `json:"age"`
	Tags []string `json:"tags"`
}
type bBStructIn struct {
	Filter *bBS `query:"filter" style:"deepObject"`
}
type bBDiveIn struct {
	Filter map[string]string `query:"filter" style:"deepObject" validate:"dive,keys,min=2,endkeys,max=3"`
}
type bBMixIn struct {
	Page   int               `query:"page"`
	Filter map[string]string `query:"filter" style:"deepObject"`
}
type bBBodyIn struct {
	Name   string            `json:"name"`
	Filter map[string]string `query:"filter" style:"deepObject"`
}
type bBPathBodyIn struct {
	ID   string `path:"id"`
	Name string `json:"name"`
}
type bBCookieIn struct {
	Session string `cookie:"session"`
	Lang    string `cookie:"lang" default:"en"`
	N       int    `cookie:"n"`
}
type bBCookieReq struct {
	Token string `cookie:"token" validate:"required"`
}

// bBHandle registers h on f. Callers pass a function literal per call
// site on purpose: tonic keys routes by handler name, which generic
// instantiations would share.
func bBHandle(f *fizz.Fizz, method, path string, h gin.HandlerFunc, opts ...fizz.OperationOption) {
	f.Handle(path, method, append([]fizz.OperationOption{fizz.ID(method + path)}, opts...), h)
}

// bBNewApp returns a fizz engine serving every Batch B fixture.
func bBNewApp() *fizz.Fizz {
	f := fizz.New()
	bBHandle(f, "GET", "/map", tonic.Handler(func(c *gin.Context, in *bBMapIn) (*bBMapIn, error) { return in, nil }, 200))
	bBHandle(f, "GET", "/req", tonic.Handler(func(c *gin.Context, in *bBReqIn) (*bBReqIn, error) { return in, nil }, 200))
	bBHandle(f, "GET", "/int", tonic.Handler(func(c *gin.Context, in *bBIntIn) (*bBIntIn, error) { return in, nil }, 200))
	bBHandle(f, "GET", "/slice", tonic.Handler(func(c *gin.Context, in *bBSliceIn) (*bBSliceIn, error) { return in, nil }, 200))
	bBHandle(f, "GET", "/struct", tonic.Handler(func(c *gin.Context, in *bBStructIn) (*bBStructIn, error) { return in, nil }, 200))
	bBHandle(f, "GET", "/dive", tonic.Handler(func(c *gin.Context, in *bBDiveIn) (*bBDiveIn, error) { return in, nil }, 200))
	bBHandle(f, "GET", "/mix", tonic.Handler(func(c *gin.Context, in *bBMixIn) (*bBMixIn, error) { return in, nil }, 200))
	bBHandle(f, "POST", "/body", tonic.Handler(func(c *gin.Context, in *bBBodyIn) (*bBBodyIn, error) { return in, nil }, 200))
	bBHandle(f, "DELETE", "/i/:id", tonic.Handler(func(c *gin.Context, in *bBPathBodyIn) (*bBPathBodyIn, error) { return in, nil }, 200))
	bBHandle(f, "GET", "/i/:id", tonic.Handler(func(c *gin.Context, in *bBPathBodyIn) (*bBPathBodyIn, error) { return in, nil }, 200))
	bBHandle(f, "GET", "/cookie", tonic.Handler(func(c *gin.Context, in *bBCookieIn) (*bBCookieIn, error) { return in, nil }, 200))
	bBHandle(f, "GET", "/cookiereq", tonic.Handler(func(c *gin.Context, in *bBCookieReq) (*bBCookieReq, error) { return in, nil }, 200))
	return f
}

// bBDecode decodes the JSON body of w into any.
func bBDecode(tb testing.TB, w *httptest.ResponseRecorder) any {
	tb.Helper()
	var v any
	if err := json.Unmarshal(w.Body.Bytes(), &v); err != nil {
		tb.Fatalf("response is not JSON (%d): %q", w.Code, w.Body.String())
	}
	return v
}

// bBOK asserts a 200 whose body equals the JSON text want, semantically.
func bBOK(tb testing.TB, w *httptest.ResponseRecorder, want string) {
	tb.Helper()
	if w.Code != http.StatusOK {
		tb.Fatalf("status = %d, want 200; body %q", w.Code, w.Body.String())
	}
	var wantV any
	if err := json.Unmarshal([]byte(want), &wantV); err != nil {
		tb.Fatalf("bad want JSON %q: %v", want, err)
	}
	if got := bBDecode(tb, w); !reflect.DeepEqual(got, wantV) {
		tb.Fatalf("body = %s, want %s", w.Body.String(), want)
	}
}

// bB400 asserts a 400 whose error text contains every sub.
func bB400(tb testing.TB, w *httptest.ResponseRecorder, sub ...string) {
	tb.Helper()
	if w.Code != http.StatusBadRequest {
		tb.Fatalf("status = %d, want 400; body %q", w.Code, w.Body.String())
	}
	obj, _ := bBDecode(tb, w).(map[string]any)
	msg, _ := obj["error"].(string)
	for _, s := range sub {
		if !strings.Contains(msg, s) {
			tb.Fatalf("error %q does not contain %q", msg, s)
		}
	}
}

// B01
func TestBBDeepObjectMapBinds(t *testing.T) {
	at.Pending(t, "P7.2b", func(tb testing.TB) {
		bBOK(tb, at.Do(bBNewApp(), "GET", "/map?filter[name]=x&filter[age]=3"), `{"Filter":{"name":"x","age":"3"}}`)
	})
}

// B02
func TestBBDeepObjectDottedKeys(t *testing.T) {
	at.Pending(t, "P7.2b", func(tb testing.TB) {
		bBOK(tb, at.Do(bBNewApp(), "GET", "/map?filter[a.b]=x&filter[attr.sss]=v"), `{"Filter":{"a.b":"x","attr.sss":"v"}}`)
	})
}

// B03
func TestBBDeepObjectEncodedBrackets(t *testing.T) {
	at.Pending(t, "P7.2b", func(tb testing.TB) {
		bBOK(tb, at.Do(bBNewApp(), "GET", "/map?filter%5Bname%5D=x"), `{"Filter":{"name":"x"}}`)
	})
}

// B04
func TestBBDeepObjectNestedKey400(t *testing.T) {
	at.Pending(t, "P7.2b", func(tb testing.TB) {
		bB400(tb, at.Do(bBNewApp(), "GET", "/map?filter[a][b]=x"), "filter[a][b]")
	})
}

// B05 guard: no keys leaves the map nil (not empty). Passes today.
func TestBBDeepObjectNoKeysNilMap(t *testing.T) {
	bBOK(t, at.Do(bBNewApp(), "GET", "/map"), `{"Filter":null}`)
}

// B06
func TestBBDeepObjectValidateRequired(t *testing.T) {
	at.Pending(t, "P7.2b", func(tb testing.TB) {
		h := bBNewApp()
		bB400(tb, at.Do(h, "GET", "/req"))
		bBOK(tb, at.Do(h, "GET", "/req?filter[k]=v"), `{"Filter":{"k":"v"}}`)
	})
}

// B07
func TestBBDeepObjectValueConversion(t *testing.T) {
	at.Pending(t, "P7.2b", func(tb testing.TB) {
		h := bBNewApp()
		bBOK(tb, at.Do(h, "GET", "/int?filter[a]=3"), `{"Filter":{"a":3}}`)
		bB400(tb, at.Do(h, "GET", "/int?filter[a]=x"), "filter[a]")
	})
}

// B08 (§5 O14): repeated keys collect into slices, and are an error
// for scalar values.
func TestBBDeepObjectRepeatedKeys(t *testing.T) {
	at.Pending(t, "P7.2b", func(tb testing.TB) {
		h := bBNewApp()
		bBOK(tb, at.Do(h, "GET", "/slice?filter[t]=a&filter[t]=b"), `{"Filter":{"t":["a","b"]}}`)
		bB400(tb, at.Do(h, "GET", "/map?filter[t]=a&filter[t]=b"))
	})
}

// B09 (§5 O14): struct targets, exact json names, unknown keys ignored.
func TestBBDeepObjectStructTarget(t *testing.T) {
	at.Pending(t, "P7.2b", func(tb testing.TB) {
		h := bBNewApp()
		bBOK(tb, at.Do(h, "GET", "/struct?filter[name]=x&filter[age]=3&filter[tags]=a&filter[zzz]=1"),
			`{"Filter":{"name":"x","age":3,"tags":["a"]}}`)
		bB400(tb, at.Do(h, "GET", "/struct?filter[age]=x"))
		bBOK(tb, at.Do(h, "GET", "/struct"), `{"Filter":null}`)
		// Case-sensitive, unlike encoding/json: the key is ignored and
		// the struct stays zero-valued.
		bBOK(tb, at.Do(h, "GET", "/struct?filter[NAME]=x"), `{"Filter":{"name":"","age":0,"tags":null}}`)
	})
}

// B10: validation runs after the hook, so dive/keys sees the bound map.
func TestBBDeepObjectDiveRunsAfterBind(t *testing.T) {
	at.Pending(t, "P7.2b", func(tb testing.TB) {
		h := bBNewApp()
		bB400(tb, at.Do(h, "GET", "/dive?filter[a]=x"))
		bBOK(tb, at.Do(h, "GET", "/dive?filter[ab]=x"), `{"Filter":{"ab":"x"}}`)
	})
}

// B11 (§5 O14): malformed keys, one row each.
func TestBBDeepObjectMalformedKeys(t *testing.T) {
	for _, row := range []struct{ name, query string }{
		{"empty-brackets", "filter[]=x"},
		{"unclosed-bracket", "filter[a=x"},
	} {
		t.Run(row.name, func(t *testing.T) {
			at.Pending(t, "P7.2b", func(tb testing.TB) {
				bB400(tb, at.Do(bBNewApp(), "GET", "/map?"+row.query))
			})
		})
	}
	// Guard: a bare name already fails today, in tonic's own conversion
	// of a string into a map. It must keep answering 400.
	t.Run("bare-name", func(t *testing.T) {
		bB400(t, at.Do(bBNewApp(), "GET", "/map?filter=x"))
	})
	// Guard: a different parameter name is not ours. Passes today.
	t.Run("other-name-ignored", func(t *testing.T) {
		bBOK(t, at.Do(bBNewApp(), "GET", "/map?filterx[a]=1"), `{"Filter":null}`)
	})
}

// B12
func TestBBDeepObjectWithRegularParams(t *testing.T) {
	at.Pending(t, "P7.2b", func(tb testing.TB) {
		bBOK(tb, at.Do(bBNewApp(), "GET", "/mix?page=2&filter[a]=b"), `{"Page":2,"Filter":{"a":"b"}}`)
	})
}

// B13: the hook calls next first, so the JSON body still binds.
func TestBBDeepObjectWithJSONBody(t *testing.T) {
	at.Pending(t, "P7.2b", func(tb testing.TB) {
		w := at.Do(bBNewApp(), "POST", "/body?filter[a]=b", at.WithBody("application/json", `{"name":"n"}`))
		bBOK(tb, w, `{"name":"n","Filter":{"a":"b"}}`)
	})
}

// B14: the binding error reaches the error hook as a tonic.BindError.
func TestBBDeepObjectErrorIsBindError(t *testing.T) {
	at.Pending(t, "P7.2b", func(tb testing.TB) {
		prev := tonic.GetErrorHook()
		tb.Cleanup(func() { tonic.SetErrorHook(prev) })
		var called, isBind bool
		tonic.SetErrorHook(func(c *gin.Context, err error) (int, any) {
			called = true
			_, isBind = err.(tonic.BindError)
			return http.StatusBadRequest, gin.H{"error": err.Error()}
		})
		at.Do(bBNewApp(), "GET", "/map?filter[a][b]=x")
		if !called {
			tb.Fatal("error hook was not called")
		}
		if !isBind {
			tb.Error("error is not a tonic.BindError")
		}
	})
}

// B15: 32 concurrent requests, each echoing its own key.
func TestBBDeepObjectConcurrentRequests(t *testing.T) {
	at.Pending(t, "P7.2b", func(tb testing.TB) {
		h := bBNewApp()
		var wg sync.WaitGroup
		for i := range 32 {
			wg.Go(func() {
				// Errorf only: Fatalf must not run off the test goroutine.
				w := at.Do(h, "GET", fmt.Sprintf("/map?filter[k]=%d", i))
				want := fmt.Sprintf(`{"Filter":{"k":"%d"}}`, i)
				var got, wantV any
				_ = json.Unmarshal(w.Body.Bytes(), &got)
				_ = json.Unmarshal([]byte(want), &wantV)
				if w.Code != 200 || !reflect.DeepEqual(got, wantV) {
					tb.Errorf("request %d: %d %s, want %s", i, w.Code, w.Body.String(), want)
				}
			})
		}
		wg.Wait()
	})
}

// B16: a handler input that differs from InputModel skips the plan and
// warns, naming InputModel and the operation ID.
func TestBBInputModelMismatchSkipsPlan(t *testing.T) {
	at.Pending(t, "P7.2b", func(tb testing.TB) {
		f := fizz.New()
		bBHandle(f, "GET", "/mismatch",
			tonic.Handler(func(c *gin.Context, in *bBMixIn) (*bBMixIn, error) { return in, nil }, 200),
			fizz.ID("mismatchOp"), fizz.InputModel(&bBMapIn{}))
		found := false
		for _, err := range f.Errors() {
			if s := err.Error(); strings.Contains(s, "InputModel") && strings.Contains(s, "mismatchOp") {
				found = true
			}
		}
		if !found {
			tb.Errorf("no warning naming InputModel and mismatchOp in %v", f.Errors())
		}
		bBOK(tb, at.Do(f, "GET", "/mismatch?filter[a]=b"), `{"Page":0,"Filter":null}`)
	})
}

// B17 guard (D13 rationale): tonic binds a DELETE body. Passes today.
func TestBBDeleteBodyBoundByTonic(t *testing.T) {
	w := at.Do(bBNewApp(), "DELETE", "/i/1", at.WithBody("application/json", `{"name":"x"}`))
	bBOK(t, w, `{"ID":"1","name":"x"}`)
}

// B18 guard (D13 rationale): tonic does not bind a GET body. Passes today.
func TestBBGetBodyNotBoundByTonic(t *testing.T) {
	w := at.Do(bBNewApp(), "GET", "/i/1", at.WithBody("application/json", `{"name":"x"}`))
	bBOK(t, w, `{"ID":"1","name":""}`)
}

// B19
func TestBBCookieBinds(t *testing.T) {
	at.Pending(t, "P7.2c", func(tb testing.TB) {
		w := at.Do(bBNewApp(), "GET", "/cookie", at.WithCookie("session", "abc"), at.WithCookie("n", "5"))
		bBOK(tb, w, `{"Session":"abc","Lang":"en","N":5}`)
	})
}

// B20
func TestBBCookieDefaultWhenAbsent(t *testing.T) {
	at.Pending(t, "P7.2c", func(tb testing.TB) {
		h := bBNewApp()
		bBOK(tb, at.Do(h, "GET", "/cookie"), `{"Session":"","Lang":"en","N":0}`)
		bBOK(tb, at.Do(h, "GET", "/cookie", at.WithCookie("lang", "fr")), `{"Session":"","Lang":"fr","N":0}`)
	})
}

// B21
func TestBBCookieRequired400(t *testing.T) {
	at.Pending(t, "P7.2c", func(tb testing.TB) {
		h := bBNewApp()
		bB400(tb, at.Do(h, "GET", "/cookiereq"))
		bBOK(tb, at.Do(h, "GET", "/cookiereq", at.WithCookie("token", "t")), `{"Token":"t"}`)
	})
}

var bBWordN = regexp.MustCompile(`\bn\b`)

// B22: the error names the cookie.
func TestBBCookieBadType400(t *testing.T) {
	at.Pending(t, "P7.2c", func(tb testing.TB) {
		w := at.Do(bBNewApp(), "GET", "/cookie", at.WithCookie("n", "abc"))
		bB400(tb, w)
		if !bBWordN.MatchString(w.Body.String()) {
			tb.Errorf("error does not mention cookie n: %s", w.Body.String())
		}
	})
}

// B23 (§5 O15): present-but-empty keeps "", the default applies only
// when absent. session is sent too, so the test is not vacuous before
// cookies are bound at all (without it the assertion passes today).
func TestBBCookieEmptyVsAbsent(t *testing.T) {
	at.Pending(t, "P7.2c", func(tb testing.TB) {
		w := at.Do(bBNewApp(), "GET", "/cookie", at.WithCookie("session", "s"), at.WithCookie("lang", ""))
		bBOK(tb, w, `{"Session":"s","Lang":"","N":0}`)
	})
}

// B24 (§5 O15): cookie values are raw, with no URL decoding.
func TestBBCookieRawValue(t *testing.T) {
	at.Pending(t, "P7.2c", func(tb testing.TB) {
		w := at.Do(bBNewApp(), "GET", "/cookie", at.WithCookie("session", "a%20b"))
		bBOK(tb, w, `{"Session":"a%20b","Lang":"en","N":0}`)
	})
}
