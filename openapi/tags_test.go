package openapi

import (
	"reflect"
	"strings"
	"testing"
)

func TestTagNamesResolved(t *testing.T) {
	if got := (TagNames{}).resolved(); got != defaultTagNames {
		t.Errorf("zero resolves to %+v, want defaults %+v", got, defaultTagNames)
	}
	for i := range reflect.TypeFor[TagNames]().NumField() {
		if reflect.ValueOf(defaultTagNames).Field(i).String() == "" {
			t.Errorf("default for field %d is empty", i)
		}
	}
	got := TagNames{Description: "doc", Format: "-"}.resolved()
	if got.Description != "doc" || got.Format != "-" {
		t.Errorf("custom names lost: %+v", got)
	}
	if got.Example != "example" || got.Deprecated != "deprecated" || got.Cookie != "cookie" {
		t.Errorf("zero fields not defaulted: %+v", got)
	}
}

func TestTagLookup(t *testing.T) {
	sf := reflect.StructField{Tag: `doc:"x" description:"y" format:"email"`}
	for _, c := range []struct {
		name, want string
		ok         bool
	}{
		{"doc", "x", true},
		{"description", "y", true},
		{"missing", "", false},
		{"-", "", false},
	} {
		got, ok := tagLookup(sf, c.name)
		if got != c.want || ok != c.ok {
			t.Errorf("tagLookup(%q) = %q, %v; want %q, %v", c.name, got, ok, c.want, c.ok)
		}
	}
}

func TestNewGeneratorBinderTags(t *testing.T) {
	b := BinderTags
	full := func() SpecGenConfig {
		return SpecGenConfig{
			ValidatorTag: b.Validate, PathLocationTag: b.Path, QueryLocationTag: b.Query,
			HeaderLocationTag: b.Header, EnumTag: b.Enum, DefaultTag: b.Default,
		}
	}
	tests := []struct {
		field string
		set   func(*SpecGenConfig, string)
		want  string
	}{
		{"ValidatorTag", func(c *SpecGenConfig, v string) { c.ValidatorTag = v }, b.Validate},
		{"PathLocationTag", func(c *SpecGenConfig, v string) { c.PathLocationTag = v }, b.Path},
		{"QueryLocationTag", func(c *SpecGenConfig, v string) { c.QueryLocationTag = v }, b.Query},
		{"HeaderLocationTag", func(c *SpecGenConfig, v string) { c.HeaderLocationTag = v }, b.Header},
		{"EnumTag", func(c *SpecGenConfig, v string) { c.EnumTag = v }, b.Enum},
		{"DefaultTag", func(c *SpecGenConfig, v string) { c.DefaultTag = v }, b.Default},
	}
	if _, err := NewGenerator(&SpecGenConfig{}); err != nil {
		t.Errorf("empty config: %v", err)
	}
	for _, tc := range tests {
		t.Run(tc.field, func(t *testing.T) {
			conf := full()
			tc.set(&conf, "")
			if _, err := NewGenerator(&conf); err != nil {
				t.Errorf("empty: %v", err)
			}
			conf = full()
			tc.set(&conf, tc.want)
			if _, err := NewGenerator(&conf); err != nil {
				t.Errorf("tonic value: %v", err)
			}
			conf = full()
			tc.set(&conf, "other")
			before := conf
			_, err := NewGenerator(&conf)
			if err == nil {
				t.Fatal("different value: want an error")
			}
			for _, sub := range []string{"SpecGenConfig." + tc.field, `"other"`, `"` + tc.want + `"`} {
				if !strings.Contains(err.Error(), sub) {
					t.Errorf("error %q lacks %q", err, sub)
				}
			}
			if conf != before {
				t.Errorf("caller config mutated: %+v, was %+v", conf, before)
			}
		})
	}
	t.Run("first in declaration order", func(t *testing.T) {
		_, err := NewGenerator(&SpecGenConfig{ValidatorTag: "v", DefaultTag: "d"})
		if err == nil || !strings.Contains(err.Error(), "ValidatorTag") {
			t.Errorf("want ValidatorTag error first, got %v", err)
		}
	})
}
