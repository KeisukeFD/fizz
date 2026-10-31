package contract

import (
	"fmt"
	"maps"
	"math"
	"slices"
	"strconv"
	"strings"
)

// Doc is a served spec, decoded generically.
type Doc map[string]any

// Resolve follows the local $ref of m (any number of hops). A schema or
// parameter without $ref is returned as is.
func (d Doc) Resolve(m map[string]any) map[string]any {
	for range 16 {
		ref, ok := m["$ref"].(string)
		if !ok {
			return m
		}
		ptr, ok := strings.CutPrefix(ref, "#")
		if !ok {
			return m // external: not followed
		}
		var cur any = map[string]any(d)
		for seg := range strings.SplitSeq(strings.TrimPrefix(ptr, "/"), "/") {
			seg = strings.NewReplacer("~1", "/", "~0", "~").Replace(seg)
			next, ok := cur.(map[string]any)[seg]
			if !ok {
				return m
			}
			cur = next
		}
		r, ok := cur.(map[string]any)
		if !ok {
			return m
		}
		m = r
	}
	return m
}

// types returns the non-null types of a schema: "type" is a string or an
// array in 3.1+, and a nullable schema may also be an anyOf/oneOf with a
// null branch.
func types(s map[string]any) []string {
	var out []string
	switch t := s["type"].(type) {
	case string:
		out = append(out, t)
	case []any:
		for _, e := range t {
			if str, _ := e.(string); str != "" && str != "null" {
				out = append(out, str)
			}
		}
	}
	return out
}

// branch returns the schema to sample when s composes others: the first
// non-null branch of allOf, then oneOf, then anyOf.
func (d Doc) branch(s map[string]any) (map[string]any, bool) {
	for _, k := range []string{"allOf", "oneOf", "anyOf"} {
		list, _ := s[k].([]any)
		for _, e := range list {
			b, ok := e.(map[string]any)
			if !ok {
				continue
			}
			b = d.Resolve(b)
			if t, _ := b["type"].(string); t == "null" {
				continue
			}
			return b, true
		}
	}
	return nil, false
}

// ParamSample returns the sample of a Parameter Object, in the order
// of the spec's fields: the parameter's example, its examples, then the
// schema's (see Sample).
func (d Doc) ParamSample(param map[string]any) any {
	if v, ok := param["example"]; ok {
		return v
	}
	if ex, ok := param["examples"].(map[string]any); ok {
		for _, k := range slices.Sorted(maps.Keys(ex)) {
			e, _ := ex[k].(map[string]any)
			e = d.Resolve(e)
			for _, f := range []string{"dataValue", "value"} {
				if v, ok := e[f]; ok {
					return v
				}
			}
		}
	}
	s, _ := param["schema"].(map[string]any)
	return d.Sample(s)
}

// Sample returns a value for schema s: its example, else examples, else
// default, else the first enum value, else a value of its type (string "s1",
// integer 7, number 1.5, boolean true, arrays of two elements, objects built
// from the properties). A candidate that does not fit the type of the schema
// is skipped, so a wrong example or default does not hide the type value.
func (d Doc) Sample(s map[string]any) any { return d.sample(s, 0, 0) }

func (d Doc) sample(s map[string]any, i, depth int) any {
	s = d.Resolve(s)
	if s == nil || depth > 5 {
		return nil
	}
	if b, ok := d.branch(s); ok && len(types(s)) == 0 {
		return d.sample(b, i, depth+1)
	}
	if v, ok := s["example"]; ok && d.fits(s, v) {
		return v
	}
	if list, _ := s["examples"].([]any); len(list) > 0 && d.fits(s, list[0]) {
		return list[0]
	}
	if v, ok := s["default"]; ok && v != nil && d.fits(s, v) {
		return v
	}
	if list, _ := s["enum"].([]any); len(list) > 0 {
		for _, v := range list {
			if v != nil {
				return v
			}
		}
	}
	if v, ok := s["const"]; ok {
		return v
	}
	ts := types(s)
	if len(ts) == 0 {
		if s["properties"] != nil || s["additionalProperties"] != nil {
			ts = []string{"object"}
		} else if s["items"] != nil {
			ts = []string{"array"}
		} else {
			return "s1"
		}
	}
	switch ts[0] {
	case "string":
		switch s["format"] {
		case "date":
			return "2026-01-02"
		case "date-time":
			return "2026-01-02T03:04:05Z"
		case "uuid":
			return "123e4567-e89b-12d3-a456-426614174000"
		case "email":
			return "a@example.com"
		}
		return "s" + strconv.Itoa(i+1)
	case "integer":
		return clamp(s, float64(7+i), true)
	case "number":
		return clamp(s, 1.5+float64(i), false)
	case "boolean":
		return i%2 == 0
	case "array":
		n := 2
		if m, ok := s["minItems"].(float64); ok && int(m) > n {
			n = int(m)
		}
		if m, ok := s["maxItems"].(float64); ok && int(m) < n {
			n = int(m)
		}
		items, _ := s["items"].(map[string]any)
		out := make([]any, n)
		for k := range out {
			out[k] = d.sample(items, k, depth+1)
		}
		if list, _ := d.Resolve(items)["enum"].([]any); len(list) > 0 && n > len(list) {
			out = out[:len(list)]
			for k := range out {
				out[k] = list[k]
			}
		}
		return out
	case "object":
		out := map[string]any{}
		props, _ := s["properties"].(map[string]any)
		for _, k := range slices.Sorted(maps.Keys(props)) {
			ps, _ := props[k].(map[string]any)
			out[k] = d.sample(ps, 0, depth+1)
		}
		if ap, ok := s["additionalProperties"].(map[string]any); ok && len(props) == 0 {
			out["k1"] = d.sample(ap, 0, depth+1)
		}
		return out
	}
	return "s1"
}

// clamp keeps v within minimum and maximum.
func clamp(s map[string]any, v float64, integer bool) any {
	if m, ok := s["minimum"].(float64); ok && v < m {
		v = m
	}
	if m, ok := s["exclusiveMinimum"].(float64); ok && v <= m {
		v = m + 1
	}
	if m, ok := s["maximum"].(float64); ok && v > m {
		v = m
	}
	if m, ok := s["exclusiveMaximum"].(float64); ok && v >= m {
		v = m - 1
	}
	if integer {
		v = math.Trunc(v)
	}
	return v
}

// fits reports whether v is a possible value of s by type only.
func (d Doc) fits(s map[string]any, v any) bool {
	_, err := d.Coerce(s, v)
	return err == nil
}

// Coerce converts v into the JSON data model value that a server bound to
// the Go type documented by s produces once it echoes it: integers and
// numbers become float64, objects and arrays are converted member by
// member. It fails when v cannot be a value of the documented type. For
// example a string where s says array, which is a documented default that a
// generated client could not use.
func (d Doc) Coerce(s map[string]any, v any) (any, error) {
	return d.coerce(s, v, 0)
}

func (d Doc) coerce(s map[string]any, v any, depth int) (any, error) {
	s = d.Resolve(s)
	if v == nil {
		return nil, nil
	}
	if depth > 6 {
		return v, nil
	}
	ts := types(s)
	if len(ts) == 0 {
		if b, ok := d.branch(s); ok {
			return d.coerce(b, v, depth+1)
		}
		return v, nil
	}
	var firstErr error
	for _, t := range ts {
		out, err := d.coerceAs(s, t, v, depth)
		if err == nil {
			return out, nil
		}
		if firstErr == nil {
			firstErr = err
		}
	}
	return nil, firstErr
}

func (d Doc) coerceAs(s map[string]any, t string, v any, depth int) (any, error) {
	switch t {
	case "string":
		if x, ok := v.(string); ok {
			return x, nil
		}
	case "integer", "number":
		switch x := v.(type) {
		case float64:
			if t == "integer" && x != math.Trunc(x) {
				return nil, fmt.Errorf("%v is not an integer", v)
			}
			return x, nil
		case int:
			return float64(x), nil
		case string:
			if f, err := strconv.ParseFloat(x, 64); err == nil {
				return f, nil
			}
		}
	case "boolean":
		if x, ok := v.(bool); ok {
			return x, nil
		}
	case "array":
		list, ok := v.([]any)
		if !ok {
			break
		}
		items, _ := s["items"].(map[string]any)
		out := make([]any, len(list))
		for i, e := range list {
			c, err := d.coerce(items, e, depth+1)
			if err != nil {
				return nil, fmt.Errorf("item %d: %w", i, err)
			}
			out[i] = c
		}
		return out, nil
	case "object":
		m, ok := v.(map[string]any)
		if !ok {
			break
		}
		props, _ := s["properties"].(map[string]any)
		ap, _ := s["additionalProperties"].(map[string]any)
		out := make(map[string]any, len(m))
		for k, e := range m {
			ps, ok := props[k].(map[string]any)
			if !ok {
				ps = ap
			}
			c, err := d.coerce(ps, e, depth+1)
			if err != nil {
				return nil, fmt.Errorf("property %q: %w", k, err)
			}
			out[k] = c
		}
		return out, nil
	}
	return nil, fmt.Errorf("%#v is not a %s", v, t)
}
