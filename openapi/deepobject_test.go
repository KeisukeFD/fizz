package openapi

import (
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"
)

type (
	doExplodeFalse struct {
		F map[string]string `query:"f" style:"deepObject" explode:"false"`
	}
	doMember struct {
		A string `json:"a" validate:"required"`
		B int    `json:"b"`
		C string `json:"c" validate:"required"`
		D string `json:"-" validate:"required"`
	}
	doRequiredMembers struct {
		F doMember `query:"f" style:"deepObject"`
	}
)

func TestDeepObjectExplodeFalseWarns(t *testing.T) {
	g, op := build(t, reflect.TypeFor[doExplodeFalse]())
	p := op.Parameters[0].Value
	if p.Explode == nil || !*p.Explode || p.Style != "deepObject" {
		t.Errorf("style=%q explode=%v, want deepObject and true", p.Style, p.Explode)
	}
	var w *Warning
	found := slices.ContainsFunc(g.Errors(), func(e error) bool {
		return errors.As(e, &w) && strings.Contains(w.Msg, `"f"`) && strings.Contains(w.Msg, "explode")
	})
	if !found {
		t.Errorf("no warning naming the field; errors = %v", errorsText(g))
	}
}

func TestDeepObjectRequiredMembers(t *testing.T) {
	_, op := build(t, reflect.TypeFor[doRequiredMembers]())
	s := op.Parameters[0].Value.Schema
	if want := []string{"a", "c"}; !slices.Equal(s.Required, want) {
		t.Errorf("required = %v, want %v", s.Required, want)
	}
	if _, ok := s.Properties["d"]; ok || len(s.Properties) != 3 {
		t.Errorf("properties = %v", s.Properties)
	}
}
