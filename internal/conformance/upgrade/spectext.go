package upgrade

import (
	"bufio"
	"bytes"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"
)

// Block is one ```yaml fenced block of the specification text.
type Block struct {
	Name    string // stable file stem: heading anchor plus per-anchor index
	Heading string // text of the nearest heading above the block
	Line    int    // 1-based line of the opening fence
	Text    string // the block body
}

// ExtractBlocks returns the ```yaml fenced blocks of a Markdown document in
// source order. Headings are only recognised outside fences, so "#" comment
// lines inside code blocks are not mistaken for sections. Block names are
// "<anchor>-<n>" where anchor is the GitHub-style slug of the nearest
// heading and n counts every block of that anchor across the document.
func ExtractBlocks(md []byte) []Block {
	var (
		blocks  []Block
		heading = "preamble"
		counts  = map[string]int{}
		inFence bool
		isYAML  bool
		cur     Block
		body    strings.Builder
	)
	sc := bufio.NewScanner(bytes.NewReader(md))
	sc.Buffer(make([]byte, 0, 64*1024), 4<<20)
	for line := 1; sc.Scan(); line++ {
		text := sc.Text()
		trimmed := strings.TrimSpace(text)
		if strings.HasPrefix(trimmed, "```") {
			if !inFence {
				inFence = true
				lang := strings.TrimSpace(strings.TrimPrefix(trimmed, "```"))
				isYAML = lang == "yaml"
				if isYAML {
					slug := Slug(heading)
					counts[slug]++
					cur = Block{Name: slug + "-" + strconv.Itoa(counts[slug]), Heading: heading, Line: line}
					body.Reset()
				}
			} else {
				if isYAML {
					cur.Text = body.String()
					blocks = append(blocks, cur)
				}
				inFence, isYAML = false, false
			}
			continue
		}
		if inFence {
			if isYAML {
				body.WriteString(text)
				body.WriteByte('\n')
			}
			continue
		}
		if h, ok := headingText(text); ok {
			heading = h
		}
	}
	return blocks
}

func headingText(line string) (string, bool) {
	n := 0
	for n < len(line) && line[n] == '#' {
		n++
	}
	if n == 0 || n > 6 || len(line) <= n || line[n] != ' ' {
		return "", false
	}
	return strings.TrimSpace(line[n:]), true
}

// Slug converts heading text to a GitHub-style anchor: lower case, spaces to
// hyphens, everything but letters, digits, hyphens and underscores dropped.
func Slug(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_':
			b.WriteRune(r)
		case r == ' ':
			b.WriteByte('-')
		}
	}
	if b.Len() == 0 {
		return "section"
	}
	return b.String()
}

// Wrapped is a fragment placed in a minimal 3.2.1 document. Only the wrapper
// is synthesized; the fragment content is used as decoded.
type Wrapped struct {
	Kind string         // what the fragment is, e.g. "components/schemas"
	Doc  map[string]any // the complete document
}

// Wrap classifies the decoded block v by its shape and the heading it sits
// under, and wraps it into a minimal OpenAPI 3.2.1 document. A block that
// already has an "openapi" key is returned as is (Kind "document"). When the
// block cannot be classified, Wrap returns a non-empty reason instead.
func Wrap(heading, title string, v any) (Wrapped, string) {
	m, ok := v.(map[string]any)
	if !ok {
		return Wrapped{}, "fragment of an enclosing object (not a mapping)"
	}
	if _, ok := m["openapi"]; ok {
		return Wrapped{Kind: "document", Doc: m}, ""
	}
	h := strings.ToLower(heading)
	has := func(keys ...string) bool {
		for _, k := range keys {
			if _, ok := m[k]; !ok {
				return false
			}
		}
		return true
	}
	hasAny := func(keys ...string) bool {
		for _, k := range keys {
			if _, ok := m[k]; ok {
				return true
			}
		}
		return false
	}
	subset := func(keys ...string) bool {
		if len(m) == 0 {
			return false
		}
		for k := range m {
			found := false
			for _, ok := range keys {
				if k == ok {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
		return true
	}
	inHeading := func(s string) bool { return strings.Contains(h, s) }

	doc := func() map[string]any {
		return map[string]any{
			"openapi": TargetVersion,
			"info":    map[string]any{"title": title, "version": "1.0.0"},
		}
	}
	component := func(kind string) (Wrapped, string) {
		d := doc()
		d["components"] = map[string]any{kind: map[string]any{"Example": m}}
		return Wrapped{Kind: "components/" + kind, Doc: d}, ""
	}
	path := func(kind string, item any) (Wrapped, string) {
		d := doc()
		d["paths"] = map[string]any{pathTemplate(m, item): item}
		return Wrapped{Kind: kind, Doc: d}, ""
	}
	rootField := func(field string, val any) (Wrapped, string) {
		d := doc()
		d[field] = val
		d["paths"] = map[string]any{}
		return Wrapped{Kind: field, Doc: d}, ""
	}

	switch {
	case subset("$self", "components", "externalDocs", "info", "jsonSchemaDialect",
		"paths", "security", "servers", "tags", "webhooks"):
		d := doc()
		for k, val := range m {
			d[k] = val
		}
		if !hasAny("paths", "components", "webhooks") {
			d["paths"] = map[string]any{}
		}
		return Wrapped{Kind: "document-fields", Doc: d}, ""
	case allKeys(m, func(k string) bool { return strings.HasPrefix(k, "/") }):
		d := doc()
		d["paths"] = m
		return Wrapped{Kind: "paths", Doc: d}, ""
	case inHeading("contact") && subset("name", "url", "email"):
		d := doc()
		d["info"] = map[string]any{"title": title, "version": "1.0.0", "contact": m}
		d["paths"] = map[string]any{}
		return Wrapped{Kind: "info/contact", Doc: d}, ""
	case inHeading("license") && subset("name", "identifier", "url"):
		d := doc()
		d["info"] = map[string]any{"title": title, "version": "1.0.0", "license": m}
		d["paths"] = map[string]any{}
		return Wrapped{Kind: "info/license", Doc: d}, ""
	case inHeading("external documentation") && subset("description", "url"):
		return rootField("externalDocs", m)
	case inHeading("server") && has("url") && subset("url", "description", "name", "variables"):
		return rootField("servers", []any{m})
	case has("title", "version"):
		d := doc()
		d["info"] = m
		d["paths"] = map[string]any{}
		return Wrapped{Kind: "info", Doc: d}, ""
	case inHeading("tag") && has("name") && subset("name", "summary", "description", "externalDocs", "parent", "kind"):
		return rootField("tags", []any{m})
	case isSecurityScheme(m):
		return component("securitySchemes")
	case isSchema(m):
		return component("schemas")
	case has("name", "in"):
		return component("parameters")
	case inHeading("link") && hasAny("operationId", "operationRef"):
		return component("links")
	case hasAny("additionalOperations") || hasAny(methods...):
		return path("paths/path-item", m)
	case hasAny("operationId", "responses", "requestBody", "callbacks"):
		return path("paths/operation", map[string]any{"get": m})
	case allKeys(m, isStatusCode):
		return path("paths/operation/responses", map[string]any{"get": map[string]any{"responses": m}})
	case allKeys(m, isMediaType):
		d := doc()
		d["components"] = map[string]any{"requestBodies": map[string]any{"Example": map[string]any{"content": m}}}
		return Wrapped{Kind: "components/requestBodies (content)", Doc: d}, ""
	case subset("content") || (subset("description", "content", "required") && (has("required") || inHeading("request body"))):
		return component("requestBodies")
	case inHeading("response") && subset("description", "content", "headers", "links"):
		return component("responses")
	case subset("parameters"):
		return path("paths/path-item (parameters)", m)
	case subset("links"):
		d := doc()
		d["components"] = map[string]any{"responses": map[string]any{"Example": map[string]any{"description": "Example response.", "links": m["links"]}}}
		return Wrapped{Kind: "components/responses (links)", Doc: d}, ""
	case inHeading("security requirement"):
		return rootField("security", []any{m})
	case inHeading("header") && isNamedMap(m, func(v map[string]any) bool { _, ok := v["schema"]; return ok }):
		return named(doc(), "headers", m)
	case inHeading("callback") && isNamedMap(m, func(map[string]any) bool { return true }):
		return named(doc(), "callbacks", m)
	case isNamedMap(m, isSchema):
		return named(doc(), "schemas", m)
	case inHeading("example") && subset("summary", "description", "value", "dataValue", "serializedValue", "externalValue") &&
		hasAny("value", "dataValue", "serializedValue", "externalValue"):
		return component("examples")
	case inHeading("header") && subset("description", "required", "deprecated", "schema", "content", "style", "explode", "example", "examples") && hasAny("schema", "content"):
		return component("headers")
	case subset("schema", "itemSchema", "example", "examples", "encoding", "itemEncoding", "prefixEncoding") && hasAny("schema", "itemSchema", "examples", "example"):
		return component("mediaTypes")
	}
	if isExampleData(m) {
		return Wrapped{}, "example data (a value used in prose), not an OpenAPI object"
	}
	return Wrapped{}, "fragment of an enclosing object"
}

// pathTemplate returns the synthesized path for a Path Item or Operation
// fragment: "/example" plus a "/{name}" segment for each distinct in:path
// parameter the fragment declares, in declaration order. Parameters are read
// inline or through a "#/components/parameters/<name>" reference resolved
// against the fragment's own components. The fragment is never modified.
func pathTemplate(frag map[string]any, item any) string {
	var names []string
	add := func(params any) {
		list, _ := params.([]any)
		for _, p := range list {
			pm, ok := p.(map[string]any)
			if !ok {
				continue
			}
			if ref, ok := pm["$ref"].(string); ok {
				key, found := strings.CutPrefix(ref, "#/components/parameters/")
				if !found {
					continue
				}
				comps, _ := frag["components"].(map[string]any)
				all, _ := comps["parameters"].(map[string]any)
				pm, ok = all[key].(map[string]any)
				if !ok {
					continue
				}
			}
			if in, _ := pm["in"].(string); in != "path" {
				continue
			}
			if name, ok := pm["name"].(string); ok && !slices.Contains(names, name) {
				names = append(names, name)
			}
		}
	}
	var walk func(v any, depth int)
	walk = func(v any, depth int) {
		im, ok := v.(map[string]any)
		if !ok {
			return
		}
		add(im["parameters"])
		if depth > 0 {
			return
		}
		for _, k := range methods {
			walk(im[k], 1)
		}
		if ops, ok := im["additionalOperations"].(map[string]any); ok {
			for _, op := range ops {
				walk(op, 1)
			}
		}
	}
	walk(item, 0)
	path := "/example"
	for _, n := range names {
		path += "/{" + n + "}"
	}
	return path
}

func named(d map[string]any, kind string, m map[string]any) (Wrapped, string) {
	d["components"] = map[string]any{kind: m}
	return Wrapped{Kind: "components/" + kind + " (named map)", Doc: d}, ""
}

func allKeys(m map[string]any, pred func(string) bool) bool {
	if len(m) == 0 {
		return false
	}
	for k := range m {
		if !pred(k) {
			return false
		}
	}
	return true
}

func isNamedMap(m map[string]any, pred func(map[string]any) bool) bool {
	if len(m) == 0 {
		return false
	}
	for _, v := range m {
		vm, ok := v.(map[string]any)
		if !ok || !pred(vm) {
			return false
		}
	}
	return true
}

var statusCode = regexp.MustCompile(`^(default|[1-5]([0-9]{2}|XX))$`)

func isStatusCode(k string) bool { return statusCode.MatchString(k) }

func isMediaType(k string) bool {
	return strings.Contains(k, "/") && !strings.HasPrefix(k, "/")
}

func isSecurityScheme(m map[string]any) bool {
	t, ok := m["type"].(string)
	if !ok {
		return false
	}
	switch t {
	case "http", "apiKey", "mutualTLS", "oauth2", "openIdConnect":
		return true
	}
	return false
}

var schemaKeywords = []string{
	"$comment", "$defs", "$id", "$ref", "$schema", "additionalProperties", "allOf",
	"anyOf", "const", "discriminator", "else", "enum", "format", "if", "items",
	"maxItems", "maxLength", "maximum", "minItems", "minLength", "minimum", "not",
	"oneOf", "pattern", "patternProperties", "prefixItems", "properties", "then",
	"type", "xml",
}

func isSchema(m map[string]any) bool {
	for _, k := range schemaKeywords {
		if _, ok := m[k]; ok {
			return true
		}
	}
	return false
}

// isExampleData reports whether m looks like plain data (for instance a
// serialization sample) rather than an OpenAPI fragment: none of its keys is
// an OpenAPI field or schema keyword.
func isExampleData(m map[string]any) bool {
	for k := range m {
		if slices.Contains(oasFields, k) || slices.Contains(schemaKeywords, k) {
			return false
		}
	}
	return true
}

// oasFields are common OpenAPI field names, used only to tell plain data
// from an OpenAPI fragment.
var oasFields = []string{
	"callbacks", "components", "content", "description", "dataValue", "encoding",
	"example", "examples", "externalDocs", "headers", "in", "info", "links", "name",
	"openapi", "operationId", "parameters", "paths", "requestBody", "required",
	"responses", "schema", "security", "servers", "summary", "tags", "value",
}

// BlockTitle returns the info.title used for a wrapper.
func BlockTitle(name string) string { return fmt.Sprintf("spec text %s", name) }
