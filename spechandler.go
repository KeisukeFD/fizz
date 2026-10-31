package fizz

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/wI2L/fizz/openapi"
)

var errInvalidContentType = errors.New("invalid content type for the OpenAPI handler")

// OpenAPI returns a Gin HandlerFunc that serves
// the marshalled OpenAPI specification of the API. An invalid content
// type is recorded in Errors and the handler answers 500.
func (f *Fizz) OpenAPI(info *openapi.Info, ct string) gin.HandlerFunc {
	f.gen.SetInfo(info)

	ct = strings.ToLower(ct)
	if ct == "" {
		ct = "json"
	}
	switch ct {
	case "json":
		return func(c *gin.Context) {
			b, err := openapi.MarshalJSON(f.gen.API())
			if err != nil {
				_ = c.AbortWithError(500, err)
				return
			}
			c.Data(200, "application/json; charset=utf-8", b)
		}
	case "yaml":
		return func(c *gin.Context) {
			b, err := openapi.MarshalYAML(f.gen.API())
			if err != nil {
				_ = c.AbortWithError(500, err)
				return
			}
			c.Data(200, "application/yaml", b)
		}
	}
	f.gen.RecordError(fmt.Errorf("invalid content type %q for the OpenAPI handler: use json or yaml", ct))
	return func(c *gin.Context) {
		_ = c.AbortWithError(http.StatusInternalServerError, errInvalidContentType)
	}
}
