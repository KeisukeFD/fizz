package acceptance

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/loopfz/gadgeto/tonic"

	"github.com/KeisukeFD/fizz"
	"github.com/KeisukeFD/fizz/bind"
)

// Runtime follows the document (D26): label and matrix path parameters
// (P10.7k), through plain HTTP.

type bLabelIn struct {
	L string `path:"l" style:"label"`
	N int    `path:"n" style:"label"`
}

type bMatrixIn struct {
	M string `path:"m" style:"matrix"`
	P *int   `path:"p" style:"matrix"`
}

type bSimpleIn struct {
	S string `path:"s"`
}

func TestLabelMatrixPathParams(t *testing.T) {
	bGGinTestMode(t)
	f := fizz.New()
	bBHandle(f, "GET", "/label/:l/:n", tonic.Handler(func(c *gin.Context, in *bLabelIn) (*bLabelIn, error) { return in, nil }, 200))
	bBHandle(f, "GET", "/matrix/:m/:p", tonic.Handler(func(c *gin.Context, in *bMatrixIn) (*bMatrixIn, error) { return in, nil }, 200))
	bBHandle(f, "GET", "/simple/:s", tonic.Handler(func(c *gin.Context, in *bSimpleIn) (*bSimpleIn, error) { return in, nil }, 200))
	rows := []struct {
		name, target, want string
		status             int
	}{
		{"label", "/label/.blue/.5", `{"L":"blue","N":5}`, 200},
		{"label empty", "/label/./.1", `{"L":"","N":1}`, 200},
		{"label encoded", "/label/.a%20b/.1", `{"L":"a b","N":1}`, 200},
		{"label keeps inner dots", "/label/.a.b/.1", `{"L":"a.b","N":1}`, 200},
		{"label without the dot", "/label/blue/.1", "", 400},
		{"label bad type", "/label/.x/.y", "", 400},
		{"matrix", "/matrix/;m=blue/;p=7", `{"M":"blue","P":7}`, 200},
		{"matrix empty value", "/matrix/;m/;p=7", `{"M":"","P":7}`, 200},
		{"matrix wrong name", "/matrix/;x=blue/;p=7", "", 400},
		{"matrix without the name", "/matrix/blue/;p=7", "", 400},
		{"matrix bad type", "/matrix/;m=a/;p=x", "", 400},
		{"simple is untouched", "/simple/.x", `{"S":".x"}`, 200},
	}
	for _, r := range rows {
		t.Run(r.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			f.ServeHTTP(w, httptest.NewRequest(http.MethodGet, r.target, nil))
			if r.status == 200 {
				bBOK(t, w, r.want)
			} else {
				bB400(t, w)
			}
		})
	}
}

// TestPathStyleDriftGuard keeps the rule that picks the path parameters to
// strip in agreement with the document: bind plans a parameter exactly
// when the generator emits style label or matrix for a scalar.
func TestPathStyleDriftGuard(t *testing.T) {
	bGGinTestMode(t)
	type (
		label struct {
			A string `path:"a" style:"label"`
		}
		matrix struct {
			A int `path:"a" style:"matrix"`
		}
		ptr struct {
			A *string `path:"a" style:"matrix"`
		}
		simple struct {
			A string `path:"a" style:"simple"`
		}
		unstyled struct {
			A string `path:"a"`
		}
	)
	rows := []struct {
		name  string
		typ   reflect.Type
		value string
		app   func() *fizz.Fizz
	}{
		{"label", reflect.TypeFor[label](), ".x", func() *fizz.Fizz { return driftPathApp(func(*gin.Context, *label) error { return nil }) }},
		{"matrix", reflect.TypeFor[matrix](), ";a=1", func() *fizz.Fizz { return driftPathApp(func(*gin.Context, *matrix) error { return nil }) }},
		{"pointer", reflect.TypeFor[ptr](), ";a=x", func() *fizz.Fizz { return driftPathApp(func(*gin.Context, *ptr) error { return nil }) }},
		{"simple", reflect.TypeFor[simple](), ".x", func() *fizz.Fizz { return driftPathApp(func(*gin.Context, *simple) error { return nil }) }},
		{"unstyled", reflect.TypeFor[unstyled](), ".x", func() *fizz.Fizz { return driftPathApp(func(*gin.Context, *unstyled) error { return nil }) }},
	}
	for _, r := range rows {
		t.Run(r.name, func(t *testing.T) {
			p, err := bind.PlanFor(r.typ, bBDelimTags)
			if err != nil {
				t.Fatal(err)
			}
			_, planned, _ := bind.RewritePath("a", r.value, p)
			f := r.app()
			item := f.Generator().API().Paths["/x/{a}"]
			if item == nil || item.GET == nil || len(item.GET.Parameters) != 1 {
				t.Fatalf("GET /x/{a} has no single parameter: %v", f.Errors())
			}
			style := item.GET.Parameters[0].Value.Style
			doc := style == "label" || style == "matrix"
			if planned != doc {
				t.Errorf("bind rewrites = %v, document style is %q", planned, style)
			}
		})
	}
}

func driftPathApp[T any](h func(*gin.Context, *T) error) *fizz.Fizz {
	f := fizz.New()
	f.Handle("/x/:a", "GET", []fizz.OperationOption{fizz.ID("x")}, tonic.Handler(h, 200))
	return f
}
