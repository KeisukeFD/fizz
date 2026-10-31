package openapi

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestSchemaMarshal(t *testing.T) {
	yes := true
	tests := []struct {
		name string
		in   *Schema
		want string
	}{
		{"ref with sibling", &Schema{Ref: "#/c", Description: "d"}, `{"$ref":"#/c","description":"d"}`},
		{"single type", &Schema{Type: Types("string")}, `{"type":"string"}`},
		{"type array", &Schema{Type: Types("string", "null")}, `{"type":["string","null"]}`},
		{"empty type", &Schema{Type: Types()}, `{}`},
		{"bool true", &Schema{Bool: &yes}, `true`},
		{"bool false", &Schema{Bool: new(false)}, `false`},
		{"float exact", &Schema{Minimum: Num(0.1)}, `{"minimum":0.1}`},
		{"float32 exact", &Schema{Maximum: Num(float32(0.1))}, `{"maximum":0.1}`},
		{"uint64 exact", &Schema{Maximum: Num(uint64(18446744073709551615))}, `{"maximum":18446744073709551615}`},
		{"negative int", &Schema{ExclusiveMinimum: Num(-3)}, `{"exclusiveMinimum":-3}`},
		{"zero minLength", &Schema{MinLength: new(uint64(0))}, `{"minLength":0}`},
		{"nested bool", &Schema{Properties: map[string]*Schema{"a": {Bool: &yes}}}, `{"properties":{"a":true}}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := json.Marshal(tt.in)
			if err != nil {
				t.Fatal(err)
			}
			if string(b) != tt.want {
				t.Errorf("got %s, want %s", b, tt.want)
			}
		})
	}
}

func TestSchemaUnmarshalRoundTrip(t *testing.T) {
	const doc = `{"$ref":"#/c","type":["string","null"],"allOf":[true,{"type":"integer"}],` +
		`"not":false,"const":"x","examples":[1,"a"],"multipleOf":0.1,` +
		`"maximum":18446744073709551615,"exclusiveMinimum":-3,"maxLength":10,"minLength":0,` +
		`"readOnly":true,"contentMediaType":"image/png","contentEncoding":"base64",` +
		`"propertyNames":{"pattern":"^a"},"prefixItems":[{"type":"string"}],` +
		`"discriminator":{"propertyName":"k","mapping":{"a":"#/A"},"defaultMapping":"#/D"},` +
		`"xml":{"nodeType":"element","name":"n","attribute":true},` +
		`"externalDocs":{"description":"d","url":"https://x"}}`

	var s Schema
	if err := json.Unmarshal([]byte(doc), &s); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(s.Type, Types("string", "null")) {
		t.Errorf("type: %v", s.Type)
	}
	if s.AllOf[0].Bool == nil || !*s.AllOf[0].Bool || s.Not.Bool == nil || *s.Not.Bool {
		t.Error("boolean schemas not decoded")
	}
	if s.MinLength == nil || *s.MinLength != 0 {
		t.Error("minLength 0 lost")
	}
	if s.Maximum != "18446744073709551615" {
		t.Errorf("maximum: %s", s.Maximum)
	}
	b, err := json.Marshal(&s)
	if err != nil {
		t.Fatal(err)
	}
	var want, got any
	if err := json.Unmarshal([]byte(doc), &want); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(want, got) {
		t.Errorf("round trip mismatch:\n got %s\nwant %s", b, doc)
	}
	var single Schema
	if err := json.Unmarshal([]byte(`{"type":"string"}`), &single); err != nil || !reflect.DeepEqual(single.Type, Types("string")) {
		t.Errorf("single type: %v %v", single.Type, err)
	}
}

func TestMakeNullable(t *testing.T) {
	tests := []struct {
		name string
		in   *Schema
		want string
	}{
		{"ref", &Schema{Ref: "#/c", Description: "d"},
			`{"anyOf":[{"$ref":"#/c"},{"type":"null"}],"description":"d"}`},
		{"type", &Schema{Type: Types("string")}, `{"type":["string","null"]}`},
		{"already null", &Schema{Type: Types("string", "null")}, `{"type":["string","null"]}`},
		{"no type", &Schema{Description: "d"}, `{"description":"d"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.in.MakeNullable()
			b, err := json.Marshal(tt.in)
			if err != nil {
				t.Fatal(err)
			}
			if string(b) != tt.want {
				t.Errorf("got %s, want %s", b, tt.want)
			}
		})
	}
}
