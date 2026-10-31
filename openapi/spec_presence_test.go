package openapi

import (
	"encoding/json"
	"testing"
)

// An empty "properties" object in a decoded document is written back,
// extensions included; an empty map built in Go (as the generator does)
// stays omitted.
func TestSchemaEmptyPropertiesPresence(t *testing.T) {
	var s Schema
	in := `{"type":"object","properties":{},"x-a":1}`
	if err := json.Unmarshal([]byte(in), &s); err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(&s)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(b); got != `{"type":"object","properties":{},"x-a":1}` {
		t.Errorf("decoded: got %s", got)
	}

	built := Schema{Type: Types("object"), Properties: map[string]*Schema{}}
	b, err = json.Marshal(&built)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(b); got != `{"type":"object"}` {
		t.Errorf("built: got %s", got)
	}
}

// An explicit "required": false survives a decode and encode, extensions
// included, and a Go-built false stays omitted.
func TestRequiredFalsePresence(t *testing.T) {
	var p Parameter
	if err := json.Unmarshal([]byte(`{"name":"q","in":"query","required":false,"x-a":1}`), &p); err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(&p)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(b); got != `{"name":"q","in":"query","required":false,"x-a":1}` {
		t.Errorf("parameter: got %s", got)
	}

	var h Header
	if err := json.Unmarshal([]byte(`{"required":false}`), &h); err != nil {
		t.Fatal(err)
	}
	if b, _ = json.Marshal(&h); string(b) != `{"required":false}` {
		t.Errorf("header: got %s", b)
	}

	b, _ = json.Marshal(&Parameter{Name: "q", In: InQuery})
	if got := string(b); got != `{"name":"q","in":"query"}` {
		t.Errorf("built: got %s", got)
	}
}

// Empty "parameters" containers on a path item and a link survive a
// decode and encode; nil stays omitted.
func TestEmptyParametersContainerPresence(t *testing.T) {
	tests := []struct {
		name string
		in   string
		v    any
	}{
		{"path item", `{"parameters":[],"x-a":1}`, new(PathItem)},
		{"link", `{"operationId":"o","parameters":{},"x-a":1}`, new(Link)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := json.Unmarshal([]byte(tt.in), tt.v); err != nil {
				t.Fatal(err)
			}
			b, err := json.Marshal(tt.v)
			if err != nil {
				t.Fatal(err)
			}
			if got := string(b); got != tt.in {
				t.Errorf("got %s, want %s", got, tt.in)
			}
		})
	}

	for name, v := range map[string]any{"path item": &PathItem{}, "link": &Link{}} {
		b, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		if got := string(b); got != `{}` {
			t.Errorf("%s nil: got %s, want {}", name, got)
		}
	}
}
