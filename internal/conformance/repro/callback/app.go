// Package callback rebuilds learn.openapis.org
// examples/v3.0/callback-example.json with the public fizz API.
package callback

import (
	"github.com/gin-gonic/gin"
	"github.com/loopfz/gadgeto/tonic"

	"github.com/KeisukeFD/fizz"
	"github.com/KeisukeFD/fizz/openapi"
)

// subscribeIn is the input of the subscription.
type subscribeIn struct {
	CallbackURL string `query:"callbackUrl" validate:"required" format:"uri" example:"https://tonys-server.com" description:"the location where data will be sent.  Must be network accessible\nby the source server\n"`
}

// subscription is the response body. Its type is anonymous, so that the
// schema is inlined as in the official document.
type subscription = struct {
	SubscriptionID string `json:"subscriptionId" validate:"required" example:"2531329f-fb09-4ef7-887e-84e648214436" description:"this unique identifier allows management of the subscription"`
}

// onDataCallback adds the onData callback to the operation. fizz has no
// option for callbacks: CustomizeOperation is the documented way.
func onDataCallback(op *openapi.Operation) error {
	op.Callbacks = map[string]*openapi.CallbackOrRef{
		"onData": {Value: &openapi.Callback{Expressions: map[string]*openapi.PathItem{
			"{$request.query.callbackUrl}/data": {POST: &openapi.Operation{
				RequestBody: &openapi.RequestBody{
					Description: "subscription payload",
					Content: map[string]*openapi.MediaType{
						"application/json": {Schema: &openapi.Schema{
							Type: openapi.Types("object"),
							Properties: map[string]*openapi.Schema{
								"timestamp": {Type: openapi.Types("string"), Format: "date-time"},
								"userData":  {Type: openapi.Types("string")},
							},
						}},
					},
				},
				Responses: openapi.Responses{
					"202": {Value: &openapi.Response{Description: "Your server implementation should return this HTTP status code\nif the data was received successfully\n"}},
					"204": {Value: &openapi.Response{Description: "Your server should return this HTTP status code if no longer interested\nin further updates\n"}},
				},
			}},
		}}},
	}
	return nil
}

// New returns the app: a fizz whose spec is the callback example.
func New() *fizz.Fizz {
	f := fizz.New()
	f.Generator().SetInfo(&openapi.Info{Title: "Callback Example", Version: "1.0.0"})

	f.POST("/streams", []fizz.OperationOption{
		fizz.ID("subscribe"),
		fizz.Description("subscribes a client to receive out-of-band data"),
		fizz.StatusDescription("subscription successfully created"),
		fizz.CustomizeOperation(onDataCallback),
	}, tonic.Handler(func(*gin.Context, *subscribeIn) (*subscription, error) { return nil, nil }, 201))
	return f
}
