package kitchensink

import (
	"encoding/json"
	"maps"
	"slices"
	"testing"

	"github.com/KeisukeFD/fizz"
	"github.com/KeisukeFD/fizz/internal/plan"
)

// kDiffSample is a kitchen-sink sample type with a fizz registration
// that returns it. The test compares the keys encoding/json emits for
// the zero value with the keys of the generated schema.
type kDiffSample struct {
	name     string
	value    any
	register func(*fizz.Fizz)
}

func kSample[T any](name string) kDiffSample {
	return kDiffSample{name: name, value: *new(T), register: kGet[T]}
}

// TestDifferentialJSONKeys (K06) checks, for every plain-struct T* and
// F* sample, that the keys of json.Marshal(sample) are exactly the
// properties of its schema.
func TestDifferentialJSONKeys(t *testing.T) {
	samples := []kDiffSample{
		kSample[TInts]("TInts"),
		kSample[TDur]("TDur"),
		kSample[TStd]("TStd"),
		kSample[TRaw]("TRaw"),
		kSample[TElems]("TElems"),
		kSample[TSelf]("TSelf"),
		kSample[FDomOuter]("FDomOuter"),
		kSample[FTagWins]("FTagWins"),
		kSample[FAmbiguous]("FAmbiguous"),
		kSample[FDash]("FDash"),
		kSample[FDashComma]("FDashComma"),
		kSample[FStringOpt]("FStringOpt"),
		kSample[FTaggedEmb]("FTaggedEmb"),
		kSample[FEmbedStr]("FEmbedStr"),
		kSample[FHolder]("FHolder"),
		kSample[FNullable]("FNullable"),
		kSample[FCycA]("FCycA"),
	}
	plan.Pending(t, "P3.1a", func(tb testing.TB) {
		for _, s := range samples {
			f := fizz.New()
			s.register(f)
			doc, err := BuildSpec(f)
			if err != nil {
				tb.Errorf("%s: build spec: %v", s.name, err)
				continue
			}
			var spec struct {
				Components struct {
					Schemas map[string]struct {
						Properties map[string]json.RawMessage `json:"properties"`
					} `json:"schemas"`
				} `json:"components"`
			}
			if err := json.Unmarshal(doc, &spec); err != nil {
				tb.Errorf("%s: %v", s.name, err)
				continue
			}
			schema, ok := spec.Components.Schemas["Kitchensink"+s.name]
			if !ok {
				tb.Errorf("%s: no component", s.name)
				continue
			}
			raw, err := json.Marshal(s.value)
			if err != nil {
				tb.Errorf("%s: marshal: %v", s.name, err)
				continue
			}
			var got map[string]json.RawMessage
			if err := json.Unmarshal(raw, &got); err != nil {
				tb.Errorf("%s: %v", s.name, err)
				continue
			}
			want := slices.Sorted(maps.Keys(got))
			have := slices.Sorted(maps.Keys(schema.Properties))
			if !slices.Equal(want, have) {
				tb.Errorf("%s: json keys %v, schema properties %v", s.name, want, have)
			}
		}
	})
}
