package acceptance

import (
	"errors"
	"reflect"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/loopfz/gadgeto/tonic"

	"github.com/KeisukeFD/fizz"
	"github.com/KeisukeFD/fizz/bind"
	"github.com/KeisukeFD/fizz/openapi"
)

// dTU is a scalar type that decodes through encoding.TextUnmarshaler.
type dTU struct{ v string }

func (d *dTU) UnmarshalText(b []byte) error { d.v = string(b); return nil }

type (
	dScalars struct {
		Name string   `json:"name"`
		Age  int      `json:"age"`
		Tags []string `json:"tags"`
	}
	dNestedStruct struct {
		In struct{ A string } `json:"in"`
	}
	dMapStringIn struct {
		F map[string]string `query:"f" style:"deepObject"`
	}
	dMapIntsIn struct {
		F map[string][]int `query:"f" style:"deepObject"`
	}
	dMapEmptyIn struct {
		F map[string]struct{} `query:"f" style:"deepObject"`
	}
	dMapMapIn struct {
		F map[string]map[string]string `query:"f" style:"deepObject"`
	}
	dStructScalarsIn struct {
		F dScalars `query:"f" style:"deepObject"`
	}
	dStructNestedIn struct {
		F dNestedStruct `query:"f" style:"deepObject"`
	}
	dPtrMapIn struct {
		F *map[string]string `query:"f" style:"deepObject"`
	}
	dMapTextIn struct {
		F map[string]dTU `query:"f" style:"deepObject"`
	}
	dMapChanIn struct {
		F map[string]chan int `query:"f" style:"deepObject"`
	}
)

// driftRow reports whether the generator emits the deepObject parameter of
// T without an error.
func driftGenerates[T any](t *testing.T) bool {
	t.Helper()
	f := fizz.New()
	f.GET("/x", bGOpts(fizz.ID("x")), tonic.Handler(func(*gin.Context, *T) error { return nil }, 200))
	for _, err := range f.Errors() {
		var w *openapi.Warning
		if !errors.As(err, &w) {
			return false
		}
	}
	return true
}

func driftRow[T any](t *testing.T) (generates, binds bool) {
	t.Helper()
	_, err := bind.PlanFor(reflect.TypeFor[T](), bBTags)
	return driftGenerates[T](t), err == nil
}

// TestDeepObjectDriftGuard keeps the scalar rule that openapi duplicates
// (openapi/deepobject.go) in agreement with bind.PlanFor (bind/plan.go).
func TestDeepObjectDriftGuard(t *testing.T) {
	bGGinTestMode(t)
	rows := []struct {
		name string
		run  func(*testing.T) (bool, bool)
	}{
		{"map[string]string", driftRow[dMapStringIn]},
		{"map[string][]int", driftRow[dMapIntsIn]},
		{"map[string]struct{}", driftRow[dMapEmptyIn]},
		{"map[string]map[string]string", driftRow[dMapMapIn]},
		{"struct of scalars", driftRow[dStructScalarsIn]},
		{"struct with nested struct", driftRow[dStructNestedIn]},
		{"*map[string]string", driftRow[dPtrMapIn]},
		{"map[string]TextUnmarshaler", driftRow[dMapTextIn]},
		{"map[string]chan int", driftRow[dMapChanIn]},
	}
	for _, r := range rows {
		t.Run(r.name, func(t *testing.T) {
			gen, bnd := r.run(t)
			t.Logf("generator accepts=%v, bind accepts=%v", gen, bnd)
			if gen != bnd {
				t.Errorf("disagreement: generator accepts=%v, bind accepts=%v", gen, bnd)
			}
		})
	}
}
