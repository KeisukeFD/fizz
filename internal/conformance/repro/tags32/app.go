// Package tags32 rebuilds learn.openapis.org examples/v3.2/3.2-tags-example.json
// with the public fizz API.
package tags32

import (
	"github.com/gin-gonic/gin"
	"github.com/loopfz/gadgeto/tonic"

	"github.com/wI2L/fizz"
	"github.com/wI2L/fizz/openapi"
)

// New returns the app: a fizz whose spec is the 3.2 tags example.
func New() *fizz.Fizz {
	f := fizz.New()
	g := f.Generator()
	// The official tags are in declaration order, not sorted.
	g.SetSortTags(false)
	g.SetInfo(&openapi.Info{Title: "Flight API", Version: "1.0.0"})

	g.SetTag(&openapi.Tag{
		Name:        "flights",
		Summary:     "Flights",
		Description: "Core flight operations",
		Kind:        "nav",
	})
	g.SetTag(&openapi.Tag{
		Name:        "international",
		Summary:     "International",
		Description: "Flights that cross country borders",
		Parent:      "flights",
		Kind:        "nav",
	})
	g.SetTag(&openapi.Tag{
		Name:        "domestic",
		Summary:     "Domestic",
		Description: "Flights within a single country",
		Parent:      "flights",
		Kind:        "nav",
	})
	g.SetTag(&openapi.Tag{
		Name:         "delays",
		Summary:      "Delays",
		Description:  "Information about flight delays",
		Kind:         "badge",
		ExternalDocs: &openapi.ExternalDocs{URL: "https://docs.example.com/delay-policies", Description: "Delay compensation policies"},
	})

	handler := tonic.Handler(func(*gin.Context) error { return nil }, 200)
	f.GET("/flights", []fizz.OperationOption{
		fizz.ID("listFlights"),
		fizz.Summary("List all flights"),
		fizz.Tags("flights"),
	}, handler)
	f.GET("/flights/international", []fizz.OperationOption{
		fizz.ID("listInternationalFlights"),
		fizz.Summary("List international flights"),
		fizz.Tags("international"),
	}, handler)
	f.GET("/flights/domestic", []fizz.OperationOption{
		fizz.ID("listDomesticFlights"),
		fizz.Summary("List domestic flights"),
		fizz.Tags("domestic"),
	}, handler)
	f.GET("/flights/delayed", []fizz.OperationOption{
		fizz.ID("getDelayedFlights"),
		fizz.Summary("Get delayed flights"),
		fizz.Tags("delays"),
	}, handler)
	return f
}
