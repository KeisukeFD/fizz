package openapi

import (
	"reflect"
	"slices"
	"testing"
)

type (
	tfLeafA   struct{ Name string }
	tfLeafB   struct{ Name int }
	tfTaggedA struct {
		X string `json:"V"`
	}
	tfUntagB struct{ V int }

	tfDeepName struct{ Name string }
	tfMid      struct{ tfDeepName }

	tfDominance struct {
		tfTaggedA
		tfUntagB
	}
	tfAmbiguous struct {
		tfLeafA
		tfLeafB
		Keep bool
	}
	tfShallow struct {
		tfMid
		Name float64
	}
	tfDash struct {
		A string `json:"-"`
		B string `json:"-,"`
		C string `json:"c"`
	}
	tfInvalidName struct {
		A string `json:"a\"b"`
		B string `json:"b'c"`
		C string `json:"ok-name"`
		D string `json:"é,omitempty"`
	}
	tfStringKinds struct {
		B  bool     `json:",string"`
		I  int8     `json:",string"`
		U  uint     `json:",string"`
		F  float32  `json:",string"`
		S  string   `json:",string"`
		P  *int     `json:",string"`
		St struct{} `json:",string"`
		Sl []int    `json:",string"`
	}
	tfTagEmbedded struct {
		tfLeafA `json:"emb"`
	}
	TFNamedInt  int
	tfNonStruct struct {
		TFNamedInt
		Z int
	}
	tfUnexp    struct{ Pub string }
	tfPromoted struct {
		tfUnexp
	}
	tfPtrUnexp struct {
		*tfUnexp
		Z int
	}
	tfCycA    struct{ *tfCycB }
	tfCycB    struct{ *tfCycA }
	tfCycRoot struct {
		tfCycA
		R int
	}
	tfOpts struct {
		A int `json:"a,omitempty,omitzero"`
	}
)

func TestTypeFields(t *testing.T) {
	type want struct {
		name   string
		tagged bool
		index  []int
		quoted bool
	}
	tests := []struct {
		name string
		typ  reflect.Type
		want []want
	}{
		{"tagged beats untagged", reflect.TypeFor[tfDominance](), []want{{"V", true, []int{0, 0}, false}}},
		{"ambiguous dropped", reflect.TypeFor[tfAmbiguous](), []want{{"Keep", false, []int{2}, false}}},
		{"shallower wins", reflect.TypeFor[tfShallow](), []want{{"Name", false, []int{1}, false}}},
		{"dash skipped, dash comma named", reflect.TypeFor[tfDash](), []want{{"-", true, []int{1}, false}, {"c", true, []int{2}, false}}},
		{"invalid tag name falls back", reflect.TypeFor[tfInvalidName](), []want{
			{"A", false, []int{0}, false}, {"B", false, []int{1}, false},
			{"ok-name", true, []int{2}, false}, {"é", true, []int{3}, false},
		}},
		{"string option kinds", reflect.TypeFor[tfStringKinds](), []want{
			{"B", false, []int{0}, true}, {"I", false, []int{1}, true},
			{"U", false, []int{2}, true}, {"F", false, []int{3}, true},
			{"S", false, []int{4}, true}, {"P", false, []int{5}, true},
			{"St", false, []int{6}, false}, {"Sl", false, []int{7}, false},
		}},
		{"tagged embedded struct is a named field", reflect.TypeFor[tfTagEmbedded](), []want{{"emb", true, []int{0}, false}}},
		{"embedded non-struct named by type", reflect.TypeFor[tfNonStruct](), []want{
			{"TFNamedInt", false, []int{0}, false}, {"Z", false, []int{1}, false},
		}},
		{"unexported embedded struct promoted", reflect.TypeFor[tfPromoted](), []want{{"Pub", false, []int{0, 0}, false}}},
		{"embedded pointer to unexported struct promoted", reflect.TypeFor[tfPtrUnexp](), []want{{"Pub", false, []int{0, 0}, false}, {"Z", false, []int{1}, false}}},
		{"embedded cycle terminates", reflect.TypeFor[tfCycRoot](), []want{{"R", false, []int{1}, false}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &Generator{}
			fs := g.typeFields(tt.typ, "json")
			var got []want
			for _, f := range fs {
				got = append(got, want{f.name, f.tagged, f.index, f.quoted})
			}
			if !slices.EqualFunc(got, tt.want, func(a, b want) bool {
				return a.name == b.name && a.tagged == b.tagged && a.quoted == b.quoted && slices.Equal(a.index, b.index)
			}) {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
			// Cached result is the same slice.
			if again := g.typeFields(tt.typ, "json"); len(again) > 0 && &again[0] != &fs[0] {
				t.Error("typeFields result is not cached")
			}
		})
	}
}

func TestTypeFieldsOptions(t *testing.T) {
	g := &Generator{}
	fs := g.typeFields(reflect.TypeFor[tfOpts](), "json")
	if len(fs) != 1 || !fs[0].omitEmpty || !fs[0].omitZero || fs[0].name != "a" {
		t.Errorf("unexpected fields: %+v", fs)
	}
	var n int
	for range g.fieldsOf(reflect.TypeFor[tfOpts](), "json") {
		n++
	}
	if n != 1 {
		t.Errorf("fieldsOf yielded %d fields, want 1", n)
	}
	// Another tag selects another field set.
	if fs := g.typeFields(reflect.TypeFor[tfOpts](), "xml"); len(fs) != 1 || fs[0].name != "A" {
		t.Errorf("xml fields: %+v", fs)
	}
}
