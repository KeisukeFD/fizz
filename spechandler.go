package fizz

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/wI2L/fizz/openapi"
)

var errInvalidContentType = errors.New("invalid content type for the OpenAPI handler")

// specEntry is the serialized document of one generation of the
// generator. The encodings are built on first use, once each.
type specEntry struct {
	gen        uint64
	json, yaml func() ([]byte, error)
}

// newSpecEntry returns the entry of generation gen. A build takes a
// snapshot of the document under the lock of the generator, which may
// be of a later generation than gen: a stale label only costs a
// rebuild, a document is never served under a label newer than itself.
// The encoding is done outside the lock.
func newSpecEntry(gen *openapi.Generator, n uint64) *specEntry {
	build := func(encode func(*openapi.OpenAPI) ([]byte, error)) func() ([]byte, error) {
		return sync.OnceValues(func() ([]byte, error) {
			doc, _, err := gen.Snapshot()
			if err != nil {
				return nil, err
			}
			return encode(doc)
		})
	}
	return &specEntry{
		gen:  n,
		json: build(openapi.MarshalJSON),
		yaml: build(openapi.MarshalYAML),
	}
}

// spec returns the entry of the current generation, building a new one
// if the generator has changed since the cached one.
func (f *Fizz) spec() *specEntry {
	for {
		cur := f.specCache.Load()
		n := f.gen.Generation()
		if cur != nil && cur.gen == n {
			return cur
		}
		if f.specCache.CompareAndSwap(cur, newSpecEntry(f.gen, n)) {
			return f.specCache.Load()
		}
	}
}

// SpecHandler returns an http.Handler that serves the OpenAPI
// specification of the API, as "json" or "yaml" (the case is ignored).
// It serves the document as of the time of each request, with the Info
// of the generator (see Generator().SetInfo), encoded once per change
// to the generator and shared by concurrent requests.
//
// Any other format is recorded in Errors, once, and the handler
// answers 500 Internal Server Error.
func (f *Fizz) SpecHandler(format string) http.Handler {
	var (
		ct     string
		encode func(*specEntry) func() ([]byte, error)
	)
	switch strings.ToLower(format) {
	case "json":
		ct = "application/json; charset=utf-8"
		encode = func(e *specEntry) func() ([]byte, error) { return e.json }
	case "yaml":
		ct = "application/yaml"
		encode = func(e *specEntry) func() ([]byte, error) { return e.yaml }
	default:
		f.gen.RecordError(fmt.Errorf("invalid content type %q for the OpenAPI handler: use json or yaml", format))
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, errInvalidContentType.Error(), http.StatusInternalServerError)
		})
	}
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		b, err := encode(f.spec())()
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		h := w.Header()
		h.Set("Content-Type", ct)
		h.Set("Content-Length", strconv.Itoa(len(b)))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(b)
	})
}

// OpenAPI returns a Gin HandlerFunc that serves the marshalled OpenAPI
// specification of the API: it sets info as the Info of the document
// and wraps SpecHandler(ct), where ct is "json" (the default when
// empty) or "yaml". An invalid content type is recorded in Errors and
// the handler answers 500.
func (f *Fizz) OpenAPI(info *openapi.Info, ct string) gin.HandlerFunc {
	f.gen.SetInfo(info)

	if ct == "" {
		ct = "json"
	}
	return gin.WrapH(f.SpecHandler(ct))
}
