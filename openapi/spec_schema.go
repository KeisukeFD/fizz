package openapi

import (
	"bytes"
	"encoding/json"
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

// MarshalJSON implements json.Marshaler for TypeSet.
func (ts TypeSet) MarshalJSON() ([]byte, error) {
	if len(ts) == 1 {
		return json.Marshal(ts[0])
	}
	return json.Marshal([]string(ts))
}

// UnmarshalJSON implements json.Unmarshaler for TypeSet. It
// accepts both a string and an array of strings.
func (ts *TypeSet) UnmarshalJSON(b []byte) error {
	if b = bytes.TrimSpace(b); len(b) > 0 && b[0] == '"' {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		*ts = TypeSet{s}
		return nil
	}
	var a []string
	if err := json.Unmarshal(b, &a); err != nil {
		return err
	}
	*ts = TypeSet(a)
	return nil
}

// MarshalYAML implements yaml.Marshaler for TypeSet.
func (ts TypeSet) MarshalYAML() (any, error) {
	if len(ts) == 1 {
		return ts[0], nil
	}
	return []string(ts), nil
}

// Discriminator aids in serializing, deserializing and
// validating request bodies or responses that may be one
// of several schemas.
type Discriminator struct {
	PropertyName   string            `json:"propertyName" yaml:"propertyName"`
	Mapping        map[string]string `json:"mapping,omitempty" yaml:"mapping,omitempty"`
	DefaultMapping string            `json:"defaultMapping,omitzero" yaml:"defaultMapping,omitempty"`
}

// XML describes the XML representation of a schema.
type XML struct {
	NodeType  string `json:"nodeType,omitzero" yaml:"nodeType,omitempty"`
	Name      string `json:"name,omitzero" yaml:"name,omitempty"`
	Namespace string `json:"namespace,omitzero" yaml:"namespace,omitempty"`
	Prefix    string `json:"prefix,omitzero" yaml:"prefix,omitempty"`
	Attribute bool   `json:"attribute,omitzero" yaml:"attribute,omitempty"`
	Wrapped   bool   `json:"wrapped,omitzero" yaml:"wrapped,omitempty"`
}

// ExternalDocs allows referencing an external resource for
// extended documentation.
type ExternalDocs struct {
	Description string `json:"description,omitzero" yaml:"description,omitempty"`
	URL         string `json:"url" yaml:"url"`
}

// Schema represents the definition of input and output data
// types of the API, as a JSON Schema 2020-12 object. A Schema
// that has Ref set is a reference; sibling keywords are allowed.
type Schema struct {
	Ref string `json:"$ref,omitzero" yaml:"$ref,omitempty"`

	// Bool, when non-nil, makes the schema the boolean schema
	// true or false. All other fields are then ignored.
	Bool *bool `json:"-" yaml:"-"`

	Type                 TypeSet            `json:"type,omitzero" yaml:"type,omitempty"`
	AllOf                []*Schema          `json:"allOf,omitempty" yaml:"allOf,omitempty"`
	OneOf                []*Schema          `json:"oneOf,omitempty" yaml:"oneOf,omitempty"`
	AnyOf                []*Schema          `json:"anyOf,omitempty" yaml:"anyOf,omitempty"`
	Not                  *Schema            `json:"not,omitempty" yaml:"not,omitempty"`
	Items                *Schema            `json:"items,omitempty" yaml:"items,omitempty"`
	PrefixItems          []*Schema          `json:"prefixItems,omitempty" yaml:"prefixItems,omitempty"`
	Properties           map[string]*Schema `json:"properties,omitempty" yaml:"properties,omitempty"`
	AdditionalProperties *Schema            `json:"additionalProperties,omitempty" yaml:"additionalProperties,omitempty"`
	PropertyNames        *Schema            `json:"propertyNames,omitempty" yaml:"propertyNames,omitempty"`
	Description          string             `json:"description,omitempty" yaml:"description,omitempty"`
	Format               string             `json:"format,omitempty" yaml:"format,omitempty"`
	Default              any                `json:"default,omitempty" yaml:"default,omitempty"`

	// Const holds the "const" keyword. It cannot express
	// "const": null yet, since a nil value is omitted.
	Const    any   `json:"const,omitzero" yaml:"const,omitempty"`
	Examples []any `json:"examples,omitempty" yaml:"examples,omitempty"`

	Title            string      `json:"title,omitempty" yaml:"title,omitempty"`
	MultipleOf       json.Number `json:"multipleOf,omitzero" yaml:"multipleOf,omitempty"`
	Maximum          json.Number `json:"maximum,omitzero" yaml:"maximum,omitempty"`
	ExclusiveMaximum json.Number `json:"exclusiveMaximum,omitzero" yaml:"exclusiveMaximum,omitempty"`
	Minimum          json.Number `json:"minimum,omitzero" yaml:"minimum,omitempty"`
	ExclusiveMinimum json.Number `json:"exclusiveMinimum,omitzero" yaml:"exclusiveMinimum,omitempty"`
	MaxLength        *uint64     `json:"maxLength,omitempty" yaml:"maxLength,omitempty"`
	MinLength        *uint64     `json:"minLength,omitempty" yaml:"minLength,omitempty"`
	Pattern          string      `json:"pattern,omitempty" yaml:"pattern,omitempty"`
	MaxItems         *uint64     `json:"maxItems,omitempty" yaml:"maxItems,omitempty"`
	MinItems         *uint64     `json:"minItems,omitempty" yaml:"minItems,omitempty"`
	UniqueItems      bool        `json:"uniqueItems,omitempty" yaml:"uniqueItems,omitempty"`
	MaxProperties    *uint64     `json:"maxProperties,omitempty" yaml:"maxProperties,omitempty"`
	MinProperties    *uint64     `json:"minProperties,omitempty" yaml:"minProperties,omitempty"`
	Required         []string    `json:"required,omitempty" yaml:"required,omitempty"`
	Enum             []any       `json:"enum,omitempty" yaml:"enum,omitempty"`
	Deprecated       bool        `json:"deprecated,omitempty" yaml:"deprecated,omitempty"`
	ReadOnly         bool        `json:"readOnly,omitzero" yaml:"readOnly,omitempty"`
	WriteOnly        bool        `json:"writeOnly,omitzero" yaml:"writeOnly,omitempty"`

	ContentMediaType string `json:"contentMediaType,omitzero" yaml:"contentMediaType,omitempty"`
	ContentEncoding  string `json:"contentEncoding,omitzero" yaml:"contentEncoding,omitempty"`

	Discriminator *Discriminator `json:"discriminator,omitempty" yaml:"discriminator,omitempty"`
	XML           *XML           `json:"xml,omitempty" yaml:"xml,omitempty"`
	ExternalDocs  *ExternalDocs  `json:"externalDocs,omitempty" yaml:"externalDocs,omitempty"`
}

// schemaAlias has the fields of Schema but none of its methods.
type schemaAlias Schema

// MarshalJSON implements json.Marshaler for Schema.
func (s Schema) MarshalJSON() ([]byte, error) {
	if s.Bool != nil {
		return json.Marshal(*s.Bool)
	}
	return json.Marshal(schemaAlias(s))
}

// UnmarshalJSON implements json.Unmarshaler for Schema. The
// literals true and false decode to boolean schemas.
func (s *Schema) UnmarshalJSON(b []byte) error {
	switch lit := string(bytes.TrimSpace(b)); lit {
	case "true", "false":
		v := lit == "true"
		*s = Schema{Bool: &v}
		return nil
	}
	var a schemaAlias
	if err := json.Unmarshal(b, &a); err != nil {
		return err
	}
	*s = Schema(a)
	return nil
}

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
