package fizz

import (
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"

	"github.com/gin-gonic/gin"
	"github.com/loopfz/gadgeto/tonic"

	"github.com/wI2L/fizz/openapi"
)

// buildCounter is an extension value that counts how many times the
// document is encoded from the generator's own state. The cache takes a
// clone under the lock, which encodes it once; the copy holds a plain
// number and the later encodings do not count.
type buildCounter struct{ n atomic.Int64 }

func (c *buildCounter) MarshalJSON() ([]byte, error) {
	c.n.Add(1)
	return []byte(`1`), nil
}

func serveSpec(h http.Handler) string {
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	return w.Body.String()
}

// TestSpecCacheInvalidation checks that concurrent readers share one
// build per generation, and that a change is served by the next read.
func TestSpecCacheInvalidation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		counter := new(buildCounter)
		info := &openapi.Info{Title: "cache", Version: "1"}
		if err := info.SetExtension("x-builds", counter); err != nil {
			t.Fatal(err)
		}
		f := New()
		f.Generator().SetInfo(info)
		h := f.SpecHandler("json")

		readAll := func() []string {
			var wg sync.WaitGroup
			out := make([]string, 8)
			for i := range out {
				wg.Go(func() { out[i] = serveSpec(h) })
			}
			wg.Wait()
			return out
		}
		same := func(bodies []string) {
			t.Helper()
			for _, b := range bodies[1:] {
				if b != bodies[0] {
					t.Fatalf("concurrent reads differ:\n%s\n%s", bodies[0], b)
				}
			}
		}

		first := readAll()
		same(first)
		if n := counter.n.Load(); n != 1 {
			t.Fatalf("%d builds for the first generation, want 1", n)
		}
		if serveSpec(h) != first[0] || counter.n.Load() != 1 {
			t.Fatal("a repeated read rebuilt the document")
		}
		if strings.Contains(first[0], "/added") {
			t.Fatal("unexpected path")
		}

		// A change after a serve shows up in the next one.
		f.GET("/added", []OperationOption{ID("added")}, tonic.Handler(func(*gin.Context, *struct{}) error { return nil }, 200))
		second := readAll()
		same(second)
		if !strings.Contains(second[0], `"/added"`) {
			t.Fatalf("change not served: %s", second[0])
		}
		if n := counter.n.Load(); n != 2 {
			t.Fatalf("%d builds after one change, want 2", n)
		}

		f.Generator().AddTag("late", "a late tag")
		if b := serveSpec(h); !strings.Contains(b, `"late"`) {
			t.Fatalf("tag not served: %s", b)
		}
		if n := counter.n.Load(); n != 3 {
			t.Fatalf("%d builds after two changes, want 3", n)
		}
	})
}

// TestSnapshotErrorRecordedOncePerGeneration: an unencodable document
// makes API return nil and the handler answer 500, with one error.
func TestSnapshotErrorRecordedOncePerGeneration(t *testing.T) {
	f := New()
	info := &openapi.Info{Title: "nan", Version: "1"}
	if err := info.SetExtension("x-nan", math.NaN()); err != nil {
		t.Fatal(err)
	}
	f.Generator().SetInfo(info)
	h := f.SpecHandler("json")
	for range 2 {
		if f.Generator().API() != nil {
			t.Fatal("API() should be nil")
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("status %d, want 500", w.Code)
		}
	}
	if n := len(f.Errors()); n != 1 {
		t.Fatalf("%d errors, want 1: %v", n, f.Errors())
	}
}

// TestSpecHandlerUnknownFormat records the error once, at construction.
func TestSpecHandlerUnknownFormat(t *testing.T) {
	f := New()
	h := f.SpecHandler("xml")
	if n := len(f.Errors()); n != 1 {
		t.Fatalf("%d errors recorded, want 1", n)
	}
	for range 2 {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("status %d, want 500", w.Code)
		}
	}
	if n := len(f.Errors()); n != 1 {
		t.Fatalf("%d errors after serving, want 1", n)
	}
}
