package specvalidate

import (
	"bytes"
	"encoding/json"
	"fmt"
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
