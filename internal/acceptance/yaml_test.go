package acceptance

import (
	"encoding/json"
	"mime"
	"net/http"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/loopfz/gadgeto/tonic"

	"github.com/KeisukeFD/fizz"
	"github.com/KeisukeFD/fizz/internal/acceptance/at"
	"github.com/KeisukeFD/fizz/internal/kitchensink"
	"github.com/KeisukeFD/fizz/internal/plan"
	"github.com/KeisukeFD/fizz/openapi"
)

// Batch G (YAML): the served YAML document must be the served JSON
// document, re-encoded.

type (
	bGAddress struct {
		Street string   `json:"street" description:"Street line"`
		Tags   []string `json:"tags,omitempty"`
	}
	bGPet struct {
		ID       int               `json:"id" description:"Pet id" validate:"required"`
		Name     string            `json:"name" validate:"required,min=1,max=20"`
		Kind     string            `json:"kind" enum:"cat,dog" default:"cat"`
		Score    float64           `json:"score"`
		Owner    *bGAddress        `json:"owner,omitempty"`
		Labels   map[string]string `json:"labels,omitempty"`
		Secret   string            `json:"-" yaml:"secret"`
		Weights  []float64         `json:"weights"`
		Archived bool              `json:"archived"`
	}
	bGPetIn struct {
		ID    string `path:"id" description:"Pet id"`
		Limit int    `query:"limit" default:"10" description:"Page size"`
		Verb  string `query:"verb" enum:"a,b"`
	}
	bGPetCreate struct {
		Pet bGPet `json:"pet"`
	}
	bGAmbiguous struct {
		Values []string `json:"values"`
	}
	bGYAMLOnly struct {
		A string `json:"a"`
	}
)

// MarshalYAML must be ignored: the YAML document derives from the JSON one.
func (bGYAMLOnly) MarshalYAML() (any, error) { return "yaml-only", nil }

func bGGetPet(*gin.Context, *bGPetIn) (*bGPet, error)      { return &bGPet{}, nil }
func bGPostPet(*gin.Context, *bGPetCreate) (*bGPet, error) { return &bGPet{}, nil }
func bGListPets(*gin.Context, *bGPetIn) ([]bGPet, error)   { return nil, nil }

// bGYAMLFizz serves the spec at /openapi.json and /openapi.yaml after
// register has run.
func bGYAMLFizz(t testing.TB, register func(f *fizz.Fizz)) *fizz.Fizz {
	t.Helper()
	bGGinTestMode(t)
	f := fizz.New()
	register(f)
	info := &openapi.Info{Title: "yaml", Version: "1.0.0", Description: "Parity: 1e3, ~, null"}
	f.GET("/openapi.json", nil, f.OpenAPI(info, "json"))
	f.GET("/openapi.yaml", nil, f.OpenAPI(info, "yaml"))
	return f
}

func bGParityRegister(f *fizz.Fizz) {
	g := f.Group("/pets", "Pets", "Pet operations")
	g.GET("", bGOpts(fizz.ID("listPets"), fizz.Summary("List"),
		fizz.Header("X-Rate-Limit", "Requests left", 0)), tonic.Handler(bGListPets, 200))
	g.GET("/:id", bGOpts(fizz.ID("getPet"), fizz.Description("Fetch one pet."),
		fizz.Response("404", "Not found", nil, nil, nil),
		fizz.ResponseWithExamples("400", "Bad", new(bGPet), nil, map[string]any{
			"empty": bGPet{Name: "n", Score: 1.5, Weights: []float64{1, 2.5}},
		})), tonic.Handler(bGGetPet, 200))
	g.POST("", bGOpts(fizz.ID("createPet")), tonic.Handler(bGPostPet, 201))
}

// bGNavigate walks keys through nested objects of a decoded document.
func bGNavigate(tb testing.TB, doc any, keys ...string) any {
	tb.Helper()
	cur := doc
	for i, k := range keys {
		m, ok := cur.(map[string]any)
		if !ok {
			tb.Fatalf("at %s: %T is not an object", strings.Join(keys[:i], "/"), cur)
		}
		if cur, ok = m[k]; !ok {
			tb.Fatalf("missing key %q under %s", k, strings.Join(keys[:i], "/"))
		}
	}
	return cur
}

func TestYAMLParityServed(t *testing.T) {
	at.Pending(t, "P2.5", func(tb testing.TB) {
		f := bGYAMLFizz(tb, bGParityRegister)
		j := at.Do(f, http.MethodGet, "/openapi.json")
		y := at.Do(f, http.MethodGet, "/openapi.yaml")
		if j.Code != http.StatusOK || y.Code != http.StatusOK {
			tb.Fatalf("status json=%d yaml=%d", j.Code, y.Code)
		}
		fromJSON := at.FromJSON[any](j.Body.String())
		fromYAML := at.YAMLToAny(tb, y.Body.Bytes())
		if !reflect.DeepEqual(fromJSON, fromYAML) {
			tb.Errorf("YAML and JSON documents differ:\njson %v\nyaml %v", fromJSON, fromYAML)
		}
		if plan.IsDone("P2.6") {
			doc, err := json.Marshal(fromYAML)
			if err != nil {
				tb.Fatal(err)
			}
			for _, p := range kitchensink.MetaProblems(doc) {
				tb.Error("yaml document: " + p)
			}
		}
	})
}

// TestYAMLAmbiguousStrings passes today (gin quotes such strings); it
// guards the P2.5 re-encoding against regressing it.
func TestYAMLAmbiguousStrings(t *testing.T) {
	want := []any{"123", "true", "null", "~", "1e3", "", "0x1F", "yes", "on", "2026-10-30"}
	values := make([]string, len(want))
	for i, w := range want {
		values[i] = w.(string)
	}
	f := bGYAMLFizz(t, func(f *fizz.Fizz) {
		f.GET("/x", bGOpts(fizz.ID("x"),
			fizz.Response("400", "bad", new(bGAmbiguous), nil, bGAmbiguous{Values: values})),
			tonic.Handler(bGHandler[bGEmpty], 200))
	})
	y := at.Do(f, http.MethodGet, "/openapi.yaml")
	doc := at.YAMLToAny(t, y.Body.Bytes())
	got := bGNavigate(t, doc, "paths", "/x", "get", "responses", "400", "content", "application/json", "example", "values")
	if !reflect.DeepEqual(got, any(want)) {
		t.Errorf("values decoded from YAML = %#v, want %#v", got, want)
	}
}

func TestYAMLIgnoresUserMarshalYAML(t *testing.T) {
	at.Pending(t, "P2.5", func(tb testing.TB) {
		f := bGYAMLFizz(tb, func(f *fizz.Fizz) {
			f.GET("/x", bGOpts(fizz.ID("x"),
				fizz.Response("400", "bad", new(bGYAMLOnly), nil, bGYAMLOnly{A: "json-value"})),
				tonic.Handler(bGHandler[bGEmpty], 200))
		})
		body := at.Do(f, http.MethodGet, "/openapi.yaml").Body.String()
		if strings.Contains(body, "yaml-only") {
			tb.Error("served YAML contains the output of the example's MarshalYAML")
		}
		if !strings.Contains(body, "json-value") {
			tb.Error("served YAML lacks the JSON value of the example")
		}
	})
}

// bGFlowStart matches a flow collection used as a value or sequence
// item (path keys such as "/pets/{id}:" do not match).
var bGFlowStart = regexp.MustCompile(`(?:^|[:-]\s)\s*([\[{].*)$`)

// TestYAMLBlockStyle passes today; it guards the P2.5 re-encoding.
func TestYAMLBlockStyle(t *testing.T) {
	f := bGYAMLFizz(t, bGParityRegister)
	body := at.Do(f, http.MethodGet, "/openapi.yaml").Body.String()
	prev, sawTwo := 0, false
	for i, line := range strings.Split(body, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if m := bGFlowStart.FindStringSubmatch(line); m != nil && m[1] != "{}" && m[1] != "[]" {
			t.Errorf("line %d: flow collection: %q", i+1, line)
		}
		indent := len(line) - len(strings.TrimLeft(line, " "))
		switch {
		case strings.HasPrefix(line[indent:], "\t"):
			t.Errorf("line %d: tab indentation", i+1)
		case indent%2 != 0, indent > prev+2:
			t.Errorf("line %d: indent %d after %d, want steps of 2: %q", i+1, indent, prev, line)
		}
		sawTwo = sawTwo || indent == 2
		prev = indent
	}
	if !sawTwo {
		t.Error("no line is indented by exactly 2 spaces")
	}
}

func TestYAMLContentType(t *testing.T) {
	at.Pending(t, "P2.5", func(tb testing.TB) {
		f := bGYAMLFizz(tb, bGParityRegister)
		w := at.Do(f, http.MethodGet, "/openapi.yaml")
		mt, _, err := mime.ParseMediaType(w.Header().Get("Content-Type"))
		if err != nil || mt != "application/yaml" {
			tb.Errorf("Content-Type = %q, want application/yaml", w.Header().Get("Content-Type"))
		}
	})
}
