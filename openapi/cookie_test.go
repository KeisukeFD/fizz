package openapi

import (
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"
)

type (
	ckName   string
	ckScalar struct {
		S  string        `cookie:"s"`
		B  bool          `cookie:"b"`
		I  int           `cookie:"i"`
		U8 uint8         `cookie:"u8"`
		F  float64       `cookie:"f"`
		N  ckName        `cookie:"n"`
		P  *int          `cookie:"p"`
		T  time.Time     `cookie:"t"`
		D  time.Duration `cookie:"d"`
	}
	ckStruct struct {
		A struct{ X string } `cookie:"a"`
		K string             `cookie:"k"`
	}
	ckSlice struct {
		A []string `cookie:"a"`
	}
	ckMap struct {
		A map[string]string `cookie:"a"`
	}
	ckSkip struct {
		A string `cookie:"-"`
		B string `cookie:""`
		C string `cookie:"kept"`
	}
	ckConflict struct {
		A string `cookie:"a" query:"a"`
	}
	ckReqDef struct {
		R string `cookie:"r,required"`
		D string `cookie:"d,required,default=x"`
	}
)

func TestCookieScalars(t *testing.T) {
	g, op := build(t, reflect.TypeFor[ckScalar]())
	if len(g.Errors()) != 0 {
		t.Fatalf("errors = %v", errorsText(g))
	}
	if len(op.Parameters) != 9 {
		t.Fatalf("params = %v", paramKeys(op))
	}
	for _, p := range op.Parameters {
		v := p.Value
		if v.In != "cookie" || v.Style != "" || v.Explode != nil {
			t.Errorf("%s: in=%q style=%q explode=%v", v.Name, v.In, v.Style, v.Explode)
		}
	}
}

func TestCookieNonScalar(t *testing.T) {
	for _, typ := range []reflect.Type{
		reflect.TypeFor[ckStruct](), reflect.TypeFor[ckSlice](), reflect.TypeFor[ckMap](),
	} {
		t.Run(typ.Name(), func(t *testing.T) {
			g, op := build(t, typ)
			var fe *FieldError
			if len(g.Errors()) != 1 || !errors.As(g.Errors()[0], &fe) {
				t.Fatalf("errors = %v", errorsText(g))
			}
			if fe.ParameterLocation != "cookie" || fe.Field != "A" || !strings.Contains(fe.Error(), "cookie") {
				t.Errorf("error = %v", fe)
			}
			for _, p := range op.Parameters {
				if p.Value.Name == "a" {
					t.Error("non-scalar field was not skipped")
				}
			}
		})
	}
	_, op := build(t, reflect.TypeFor[ckStruct]())
	if got := paramKeys(op); !slices.Equal(got, []string{"cookie:k"}) {
		t.Errorf("params = %v", got)
	}
}

func TestCookieSkippedAndDisabled(t *testing.T) {
	_, op := build(t, reflect.TypeFor[ckSkip]())
	if got := paramKeys(op); !slices.Equal(got, []string{"cookie:kept"}) {
		t.Errorf("params = %v", got)
	}
	for _, name := range []string{"-", "ck"} {
		g := gen(t)
		g.SetTagNames(TagNames{Cookie: name})
		op := &Operation{}
		if err := g.buildParams(op, reflect.TypeFor[ckSkip](), false); err != nil {
			t.Fatal(err)
		}
		if len(op.Parameters) != 0 {
			t.Errorf("Cookie=%q: params = %v", name, paramKeys(op))
		}
	}
}

func TestCookieConflict(t *testing.T) {
	g := gen(t)
	err := g.buildParams(&Operation{}, reflect.TypeFor[ckConflict](), true)
	var fe *FieldError
	if !errors.As(err, &fe) || fe.Message != "conflicting parameter location" {
		t.Errorf("err = %v", err)
	}
}

func TestCookieRequiredDefault(t *testing.T) {
	g, op := build(t, reflect.TypeFor[ckReqDef]())
	byName := map[string]*Parameter{}
	for _, p := range op.Parameters {
		byName[p.Value.Name] = p.Value
	}
	if !byName["r"].Required {
		t.Error("r must be required")
	}
	d := byName["d"]
	if d.Required || d.Schema.Default != "x" {
		t.Errorf("d: required=%v default=%v", d.Required, d.Schema.Default)
	}
	var warned bool
	for _, e := range g.Errors() {
		var w *Warning
		warned = warned || errors.As(e, &w)
	}
	if !warned {
		t.Errorf("no warning: %v", errorsText(g))
	}
}
