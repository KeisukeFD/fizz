// Package query32 rebuilds learn.openapis.org examples/v3.2/3.2-query-example.json
// with the public fizz API.
package query32

import (
	"reflect"

	"github.com/gin-gonic/gin"
	"github.com/loopfz/gadgeto/tonic"

	"github.com/KeisukeFD/fizz"
	"github.com/KeisukeFD/fizz/openapi"
)

// searchBody is the body of the search. Its type is anonymous, so that
// the schema is inlined as in the official document.
var searchBody = struct {
	Origin        string `json:"origin" validate:"required" example:"ATL"`
	Destination   string `json:"destination" validate:"required" example:"LHR"`
	DepartureDate string `json:"departureDate" validate:"required" format:"date" example:"2026-06-15"`
	ReturnDate    string `json:"returnDate,omitempty" format:"date" example:"2026-06-25"`
	Passengers    int    `json:"passengers,omitempty" validate:"min=1" example:"2"`
	CabinClass    string `json:"cabinClass,omitempty" enum:"economy,premium-economy,business,first" example:"business"`
	NonStopOnly   bool   `json:"nonStopOnly,omitempty" example:"true"`
}{}

// searchResult is the response of the search.
type searchResult = struct {
	Flights []struct {
		FlightNumber  string  `json:"flightNumber,omitempty" example:"DL30"`
		Origin        string  `json:"origin,omitempty" example:"ATL"`
		Destination   string  `json:"destination,omitempty" example:"LHR"`
		DepartureTime string  `json:"departureTime,omitempty" format:"date-time" example:"2026-06-15T22:30:00Z"`
		ArrivalTime   string  `json:"arrivalTime,omitempty" format:"date-time" example:"2026-06-16T10:15:00Z"`
		Price         float32 `json:"price,omitempty" example:"1249.99"`
	} `json:"flights,omitempty"`
}

// New returns the app: a fizz whose spec is the 3.2 query example.
func New() *fizz.Fizz {
	f := fizz.New()
	g := f.Generator()
	g.SetInfo(&openapi.Info{Title: "Flight API", Version: "1.0.0"})
	// The official integers carry no format; fizz writes int64 for int.
	if err := g.OverrideDataType(reflect.TypeFor[int](), "integer", ""); err != nil {
		panic(err)
	}

	f.Handle("/flights/search", "QUERY", []fizz.OperationOption{
		fizz.ID("searchFlights"),
		fizz.Summary("Search flights with complex criteria"),
		fizz.Description("Uses the HTTP QUERY method to perform a safe, idempotent flight search with filters that are too complex for a URL query string.\n"),
		fizz.StatusDescription("Matching flights"),
		fizz.RequestBody("", true, fizz.Content("application/json", searchBody)),
	}, tonic.Handler(func(*gin.Context) (*searchResult, error) { return nil, nil }, 200))
	return f
}
