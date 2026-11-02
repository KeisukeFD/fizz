// Package hookorder holds the Batch B tests whose outcome depends on the
// order in which the process-global tonic bind hook is installed: a user
// hook is set in TestMain before the first fizz.New (ACCEPTANCE.md B25,
// B26, B28; decision O16: New re-wraps the current hook by identity).
//
// Tests here never run in parallel and restore the global hook.
package hookorder

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/loopfz/gadgeto/tonic"

	"github.com/KeisukeFD/fizz"
	"github.com/KeisukeFD/fizz/internal/acceptance/at"
)

type bBMapIn struct {
	Filter map[string]string `query:"filter" style:"deepObject"`
}

// bBCount counts the calls of the user hook.
var bBCount atomic.Int64

// bBUserHook is the user hook: it counts, fails on demand (header
// X-Nope), and otherwise defers to tonic's default hook.
func bBUserHook(c *gin.Context, v any) error {
	bBCount.Add(1)
	if c.GetHeader("X-Nope") != "" {
		return errors.New("nope")
	}
	return tonic.DefaultBindingHook(c, v)
}

func TestMain(m *testing.M) {
	tonic.SetBindHook(bBUserHook)
	// Three New calls: with re-wrap by identity the hook must still be
	// composed exactly once, not three times.
	for range 3 {
		fizz.New()
	}
	os.Exit(m.Run())
}

func bBNewApp(opID string) *fizz.Fizz {
	f := fizz.New()
	f.GET("/map", []fizz.OperationOption{fizz.ID(opID)},
		tonic.Handler(func(c *gin.Context, in *bBMapIn) (*bBMapIn, error) { return in, nil }, 200))
	return f
}

func bBDecode(tb testing.TB, w *httptest.ResponseRecorder) map[string]any {
	tb.Helper()
	var v map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &v); err != nil {
		tb.Fatalf("response is not a JSON object (%d): %q", w.Code, w.Body.String())
	}
	return v
}

// bBWantFilter asserts a 200 echoing the given Filter value (nil: null).
func bBWantFilter(tb testing.TB, w *httptest.ResponseRecorder, want map[string]any) {
	tb.Helper()
	if w.Code != http.StatusOK {
		tb.Fatalf("status = %d, want 200; body %q", w.Code, w.Body.String())
	}
	got := bBDecode(tb, w)["Filter"]
	if want == nil {
		if got != nil {
			tb.Fatalf("Filter = %v, want null", got)
		}
		return
	}
	if !reflect.DeepEqual(got, any(want)) {
		tb.Fatalf("Filter = %v, want %v", got, want)
	}
}

// B25: a user hook set before New is composed, not replaced and not
// stacked: it runs once per request and deepObject still binds.
func TestBBUserHookComposed(t *testing.T) {
	h := bBNewApp("composed")
	bBCount.Store(0)
	w := at.Do(h, "GET", "/map?filter[a]=b")
	if n := bBCount.Load(); n != 1 {
		t.Errorf("user hook ran %d times, want 1", n)
	}
	bBWantFilter(t, w, map[string]any{"a": "b"})

}

// B26 guard: an error from the user hook short-circuits as a 400.
// Passes today.
func TestBBUserHookErrorShortCircuits(t *testing.T) {
	w := at.Do(bBNewApp("short"), "GET", "/map?filter[a]=b", at.WithHeader("X-Nope", "1"))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body %q", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "nope") {
		t.Errorf("body %q does not contain %q", w.Body.String(), "nope")
	}
}

// B28 (runs after B25): a hook installed after New replaces fizz's, so
// deepObject cannot bind. Fizz warns once, naming "bind hook" and the
// operation ID, however many requests follow.
func TestBBReplacedHookWarnsOnce(t *testing.T) {
	prevHook, prevMode, prevWriter := tonic.GetBindHook(), gin.Mode(), gin.DefaultWriter
	t.Cleanup(func() {
		tonic.SetBindHook(prevHook)
		gin.SetMode(prevMode)
		gin.DefaultWriter = prevWriter
	})
	var buf bytes.Buffer
	gin.SetMode(gin.DebugMode)
	gin.DefaultWriter = &buf

	h := bBNewApp("replacedOp")
	tonic.SetBindHook(tonic.DefaultBindingHook)

	for range 2 {
		bBWantFilter(t, at.Do(h, "GET", "/map?filter[a]=b"), nil)
	}
	n := 0
	for _, line := range strings.Split(buf.String(), "\n") {
		if strings.Contains(line, "bind hook") {
			n++
			if !strings.Contains(line, "replacedOp") {
				t.Errorf("warning does not name the operation ID: %q", line)
			}
		}
	}
	if n != 1 {
		t.Errorf("bind hook warning logged %d times, want 1; output:\n%s", n, buf.String())
	}

}
