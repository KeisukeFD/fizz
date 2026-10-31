package openapi

import (
	"encoding/json"
	"encoding/json/jsontext"
	"reflect"
	"strconv"
)

// SchemaOrRef is the former reference-or-schema wrapper.
//
// Deprecated: use Schema.
type SchemaOrRef = Schema

// TypeSet is the value of the JSON Schema "type" keyword. It
// marshals to a JSON string when it holds a single element,
// and to an array otherwise.
type TypeSet []string

// Types returns a TypeSet holding the given type names.
func Types(t ...string) TypeSet { return TypeSet(t) }

// MarshalJSONTo implements jsonv2.MarshalerTo for TypeSet.
func (ts TypeSet) MarshalJSONTo(enc *jsontext.Encoder) error {
	if len(ts) == 1 {
		return enc.WriteToken(jsontext.String(ts[0]))
	}
	return encodeValue(enc, []string(ts))
}

// UnmarshalJSONFrom implements jsonv2.UnmarshalerFrom for TypeSet.
// It accepts both a string and an array of strings.
func (ts *TypeSet) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	if dec.PeekKind() == '"' {
		var s string
		if err := decodeValue(dec, &s); err != nil {
			return err
		}
		*ts = TypeSet{s}
		return nil
	}
	var a []string
	if err := decodeValue(dec, &a); err != nil {
		return err
	}
	*ts = TypeSet(a)
	return nil
}

// MarshalJSON implements json.Marshaler for TypeSet.
func (ts TypeSet) MarshalJSON() ([]byte, error) { return marshalJSON(ts) }

// UnmarshalJSON implements json.Unmarshaler for TypeSet.
func (ts *TypeSet) UnmarshalJSON(b []byte) error { return unmarshalJSON(b, ts) }

// Discriminator aids in serializing, deserializing and
// validating request bodies or responses that may be one
// of several schemas.
type Discriminator struct {
	PropertyName   string            `json:"propertyName"`
	Mapping        map[string]string `json:"mapping,omitempty"`
	DefaultMapping string            `json:"defaultMapping,omitzero"`

	// Extensions holds the "x-" specification extensions. Keys are
	// emitted sorted after the typed members. Use SetExtension to
	// validate keys.
	Extensions map[string]any `json:",embed"`
}

// XML describes the XML representation of a schema.
type XML struct {
	NodeType  string `json:"nodeType,omitzero"`
	Name      string `json:"name,omitzero"`
	Namespace string `json:"namespace,omitzero"`
	Prefix    string `json:"prefix,omitzero"`
	Attribute bool   `json:"attribute,omitzero"`
	Wrapped   bool   `json:"wrapped,omitzero"`

	// Extensions holds the "x-" specification extensions. Keys are
	// emitted sorted after the typed members. Use SetExtension to
	// validate keys.
	Extensions map[string]any `json:",embed"`
}

// ExternalDocs allows referencing an external resource for
// extended documentation.
type ExternalDocs struct {
	Description string `json:"description,omitempty"`
	URL         string `json:"url"`

	// Extensions holds the "x-" specification extensions. Keys are
	// emitted sorted after the typed members. Use SetExtension to
	// validate keys.
	Extensions map[string]any `json:",embed"`
}

// Schema represents the definition of input and output data
// types of the API, as a JSON Schema 2020-12 object. A Schema
// that has Ref set is a reference; sibling keywords are allowed.
type Schema struct {
	Ref string `json:"$ref,omitzero"`

	// Bool, when non-nil, makes the schema the boolean schema
	// true or false. All other fields are then ignored.
	Bool *bool `json:"-"`

	Type                 TypeSet            `json:"type,omitzero"`
	AllOf                []*Schema          `json:"allOf,omitempty"`
	OneOf                []*Schema          `json:"oneOf,omitempty"`
	AnyOf                []*Schema          `json:"anyOf,omitempty"`
	Not                  *Schema            `json:"not,omitempty"`
	Items                *Schema            `json:"items,omitempty"`
	PrefixItems          []*Schema          `json:"prefixItems,omitempty"`
	Properties           map[string]*Schema `json:"properties,omitempty"`
	AdditionalProperties *Schema            `json:"additionalProperties,omitempty"`
	PropertyNames        *Schema            `json:"propertyNames,omitempty"`
	Description          string             `json:"description,omitempty"`
	Format               string             `json:"format,omitempty"`
	Default              any                `json:"default,omitempty"`

	// Const holds the "const" keyword. It cannot express
	// "const": null yet, since a nil value is omitted.
	Const    any   `json:"const,omitzero"`
	Examples []any `json:"examples,omitempty"`

	Title            string      `json:"title,omitempty"`
	MultipleOf       json.Number `json:"multipleOf,omitzero"`
	Maximum          json.Number `json:"maximum,omitzero"`
	ExclusiveMaximum json.Number `json:"exclusiveMaximum,omitzero"`
	Minimum          json.Number `json:"minimum,omitzero"`
	ExclusiveMinimum json.Number `json:"exclusiveMinimum,omitzero"`
	MaxLength        *uint64     `json:"maxLength,omitempty"`
	MinLength        *uint64     `json:"minLength,omitempty"`
	Pattern          string      `json:"pattern,omitempty"`
	MaxItems         *uint64     `json:"maxItems,omitempty"`
	MinItems         *uint64     `json:"minItems,omitempty"`
	UniqueItems      bool        `json:"uniqueItems,omitempty"`
	MaxProperties    *uint64     `json:"maxProperties,omitempty"`
	MinProperties    *uint64     `json:"minProperties,omitempty"`
	Required         []string    `json:"required,omitempty"`
	Enum             []any       `json:"enum,omitempty"`
	Deprecated       bool        `json:"deprecated,omitempty"`
	ReadOnly         bool        `json:"readOnly,omitzero"`
	WriteOnly        bool        `json:"writeOnly,omitzero"`

	ContentMediaType string `json:"contentMediaType,omitzero"`
	ContentEncoding  string `json:"contentEncoding,omitzero"`

	Discriminator *Discriminator `json:"discriminator,omitempty"`
	XML           *XML           `json:"xml,omitempty"`
	ExternalDocs  *ExternalDocs  `json:"externalDocs,omitempty"`

	// Extensions holds the "x-" specification extensions. Keys are
	// emitted sorted after the typed members. Use SetExtension to
	// validate keys.
	Extensions map[string]any `json:",embed"`
}

// schemaAlias has the fields of Schema but none of its methods.
type schemaAlias Schema

// MarshalJSONTo implements jsonv2.MarshalerTo for Schema.
func (s Schema) MarshalJSONTo(enc *jsontext.Encoder) error {
	if s.Bool != nil {
		return enc.WriteToken(jsontext.Bool(*s.Bool))
	}
	return encodeValue(enc, schemaAlias(s))
}

// UnmarshalJSONFrom implements jsonv2.UnmarshalerFrom for Schema. The
// literals true and false decode to boolean schemas.
func (s *Schema) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	switch dec.PeekKind() {
	case 't', 'f':
		tok, err := dec.ReadToken()
		if err != nil {
			return err
		}
		v := tok.Bool()
		*s = Schema{Bool: &v}
		return nil
	}
	var a schemaAlias
	if err := decodeValue(dec, &a); err != nil {
		return err
	}
	*s = Schema(a)
	return nil
}

// MarshalJSON implements json.Marshaler for Schema.
func (s Schema) MarshalJSON() ([]byte, error) { return marshalJSON(s) }

// UnmarshalJSON implements json.Unmarshaler for Schema.
func (s *Schema) UnmarshalJSON(b []byte) error { return unmarshalJSON(b, s) }

// MakeNullable allows the schema to also validate null. A
// reference is moved into anyOf next to a null type schema,
// keeping the other keywords of s in place. Otherwise "null"
// is added to the type set, if any.
func (s *Schema) MakeNullable() {
	if s.Ref != "" {
		s.AnyOf = []*Schema{{Ref: s.Ref}, {Type: Types("null")}}
		s.Ref = ""
		return
	}
	if len(s.Type) == 0 {
		return
	}
	for _, t := range s.Type {
		if t == "null" {
			return
		}
	}
	s.Type = append(s.Type[:len(s.Type):len(s.Type)], "null")
}

// Num returns v as a json.Number, preserving its exact
// decimal representation.
func Num[T ~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~float32 | ~float64](v T) json.Number {
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Float32:
		return json.Number(strconv.FormatFloat(rv.Float(), 'g', -1, 32))
	case reflect.Float64:
		return json.Number(strconv.FormatFloat(rv.Float(), 'g', -1, 64))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return json.Number(strconv.FormatUint(rv.Uint(), 10))
	default:
		return json.Number(strconv.FormatInt(rv.Int(), 10))
	}
}

// u64 returns a pointer to v.
func u64(v uint64) *uint64 { return &v }
