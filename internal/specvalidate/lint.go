package specvalidate

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strings"
)

// nullAllowed lists the keys whose subtree may legitimately contain null.
var nullAllowed = map[string]bool{
	"example": true, "examples": true, "default": true, "const": true,
	"enum": true, "value": true, "dataValue": true,
}

// LintNoStrayNull fails on any JSON null in doc outside the subtrees of
// example, examples, default, const, enum, value and dataValue. The error
// lists the JSON pointer of every offending null.
func LintNoStrayNull(doc []byte) error {
	dec := json.NewDecoder(bytes.NewReader(doc))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return fmt.Errorf("specvalidate: invalid JSON: %w", err)
	}
	var found []string
	walk(v, "", &found)
	if len(found) > 0 {
		slices.Sort(found)
		return fmt.Errorf("stray null at: %s", strings.Join(found, ", "))
	}
	return nil
}

func walk(v any, ptr string, found *[]string) {
	switch t := v.(type) {
	case nil:
		if ptr == "" {
			ptr = "/"
		}
		*found = append(*found, ptr)
	case map[string]any:
		for k, c := range t {
			if nullAllowed[k] {
				continue
			}
			walk(c, ptr+"/"+escape(k), found)
		}
	case []any:
		for i, c := range t {
			walk(c, fmt.Sprintf("%s/%d", ptr, i), found)
		}
	}
}

func escape(k string) string {
	return strings.NewReplacer("~", "~0", "/", "~1").Replace(k)
}

// LintNoNullable fails on any `"nullable": true|false` keyword outside the
// subtrees of example, examples, default, const, enum, value and dataValue.
// `nullable` is an OAS 3.0 keyword that the 3.2 schema silently accepts as
// an unknown keyword, so only a lint can catch it; 3.2 expresses
// nullability as a type array containing "null". A property that is merely
// named "nullable" has an object value and is not reported.
func LintNoNullable(doc []byte) error {
	var v any
	if err := json.Unmarshal(doc, &v); err != nil {
		return fmt.Errorf("specvalidate: invalid JSON: %w", err)
	}
	var found []string
	walkNullable(v, "", &found)
	if len(found) > 0 {
		slices.Sort(found)
		return fmt.Errorf("3.0 nullable keyword at: %s", strings.Join(found, ", "))
	}
	return nil
}

func walkNullable(v any, ptr string, found *[]string) {
	switch t := v.(type) {
	case map[string]any:
		for k, c := range t {
			if nullAllowed[k] {
				continue
			}
			p := ptr + "/" + escape(k)
			if _, isBool := c.(bool); isBool && k == "nullable" {
				*found = append(*found, p)
				continue
			}
			walkNullable(c, p, found)
		}
	case []any:
		for i, c := range t {
			walkNullable(c, fmt.Sprintf("%s/%d", ptr, i), found)
		}
	}
}

// componentKeyRE is the key pattern the OAS 3.2 schema enforces on the
// Components Object maps.
var componentKeyRE = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)

// LintComponentNames fails on any key of a components map whose name does
// not match ^[a-zA-Z0-9._-]+$. The 3.2 schema already enforces this, but
// reports an empty instance pointer; this lint names the offending key.
func LintComponentNames(doc []byte) error {
	var d struct {
		Components map[string]json.RawMessage `json:"components"`
	}
	if err := json.Unmarshal(doc, &d); err != nil {
		return fmt.Errorf("specvalidate: invalid JSON: %w", err)
	}
	var bad []string
	for kind, raw := range d.Components {
		var m map[string]json.RawMessage
		if json.Unmarshal(raw, &m) != nil {
			continue // not a map (for example an extension value)
		}
		if strings.HasPrefix(kind, "x-") {
			continue
		}
		for name := range m {
			if !componentKeyRE.MatchString(name) {
				bad = append(bad, "/components/"+escape(kind)+"/"+escape(name))
			}
		}
	}
	if len(bad) > 0 {
		slices.Sort(bad)
		return fmt.Errorf("invalid component name at: %s", strings.Join(bad, ", "))
	}
	return nil
}
