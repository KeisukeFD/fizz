package openapi

import (
	"bytes"
	"encoding/json"
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

// MarshalJSON implements json.Marshaler. A non-nil Ref wins over
// Value. It returns ErrNilOrRef when both are nil.
func (o OrRef[T]) MarshalJSON() ([]byte, error) {
	switch {
	case o.Ref != nil:
		return json.Marshal(o.Ref)
	case o.Value != nil:
		return json.Marshal(o.Value)
	}
	return nil, ErrNilOrRef
}

// UnmarshalJSON implements json.Unmarshaler. The presence of a
// top-level "$ref" key decides between Ref and Value.
func (o *OrRef[T]) UnmarshalJSON(b []byte) error {
	var probe map[string]json.RawMessage
	if err := json.Unmarshal(bytes.TrimSpace(b), &probe); err != nil {
		return err
	}
	*o = OrRef[T]{}
	if _, ok := probe["$ref"]; ok {
		var r Reference
		if err := json.Unmarshal(b, &r); err != nil {
			return err
		}
		o.Ref = &r
		return nil
	}
	var v T
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	o.Value = &v
	return nil
}

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
