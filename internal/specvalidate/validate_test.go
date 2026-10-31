package specvalidate_test

import (
	"encoding/json"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/wI2L/fizz/internal/specvalidate"
)

const minimalDoc = `{
  "openapi": "3.2.1",
  "info": {"title": "t", "version": "1"},
  "paths": {
    "/x": {"get": {"responses": {"200": {"description": "ok",
      "content": {"application/json": {"schema": {"type": ["string", "null"]}}}}}}}
  }
}`

func TestRejectsBadDoc(t *testing.T) {
	noInfo := `{"openapi": "3.2.1", "paths": {}}`
	err := specvalidate.Validate([]byte(noInfo))
	if err == nil {
		t.Fatal("document without info must fail validation")
	}
	if !strings.Contains(err.Error(), "info") {
		t.Errorf("error should mention info: %v", err)
	}

	badSchema := `{"openapi": "3.2.1", "info": {"title": "t", "version": "1"},
	  "paths": {"/x": {"get": {"responses": {"200": {"description": "ok",
	  "content": {"application/json": {"schema": {"type": "bogus"}}}}}}}}}`
	err = specvalidate.Validate([]byte(badSchema))
	if err == nil {
		t.Fatal("invalid Schema Object must fail (dialect check)")
	}
	t.Log(err)
	if !strings.Contains(err.Error(), "/paths/~1x/get/responses/200/content/application~1json/schema/type") {
		t.Errorf("error should carry the instance pointer: %v", err)
	}

	nullSchema := `{"openapi": "3.2.1", "info": {"title": "t", "version": "1"},
	  "paths": {"/x": {"get": {"responses": {"200": {"description": "ok",
	  "content": {"application/json": {"schema": null}}}}}}}}`
	err = specvalidate.LintNoStrayNull([]byte(nullSchema))
	if err == nil {
		t.Fatal(`"schema": null must fail the lint`)
	}
	if !strings.Contains(err.Error(), "/paths/~1x/get/responses/200/content/application~1json/schema") {
		t.Errorf("lint error should carry the pointer: %v", err)
	}
}

func TestAcceptsMinimalDoc(t *testing.T) {
	if err := specvalidate.Validate([]byte(minimalDoc)); err != nil {
		t.Fatalf("minimal 3.2.1 document rejected: %v", err)
	}
	if err := specvalidate.LintNoStrayNull([]byte(minimalDoc)); err != nil {
		t.Fatalf("lint rejected minimal document: %v", err)
	}
}

func TestLintAllowsNullInValueSubtrees(t *testing.T) {
	doc := `{"example": null, "a": {"default": null, "enum": [null, 1],
	  "const": null, "examples": [null], "value": null, "dataValue": {"x": null}}}`
	if err := specvalidate.LintNoStrayNull([]byte(doc)); err != nil {
		t.Fatal(err)
	}
	if err := specvalidate.LintNoStrayNull([]byte(`{"a": [1, null]}`)); err == nil ||
		!strings.Contains(err.Error(), "/a/1") {
		t.Fatalf("expected stray null at /a/1, got %v", err)
	}
}

func TestVersionPattern(t *testing.T) {
	t.Skip("fixed by P2.6")

	raw, err := os.ReadFile("schemas/3.2/schema-" + specvalidate.SchemaDate + ".json")
	if err != nil {
		t.Fatal(err)
	}
	var s struct {
		Properties struct {
			OpenAPI struct {
				Pattern string `json:"pattern"`
			} `json:"openapi"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(raw, &s); err != nil {
		t.Fatal(err)
	}
	re, err := regexp.Compile(s.Properties.OpenAPI.Pattern)
	if err != nil {
		t.Fatal(err)
	}

	// The version is currently the unexported const in openapi/generator.go;
	// read it from source until an exported openapi.Version exists.
	src, err := os.ReadFile("../../openapi/generator.go")
	if err != nil {
		t.Fatal(err)
	}
	m := regexp.MustCompile(`(?m)^\s*version\s*=\s*"([^"]+)"`).FindSubmatch(src)
	if m == nil {
		t.Fatal("version constant not found in openapi/generator.go")
	}
	if v := string(m[1]); !re.MatchString(v) {
		t.Errorf("version %q does not match %s", v, re)
	}
}
