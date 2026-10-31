package vtag

import (
	"reflect"
	"testing"
)

func r(name, param string) Rule { return Rule{Name: name, Param: param} }
func g(rs ...Rule) Group        { return Group(rs) }

func TestParse(t *testing.T) {
	tests := []struct {
		name string
		tag  string
		want *Tag
	}{
		{"empty", "", &Tag{}},
		{"skip", "-", &Tag{Skip: true}},
		{"required", "required", &Tag{Root: Level{Rules: []Group{g(r("required", ""))}}}},
		{"omitempty then required", "omitempty,required", &Tag{Root: Level{OmitEmpty: true, Rules: []Group{g(r("required", ""))}}}},
		{"omitempty min max", "omitempty,min=1,max=10", &Tag{Root: Level{OmitEmpty: true, Rules: []Group{g(r("min", "1")), g(r("max", "10"))}}}},
		{"oneof or eq", "oneof=a b|eq=c", &Tag{Root: Level{Rules: []Group{g(r("oneof", "a b"), r("eq", "c"))}}}},
		{"gt or lt", "gt=0|lt=-5", &Tag{Root: Level{Rules: []Group{g(r("gt", "0"), r("lt", "-5"))}}}},
		{"len", "len=3", &Tag{Root: Level{Rules: []Group{g(r("len", "3"))}}}},
		{"required_if", "required_if=Field val", &Tag{Root: Level{Rules: []Group{g(r("required_if", "Field val"))}}}},
		{"escaped comma", "startswith=a0x2Cb", &Tag{Root: Level{Rules: []Group{g(r("startswith", "a,b"))}}}},
		{"escaped pipe", "contains=0x7C", &Tag{Root: Level{Rules: []Group{g(r("contains", "|"))}}}},
		{"param keeps later equals", "eqfield=a=b", &Tag{Root: Level{Rules: []Group{g(r("eqfield", "a=b"))}}}},
		{"empty param", "min=", &Tag{Root: Level{Rules: []Group{g(r("min", ""))}}}},
		{"no trim", " required", &Tag{Root: Level{Rules: []Group{g(r(" required", ""))}}}},
		{"dive required", "dive,required", &Tag{Root: Level{Dive: &Level{Rules: []Group{g(r("required", ""))}}}}},
		{"rules before and after dive", "min=1,dive,max=3", &Tag{Root: Level{
			Rules: []Group{g(r("min", "1"))},
			Dive:  &Level{Rules: []Group{g(r("max", "3"))}},
		}}},
		{"nested dive", "min=1,dive,dive,required", &Tag{Root: Level{
			Rules: []Group{g(r("min", "1"))},
			Dive:  &Level{Dive: &Level{Rules: []Group{g(r("required", ""))}}},
		}}},
		{"dive omitempty", "dive,omitempty,min=1", &Tag{Root: Level{
			Dive: &Level{OmitEmpty: true, Rules: []Group{g(r("min", "1"))}},
		}}},
		{"keys", "dive,keys,min=1,endkeys,max=3", &Tag{Root: Level{Dive: &Level{
			Keys:  &Level{Rules: []Group{g(r("min", "1"))}},
			Rules: []Group{g(r("max", "3"))},
		}}}},
		{"keys only", "dive,keys,omitempty,min=1,endkeys", &Tag{Root: Level{Dive: &Level{
			Keys: &Level{OmitEmpty: true, Rules: []Group{g(r("min", "1"))}},
		}}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.tag)
			if err != nil {
				t.Fatalf("Parse(%q): %v", tt.tag, err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parse(%q)\n got %+v\nwant %+v", tt.tag, got, tt.want)
			}
		})
	}
}

func TestParseErrors(t *testing.T) {
	tests := []struct{ name, tag string }{
		{"empty middle item", "a,,b"},
		{"trailing comma", "a,"},
		{"leading comma", ",a"},
		{"empty or alternative", "a||b"},
		{"trailing pipe", "a|"},
		{"keys without dive", "keys,min=1"},
		{"keys after rule", "dive,min=1,keys,endkeys"},
		{"missing endkeys", "dive,keys,min=1"},
		{"endkeys without keys", "dive,endkeys"},
		{"endkeys at root", "endkeys"},
		{"dive in keys", "dive,keys,dive,endkeys"},
		{"omitempty in or", "min=1|omitempty"},
		{"dive in or", "dive|min=1"},
		{"keys in or", "dive,keys|min=1"},
		{"endkeys in or", "dive,keys,min=1|endkeys"},
		{"empty name", "=5"},
		{"empty name in or", "min=1|=5"},
		{"keys twice", "dive,keys,endkeys,keys,endkeys"},
		{"keys inside keys", "dive,keys,keys,endkeys"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.tag)
			if err == nil {
				t.Fatalf("Parse(%q) = %+v, want error", tt.tag, got)
			}
			if got != nil {
				t.Errorf("Parse(%q) returned non-nil tag with error", tt.tag)
			}
		})
	}
}

func TestLevelHas(t *testing.T) {
	tag, err := Parse("required,min=1|max=3,dive,eq=1")
	if err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]bool{
		"required": true, "min": true, "max": true,
		"eq": false, "dive": false, "omitempty": false, "": false,
	} {
		if got := tag.Root.Has(name); got != want {
			t.Errorf("Root.Has(%q) = %v, want %v", name, got, want)
		}
	}
	if !tag.Root.Dive.Has("eq") || tag.Root.Dive.Has("required") {
		t.Error("Dive level Has mismatch")
	}
	var nilLevel *Level
	if nilLevel.Has("x") {
		t.Error("nil Level.Has should be false")
	}
}
