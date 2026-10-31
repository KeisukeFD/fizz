package openapi

import (
	"errors"
	"net/http"
	"reflect"
	"testing"
)

type (
	rbTwo struct {
		A string `json:"a" validate:"required"`
		B string `json:"b"`
	}
	rbOpt struct {
		A string `json:"a"`
		B string `json:"b"`
	}
	rbOther struct {
		C string `json:"c"`
		D string `json:"d"`
	}
	rbPet struct {
		Name string `json:"name"`
	}
	rbEmbedded struct {
		rbOpt
	}
	rbPtrEmbedded struct {
		*rbOpt
	}
	rbWithParam struct {
		ID string `path:"id"`
		rbPet
	}
	rbParamPet struct {
		Q    string `query:"q"`
		Name string `json:"name"`
	}
)

const rbJSON = "application/json"

func rbAdd(t *testing.T, g *Generator, method, path, id string, in reflect.Type) *Operation {
	t.Helper()
	op, err := g.AddOperation(path, method, "", in, nil, &OperationInfo{ID: id, StatusCode: 200})
	if err != nil {
		t.Fatal(err)
	}
	return op
}

func rbRef(op *Operation) string {
	if op.RequestBody == nil {
		return ""
	}
	return op.RequestBody.Content[rbJSON].Schema.Ref
}

func TestRequestBodyDefaultName(t *testing.T) {
	g := gen(t)
	op := rbAdd(t, g, http.MethodPost, "/a", "createThing", reflect.TypeFor[rbTwo]())
	if got, want := rbRef(op), "#/components/schemas/CreateThingInput"; got != want {
		t.Errorf("ref %q, want %q", got, want)
	}
	if g.API().Components.Schemas["CreateThingInput"] == nil {
		t.Error("component CreateThingInput missing")
	}
}

func TestRequestBodyNamer(t *testing.T) {
	g := gen(t)
	g.SetRequestBodyNamer(func(opID string, _ reflect.Type) string { return opID + " Body!" })
	op := rbAdd(t, g, http.MethodPost, "/a", "make", reflect.TypeFor[rbTwo]())
	if got, want := rbRef(op), "#/components/schemas/make_Body_"; got != want {
		t.Errorf("ref %q, want %q", got, want)
	}
	// An empty name selects the default.
	g = gen(t)
	g.SetRequestBodyNamer(func(string, reflect.Type) string { return "" })
	op = rbAdd(t, g, http.MethodPost, "/a", "make", reflect.TypeFor[rbTwo]())
	if got, want := rbRef(op), "#/components/schemas/MakeInput"; got != want {
		t.Errorf("ref %q, want %q", got, want)
	}
}

func TestRequestBodyCollision(t *testing.T) {
	g := gen(t)
	a := rbAdd(t, g, http.MethodPost, "/a", "createUser", reflect.TypeFor[rbOpt]())
	b := rbAdd(t, g, http.MethodPost, "/b", "CreateUser", reflect.TypeFor[rbOther]())
	if got, want := rbRef(a), "#/components/schemas/CreateUserInput"; got != want {
		t.Errorf("first ref %q, want %q", got, want)
	}
	if rbRef(b) != "" {
		t.Errorf("second body is a ref %q, want inline", rbRef(b))
	}
	if b.RequestBody.Content[rbJSON].Schema.Properties["c"] == nil {
		t.Error("inlined body lacks property c")
	}
	if g.API().Components.Schemas["CreateUserInput"].Properties["a"] == nil {
		t.Error("component was overwritten")
	}
	var ce *CollisionError
	found := false
	for _, err := range g.Errors() {
		if errors.As(err, &ce) {
			found = true
		}
	}
	if !found {
		t.Fatalf("no CollisionError in %v", g.Errors())
	}
	if ce.Name != "CreateUserInput" || ce.Kept != reflect.TypeFor[rbOpt]() || ce.Rejected != reflect.TypeFor[rbOther]() {
		t.Errorf("unexpected collision %+v", ce)
	}
}

func TestRequestBodyRequired(t *testing.T) {
	g := gen(t)
	if op := rbAdd(t, g, http.MethodPost, "/a", "a", reflect.TypeFor[rbTwo]()); !op.RequestBody.Required {
		t.Error("required field: body not required")
	}
	if op := rbAdd(t, g, http.MethodPost, "/b", "b", reflect.TypeFor[rbOpt]()); op.RequestBody.Required {
		t.Error("no required field: body required")
	}
}

func TestRequestBodyDirectRef(t *testing.T) {
	g := gen(t)
	// A whole input struct with one field is synthesized.
	if op := rbAdd(t, g, http.MethodPost, "/a", "a", reflect.TypeFor[rbPet]()); rbRef(op) != "#/components/schemas/AInput" {
		t.Errorf("one-field input: ref %q", rbRef(op))
	}
	if op := rbAdd(t, g, http.MethodPost, "/e", "e", reflect.TypeFor[rbPtrEmbedded]()); rbRef(op) != "#/components/schemas/RbOpt" {
		t.Errorf("pointer-embedded struct: ref %q", rbRef(op))
	}
	if op := rbAdd(t, g, http.MethodPost, "/b", "b", reflect.TypeFor[rbEmbedded]()); rbRef(op) != "#/components/schemas/RbOpt" {
		t.Errorf("embedded struct: ref %q", rbRef(op))
	}
	// With a parameter the body is synthesized.
	if op := rbAdd(t, g, http.MethodPost, "/c/{id}", "c", reflect.TypeFor[rbWithParam]()); rbRef(op) != "#/components/schemas/CInput" {
		t.Errorf("with param: ref %q", rbRef(op))
	}
	if op := rbAdd(t, g, http.MethodPost, "/d", "d", reflect.TypeFor[rbParamPet]()); rbRef(op) != "#/components/schemas/DInput" {
		t.Errorf("with query: ref %q", rbRef(op))
	}
}

func TestRequestBodyMethods(t *testing.T) {
	tests := []struct {
		method string
		body   bool
	}{
		{http.MethodPost, true},
		{http.MethodPut, true},
		{http.MethodPatch, true},
		{http.MethodDelete, true},
		{http.MethodGet, false},
		{http.MethodHead, false},
	}
	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			g := gen(t)
			op := rbAdd(t, g, tt.method, "/x", "op", reflect.TypeFor[rbTwo]())
			if got := op.RequestBody != nil; got != tt.body {
				t.Errorf("body present = %v, want %v", got, tt.body)
			}
		})
	}
}
