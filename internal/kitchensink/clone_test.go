package kitchensink

import (
	"bytes"
	"encoding/json"
	"math"
	"math/rand/v2"
	"strconv"
	"testing"

	"github.com/wI2L/fizz/openapi"
)

// numberSeeds are numbers that a float64 cannot hold.
var numberSeeds = []string{
	"18446744073709551615", "-9223372036854775808", "0.1", "1e400", "-0", "0",
	"123456789012345678901234567890", "1E+2", "3.141592653589793238462643383279",
}

// randomValue returns a random JSON-like value for an "any" field.
func randomValue(r *rand.Rand, depth int) any {
	n := 9
	if depth <= 0 {
		n = 7
	}
	switch r.IntN(n) {
	case 0:
		return nil
	case 1:
		return r.IntN(2) == 0
	case 2:
		return "s<&> " + strconv.Itoa(r.IntN(100))
	case 3:
		return json.Number(numberSeeds[r.IntN(len(numberSeeds))])
	case 4:
		return []any(nil)
	case 5:
		return []int64{math.MaxInt64, math.MinInt64, int64(r.IntN(10))}
	case 6:
		return []uint64{math.MaxUint64, 1}
	case 7:
		s := make([]any, r.IntN(4))
		for i := range s {
			s[i] = randomValue(r, depth-1)
		}
		return s
	default:
		m := make(map[string]any, 4)
		for range r.IntN(4) {
			m["k"+strconv.Itoa(r.IntN(10))] = randomValue(r, depth-1)
		}
		return m
	}
}

func mustMarshal(tb testing.TB, d *openapi.OpenAPI) []byte {
	tb.Helper()
	b, err := openapi.MarshalJSON(d)
	if err != nil {
		tb.Fatalf("MarshalJSON: %v", err)
	}
	return b
}

func checkClone(tb testing.TB, d *openapi.OpenAPI) *openapi.OpenAPI {
	tb.Helper()
	want := mustMarshal(tb, d)
	c, err := d.Clone()
	if err != nil {
		tb.Fatalf("Clone: %v", err)
	}
	if got := mustMarshal(tb, c); !bytes.Equal(got, want) {
		tb.Fatalf("Marshal(Clone(d)) != Marshal(d)\nwant %s\ngot  %s", want, got)
	}
	return c
}

// TestCloneRoundTrip checks Marshal(Clone(d)) == Marshal(d) over the
// kitchen sink document, with extension values added at several levels.
func TestCloneRoundTrip(t *testing.T) {
	base := NewWholeFizz().Generator().API()
	if base == nil {
		t.Fatal("API() returned nil")
	}
	checkClone(t, base)

	r := rand.New(rand.NewPCG(1, 2))
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	for i := range 300 {
		d, err := base.Clone()
		must(err)
		must(d.SetExtension("x-doc", randomValue(r, 4)))
		must(d.Info.SetExtension("x-info", randomValue(r, 4)))
		for _, item := range d.Paths {
			if item.GET != nil {
				must(item.GET.SetExtension("x-op", randomValue(r, 4)))
			}
		}
		c := checkClone(t, d)
		if i%50 == 0 {
			// The copy is independent of the original.
			must(c.Info.SetExtension("x-only-in-copy", 1))
			if bytes.Contains(mustMarshal(t, d), []byte("x-only-in-copy")) {
				t.Fatal("clone shares state with the original")
			}
		}
	}
}

// FuzzCloneRoundTrip uses any JSON text as the value of an extension.
func FuzzCloneRoundTrip(f *testing.F) {
	for _, n := range numberSeeds {
		f.Add(n)
		f.Add(`{"a":[` + n + `,null,true],"b":{"c":` + n + `}}`)
	}
	f.Add(`"str"`)
	f.Add(`[]`)
	f.Add(`{}`)
	f.Fuzz(func(t *testing.T, s string) {
		dec := json.NewDecoder(bytes.NewReader([]byte(s)))
		dec.UseNumber()
		var v any
		if err := dec.Decode(&v); err != nil {
			t.Skip()
		}
		d := &openapi.OpenAPI{OpenAPI: openapi.Version, Info: &openapi.Info{Title: "t", Version: "1"}}
		if err := d.SetExtension("x-fuzz", v); err != nil {
			t.Fatal(err)
		}
		if _, err := openapi.MarshalJSON(d); err != nil {
			t.Skip() // not encodable
		}
		checkClone(t, d)
	})
}

func TestCloneEncodeFailure(t *testing.T) {
	d := &openapi.OpenAPI{OpenAPI: openapi.Version, Info: &openapi.Info{Title: "t", Version: "1"}}
	if err := d.SetExtension("x-nan", math.NaN()); err != nil {
		t.Fatal(err)
	}
	if c, err := d.Clone(); err == nil || c != nil {
		t.Fatalf("Clone = %v, %v; want nil and an error", c, err)
	}
}
