package acceptance

// Batch D: the Parameter option. See docs/plan/ACCEPTANCE.md, D05-D08.
//
// Every package-level identifier is prefixed bPO, or is a dX type, as
// sibling files share the package.

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/loopfz/gadgeto/tonic"

	"github.com/KeisukeFD/fizz"
	"github.com/KeisukeFD/fizz/internal/acceptance/at"
	"github.com/KeisukeFD/fizz/openapi"
)

type dQuery struct {
	Q string `query:"q"`
}

const bPOParams = "/paths/~1p/get/parameters"

func bPOGet(f *fizz.Fizz, opts ...fizz.OperationOption) {
	f.GET("/p", append([]fizz.OperationOption{fizz.ID("bPOGet")}, opts...),
		tonic.Handler(func(*gin.Context, *dQuery) error { return nil }, 200))
}

func bPOParam(js string) fizz.OperationOption {
	p := at.FromJSON[openapi.Parameter](js)
	return fizz.Parameter(&p)
}

// D05.
func TestParameterMergeByNameIn(t *testing.T) {
	f := fizz.New()
	bPOGet(f, bPOParam(`{"name":"q","in":"query","description":"m","deprecated":true}`))
	doc := at.Spec(t, f)
	at.Frag(t, doc, bPOParams, `[{"name":"q","in":"query","description":"m","deprecated":true,"schema":{"type":"string"}}]`)

}

// D06.
func TestParameterAppendsWhenNew(t *testing.T) {
	f := fizz.New()
	bPOGet(f, bPOParam(`{"name":"X-Trace","in":"header","schema":{"type":"string"}}`))
	doc := at.Spec(t, f)
	at.Frag(t, doc, bPOParams+"/1", `{"name":"X-Trace","in":"header","schema":{"type":"string"}}`)
	at.Absent(t, doc, bPOParams+"/2")

}

// D07.
func TestParameterMergeLastWriterWins(t *testing.T) {
	f := fizz.New()
	bPOGet(f,
		bPOParam(`{"name":"q","in":"query","description":"a"}`),
		bPOParam(`{"name":"q","in":"query","description":"b"}`))
	doc := at.Spec(t, f)
	at.Frag(t, doc, bPOParams+"/0/description", `"b"`)
	at.Absent(t, doc, bPOParams+"/1")

}

// D08.
func TestParameterMergeKeyIncludesIn(t *testing.T) {
	f := fizz.New()
	bPOGet(f, bPOParam(`{"name":"q","in":"header","description":"h","schema":{"type":"string"}}`))
	doc := at.Spec(t, f)
	at.Frag(t, doc, bPOParams+"/0", `{"name":"q","in":"query","schema":{"type":"string"}}`)
	at.Frag(t, doc, bPOParams+"/1/in", `"header"`)
	at.Frag(t, doc, bPOParams+"/1/description", `"h"`)

}
