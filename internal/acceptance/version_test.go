package acceptance

import (
	"net/http"
	"testing"

	"github.com/KeisukeFD/fizz"
	"github.com/KeisukeFD/fizz/internal/acceptance/at"
	"github.com/KeisukeFD/fizz/openapi"
)

// TestVersionConstant (G22): openapi.Version is "3.2.1" and is the
// version the served document declares.
func TestVersionConstant(t *testing.T) {
	if openapi.Version != "3.2.1" {
		t.Fatalf("openapi.Version = %q, want 3.2.1", openapi.Version)
	}
	bGGinTestMode(t)
	f := fizz.New()
	f.GET("/openapi.json", nil, f.OpenAPI(&openapi.Info{Title: "v", Version: "1"}, "json"))
	w := at.Do(f, http.MethodGet, "/openapi.json")
	if w.Code != http.StatusOK {
		t.Fatalf("status %d", w.Code)
	}
	doc := at.FromJSON[map[string]any](w.Body.String())
	if got := doc["openapi"]; got != openapi.Version {
		t.Errorf("served openapi = %v, want %q", got, openapi.Version)
	}
}
