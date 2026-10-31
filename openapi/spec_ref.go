package openapi

import (
	"encoding/json/jsontext"
	jsonv2 "encoding/json/v2"
	"errors"
)

// ErrNilOrRef is returned when an OrRef has neither a $ref nor a value.
var ErrNilOrRef = errors.New("openapi: OrRef has neither $ref nor value")

// Reference is a simple object to allow referencing
// other components in the specification, internally and
// externally. Summary and Description override those of
// the referenced component.
type Reference struct {
	Ref         string `json:"$ref" yaml:"$ref"`
	Summary     string `json:"summary,omitempty" yaml:"summary,omitempty"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
}

// OrRef holds either a Reference or an inline Value of type T.
// Exactly one of the two must be set; see Validate.
type OrRef[T any] struct {
	Ref   *Reference
	Value *T
}

// MarshalJSONTo implements jsonv2.MarshalerTo. A non-nil Ref wins
// over Value. It returns ErrNilOrRef when both are nil.
func (o OrRef[T]) MarshalJSONTo(enc *jsontext.Encoder) error {
	switch {
	case o.Ref != nil:
		return encodeValue(enc, o.Ref)
	case o.Value != nil:
		return encodeValue(enc, o.Value)
	}
	return ErrNilOrRef
}

// UnmarshalJSONFrom implements jsonv2.UnmarshalerFrom. The presence
// of a top-level "$ref" key decides between Ref and Value.
func (o *OrRef[T]) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	raw, err := dec.ReadValue()
	if err != nil {
		return err
	}
	var probe map[string]jsontext.Value
	if err := jsonv2.Unmarshal(raw, &probe, unmarshalOptions); err != nil {
		return err
	}
	*o = OrRef[T]{}
	if _, ok := probe["$ref"]; ok {
		var r Reference
		if err := jsonv2.Unmarshal(raw, &r, unmarshalOptions); err != nil {
			return err
		}
		o.Ref = &r
		return nil
	}
	var v T
	if err := jsonv2.Unmarshal(raw, &v, unmarshalOptions); err != nil {
		return err
	}
	o.Value = &v
	return nil
}

// MarshalJSON implements json.Marshaler by delegating to v2.
func (o OrRef[T]) MarshalJSON() ([]byte, error) { return marshalJSON(o) }

// UnmarshalJSON implements json.Unmarshaler by delegating to v2.
func (o *OrRef[T]) UnmarshalJSON(b []byte) error { return unmarshalJSON(b, o) }

// MarshalYAML implements yaml.Marshaler with the same choice logic
// as MarshalJSON.
func (o OrRef[T]) MarshalYAML() (any, error) {
	switch {
	case o.Ref != nil:
		return o.Ref, nil
	case o.Value != nil:
		return o.Value, nil
	}
	return nil, ErrNilOrRef
}

// Validate reports an error unless exactly one of Ref and Value is
// set. It returns ErrNilOrRef when both are nil.
func (o *OrRef[T]) Validate() error {
	switch {
	case o.Ref == nil && o.Value == nil:
		return ErrNilOrRef
	case o.Ref != nil && o.Value != nil:
		return errors.New("both $ref and value set")
	}
	return nil
}
