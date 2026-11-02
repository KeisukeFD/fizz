package kitchensink

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/loopfz/gadgeto/tonic"

	"github.com/KeisukeFD/fizz"
	"github.com/KeisukeFD/fizz/openapi"
)

// TestConcurrentServeAndRegister serves the spec from several goroutines
// while another registers new routes on the same generator.
func TestConcurrentServeAndRegister(t *testing.T) {
	concurrentServeAndRegister(t)
}

type raceGetIn struct {
	ID int `path:"id"`
}

func raceGet(*gin.Context, *raceGetIn) error { return nil }

func concurrentServeAndRegister(t *testing.T) {

	const (
		readers = 4
		routes  = 20
	)
	gin.SetMode(gin.TestMode)

	f := NewWholeFizz()
	r := gin.New()
	r.GET("/openapi.json", f.OpenAPI(&openapi.Info{Title: "race", Version: "0"}, "json"))
	r.GET("/openapi.yaml", f.OpenAPI(&openapi.Info{Title: "race", Version: "0"}, "yaml"))

	// tonic.Handler writes a global map without a lock, so every handler
	// is created before any goroutine starts.
	handlers := make([]gin.HandlerFunc, routes)
	for i := range handlers {
		handlers[i] = tonic.Handler(raceGet, 200)
	}

	ctx := t.Context()
	var wg sync.WaitGroup
	for i := range readers {
		path := "/openapi.json"
		if i%2 == 1 {
			path = "/openapi.yaml"
		}
		wg.Go(func() {
			for range routes {
				if ctx.Err() != nil {
					return
				}
				rec := httptest.NewRecorder()
				req := httptest.NewRequestWithContext(ctx, http.MethodGet, path, nil)
				r.ServeHTTP(rec, req)
				if rec.Code != http.StatusOK {
					t.Errorf("%s: status %d", path, rec.Code)
					return
				}
			}
		})
	}
	wg.Go(func() {
		g := f.Group("/extra", "Extra", "Concurrently added")
		for i, h := range handlers {
			g.GET("/"+strconv.Itoa(i)+"/:id", []fizz.OperationOption{
				fizz.ID("extra" + strconv.Itoa(i)),
			}, h)
		}
	})
	wg.Wait()
}
