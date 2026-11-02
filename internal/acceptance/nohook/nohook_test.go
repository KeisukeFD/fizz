// Package nohook holds the Batch B tests that need a process in which no
// fizz.New ever auto-installs the tonic bind hook (ACCEPTANCE.md B29,
// B30). Every fizz.New here passes fizz.WithoutBindHook.
//
// Tests here never run in parallel; B30 runs after B29 and leaves the
// hook installed.
package nohook

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
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

// bBCount counts the calls of the counting hook.
var bBCount atomic.Int64

func bBCountingHook(c *gin.Context, v any) error {
	bBCount.Add(1)
	return tonic.DefaultBindingHook(c, v)
}

func bBNewApp(opID string) *fizz.Fizz {
	f := fizz.New(fizz.WithoutBindHook())
	f.GET("/map", []fizz.OperationOption{fizz.ID(opID)},
		tonic.Handler(func(c *gin.Context, in *bBMapIn) (*bBMapIn, error) { return in, nil }, 200))
	return f
}

func bBFilter(tb testing.TB, w *httptest.ResponseRecorder) any {
	tb.Helper()
	if w.Code != http.StatusOK {
		tb.Fatalf("status = %d, want 200; body %q", w.Code, w.Body.String())
	}
	var v map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &v); err != nil {
		tb.Fatalf("response is not a JSON object: %q", w.Body.String())
	}
	return v["Filter"]
}

func bBPC(h tonic.BindHook) uintptr { return reflect.ValueOf(h).Pointer() }

// B29: WithoutBindHook leaves tonic's hook alone, and deepObject does not
// bind.
func TestBBWithoutBindHook(t *testing.T) {
	h := bBNewApp("without")
	if got, want := bBPC(tonic.GetBindHook()), bBPC(tonic.DefaultBindingHook); got != want {
		t.Fatal("tonic bind hook is not tonic.DefaultBindingHook")
	}
	if f := bBFilter(t, at.Do(h, "GET", "/map?filter[a]=b")); f != nil {
		t.Errorf("Filter = %v, want null", f)
	}

}

// B30 (runs after B29): InstallBindHook binds deepObject on the next
// request, and a second call does not wrap again.
func TestBBInstallBindHookExplicit(t *testing.T) {
	prev := tonic.GetBindHook()
	t.Cleanup(func() { tonic.SetBindHook(prev) })
	tonic.SetBindHook(bBCountingHook)

	h := bBNewApp("explicit")
	fizz.InstallBindHook()
	fizz.InstallBindHook()

	bBCount.Store(0)
	f, _ := bBFilter(t, at.Do(h, "GET", "/map?filter[a]=b")).(map[string]any)
	if !reflect.DeepEqual(f, map[string]any{"a": "b"}) {
		t.Errorf("Filter = %v, want map[a:b]", f)
	}
	if n := bBCount.Load(); n != 1 {
		t.Errorf("counting hook ran %d times, want 1", n)
	}

}
