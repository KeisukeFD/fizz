package openapi

import (
	"bytes"
	"encoding/json/jsontext"
	jsonv2 "encoding/json/v2"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"go.yaml.in/yaml/v3"
)

// marshalOptions are the options of encoding/json/v2 that the document is
// marshaled with: the v2 defaults, with sorted map keys (Deterministic) and
// the escaping and the nil slice and map as null of encoding/json (v1), which
// keeps the output of the user values in "any" fields (a nil slice in an
// example stays null, as the API's own encoding/json responses do) and the
// output of encoding/json (v1) callers byte-identical: v1 escapes the
// output of the json.Marshaler methods of the model itself. The same
// options apply to the user values held in "any" fields, whichever package
// the caller uses: encoding/json (v1) calls the v2 methods of the model,
// and encoding/json/v2 callers get them from the methods.
//
// The model relies on the v2 semantics: "omitzero" on struct-valued and
// bool fields (v2 "omitempty" drops an empty struct such as
// "itemSchema: {}" or "components: {}", and a nil slice is an empty array,
// not null), and no time.Duration in an "any" field (the defaults and
// enums hold nanoseconds as int64).
var marshalOptions = jsonv2.JoinOptions(
	jsonv2.DefaultOptionsV2(),
	jsonv2.Deterministic(true),
	jsonv2.FormatNilSliceAsNull(true),
	jsonv2.FormatNilMapAsNull(true),
	jsontext.EscapeForHTML(true),
	jsontext.EscapeForJS(true),
)

// unmarshalOptions pin the v2 decoding semantics: member names are matched
// case-sensitively, as the specification requires, and duplicate names are
// rejected.
var unmarshalOptions = jsonv2.DefaultOptionsV2()

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
	return jsonv2.Marshal(doc, marshalOptions)
}

// MarshalYAML returns the YAML encoding of doc, in block style with an
// indentation of two spaces.
//
// The YAML document is derived from the JSON one: doc is encoded with
// MarshalJSON, and that output is re-encoded as YAML. The two documents
// therefore hold the same values, and any MarshalYAML method of a user
// type (in an example, for instance) is ignored. Scalar tags are kept,
// so strings that would read as another type in YAML ("123", "true",
// "null", "~", "yes", "2026-10-30") are quoted.
func MarshalYAML(doc *OpenAPI) ([]byte, error) {
	j, err := MarshalJSON(doc)
	if err != nil {
		return nil, err
	}
	var n yaml.Node
	if err := yaml.Unmarshal(j, &n); err != nil {
		return nil, fmt.Errorf("openapi: decoding the JSON document as YAML: %w", err)
	}
	clearStyle(&n)

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(&n); err != nil {
		return nil, fmt.Errorf("openapi: encoding the YAML document: %w", err)
	}
	if err := enc.Close(); err != nil {
		return nil, fmt.Errorf("openapi: encoding the YAML document: %w", err)
	}
	return buf.Bytes(), nil
}

// clearStyle resets the style of every node, which turns the flow
// collections and quoted scalars of the JSON text into block style.
// The tags are kept: the encoder quotes a string scalar only when its
// plain form would resolve to another tag. The one exception is "<<",
// which its resolver reports as a string while the decoder reads the
// plain form as a merge key, so it is quoted here.
func clearStyle(n *yaml.Node) {
	n.Style = 0
	if n.Kind == yaml.ScalarNode && n.Tag == "!!str" && n.Value == "<<" {
		n.Style = yaml.DoubleQuotedStyle
	}
	for _, c := range n.Content {
		clearStyle(c)
	}
}

func marshalJSON(v any) ([]byte, error) {
	return jsonv2.Marshal(v, marshalOptions)
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

// hasMember reports whether the JSON object v has a member named name.
// v must be a valid JSON value; anything but an object has no members.
func hasMember(v jsontext.Value, name string) bool {
	dec := jsontext.NewDecoder(bytes.NewReader(v))
	if tok, err := dec.ReadToken(); err != nil || tok.Kind() != '{' {
		return false
	}
	for dec.PeekKind() == '"' {
		tok, err := dec.ReadToken()
		if err != nil {
			return false
		}
		if tok.String() == name {
			return true
		}
		if dec.SkipValue() != nil {
			return false
		}
	}
	return false
}
