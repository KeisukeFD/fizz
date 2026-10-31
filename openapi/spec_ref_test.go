package openapi

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"
)

func TestOrRefMarshal(t *testing.T) {
	ref := &Reference{Ref: "#/components/parameters/p", Summary: "s", Description: "d"}
	val := &Parameter{Name: "n", In: "query"}

	t.Run("ref", func(t *testing.T) {
		b, err := json.Marshal(&ParameterOrRef{Ref: ref})
		if err != nil {
			t.Fatal(err)
		}
		const want = `{"$ref":"#/components/parameters/p","summary":"s","description":"d"}`
		if string(b) != want {
			t.Errorf("got %s, want %s", b, want)
		}
	})
	t.Run("value", func(t *testing.T) {
		b, err := json.Marshal(&ParameterOrRef{Value: val})
		if err != nil {
			t.Fatal(err)
		}
		want, _ := json.Marshal(val)
		if string(b) != string(want) {
			t.Errorf("got %s, want %s", b, want)
		}
	})
	t.Run("both nil", func(t *testing.T) {
		_, err := json.Marshal(&ParameterOrRef{})
		if !errors.Is(err, ErrNilOrRef) {
			t.Errorf("got %v, want ErrNilOrRef", err)
		}
	})
	t.Run("round trip", func(t *testing.T) {
		for _, in := range []*ParameterOrRef{{Ref: ref}, {Value: val}} {
			b, err := json.Marshal(in)
			if err != nil {
				t.Fatal(err)
			}
			var out ParameterOrRef
			if err := json.Unmarshal(b, &out); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(&out, in) {
				t.Errorf("got %+v, want %+v", out, *in)
			}
		}
	})
}

func TestOrRefValidate(t *testing.T) {
	both := &ParameterOrRef{Ref: &Reference{}, Value: &Parameter{}}
	tests := []struct {
		name    string
		in      *ParameterOrRef
		wantErr bool
		isNil   bool
	}{
		{"nil", &ParameterOrRef{}, true, true},
		{"both", both, true, false},
		{"ref", &ParameterOrRef{Ref: &Reference{Ref: "#/x"}}, false, false},
		{"value", &ParameterOrRef{Value: &Parameter{}}, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.in.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if errors.Is(err, ErrNilOrRef) != tt.isNil {
				t.Errorf("errors.Is(ErrNilOrRef) = %v, want %v", !tt.isNil, tt.isNil)
			}
		})
	}
}

func TestOrRefMarshalYAML(t *testing.T) {
	ref := &Reference{Ref: "#/x"}
	val := &Parameter{Name: "n"}
	if got, err := (&ParameterOrRef{Ref: ref}).MarshalYAML(); err != nil || got != any(ref) {
		t.Errorf("ref: got %v, %v", got, err)
	}
	if got, err := (&ParameterOrRef{Value: val}).MarshalYAML(); err != nil || got != any(val) {
		t.Errorf("value: got %v, %v", got, err)
	}
	if _, err := (&ParameterOrRef{}).MarshalYAML(); !errors.Is(err, ErrNilOrRef) {
		t.Errorf("nil: got %v", err)
	}
}
