// Package nonoauthscopes rebuilds learn.openapis.org
// examples/v3.1/non-oauth-scopes.json with the public fizz API.
package nonoauthscopes

import (
	"github.com/gin-gonic/gin"
	"github.com/loopfz/gadgeto/tonic"

	"github.com/KeisukeFD/fizz"
	"github.com/KeisukeFD/fizz/openapi"
)

// New returns the app: a fizz whose spec is the non-oauth-scopes example.
func New() *fizz.Fizz {
	f := fizz.New()
	g := f.Generator()
	g.SetInfo(&openapi.Info{Title: "Non-oAuth Scopes example", Version: "1.0.0"})
	g.SetSecuritySchemes(map[string]*openapi.SecuritySchemeOrRef{
		"bearerAuth": {Value: &openapi.SecurityScheme{
			Type:         "http",
			Scheme:       "bearer",
			BearerFormat: "jwt",
			Description:  "note: non-oauth scopes are not defined at the securityScheme level",
		}},
	})

	f.GET("/users", []fizz.OperationOption{
		fizz.Security(&openapi.SecurityRequirement{"bearerAuth": {"read:users", "public"}}),
	}, tonic.Handler(func(*gin.Context) error { return nil }, 200))
	return f
}
