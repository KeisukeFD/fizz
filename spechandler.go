package fizz

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/KeisukeFD/fizz/openapi"
	"github.com/gin-gonic/gin"
)

// specEntry is the serialized document of one generation of the
// generator. The encodings are built on first use, once each.
type specEntry struct {
	gen        uint64
	doc        func() (*openapi.OpenAPI, error) // hooked document, built once
	json, yaml func() ([]byte, error)
}

// newSpecEntry returns the entry of generation n. A build takes a
// copy of the document under the lock of the generator, which may be of
// a later generation than n: a stale label only costs a rebuild, a
// document is never served under a label newer than itself. The
// DocumentHook and the encoding run outside the lock, once per entry,
// and the JSON and YAML encodings share the hooked document.
func (f *Fizz) newSpecEntry(n uint64) *specEntry {
	doc := sync.OnceValues(func() (*openapi.OpenAPI, error) {
		doc, built, err := f.gen.Document()
		if err != nil {
			// A copy failure is recorded by the generator. A hook
			// error is recorded once per generation, even if an
			// entry is rebuilt for it.
			if errors.Is(err, openapi.ErrDocumentHook) && f.docErrGen.Swap(built+1) != built+1 {
				f.gen.RecordError(err)
			}
			return nil, err
		}
		return doc, nil
	})
	encode := func(enc func(*openapi.OpenAPI) ([]byte, error)) func() ([]byte, error) {
		return sync.OnceValues(func() ([]byte, error) {
			d, err := doc()
			if err != nil {
				return nil, err
			}
			return enc(d)
		})
	}
	return &specEntry{
		gen:  n,
		doc:  doc,
		json: encode(openapi.MarshalJSON),
		yaml: encode(openapi.MarshalYAML),
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
		if f.specCache.CompareAndSwap(cur, f.newSpecEntry(n)) {
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
			http.Error(w, "invalid content type for the OpenAPI handler", http.StatusInternalServerError)
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

// Document returns the document as served by SpecHandler: a deep copy
// with the DocumentHook applied, which the caller may change. It is
// built once per generation, like the served encodings. An error of the
// hook or of the copy is returned, and recorded in Errors once per
// generation.
func (f *Fizz) Document() (*openapi.OpenAPI, error) {
	doc, err := f.spec().doc()
	if err != nil {
		return nil, err
	}
	c, err := doc.Clone()
	if err != nil {
		return nil, fmt.Errorf("copying the document: %w", err)
	}
	return c, nil
}
