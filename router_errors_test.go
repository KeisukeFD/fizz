package fizz

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/loopfz/gadgeto/tonic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KeisukeFD/fizz/openapi"
)

func ok200() gin.HandlerFunc {
	return tonic.Handler(func(c *gin.Context) error { return nil }, 200)
}

func serve(f *Fizz, method, path string) int {
	w := httptest.NewRecorder()
	f.ServeHTTP(w, httptest.NewRequest(method, path, nil))
	return w.Code
}

// TestHandleRecordsErrorsInsteadOfPanicking tests that every fatal
// condition of an operation is recorded in Errors, leaves the operation
// undocumented, and still registers the gin route.
func TestHandleRecordsErrorsInsteadOfPanicking(t *testing.T) {
	type conflictIn struct {
		A string `path:"a" query:"b"`
	}
	f := New()
	require.NotPanics(t, func() {
		f.GET("/multi", nil, ok200(), ok200())
		f.GET("/conflict/:a", nil,
			tonic.Handler(func(c *gin.Context, in *conflictIn) error { return nil }, 200))
		f.GET("/clash", []OperationOption{Response("200", "again", nil, nil, nil)}, ok200())
		f.GET("/badcode", []OperationOption{Response("1000", "bad", nil, nil, nil)}, ok200())
		f.GET("/dup1", []OperationOption{ID("dup")}, ok200())
		f.GET("/dup2", []OperationOption{ID("dup")}, ok200())
	})
	for _, p := range []string{"/multi", "/conflict/1", "/clash", "/badcode", "/dup1", "/dup2"} {
		assert.Equal(t, http.StatusOK, serve(f, "GET", p), p)
	}
	doc := f.Generator().API()
	for _, p := range []string{"/multi", "/conflict/{a}", "/clash", "/badcode", "/dup2"} {
		assert.NotContains(t, doc.Paths, p)
	}
	assert.Contains(t, doc.Paths, "/dup1")

	require.Len(t, f.Errors(), 5)
	for _, e := range f.Errors() {
		var oe *openapi.OperationError
		assert.ErrorAs(t, e, &oe)
	}
	assert.Error(t, f.Err())
}

// TestHandleFailedOperationKeepsHandlers tests that the handlers of a failed
// operation reach gin unwrapped.
func TestHandleFailedOperationKeepsHandlers(t *testing.T) {
	f := New()
	called := false
	h := func(c *gin.Context) { called = true }
	f.GET("/clash", []OperationOption{Response("200", "again", nil, nil, nil)}, ok200(), h)
	assert.Equal(t, http.StatusOK, serve(f, "GET", "/clash"))
	assert.True(t, called)
}

func TestInvalidContentTypeOpenAPIHandler(t *testing.T) {
	f := New()
	require.NotPanics(t, func() {
		f.GET("/openapi.xml", nil, f.OpenAPI(nil, "xml"))
	})
	assert.Equal(t, http.StatusInternalServerError, serve(f, "GET", "/openapi.xml"))
	require.Error(t, f.Err())
	assert.Contains(t, f.Err().Error(), "xml")
}

func TestErrExcludesWarnings(t *testing.T) {
	f := New()
	assert.NoError(t, f.Err())
	f.Generator().RecordError(&openapi.Warning{Msg: "w"})
	assert.Len(t, f.Errors(), 1)
	assert.NoError(t, f.Err())
	f.Generator().RecordError(&openapi.OperationError{Method: "GET", Path: "/x"})
	assert.ErrorContains(t, f.Err(), "GET /x")
}

// TestHandleMissingPathParamWarns tests that a path parameter missing
// from the input is synthesized with a warning, not an error, and that
// a catch-all wildcard is documented as such.
func TestHandleMissingPathParamWarns(t *testing.T) {
	f := New()
	f.GET("/u/:id/*rest", []OperationOption{ID("op")}, ok200())

	assert.NoError(t, f.Err())
	require.Len(t, f.Errors(), 2)
	for _, e := range f.Errors() {
		var w *openapi.Warning
		require.ErrorAs(t, e, &w)
		assert.Equal(t, "/u/{id}/{rest}", w.Path)
	}
	item := f.Generator().API().Paths["/u/{id}/{rest}"]
	require.NotNil(t, item)
	require.Len(t, item.GET.Parameters, 2)
	assert.Equal(t, "id", item.GET.Parameters[0].Value.Name)
	assert.Equal(t, "", item.GET.Parameters[0].Value.Description)
	assert.Equal(t, "rest", item.GET.Parameters[1].Value.Name)
	assert.Equal(t, "The value may contain '/'.", item.GET.Parameters[1].Value.Description)
}
