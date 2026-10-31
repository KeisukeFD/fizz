package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/wI2L/fizz/internal/acceptance/at"
	"github.com/wI2L/fizz/internal/kitchensink"
)

func bINewMarket(tb testing.TB) http.Handler {
	tb.Helper()
	r, err := NewRouter()
	if err != nil {
		tb.Fatalf("NewRouter: %v", err)
	}
	return r
}

// I13
func TestMarketSpecValid(t *testing.T) {
	at.Pending(t, "P9.3", func(tb testing.TB) {
		h := bINewMarket(tb)
		w := at.Do(h, "GET", "/openapi.json")
		if w.Code != 200 {
			tb.Fatalf("GET /openapi.json = %d", w.Code)
		}
		doc := w.Body.Bytes()
		for _, p := range kitchensink.MetaProblems(doc) {
			tb.Error(p)
		}
		var spec struct {
			TagGroups any `json:"x-tagGroups"`
			Paths     map[string]map[string]struct {
				Parameters []struct {
					Name  string `json:"name"`
					In    string `json:"in"`
					Style string `json:"style"`
				} `json:"parameters"`
			} `json:"paths"`
		}
		if err := json.Unmarshal(doc, &spec); err != nil {
			tb.Fatal(err)
		}
		if spec.TagGroups == nil {
			tb.Error("root x-tagGroups missing")
		}
		deep, cookies := false, 0
		for _, item := range spec.Paths {
			for method, op := range item {
				for _, p := range op.Parameters {
					if method == "get" && p.Name == "filter" && p.In == "query" && p.Style == "deepObject" {
						deep = true
					}
					if p.In == "cookie" {
						cookies++
					}
				}
			}
		}
		if !deep {
			tb.Error("ListFruits has no query parameter filter with style deepObject")
		}
		if cookies < 1 {
			tb.Error("no in: cookie parameter")
		}
		d := at.Do(h, "GET", "/docs")
		if d.Code != 200 || !strings.Contains(d.Body.String(), "@scalar/api-reference") {
			tb.Errorf("GET /docs = %d, want 200 serving @scalar/api-reference", d.Code)
		}
	})
}

// I15
func TestMarketCurrencyCookie(t *testing.T) {
	at.Pending(t, "P9.3", func(tb testing.TB) {
		h := bINewMarket(tb)
		w := at.Do(h, "GET", "/market?origin=spain")
		if w.Code != 200 {
			tb.Fatalf("default currency = %d: %s", w.Code, w.Body)
		}
		if got := w.Header().Get("X-Market-Currency"); got != "EUR" {
			tb.Errorf("default X-Market-Currency = %q, want EUR", got)
		}
		w = at.Do(h, "GET", "/market?origin=spain", at.WithCookie("currency", "USD"))
		if got := w.Header().Get("X-Market-Currency"); got != "USD" {
			tb.Errorf("X-Market-Currency with cookie = %q, want USD", got)
		}
	})
}

// I14
func TestMarketFilterEndpoint(t *testing.T) {
	at.Pending(t, "P9.3", func(tb testing.TB) {
		h := bINewMarket(tb)
		w := at.Do(h, "GET", "/market?filter[origin]=Spain")
		if w.Code != 200 {
			tb.Fatalf("filter[origin]=Spain = %d: %s", w.Code, w.Body)
		}
		var fruits []Fruit
		if err := json.Unmarshal(w.Body.Bytes(), &fruits); err != nil {
			tb.Fatal(err)
		}
		if len(fruits) == 0 {
			tb.Fatal("no Spanish fruit returned")
		}
		for _, f := range fruits {
			// The seed data stores origins in lower case.
			if !strings.EqualFold(f.Origin, "spain") {
				tb.Errorf("fruit %q has origin %q", f.Name, f.Origin)
			}
		}
		if w := at.Do(h, "GET", "/market?filter[a][b]=x"); w.Code != 400 {
			tb.Errorf("filter[a][b]=x = %d, want 400", w.Code)
		}
	})
}
