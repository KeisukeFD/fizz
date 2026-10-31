package openapi

import (
	"reflect"
	"slices"
	"testing"
)

type (
	pwDash struct {
		Hidden string `json:"-" query:"q"`
		Keep   string `json:"keep"`
	}

	PwCycA struct {
		*PwCycB
		A string `query:"a"`
	}
	PwCycB struct {
		*PwCycA
		B string `header:"b"`
	}

	pwParams struct {
		ID string `path:"id"`
	}
	pwUntagged struct {
		pwParams
		Body string `json:"body"`
	}
	pwTagged struct {
		pwParams `json:"p"`
	}
	pwBodyAndParams struct {
		Q    int    `query:"q"`
		Name string `json:"name" validate:"required"`
	}
)

func walk(t *testing.T, typ reflect.Type, allowBody bool) *Operation {
	t.Helper()
	g := gen(t)
	op := &Operation{}
	if err := g.buildParams(op, typ, allowBody); err != nil {
		t.Fatal(err)
	}
	return op
}

func paramKeys(op *Operation) []string {
	var ks []string
	for _, p := range op.Parameters {
		ks = append(ks, p.Value.In+":"+p.Value.Name)
	}
	return ks
}

func bodyProps(op *Operation) []string {
	if op.RequestBody == nil {
		return nil
	}
	var ks []string
	for _, mt := range op.RequestBody.Content {
		for k := range mt.Schema.Properties {
			ks = append(ks, k)
		}
	}
	slices.Sort(ks)
	return ks
}

func TestWalkParamsJSONDashLocationField(t *testing.T) {
	op := walk(t, reflect.TypeFor[pwDash](), true)
	if got := paramKeys(op); !slices.Equal(got, []string{"query:q"}) {
		t.Errorf("params = %v", got)
	}
	if got := bodyProps(op); !slices.Equal(got, []string{"keep"}) {
		t.Errorf("body = %v", got)
	}
}

func TestWalkParamsEmbeddedCycle(t *testing.T) {
	op := walk(t, reflect.TypeFor[PwCycA](), true)
	got := paramKeys(op)
	slices.Sort(got)
	if !slices.Equal(got, []string{"header:b", "query:a"}) {
		t.Errorf("params = %v", got)
	}
	if got := bodyProps(op); got != nil {
		t.Errorf("body = %v", got)
	}
}

func TestWalkParamsAnonymousStructs(t *testing.T) {
	for name, typ := range map[string]reflect.Type{
		"untagged": reflect.TypeFor[pwUntagged](),
		"tagged":   reflect.TypeFor[pwTagged](),
	} {
		t.Run(name, func(t *testing.T) {
			op := walk(t, typ, true)
			if got := paramKeys(op); !slices.Equal(got, []string{"path:id"}) {
				t.Errorf("params = %v", got)
			}
		})
	}
	// The tagged embedded struct is still a named body property;
	// the untagged one has its body fields promoted.
	if got := bodyProps(walk(t, reflect.TypeFor[pwUntagged](), true)); !slices.Equal(got, []string{"body"}) {
		t.Errorf("untagged body = %v", got)
	}
	if got := bodyProps(walk(t, reflect.TypeFor[pwTagged](), true)); !slices.Equal(got, []string{"p"}) {
		t.Errorf("tagged body = %v", got)
	}
}

func TestWalkParamsBodyAlongsideParams(t *testing.T) {
	op := walk(t, reflect.TypeFor[pwBodyAndParams](), true)
	if got := paramKeys(op); !slices.Equal(got, []string{"query:q"}) {
		t.Errorf("params = %v", got)
	}
	if got := bodyProps(op); !slices.Equal(got, []string{"name"}) {
		t.Errorf("body = %v", got)
	}
	for _, mt := range op.RequestBody.Content {
		if !slices.Equal(mt.Schema.Required, []string{"name"}) {
			t.Errorf("required = %v", mt.Schema.Required)
		}
	}
	if op := walk(t, reflect.TypeFor[pwBodyAndParams](), false); op.RequestBody != nil {
		t.Error("body present with allowBody=false")
	}
}
