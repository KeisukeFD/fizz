package kitchensink

import (
	"github.com/gin-gonic/gin"
	"github.com/loopfz/gadgeto/tonic"

	"github.com/KeisukeFD/fizz"
	"github.com/KeisukeFD/fizz/internal/kitchensink/a/models"
)

func getItem(*gin.Context, *struct{}) (*models.Item, error) { return &models.Item{}, nil }

func init() {
	register(
		Case{
			Name: "sample/struct-schema",
			Register: func(f *fizz.Fizz) {
				f.GET("/items", []fizz.OperationOption{fizz.ID("getItem")}, tonic.Handler(getItem, 200))
			},
			// Valid in 3.0.1 and 3.2 alike.
			Expect: []Fragment{
				{Pointer: "/paths/~1items/get/operationId", JSON: `"getItem"`},
				{Pointer: "/paths/~1items/get/responses/200/description", JSON: `"OK"`},
				{Pointer: "/paths/~1items/get/requestBody", Absent: true},
			},
		},
	)
}
