package upgrade

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"

	"go.yaml.in/yaml/v3"
)

// DecodeYAML decodes one YAML document into generic JSON-compatible values:
// map[string]any, []any, string, bool, nil and the numeric types produced by
// yaml.v3. Unlike a plain Unmarshal it never produces map[any]any, and it
// keeps unquoted timestamps as their source text instead of a time.Time, so
// the result encodes to JSON without surprises.
func DecodeYAML(src []byte) (any, error) {
	var root yaml.Node
	if err := yaml.Unmarshal(src, &root); err != nil {
		return nil, fmt.Errorf("decode yaml: %w", err)
	}
	if root.Kind == 0 {
		return nil, errors.New("decode yaml: empty document")
	}
	d := &decoder{text: map[*yaml.Node]bool{}}
	d.markInfoVersion(&root)
	return d.fromNode(&root, 0)
}

// SourceScalar is the decoded value of a scalar whose exact YAML source text
// matters. DecodeYAML produces it only for a non-string info.version, so the
// info-version-string rule can keep "1.0" as "1.0" and not "1".
type SourceScalar struct {
	Value any
	Text  string
}

// MarshalJSON encodes the underlying value.
func (s SourceScalar) MarshalJSON() ([]byte, error) { return json.Marshal(s.Value) }

type decoder struct{ text map[*yaml.Node]bool }

// markInfoVersion records the root info.version node when it is a scalar
// that is not a string.
func (d *decoder) markInfoVersion(n *yaml.Node) {
	if n.Kind == yaml.DocumentNode && len(n.Content) == 1 {
		n = n.Content[0]
	}
	info := mapValue(n, "info")
	if info == nil {
		return
	}
	if v := mapValue(info, "version"); v != nil && v.Kind == yaml.ScalarNode && v.Tag != "!!str" && v.Tag != "!!null" {
		d.text[v] = true
	}
}

func mapValue(n *yaml.Node, key string) *yaml.Node {
	if n == nil || n.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(n.Content); i += 2 {
		if n.Content[i].Value == key {
			return n.Content[i+1]
		}
	}
	return nil
}

const maxAliasDepth = 64

func (d *decoder) fromNode(n *yaml.Node, depth int) (any, error) {
	if depth > maxAliasDepth {
		return nil, errors.New("yaml nesting or alias depth too large")
	}
	switch n.Kind {
	case yaml.DocumentNode:
		if len(n.Content) != 1 {
			return nil, errors.New("yaml document has no single root")
		}
		return d.fromNode(n.Content[0], depth+1)
	case yaml.AliasNode:
		return d.fromNode(n.Alias, depth+1)
	case yaml.SequenceNode:
		out := make([]any, 0, len(n.Content))
		for _, c := range n.Content {
			v, err := d.fromNode(c, depth+1)
			if err != nil {
				return nil, err
			}
			out = append(out, v)
		}
		return out, nil
	case yaml.MappingNode:
		out := make(map[string]any, len(n.Content)/2)
		for i := 0; i+1 < len(n.Content); i += 2 {
			k, v := n.Content[i], n.Content[i+1]
			if k.Tag == "!!merge" {
				mv, err := d.fromNode(v, depth+1)
				if err != nil {
					return nil, err
				}
				mergeInto(out, mv)
				continue
			}
			if k.Kind != yaml.ScalarNode {
				return nil, fmt.Errorf("yaml line %d: non-scalar mapping key", k.Line)
			}
			val, err := d.fromNode(v, depth+1)
			if err != nil {
				return nil, err
			}
			out[k.Value] = val
		}
		return out, nil
	case yaml.ScalarNode:
		v, err := scalar(n)
		if err == nil && d.text[n] {
			return SourceScalar{Value: v, Text: n.Value}, nil
		}
		return v, err
	}
	return nil, fmt.Errorf("yaml line %d: unsupported node kind", n.Line)
}

// mergeInto applies a "<<" merge: keys already present win.
func mergeInto(dst map[string]any, v any) {
	var srcs []map[string]any
	switch t := v.(type) {
	case map[string]any:
		srcs = append(srcs, t)
	case []any:
		for _, e := range t {
			if m, ok := e.(map[string]any); ok {
				srcs = append(srcs, m)
			}
		}
	}
	for _, s := range srcs {
		for k, val := range s {
			if _, ok := dst[k]; !ok {
				dst[k] = val
			}
		}
	}
}

func scalar(n *yaml.Node) (any, error) {
	switch n.Tag {
	case "!!null":
		return nil, nil
	case "!!bool", "!!int":
		var v any
		if err := n.Decode(&v); err != nil {
			return nil, fmt.Errorf("yaml line %d: %w", n.Line, err)
		}
		return v, nil
	case "!!float":
		var f float64
		if err := n.Decode(&f); err != nil {
			return nil, fmt.Errorf("yaml line %d: %w", n.Line, err)
		}
		if math.IsInf(f, 0) || math.IsNaN(f) {
			return nil, fmt.Errorf("yaml line %d: %s is not JSON-compatible", n.Line, n.Value)
		}
		return f, nil
	}
	// !!str, !!timestamp, !!binary and custom tags: the source text.
	return n.Value, nil
}

// EncodeJSON emits v as canonical indented JSON (two spaces, sorted object
// keys, no HTML escaping) with a trailing newline.
func EncodeJSON(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return nil, fmt.Errorf("encode json: %w", err)
	}
	return buf.Bytes(), nil
}
