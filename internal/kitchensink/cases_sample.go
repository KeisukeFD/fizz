package kitchensink

import (
	"github.com/gin-gonic/gin"
	"github.com/loopfz/gadgeto/tonic"

	"github.com/wI2L/fizz"
	"github.com/wI2L/fizz/internal/kitchensink/a/models"
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
		Case{
			Name: "sample/fixed-by",
			Register: func(f *fizz.Fizz) {
				f.GET("/items", []fizz.OperationOption{fizz.ID("getItem")}, tonic.Handler(getItem, 200))
			},
			// 3.2 target: the document declares 3.2.x (today: 3.0.1).
			Expect:  []Fragment{{Pointer: "/openapi", JSON: `"3.2.1"`}},
			FixedBy: "P2.6",
		},
		Case{
			Name:     "sample/crashes",
			Register: func(*fizz.Fizz) { panic("never run") },
			FixedBy:  "P7.1",
			Crashes:  true,
		},
	)
}
