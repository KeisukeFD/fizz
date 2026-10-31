package openapi

import (
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"
)

type (
	poSkip struct {
		A string `query:"-"`
		B string `query:""`
		C string `query:",required"`
		D string `query:"kept"`
	}
	poRequired struct {
		Q string `query:"q,required"`
		H string `header:"X-H,required"`
		P string `path:"id"`
	}
	poDefault struct {
		N int `query:"n,default=3"`
	}
	poDefaultReq struct {
		N int `query:"n,required,default=3"`
	}
	poDefaultBoth struct {
		N int `query:"n,default=3" default:"7"`
	}
	poDefaultHeader struct {
		H string `header:"X-H,default=abc"`
	}
	poOmitempty struct {
		Q string `query:"q,omitempty"`
		R string `query:"r"`
	}
	poExplode struct {
		Tags []string `query:"tags" explode:"false"`
		Bad  []string `query:"bad" explode:"maybe"`
		Def  []string `query:"def"`
	}
	poUnsupported struct {
		C chan int `query:"c"`
		R string   `query:"r"`
	}
)

func errorsText(g *Generator) string {
	var sb strings.Builder
	for _, e := range g.Errors() {
		sb.WriteString(e.Error() + "\n")
	}
	return sb.String()
}

// build runs buildParams on typ and fails on a returned error.
func build(t *testing.T, typ reflect.Type) (*Generator, *Operation) {
	t.Helper()
	g := gen(t)
	op := &Operation{}
	if err := g.buildParams(op, typ, false); err != nil {
		t.Fatalf("buildParams returned %v", err)
	}
	return g, op
}

func TestParamTagSkipped(t *testing.T) {
	g, op := build(t, reflect.TypeFor[poSkip]())
	if got := paramKeys(op); !slices.Equal(got, []string{"query:kept"}) {
		t.Errorf("params = %v", got)
	}
	if len(g.Errors()) != 0 {
		t.Errorf("errors = %v", errorsText(g))
	}
}

func TestParamTagRequired(t *testing.T) {
	_, op := build(t, reflect.TypeFor[poRequired]())
	if len(op.Parameters) != 3 {
		t.Fatalf("params = %v", paramKeys(op))
	}
	for _, p := range op.Parameters {
		if !p.Value.Required {
			t.Errorf("%s:%s not required", p.Value.In, p.Value.Name)
		}
	}
}

func TestParamTagDefaultOption(t *testing.T) {
	_, op := build(t, reflect.TypeFor[poDefault]())
	p := op.Parameters[0].Value
	if got := fmt.Sprint(p.Schema.Default); got != "3" {
		t.Errorf("default = %#v", p.Schema.Default)
	}
	if p.Required {
		t.Error("required")
	}
	_, op = build(t, reflect.TypeFor[poDefaultHeader]())
	if d := op.Parameters[0].Value.Schema.Default; d != "abc" {
		t.Errorf("header default = %#v", d)
	}
}

func TestParamTagDefaultOptionWarnsWhenRequired(t *testing.T) {
	g, op := build(t, reflect.TypeFor[poDefaultReq]())
	if op.Parameters[0].Value.Required {
		t.Error("a default makes the parameter optional")
	}
	var w *Warning
	if !slices.ContainsFunc(g.Errors(), func(e error) bool { return errors.As(e, &w) }) {
		t.Errorf("no warning in %v", errorsText(g))
	}
}

// tonic applies the default option before the default tag, so the
// option wins.
func TestParamTagDefaultOptionWinsOverTag(t *testing.T) {
	_, op := build(t, reflect.TypeFor[poDefaultBoth]())
	if got := fmt.Sprint(op.Parameters[0].Value.Schema.Default); got != "3" {
		t.Errorf("default = %v", got)
	}
}

func TestParamTagUnknownOption(t *testing.T) {
	g, op := build(t, reflect.TypeFor[poOmitempty]())
	if got := paramKeys(op); !slices.Equal(got, []string{"query:r"}) {
		t.Errorf("params = %v", got)
	}
	var fe *FieldError
	if !slices.ContainsFunc(g.Errors(), func(e error) bool { return errors.As(e, &fe) }) {
		t.Fatalf("no FieldError in %v", errorsText(g))
	}
	if fe.Field != "Q" {
		t.Errorf("Field = %q", fe.Field)
	}
	if msg := fe.Error(); !strings.Contains(msg, "omitempty") || !strings.Contains(msg, "q") {
		t.Errorf("Error() = %q", msg)
	}
	if fe.Err == nil || errors.Unwrap(fe) != fe.Err {
		t.Errorf("Unwrap = %v", errors.Unwrap(fe))
	}
}

func TestFieldErrorUnwrap(t *testing.T) {
	cause := errors.New("cause")
	var err error = &FieldError{Message: "m", Name: "n", Err: cause}
	if !errors.Is(err, cause) {
		t.Error("errors.Is does not find the cause")
	}
	if errors.Unwrap(&FieldError{}) != nil {
		t.Error("Unwrap of an error without cause is not nil")
	}
}

func TestParamTagExplode(t *testing.T) {
	g, op := build(t, reflect.TypeFor[poExplode]())
	by := map[string]*Parameter{}
	for _, p := range op.Parameters {
		by[p.Value.Name] = p.Value
	}
	if len(by) != 3 {
		t.Fatalf("params = %v", paramKeys(op))
	}
	if e := by["tags"].Explode; e == nil || *e {
		t.Errorf("tags explode = %v", e)
	}
	if e := by["bad"].Explode; e == nil || !*e {
		t.Errorf("bad explode = %v (field is kept with the default)", e)
	}
	if e := by["def"].Explode; e == nil || !*e {
		t.Errorf("def explode = %v", e)
	}
	var fe *FieldError
	if !slices.ContainsFunc(g.Errors(), func(e error) bool { return errors.As(e, &fe) }) || fe.Field != "Bad" {
		t.Fatalf("errors = %v", errorsText(g))
	}
	var ne *strconv.NumError
	if !errors.As(fe, &ne) {
		t.Errorf("cause of %v is not a *strconv.NumError", fe)
	}
}

func TestParamTagUnsupportedSchema(t *testing.T) {
	g, op := build(t, reflect.TypeFor[poUnsupported]())
	if got := paramKeys(op); !slices.Equal(got, []string{"query:r"}) {
		t.Errorf("params = %v", got)
	}
	if len(g.Errors()) == 0 {
		t.Error("no error recorded")
	}
}
