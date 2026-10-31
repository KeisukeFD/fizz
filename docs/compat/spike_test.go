// Package compat holds the renderer compatibility spike (P0.6) and its
// validity test. It contains no production code.
package compat

import (
	"os"
	"strings"
	"testing"

	"github.com/wI2L/fizz/internal/specvalidate"
)

func TestSpikeDocValid(t *testing.T) {
	doc, err := os.ReadFile("spike/spec-3.2.1.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := specvalidate.Validate(doc); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if err := specvalidate.LintNoStrayNull(doc); err != nil {
		t.Fatalf("LintNoStrayNull: %v", err)
	}

	// The spike is only useful if it exercises the 3.1/3.2 features.
	// Whitespace is stripped so the check is formatting independent.
	compact := strings.NewReplacer(" ", "", "\n", "", "\t", "").Replace(string(doc))
	for _, needle := range []string{
		`"openapi":"3.2.1"`,
		`"$ref":"#/components/schemas/`,
		`"type":["string","null"]`,
		`"anyOf":[{"$ref":`,
		`{"type":"null"}`,
		`"style":"deepObject"`,
		`"explode":true`,
		`"additionalProperties":{"type":"string"}`,
		`"in":"cookie"`,
		`"contentEncoding":"base64"`,
		`"examples":[`,
		`"default":{`,
		`"summary":`,
		`"x-`,
	} {
		if !strings.Contains(compact, needle) {
			t.Errorf("spike doc missing feature %s", needle)
		}
	}
}
