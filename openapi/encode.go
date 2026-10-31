package openapi

import (
	"encoding/json"
	"encoding/json/jsontext"
	jsonv2 "encoding/json/v2"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
)

// marshalOptions are the options of encoding/json (v1) expressed for the v2
// engine: marshaling with them is what json.Marshal does. This keeps the
// output of the document, including the user values held in "any" fields
// (nil slices as null, legacy omitempty, escaping of <, >, & and U+2028/9),
// the same as before v2, and the same whichever package the caller uses:
// encoding/json (v1) calls the v2 methods of the model with these options,
// and encoding/json/v2 callers get them from the methods. Map keys are
// sorted (Deterministic is part of the set).
var marshalOptions = json.DefaultOptionsV1()

// unmarshalOptions pin the v2 decoding semantics: member names are matched
// case-sensitively, as the specification requires, and duplicate names are
// rejected.
var unmarshalOptions = jsonv2.DefaultOptionsV2()

// jsonOptions are the options of the top-level encoders.
func jsonOptions() jsonv2.Options { return marshalOptions }

func encodeValue(enc *jsontext.Encoder, v any) error {
	return jsonv2.MarshalEncode(enc, v, marshalOptions)
}

func decodeValue(dec *jsontext.Decoder, v any) error {
	return jsonv2.UnmarshalDecode(dec, v, unmarshalOptions)
}

// MarshalJSON returns the compact, deterministic JSON encoding of doc:
// map keys and "x-" extensions are sorted, and extensions are emitted
// inline after the typed members of their object.
//
// The model is encoded with encoding/json/v2. The extensible types also
// implement json.Marshaler and json.Unmarshaler by delegating to v2, so
// encoding/json (v1) produces the same bytes for a *OpenAPI, and keeps
// the extensions when it unmarshals into any of them. The delegation
// uses the options of MarshalJSON, not those of the caller. Prefer
// MarshalJSON, or marshal through encoding/json/v2, when a nested
// object is encoded on its own.
func MarshalJSON(doc *OpenAPI) ([]byte, error) {
	return jsonv2.Marshal(doc, jsonOptions())
}

func marshalJSON(v any) ([]byte, error) {
	return jsonv2.Marshal(v, jsonOptions())
}

func unmarshalJSON(b []byte, v any) error {
	return jsonv2.Unmarshal(b, v, unmarshalOptions)
}

// Specification extension keys.
const (
	extPrefix         = "x-"
	reservedOAIPrefix = "x-oai-"
	reservedOASPrefix = "x-oas-"
)

// Errors returned by SetExtension.
var (
	// ErrInvalidExtensionKey is returned for an extension key that does
	// not start with "x-" followed by a name, or that collides with a typed
	// field of the object.
	ErrInvalidExtensionKey = errors.New("openapi: invalid extension key")

	// ErrReservedExtension is returned for an extension key with the
	// "x-oai-" or "x-oas-" prefix, which the OpenAPI Initiative reserves.
	// The comparison is case-insensitive.
	ErrReservedExtension = errors.New("openapi: reserved extension key")
)

// setExtension validates key and stores value in *m, allocating the map
// on first use. typed lists the extension keys that the object models as
// fields; the typed field wins, so they are rejected.
func setExtension(m *map[string]any, key string, value any, typed ...string) error {
	lower := strings.ToLower(key)
	switch {
	case strings.HasPrefix(lower, reservedOAIPrefix), strings.HasPrefix(lower, reservedOASPrefix):
		return fmt.Errorf("%w: %q", ErrReservedExtension, key)
	case !strings.HasPrefix(key, extPrefix), len(key) == len(extPrefix):
		return fmt.Errorf("%w: %q must start with %q followed by a name", ErrInvalidExtensionKey, key, extPrefix)
	}
	for _, t := range typed {
		if key == t {
			return fmt.Errorf("%w: %q is a typed field", ErrInvalidExtensionKey, key)
		}
	}
	if *m == nil {
		*m = make(map[string]any)
	}
	(*m)[key] = value
	return nil
}

// MarshalJSONTo implements jsonv2.MarshalerTo for Callback. The
// expressions are emitted sorted, then the extensions, sorted.
func (c Callback) MarshalJSONTo(enc *jsontext.Encoder) error {
	if err := enc.WriteToken(jsontext.BeginObject); err != nil {
		return err
	}
	if err := writeSortedMembers(enc, c.Expressions); err != nil {
		return err
	}
	if err := writeSortedMembers(enc, c.Extensions); err != nil {
		return err
	}
	return enc.WriteToken(jsontext.EndObject)
}

func writeSortedMembers[V any](enc *jsontext.Encoder, m map[string]V) error {
	for _, k := range slices.Sorted(maps.Keys(m)) {
		if err := enc.WriteToken(jsontext.String(k)); err != nil {
			return err
		}
		if err := encodeValue(enc, m[k]); err != nil {
			return err
		}
	}
	return nil
}

// UnmarshalJSONFrom implements jsonv2.UnmarshalerFrom for Callback.
// Members whose name starts with "x-" are extensions; the others are
// expressions.
func (c *Callback) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	if dec.PeekKind() == 'n' {
		return dec.SkipValue()
	}
	if tok, err := dec.ReadToken(); err != nil {
		return err
	} else if tok.Kind() != '{' {
		return fmt.Errorf("openapi: callback must be a JSON object, got %v", tok.Kind())
	}
	*c = Callback{}
	for dec.PeekKind() != '}' {
		tok, err := dec.ReadToken()
		if err != nil {
			return err
		}
		name := tok.String()
		if strings.HasPrefix(name, extPrefix) {
			var v any
			if err := decodeValue(dec, &v); err != nil {
				return err
			}
			if c.Extensions == nil {
				c.Extensions = make(map[string]any)
			}
			c.Extensions[name] = v
			continue
		}
		var p *PathItem
		if err := decodeValue(dec, &p); err != nil {
			return err
		}
		if c.Expressions == nil {
			c.Expressions = make(map[string]*PathItem)
		}
		c.Expressions[name] = p
	}
	_, err := dec.ReadToken() // '}'
	return err
}

// MarshalJSON implements json.Marshaler for Callback.
func (c Callback) MarshalJSON() ([]byte, error) { return marshalJSON(c) }

// UnmarshalJSON implements json.Unmarshaler for Callback.
func (c *Callback) UnmarshalJSON(b []byte) error { return unmarshalJSON(b, c) }

// SetExtension sets the specification extension key to value. See
// OpenAPI.SetExtension for the rules.
func (c *Callback) SetExtension(key string, value any) error {
	return setExtension(&c.Extensions, key, value)
}
