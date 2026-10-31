package openapi

import (
	"encoding/json"
	"encoding/json/jsontext"
	jsonv2 "encoding/json/v2"
	"fmt"
)

// cloneOptions decode like unmarshalOptions, except that a number held
// in an "any" (an extension, an example, a default...) is kept as a
// json.Number, which is what encoding/json's UseNumber does. A float64
// would round 18446744073709551615 and turn 1e400 into an error, and the
// clone would no longer marshal to the same bytes.
var cloneOptions = jsonv2.JoinOptions(
	unmarshalOptions,
	jsonv2.WithUnmarshalers(jsonv2.UnmarshalFromFunc(func(dec *jsontext.Decoder, v *any) error {
		x, err := decodeAny(dec)
		*v = x
		return err
	})),
)

// decodeAny decodes the next value as encoding/json does into an any,
// with the numbers as json.Number.
func decodeAny(dec *jsontext.Decoder) (any, error) {
	switch dec.PeekKind() {
	case '0':
		val, err := dec.ReadValue()
		return json.Number(val), err
	case '[':
		if _, err := dec.ReadToken(); err != nil {
			return nil, err
		}
		s := []any{}
		for dec.PeekKind() != ']' {
			e, err := decodeAny(dec)
			if err != nil {
				return nil, err
			}
			s = append(s, e)
		}
		_, err := dec.ReadToken()
		return s, err
	case '{':
		if _, err := dec.ReadToken(); err != nil {
			return nil, err
		}
		m := map[string]any{}
		for dec.PeekKind() != '}' {
			k, err := dec.ReadToken()
			if err != nil {
				return nil, err
			}
			name := k.String() // valid until the next read
			e, err := decodeAny(dec)
			if err != nil {
				return nil, err
			}
			m[name] = e
		}
		_, err := dec.ReadToken()
		return m, err
	}
	t, err := dec.ReadToken() // null, bool or string
	if err != nil {
		return nil, err
	}
	switch t.Kind() {
	case 't':
		return true, nil
	case 'f':
		return false, nil
	case '"':
		return t.String(), nil
	}
	return nil, nil
}

// deepClone copies *v by encoding it with the options of MarshalJSON
// and decoding the result.
func deepClone[T any](v *T) (*T, error) {
	b, err := marshalJSON(v)
	if err != nil {
		return nil, err
	}
	var out *T
	if err := jsonv2.Unmarshal(b, &out, cloneOptions); err != nil {
		return nil, err
	}
	return out, nil
}

// Clone returns a deep copy of the document: the copy shares no map,
// slice or pointer with o. It is made by encoding the document with
// MarshalJSON and decoding it back, so the copy marshals to the same
// bytes as o, and a value that cannot be encoded (a NaN or an infinity
// in an extension, a channel...) is returned as an error, never a
// panic. Numbers held in "any" fields come back as json.Number, and
// other values as the generic types of encoding/json (a struct example
// becomes a map[string]any). A nil receiver gives a nil copy.
func (o *OpenAPI) Clone() (*OpenAPI, error) {
	if o == nil {
		return nil, nil
	}
	c, err := deepClone(o)
	if err != nil {
		return nil, fmt.Errorf("openapi: cloning the document: %w", err)
	}
	return c, nil
}

// Clone returns a deep copy of the parameter, with the rules of
// OpenAPI.Clone. A nil receiver gives a nil copy.
func (p *Parameter) Clone() (*Parameter, error) {
	if p == nil {
		return nil, nil
	}
	c, err := deepClone(p)
	if err != nil {
		return nil, fmt.Errorf("openapi: cloning the parameter: %w", err)
	}
	return c, nil
}

// Clone returns a deep copy of the operation, with the rules of
// OpenAPI.Clone. A nil receiver gives a nil copy.
func (o *Operation) Clone() (*Operation, error) {
	if o == nil {
		return nil, nil
	}
	c, err := deepClone(o)
	if err != nil {
		return nil, fmt.Errorf("openapi: cloning the operation: %w", err)
	}
	return c, nil
}
