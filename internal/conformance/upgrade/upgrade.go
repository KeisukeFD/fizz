// Package upgrade converts official OpenAPI example documents to OAS 3.2.1
// and extracts the YAML examples of the specification text.
//
// The converter works on generic decoded JSON (map[string]any and []any). It
// follows the OAI 3.0 to 3.1 migration guidance, applied only inside Schema
// Objects found by a structural walk of the document, then sets the version
// to 3.2.1 (3.2 is a superset of 3.1, so nothing else changes). Every
// modification is recorded as a Change, in a deterministic order.
package upgrade

import (
	"errors"
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/KeisukeFD/fizz/internal/specvalidate"
)

// TargetVersion is the OpenAPI version every upgraded document carries.
const TargetVersion = "3.2.1"

// Change is one modification made by the converter. Before and After are
// omitted from the JSON form when the value was absent.
type Change struct {
	Pointer string `json:"pointer"` // RFC 6901 pointer of the changed location
	Rule    string `json:"rule"`
	Before  any    `json:"before,omitempty"`
	After   any    `json:"after,omitempty"`
}

// Upgrade converts doc, a decoded OpenAPI document, to 3.2.1 in place and
// returns it with the ordered list of changes. A document whose "openapi"
// field is missing or not a "3.x" string only gets schema-independent
// treatment (none), so malformed inputs pass through untouched.
func Upgrade(doc any) (any, []Change, error) {
	m, ok := doc.(map[string]any)
	if !ok {
		return nil, nil, errors.New("upgrade: document is not an object")
	}
	w := &walker{}
	if ver, ok := m["openapi"].(string); ok && strings.HasPrefix(ver, "3.") {
		w.v30 = ver == "3.0" || strings.HasPrefix(ver, "3.0.")
		if ver != TargetVersion {
			m["openapi"] = TargetVersion
			w.record([]string{"openapi"}, "openapi-version", ver, TargetVersion)
		}
	}
	w.infoVersion(m)
	w.dialect(m, []string{"jsonSchemaDialect"}, "jsonSchemaDialect")
	w.document(m)
	return m, w.changes, nil
}

type walker struct {
	v30     bool // apply the 3.0 -> 3.1 schema rules
	changes []Change
}

func (w *walker) record(path []string, rule string, before, after any) {
	w.changes = append(w.changes, Change{Pointer: Pointer(path), Rule: rule, Before: before, After: after})
}

// Pointer renders tokens as an RFC 6901 JSON pointer.
func Pointer(tokens []string) string {
	var b strings.Builder
	for _, t := range tokens {
		b.WriteByte('/')
		t = strings.ReplaceAll(t, "~", "~0")
		b.WriteString(strings.ReplaceAll(t, "/", "~1"))
	}
	return b.String()
}

func with(path []string, tok ...string) []string {
	out := make([]string, 0, len(path)+len(tok))
	out = append(out, path...)
	return append(out, tok...)
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// each calls fn for every non-extension entry of the object v, in key order.
func each(v any, fn func(key string, val any)) {
	m, ok := v.(map[string]any)
	if !ok {
		return
	}
	for _, k := range sortedKeys(m) {
		if strings.HasPrefix(k, "x-") {
			continue
		}
		fn(k, m[k])
	}
}

// infoVersion: info.version must be a string. An unquoted YAML scalar such as
// 1.0 is made a string holding its exact source text.
func (w *walker) infoVersion(m map[string]any) {
	info, ok := m["info"].(map[string]any)
	if !ok {
		return
	}
	var text string
	switch v := info["version"].(type) {
	case SourceScalar:
		text = v.Text
	case int, int64, uint64, bool:
		text = fmt.Sprint(v)
	case float64:
		text = strconv.FormatFloat(v, 'f', -1, 64)
	default:
		return
	}
	before := info["version"]
	if ss, ok := before.(SourceScalar); ok {
		before = ss.Value
	}
	info["version"] = text
	w.record([]string{"info", "version"}, "info-version-string", before, text)
}

const dialect31Prefix = "https://spec.openapis.org/oas/3.1/dialect/"

// dialect: a 3.1 OAS dialect URI in jsonSchemaDialect or a Schema Object's
// $schema becomes the 3.2 dialect URI. Other values are left alone.
func (w *walker) dialect(m map[string]any, path []string, key string) {
	s, ok := m[key].(string)
	if !ok || !strings.HasPrefix(s, dialect31Prefix) {
		return
	}
	m[key] = specvalidate.DialectURL
	w.record(with(path[:len(path)-1], key), "dialect-uri", s, specvalidate.DialectURL)
}

func (w *walker) document(m map[string]any) {
	if c, ok := m["components"].(map[string]any); ok {
		w.components(c, []string{"components"})
	}
	w.pathItems(m["paths"], []string{"paths"})
	w.pathItems(m["webhooks"], []string{"webhooks"})
}

func (w *walker) components(c map[string]any, path []string) {
	sub := func(name string, fn func(any, []string)) {
		each(c[name], func(k string, v any) { fn(v, with(path, name, k)) })
	}
	sub("callbacks", w.callback)
	sub("headers", w.header)
	sub("mediaTypes", func(v any, p []string) { w.mediaType("", v, p) })
	sub("parameters", w.parameter)
	sub("pathItems", w.pathItem)
	sub("requestBodies", w.requestBody)
	sub("responses", w.response)
	sub("schemas", func(v any, p []string) { w.schema(v, p, false) })
}

func (w *walker) pathItems(v any, path []string) {
	each(v, func(k string, item any) { w.pathItem(item, with(path, k)) })
}

var methods = []string{"delete", "get", "head", "options", "patch", "post", "put", "query", "trace"}

func (w *walker) pathItem(v any, path []string) {
	m, ok := v.(map[string]any)
	if !ok {
		return
	}
	w.parameters(m["parameters"], with(path, "parameters"))
	for _, name := range methods {
		if op, ok := m[name]; ok {
			w.operation(op, with(path, name))
		}
	}
	each(m["additionalOperations"], func(k string, op any) {
		w.operation(op, with(path, "additionalOperations", k))
	})
}

func (w *walker) parameters(v any, path []string) {
	list, ok := v.([]any)
	if !ok {
		return
	}
	for i, p := range list {
		w.parameter(p, with(path, itoa(i)))
	}
}

func (w *walker) operation(v any, path []string) {
	m, ok := v.(map[string]any)
	if !ok {
		return
	}
	w.parameters(m["parameters"], with(path, "parameters"))
	w.requestBody(m["requestBody"], with(path, "requestBody"))
	each(m["responses"], func(k string, r any) { w.response(r, with(path, "responses", k)) })
	each(m["callbacks"], func(k string, cb any) { w.callback(cb, with(path, "callbacks", k)) })
}

func (w *walker) callback(v any, path []string) {
	each(v, func(k string, item any) { w.pathItem(item, with(path, k)) })
}

func (w *walker) parameter(v any, path []string) {
	m, ok := v.(map[string]any)
	if !ok {
		return
	}
	w.schema(m["schema"], with(path, "schema"), false)
	w.content(m["content"], with(path, "content"))
}

func (w *walker) header(v any, path []string) { w.parameter(v, path) }

func (w *walker) requestBody(v any, path []string) {
	if m, ok := v.(map[string]any); ok {
		w.content(m["content"], with(path, "content"))
	}
}

func (w *walker) response(v any, path []string) {
	m, ok := v.(map[string]any)
	if !ok {
		return
	}
	each(m["headers"], func(k string, h any) { w.header(h, with(path, "headers", k)) })
	w.content(m["content"], with(path, "content"))
}

func (w *walker) content(v any, path []string) {
	each(v, func(k string, mt any) { w.mediaType(k, mt, with(path, k)) })
}

// mediaType walks a Media Type Object. key is its media type name (empty when
// unknown); form bodies are the only place the file-field rules apply.
func (w *walker) mediaType(key string, v any, path []string) {
	m, ok := v.(map[string]any)
	if !ok {
		return
	}
	lk := strings.ToLower(key)
	form := lk == "application/x-www-form-urlencoded" || strings.HasPrefix(lk, "multipart/")
	w.schema(m["schema"], with(path, "schema"), form)
	w.schema(m["itemSchema"], with(path, "itemSchema"), form)
	each(m["encoding"], func(k string, enc any) {
		if em, ok := enc.(map[string]any); ok {
			each(em["headers"], func(hk string, h any) {
				w.header(h, with(path, "encoding", k, "headers", hk))
			})
		}
	})
}

type kind int

const (
	single kind = iota
	list
	object
	singleOrList
)

// subschemas lists, alphabetically, where a Schema Object holds schemas.
var subschemas = []struct {
	key  string
	kind kind
}{
	{"$defs", object}, {"additionalProperties", single}, {"allOf", list},
	{"anyOf", list}, {"contains", single}, {"contentSchema", single},
	{"definitions", object}, {"dependentSchemas", object}, {"else", single},
	{"if", single}, {"items", singleOrList}, {"not", single}, {"oneOf", list},
	{"patternProperties", object}, {"prefixItems", list},
	{"propertyNames", single}, {"properties", object}, {"then", single},
	{"unevaluatedItems", single}, {"unevaluatedProperties", single},
}

func (w *walker) schema(v any, path []string, form bool) {
	m, ok := v.(map[string]any)
	if !ok {
		return
	}
	w.dialect(m, with(path, "$schema"), "$schema")
	if w.v30 {
		w.nullable(m, path)
		w.exclusive(m, path, "exclusiveMinimum", "minimum")
		w.exclusive(m, path, "exclusiveMaximum", "maximum")
		if form {
			w.fileFields(m, path)
		}
	}
	for _, s := range subschemas {
		child, ok := m[s.key]
		if !ok {
			continue
		}
		p := with(path, s.key)
		switch s.kind {
		case single:
			w.schema(child, p, form)
		case list:
			w.schemaList(child, p, form)
		case singleOrList:
			if _, isList := child.([]any); isList {
				w.schemaList(child, p, form)
			} else {
				w.schema(child, p, form)
			}
		case object:
			if cm, ok := child.(map[string]any); ok {
				for _, k := range sortedKeys(cm) {
					w.schema(cm[k], with(p, k), form)
				}
			}
		}
	}
}

func (w *walker) schemaList(v any, path []string, form bool) {
	l, ok := v.([]any)
	if !ok {
		return
	}
	for i, s := range l {
		w.schema(s, with(path, itoa(i)), form)
	}
}

// nullable: "nullable: true" becomes a "null" entry in type (and enum);
// "nullable: false" is dropped. Without a type, 3.0 gives nullable no effect,
// so the keyword is only dropped.
func (w *walker) nullable(m map[string]any, path []string) {
	nv, has := m["nullable"]
	if !has {
		return
	}
	b, isBool := nv.(bool)
	if !isBool {
		return
	}
	np := with(path, "nullable")
	if !b {
		delete(m, "nullable")
		w.record(np, "nullable-false-removed", false, nil)
		return
	}
	switch t := m["type"].(type) {
	case string:
		after := []any{t, "null"}
		m["type"] = after
		w.record(with(path, "type"), "nullable-to-type-null", t, after)
	case []any:
		if !slices.Contains(t, any("null")) {
			after := append(slices.Clone(t), "null")
			m["type"] = after
			w.record(with(path, "type"), "nullable-to-type-null", t, after)
		}
	default:
		delete(m, "nullable")
		w.record(np, "nullable-without-type-removed", true, nil)
		return
	}
	if enum, ok := m["enum"].([]any); ok && !slices.Contains(enum, nil) {
		after := append(slices.Clone(enum), nil)
		m["enum"] = after
		w.record(with(path, "enum"), "nullable-enum-add-null", enum, after)
	}
	delete(m, "nullable")
	w.record(np, "nullable-removed", true, nil)
}

// exclusive: boolean exclusiveMinimum/Maximum next to minimum/maximum becomes
// the numeric 3.1 form. "true" without a numeric bound is left untouched.
func (w *walker) exclusive(m map[string]any, path []string, excl, bound string) {
	b, isBool := m[excl].(bool)
	if !isBool {
		return
	}
	ep := with(path, excl)
	if !b {
		delete(m, excl)
		w.record(ep, "exclusive-bound-false-removed", false, nil)
		return
	}
	bv, has := m[bound]
	if !has || !isNumber(bv) {
		return
	}
	m[excl] = bv
	delete(m, bound)
	w.record(ep, "exclusive-bound-to-number", true, bv)
	w.record(with(path, bound), "exclusive-bound-inclusive-removed", bv, nil)
}

func isNumber(v any) bool {
	switch v.(type) {
	case int, int64, uint64, float64:
		return true
	}
	return false
}

// fileFields: in form bodies, "type: string" with "format: binary" gets
// contentMediaType application/octet-stream and "format: base64" gets
// contentEncoding base64. type stays "string".
func (w *walker) fileFields(m map[string]any, path []string) {
	if m["type"] != "string" {
		return
	}
	var rmRule, addRule, addKey, addVal string
	switch m["format"] {
	case "binary":
		rmRule, addRule, addKey, addVal = "binary-format-removed", "binary-to-contentMediaType", "contentMediaType", "application/octet-stream"
	case "base64":
		rmRule, addRule, addKey, addVal = "base64-format-removed", "base64-to-contentEncoding", "contentEncoding", "base64"
	default:
		return
	}
	before := m["format"]
	delete(m, "format")
	w.record(with(path, "format"), rmRule, before, nil)
	if _, exists := m[addKey]; !exists {
		m[addKey] = addVal
		w.record(with(path, addKey), addRule, nil, addVal)
	}
}

func itoa(i int) string { return strconv.Itoa(i) }
