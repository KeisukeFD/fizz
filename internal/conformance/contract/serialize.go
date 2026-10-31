// Package contract holds the spec-driven contract tests (docs/plan/PLAN-v2.md,
// P10.6). A small client reads the spec a fizz app serves and sends, for each
// parameter, a request serialized exactly as the document says. The server
// must bind what was sent. A request that the server rejects, or binds to a
// different value, is a drift between the document and the runtime.
//
// This file is the serializer: OAS 3.2.1 "Style Values" and "Style Examples"
// (internal/conformance/oai/orig/oas/versions/3.2.1.md), whose table is the
// oracle of TestSerializeStyleTable.
package contract

import (
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"
)

// Prop is a property of an Object.
type Prop struct {
	Key   string
	Value any
}

// Object is an object sample that keeps the order of its properties, which
// the spec's style table depends on (R, G, B). A map[string]any is an
// object too; its properties are serialized in key order.
type Object []Prop

// Param is what the serializer needs of an OAS 3.2 Parameter Object.
type Param struct {
	Name          string
	In            string // query, path, header or cookie
	Style         string // "" means the default of In
	Explode       *bool  // nil means the default of Style
	AllowReserved bool
}

// ParamOf reads a decoded Parameter Object.
func ParamOf(m map[string]any) Param {
	p := Param{AllowReserved: m["allowReserved"] == true}
	p.Name, _ = m["name"].(string)
	p.In, _ = m["in"].(string)
	p.Style, _ = m["style"].(string)
	if b, ok := m["explode"].(bool); ok {
		p.Explode = &b
	}
	return p
}

// style returns the style, with the 3.2.1 default of the location: form for
// query and cookie, simple for path and header.
func (p Param) style() string {
	if p.Style != "" {
		return p.Style
	}
	switch p.In {
	case "query", "cookie":
		return "form"
	}
	return "simple"
}

// explode returns the explode flag, true by default for form only.
func (p Param) explode() bool {
	if p.Explode != nil {
		return *p.Explode
	}
	return p.style() == "form"
}

// Placement is where the serialized value goes. Only the field of the
// parameter's location is set.
type Placement struct {
	// Query is a raw query string fragment ("a=1&a=2"), without leading ?.
	Query string
	// Path replaces the {name} template variable, with the leading
	// delimiter of the label and matrix styles.
	Path string
	// Header is the value of the header named after the parameter.
	Header string
	// Cookie is a fragment of the Cookie header ("a=1; a=2").
	Cookie string
}

// Serialize returns the placement of v, a scalar, a []any, a map[string]any
// or an Object (nil is the undefined value), according to the style, explode
// and allowReserved of p. It follows RFC 6570 as the spec's "Style Values"
// section describes: values are percent-encoded except the unreserved
// characters (and the reserved ones with allowReserved), delimiters are not.
// The tilde is left alone, which every decoder reads the same.
//
// A combination that the spec table marks n/a is an error.
func Serialize(p Param, v any) (Placement, error) {
	style, explode := p.style(), p.explode()
	allowed := map[string][]string{
		"query":  {"form", "spaceDelimited", "pipeDelimited", "deepObject"},
		"path":   {"simple", "label", "matrix"},
		"header": {"simple"},
		"cookie": {"form", "cookie"},
	}[p.In]
	if !slices.Contains(allowed, style) {
		return Placement{}, fmt.Errorf("style %q is not defined for in: %s", style, p.In)
	}
	val, err := classify(v)
	if err != nil {
		return Placement{}, err
	}
	var s string
	switch p.In {
	case "query":
		s, err = serializeQuery(p, style, explode, val)
		return Placement{Query: s}, err
	case "path":
		s, err = serializePath(p, style, explode, val)
		return Placement{Path: s}, err
	case "header":
		s, err = serializeHeader(explode, val)
		return Placement{Header: s}, err
	default:
		s, err = serializeCookie(p, style, explode, val)
		return Placement{Cookie: s}, err
	}
}

// value is a classified sample.
type value struct {
	kind   string // undefined, scalar, array, object
	scalar string
	items  []string // array elements, as text
	props  []Prop   // object properties, values as text
}

func classify(v any) (value, error) {
	switch x := v.(type) {
	case nil:
		return value{kind: "undefined"}, nil
	case []any:
		out := value{kind: "array", items: make([]string, len(x))}
		for i, e := range x {
			s, err := text(e)
			if err != nil {
				return value{}, err
			}
			out.items[i] = s
		}
		return out, nil
	case []string:
		return classify(slices.Collect(func(yield func(any) bool) {
			for _, s := range x {
				if !yield(s) {
					return
				}
			}
		}))
	case map[string]any:
		props := make(Object, 0, len(x))
		for _, k := range slices.Sorted(maps.Keys(x)) {
			props = append(props, Prop{k, x[k]})
		}
		return classify(props)
	case Object:
		out := value{kind: "object"}
		for _, pr := range x {
			s, err := text(pr.Value)
			if err != nil {
				return value{}, err
			}
			out.props = append(out.props, Prop{pr.Key, s})
		}
		return out, nil
	}
	s, err := text(v)
	return value{kind: "scalar", scalar: s}, err
}

// text is the string form of a primitive.
func text(v any) (string, error) {
	switch x := v.(type) {
	case nil:
		return "", nil
	case string:
		return x, nil
	case bool:
		return strconv.FormatBool(x), nil
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64), nil
	case json.Number:
		return x.String(), nil
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return fmt.Sprint(x), nil
	}
	return "", fmt.Errorf("nested value %T is not a primitive: its serialization is undefined", v)
}

const (
	unreserved = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-._~"
	reserved   = ":/?#[]@!$&'()*+,;="
)

// pct percent-encodes s: everything but the unreserved characters is
// encoded; with allowReserved the reserved ones and the percent-encoded
// triples pass through.
func pct(s string, allowReserved bool) string {
	var sb strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case strings.IndexByte(unreserved, c) >= 0:
			sb.WriteByte(c)
		case allowReserved && strings.IndexByte(reserved, c) >= 0:
			sb.WriteByte(c)
		case allowReserved && c == '%' && i+2 < len(s) && isHex(s[i+1]) && isHex(s[i+2]):
			sb.WriteByte(c)
		default:
			fmt.Fprintf(&sb, "%%%02X", c)
		}
	}
	return sb.String()
}

func isHex(c byte) bool {
	return c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F'
}

func (p Param) enc(s string) string { return pct(s, p.AllowReserved) }

// pairs joins k<sep>v for every property, encoding both.
func (p Param) pairs(props []Prop, sep, join string) string {
	parts := make([]string, len(props))
	for i, pr := range props {
		parts[i] = p.enc(pr.Key) + sep + p.enc(pr.Value.(string))
	}
	return strings.Join(parts, join)
}

// flat lists the object as k,v,k,v (unexploded forms).
func (p Param) flat(props []Prop, join string) string {
	parts := make([]string, 0, 2*len(props))
	for _, pr := range props {
		parts = append(parts, p.enc(pr.Key), p.enc(pr.Value.(string)))
	}
	return strings.Join(parts, join)
}

func (p Param) encAll(items []string) []string {
	out := make([]string, len(items))
	for i, s := range items {
		out[i] = p.enc(s)
	}
	return out
}

var errNA = fmt.Errorf("the combination is n/a in the style table: its behavior is undefined")

// serializeQuery handles the query styles: form (3.2.1 §Style Values: RFC
// 6570 form expansion), spaceDelimited, pipeDelimited and deepObject.
func serializeQuery(p Param, style string, explode bool, v value) (string, error) {
	name := p.enc(p.Name)
	switch style {
	case "spaceDelimited", "pipeDelimited":
		delim := map[string]string{"spaceDelimited": "%20", "pipeDelimited": "%7C"}[style]
		if explode || (v.kind != "array" && v.kind != "object") {
			return "", errNA
		}
		if v.kind == "array" {
			return name + "=" + strings.Join(p.encAll(v.items), delim), nil
		}
		return name + "=" + p.flat(v.props, delim), nil
	case "deepObject":
		if v.kind != "object" {
			return "", errNA
		}
		parts := make([]string, len(v.props))
		for i, pr := range v.props {
			parts[i] = name + "%5B" + p.enc(pr.Key) + "%5D=" + p.enc(pr.Value.(string))
		}
		return strings.Join(parts, "&"), nil
	}
	return form(p, name, explode, v, "&"), nil
}

// form is RFC 6570 form expansion; join separates the exploded pairs.
func form(p Param, name string, explode bool, v value, join string) string {
	switch v.kind {
	case "undefined":
		return name + "="
	case "scalar":
		return name + "=" + p.enc(v.scalar)
	case "array":
		if len(v.items) == 0 {
			return ""
		}
		if explode {
			parts := make([]string, len(v.items))
			for i, it := range v.items {
				parts[i] = name + "=" + p.enc(it)
			}
			return strings.Join(parts, join)
		}
		return name + "=" + strings.Join(p.encAll(v.items), ",")
	}
	if len(v.props) == 0 {
		return ""
	}
	if explode {
		return p.pairs(v.props, "=", join)
	}
	return name + "=" + p.flat(v.props, ",")
}

// serializePath handles simple, label and matrix.
func serializePath(p Param, style string, explode bool, v value) (string, error) {
	if v.kind == "undefined" {
		return "", nil
	}
	switch style {
	case "simple":
		return simple(p, explode, v), nil
	case "label":
		sep := ","
		if explode {
			sep = "."
		}
		switch v.kind {
		case "scalar":
			return "." + p.enc(v.scalar), nil
		case "array":
			return "." + strings.Join(p.encAll(v.items), sep), nil
		}
		if explode {
			return "." + p.pairs(v.props, "=", "."), nil
		}
		return "." + p.flat(v.props, ","), nil
	}
	name := p.enc(p.Name)
	switch v.kind {
	case "scalar":
		if v.scalar == "" {
			return ";" + name, nil // RFC 6570 3.2.7: no = for an empty value
		}
		return ";" + name + "=" + p.enc(v.scalar), nil
	case "array":
		if explode {
			parts := make([]string, len(v.items))
			for i, it := range v.items {
				parts[i] = ";" + name + "=" + p.enc(it)
			}
			return strings.Join(parts, ""), nil
		}
		return ";" + name + "=" + strings.Join(p.encAll(v.items), ","), nil
	}
	if explode {
		return ";" + p.pairs(v.props, "=", ";"), nil
	}
	return ";" + name + "=" + p.flat(v.props, ","), nil
}

// simple is RFC 6570 simple expansion: path and header.
func simple(p Param, explode bool, v value) string {
	switch v.kind {
	case "scalar":
		return p.enc(v.scalar)
	case "array":
		return strings.Join(p.encAll(v.items), ",")
	case "object":
		if explode {
			return p.pairs(v.props, "=", ",")
		}
		return p.flat(v.props, ",")
	}
	return ""
}

// serializeHeader: the value is not percent-encoded. Appendix D says the
// data of a header "needs to be quoted and escaped prior to passing it to the
// OAS implementation", so the samples are sent as they are.
func serializeHeader(explode bool, v value) (string, error) {
	switch v.kind {
	case "scalar":
		return v.scalar, nil
	case "array":
		return strings.Join(v.items, ","), nil
	case "object":
		parts := make([]string, 0, 2*len(v.props))
		for _, pr := range v.props {
			if explode {
				parts = append(parts, pr.Key+"="+pr.Value.(string))
			} else {
				parts = append(parts, pr.Key, pr.Value.(string))
			}
		}
		return strings.Join(parts, ","), nil
	}
	return "", nil
}

// serializeCookie: style cookie applies no percent-encoding (3.2.1
// "Style Values"); form applies it, with the "; " separator of Appendix D.
func serializeCookie(p Param, style string, explode bool, v value) (string, error) {
	if style == "form" {
		return form(p, p.enc(p.Name), explode, v, "; "), nil
	}
	if !explode && (v.kind == "array" || v.kind == "object") {
		return "", errNA
	}
	switch v.kind {
	case "undefined":
		return p.Name + "=", nil
	case "scalar":
		return p.Name + "=" + v.scalar, nil
	case "array":
		parts := make([]string, len(v.items))
		for i, it := range v.items {
			parts[i] = p.Name + "=" + it
		}
		return strings.Join(parts, "; "), nil
	}
	parts := make([]string, len(v.props))
	for i, pr := range v.props {
		parts[i] = pr.Key + "=" + pr.Value.(string)
	}
	return strings.Join(parts, "; "), nil
}
