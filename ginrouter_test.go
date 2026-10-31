package fizz

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestGinRouterGroup(t *testing.T) {
	f := New()
	if f.GinRouterGroup() == nil {
		t.Fatal("GinRouterGroup returned nil")
	}
	f.GinRouterGroup().GET("/raw", func(c *gin.Context) { c.String(200, "ok") })
	w := httptest.NewRecorder()
	f.ServeHTTP(w, httptest.NewRequest("GET", "/raw", nil))
	if w.Code != 200 || w.Body.String() != "ok" {
		t.Errorf("got %d %q", w.Code, w.Body.String())
	}
}
