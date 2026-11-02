package acceptance

// Batch D (default build): the RequestBody option. See
// docs/plan/ACCEPTANCE.md, D01-D04.
//
// Every package-level identifier is prefixed bRB, or is a dX type, as
// sibling files share the package.

import (
	"errors"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/loopfz/gadgeto/tonic"

	"github.com/KeisukeFD/fizz"
	"github.com/KeisukeFD/fizz/internal/acceptance/at"
	"github.com/KeisukeFD/fizz/openapi"
)

type (
	dPet struct {
		Name string `json:"name"`
	}
	dReqPet struct {
		Name string `json:"name" validate:"required"`
	}
	dUpload struct {
		File string `json:"file"`
		Note string `json:"note"`
	}
)

const bRBBody = "/paths/~1p/post/requestBody"

func bRBPost[In any](f *fizz.Fizz, opts ...fizz.OperationOption) {
	f.POST("/p", append([]fizz.OperationOption{fizz.ID("bRBPost")}, opts...),
		tonic.Handler(func(*gin.Context, *In) error { return nil }, 200))
}

// D01.
func TestRequestBodyOverridesRequired(t *testing.T) {
	f := fizz.New()
	bRBPost[dPet](f, fizz.RequestBody("Create", true))
	doc := at.Spec(t, f)
	at.Frag(t, doc, bRBBody+"/description", `"Create"`)
	at.Frag(t, doc, bRBBody+"/required", `true`)
	at.Frag(t, doc, bRBBody+"/content/application~1json/schema/$ref", `"#/components/schemas/BRBPostInput"`)

	f = fizz.New()
	bRBPost[dReqPet](f, fizz.RequestBody("", false))
	doc = at.Spec(t, f)
	at.Absent(t, doc, bRBBody+"/required")
	at.Absent(t, doc, bRBBody+"/description")
	at.Frag(t, doc, bRBBody+"/content/application~1json/schema/$ref", `"#/components/schemas/BRBPostInput"`)

}

// D02.
func TestRequestBodyContentReplaces(t *testing.T) {
	f := fizz.New()
	bRBPost[dReqPet](f, fizz.RequestBody("", true, fizz.Content("application/xml", dPet{})))
	doc := at.Spec(t, f)
	at.Frag(t, doc, bRBBody+"/content", `{"application/xml":{"schema":{"$ref":"#/components/schemas/AcceptanceDPet"}}}`)
	at.Frag(t, doc, bRBBody+"/required", `true`)

}

// D03.
func TestRequestBodyEncoding(t *testing.T) {
	f := fizz.New()
	bRBPost[dPet](f, fizz.RequestBody("", false,
		fizz.Content("multipart/form-data", dUpload{}, fizz.Encoding("file", "image/png"))))
	doc := at.Spec(t, f)
	at.Frag(t, doc, bRBBody+"/content/multipart~1form-data/encoding/file/contentType", `"image/png"`)
	for _, err := range f.Generator().Errors() {
		t.Errorf("unexpected error: %v", err)
	}

}

// D04.
func TestRequestBodyOnGETIsError(t *testing.T) {
	f := fizz.New()
	f.GET("/p", []fizz.OperationOption{fizz.ID("bRBGet"), fizz.RequestBody("x", true, fizz.Content("application/json", dPet{}))},
		tonic.Handler(func(*gin.Context) error { return nil }, 200))
	doc := at.Spec(t, f)
	at.Absent(t, doc, "/paths/~1p/get/requestBody")
	var oe *openapi.OperationError
	if !errors.As(f.Err(), &oe) || oe.Method != "GET" || oe.Path != "/p" || !strings.Contains(oe.Error(), "bRBGet") {
		t.Errorf("Err = %v, want an OperationError naming GET /p (bRBGet)", f.Err())
	}

}
