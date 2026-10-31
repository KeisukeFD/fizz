package specvalidate

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"strings"
)

// Problem is one semantic rule violation: the JSON pointer of the
// offending value in the document and a human-readable message.
type Problem struct {
	Pointer string
	Message string
}

func (p Problem) String() string { return p.Pointer + ": " + p.Message }

var (
	templateVarRE = regexp.MustCompile(`\{([^{}]*)\}`)
	httpMethods   = []string{"get", "put", "post", "delete", "options", "head", "patch", "trace", "query"}
	validIn       = []string{"query", "querystring", "header", "path", "cookie"}
	reservedHdrs  = []string{"accept", "content-type", "authorization"}
)

// LintSemantics checks the OAS 3.2.1 MUST-rules that the JSON Schema cannot
// express. It reports, with the section of the specification (versions/3.2.1.md):
//
//   - Path Templating: every template expression has exactly one path
//     parameter (path item level plus operation level, the operation
//     overriding), in: path parameters are required: true and appear in the
//     template, and an expression appears once per template.
//   - Parameter Object: (name, in) unique per list, exactly one of schema and
//     content, content has one entry, a valid in, reserved header names, and
//     the querystring rules.
//   - Operation Object: operationId unique across the document, including
//     callbacks and webhooks.
//   - Paths Object: templated paths that differ only in variable names.
//   - Reference Object: every local $ref (and plain-pointer $dynamicRef)
//     resolves within the document.
//
// References to #/components/parameters/ are resolved locally for the
// parameter rules; any other reference is skipped by them. Problems are ordered by traversal (sorted keys).
// An unparsable document yields a single problem with pointer "".
func LintSemantics(doc []byte) []Problem {
	var root map[string]any
	if err := json.Unmarshal(doc, &root); err != nil {
		return []Problem{{Message: "invalid JSON object: " + err.Error()}}
	}
	l := &semLinter{root: root, opIDs: map[string]string{}}
	l.run()
	// A path item parameter can be reported once per operation.
	seen := map[Problem]bool{}
	var out []Problem
	for _, p := range l.problems {
		if !seen[p] {
			seen[p] = true
			out = append(out, p)
		}
	}
	return out
}

type semLinter struct {
	root     map[string]any
	opIDs    map[string]string // operationId -> pointer of first use
	problems []Problem
}

func (l *semLinter) add(ptr, format string, args ...any) {
	l.problems = append(l.problems, Problem{Pointer: ptr, Message: fmt.Sprintf(format, args...)})
}

func obj(v any) map[string]any { m, _ := v.(map[string]any); return m }

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

func (l *semLinter) run() {
	l.refs(l.root, "", false)
	if paths := obj(l.root["paths"]); paths != nil {
		l.paths(paths)
	}
	if wh := obj(l.root["webhooks"]); wh != nil {
		for _, k := range sortedKeys(wh) {
			if strings.HasPrefix(k, "x-") {
				continue
			}
			if item := obj(wh[k]); item != nil {
				l.pathItem("/webhooks/"+escape(k), item, "", false)
			}
		}
	}
	comps := obj(l.root["components"])
	if comps == nil {
		return
	}
	if pi := obj(comps["pathItems"]); pi != nil {
		for _, k := range sortedKeys(pi) {
			if item := obj(pi[k]); item != nil {
				l.pathItem("/components/pathItems/"+escape(k), item, "", false)
			}
		}
	}
	if cb := obj(comps["callbacks"]); cb != nil {
		for _, k := range sortedKeys(cb) {
			l.callback("/components/callbacks/"+escape(k), obj(cb[k]))
		}
	}
	if ps := obj(comps["parameters"]); ps != nil {
		for _, k := range sortedKeys(ps) {
			if p := obj(ps[k]); p != nil && p["$ref"] == nil {
				l.checkParam("/components/parameters/"+escape(k), p)
			}
		}
	}
}

// paths applies the Paths Object rule on colliding templates, then walks
// every path item.
func (l *semLinter) paths(paths map[string]any) {
	seen := map[string]string{} // normalized template -> first path
	for _, k := range sortedKeys(paths) {
		if !strings.HasPrefix(k, "/") {
			continue // specification extension
		}
		// Paths Object, /{path}: "Templated paths with the same hierarchy
		// but different templated names MUST NOT exist as they are
		// identical." Path Templating Matching lists /pets/{petId} and
		// /pets/{name} as "identical and invalid".
		norm := templateVarRE.ReplaceAllString(k, "{}")
		if first, ok := seen[norm]; ok {
			l.add("/paths/"+escape(k), "path template is identical to %q: templates that differ only in variable names MUST NOT both exist", first)
		} else {
			seen[norm] = k
		}
		if item := obj(paths[k]); item != nil {
			l.pathItem("/paths/"+escape(k), item, k, true)
		}
	}
}

// callback walks a Callback Object: a map of runtime expressions to path
// items. A callback that is a $ref is skipped.
func (l *semLinter) callback(ptr string, cb map[string]any) {
	for _, k := range sortedKeys(cb) {
		if k == "$ref" || strings.HasPrefix(k, "x-") {
			continue
		}
		if item := obj(cb[k]); item != nil {
			l.pathItem(ptr+"/"+escape(k), item, "", false)
		}
	}
}

// param is a resolved Parameter Object.
type param struct {
	ptr      string
	name, in string
	required any
	fromItem bool
}

func (p param) key() string {
	name := p.name
	if p.in == "header" {
		name = strings.ToLower(name) // RFC9110 5.1: header names are case-insensitive
	}
	return p.in + "\x00" + name
}

// resolveParams resolves a "parameters" array. ok is false when an entry
// could not be resolved (external or missing reference, malformed entry),
// in which case the caller must not conclude that a parameter is absent.
// Inline parameters are checked with checkParam when check is true.
func (l *semLinter) resolveParams(ptr string, raw any, fromItem, check bool) (ps []param, ok bool) {
	list, _ := raw.([]any)
	ok = true
	for i, e := range list {
		p := obj(e)
		pptr := fmt.Sprintf("%s/%d", ptr, i)
		if p == nil {
			ok = false
			continue
		}
		if ref, isRef := p["$ref"].(string); isRef {
			p = l.localParam(ref)
			if p == nil {
				ok = false
				continue
			}
		} else if check {
			l.checkParam(pptr, p)
		}
		name, _ := p["name"].(string)
		in, _ := p["in"].(string)
		if name == "" || in == "" {
			ok = false // reported by the meta-schema
			continue
		}
		ps = append(ps, param{ptr: pptr, name: name, in: in, required: p["required"], fromItem: fromItem})
	}
	return ps, ok
}

// localParam resolves a reference to #/components/parameters/{name}. It
// returns nil for anything else.
func (l *semLinter) localParam(ref string) map[string]any {
	for range 8 { // follow reference chains, bounded
		name, found := strings.CutPrefix(ref, "#/components/parameters/")
		if !found {
			return nil
		}
		name = strings.NewReplacer("~1", "/", "~0", "~").Replace(name)
		p := obj(obj(obj(l.root["components"])["parameters"])[name])
		if p == nil {
			return nil
		}
		next, isRef := p["$ref"].(string)
		if !isRef {
			return p
		}
		ref = next
	}
	return nil
}

// checkParam applies the rules that look at a single Parameter Object.
func (l *semLinter) checkParam(ptr string, p map[string]any) {
	name, _ := p["name"].(string)
	in, hasIn := p["in"].(string)

	// Parameter Object, in: "Possible values are "query", "querystring",
	// "header", "path" or "cookie"."
	if hasIn && !slices.Contains(validIn, in) {
		l.add(ptr+"/in", "invalid parameter location %q: must be one of query, querystring, header, path, cookie", in)
	}

	// Parameter Object: "Parameter Objects MUST include either a `content`
	// field or a `schema` field, but not both."
	_, hasSchema := p["schema"]
	content, hasContent := p["content"]
	switch {
	case hasSchema && hasContent:
		l.add(ptr, "parameter %q has both schema and content: exactly one is allowed", name)
	case !hasSchema && !hasContent:
		l.add(ptr, "parameter %q has neither schema nor content: exactly one is required", name)
	}
	// `content`: "The map MUST only contain one entry."
	if hasContent {
		if m := obj(content); m != nil && len(m) != 1 {
			l.add(ptr+"/content", "parameter %q content has %d entries: it MUST contain exactly one", name, len(m))
		}
	}

	// Parameter Object, required: "If the parameter location is "path", this
	// field is REQUIRED and its value MUST be true."
	if in == "path" && p["required"] != true {
		l.add(ptr, "path parameter %q must have required: true", name)
	}

	// Parameter Object, name: "If `in` is "header" and the `name` field is
	// "Accept", "Content-Type" or "Authorization", the parameter definition
	// SHALL be ignored."
	if in == "header" && slices.Contains(reservedHdrs, strings.ToLower(name)) {
		l.add(ptr+"/name", "header parameter %q is ignored by OAS (Accept, Content-Type and Authorization are reserved): use content or security schemes", name)
	}

	// Parameter Locations, querystring: "...a parameter that treats the entire
	// URL query string as a value which MUST be specified using the `content`
	// field".
	if in == "querystring" && !hasContent {
		l.add(ptr, "querystring parameter %q must use content", name)
	}
}

type operation struct {
	ptr   string
	op    map[string]any
	ps    []param // effective parameters (path item plus operation)
	known bool    // false when some parameter could not be resolved
}

func (l *semLinter) pathItem(ptr string, item map[string]any, tmpl string, isPath bool) {
	itemPS, itemOK := l.resolveParams(ptr+"/parameters", item["parameters"], true, true)
	l.uniqueParams(ptr+"/parameters", itemPS)

	var ops []*operation
	add := func(optr string, op map[string]any) {
		ops = append(ops, l.operation(optr, op, itemPS, itemOK))
	}
	for _, m := range httpMethods {
		if op := obj(item[m]); op != nil {
			add(ptr+"/"+m, op)
		}
	}
	if add2 := obj(item["additionalOperations"]); add2 != nil {
		for _, k := range sortedKeys(add2) {
			if op := obj(add2[k]); op != nil {
				add(ptr+"/additionalOperations/"+escape(k), op)
			}
		}
	}

	// Parameter Locations, query/querystring: neither may share "the same
	// operation (or in the operation's path-item)" with the other kind; a
	// querystring "MUST NOT appear more than once".
	l.qs(itemPS, false)

	if !isPath {
		return
	}
	if _, isRef := item["$ref"]; isRef {
		return
	}
	for _, p := range itemPS {
		l.inTemplate(tmpl, p)
	}
	for _, o := range ops {
		for _, p := range o.ps {
			if !p.fromItem {
				l.inTemplate(tmpl, p)
			}
		}
	}

	// Path Templating: "Each template expression in the path MUST correspond
	// to a path parameter that is included in the Path Item itself and/or in
	// each of the Path Item's Operations. An exception is if the path item is
	// empty... matching path parameters are not required."
	// "Each template expression MUST NOT appear more than once in a single
	// path template."
	var vars []string
	for _, m := range templateVarRE.FindAllStringSubmatch(tmpl, -1) {
		if slices.Contains(vars, m[1]) {
			l.add(ptr, "template expression {%s} appears more than once in the path template", m[1])
			continue
		}
		vars = append(vars, m[1])
	}
	if len(ops) == 0 && rawLen(item["parameters"]) == 0 {
		return
	}
	for _, v := range vars {
		if hasPathParam(itemPS, v) || !itemOK {
			continue
		}
		if len(ops) == 0 {
			l.add(ptr, "path template expression {%s} has no path parameter", v)
			continue
		}
		for _, o := range ops {
			if o.known && !hasPathParam(o.ps, v) {
				l.add(o.ptr, "path template expression {%s} has no path parameter (path item or operation)", v)
			}
		}
	}
}

func rawLen(v any) int { l, _ := v.([]any); return len(l) }

func hasPathParam(ps []param, name string) bool {
	return slices.ContainsFunc(ps, func(p param) bool { return p.in == "path" && p.name == name })
}

// inTemplate checks that a path parameter names a template expression of
// its path (Parameter Object, name: "If `in` is "path", the `name` field
// MUST correspond to a single template expression occurring within the
// path field").
func (l *semLinter) inTemplate(tmpl string, p param) {
	if p.in != "path" {
		return
	}
	for _, m := range templateVarRE.FindAllStringSubmatch(tmpl, -1) {
		if m[1] == p.name {
			return
		}
	}
	l.add(p.ptr, "path parameter %q does not appear in the path template %q", p.name, tmpl)
}

func (l *semLinter) operation(ptr string, op map[string]any, itemPS []param, itemOK bool) *operation {
	o := &operation{ptr: ptr, op: op}

	// Operation Object, operationId: "The id MUST be unique among all
	// operations described in the API."
	if id, ok := op["operationId"].(string); ok {
		if first, dup := l.opIDs[id]; dup {
			l.add(ptr+"/operationId", "operationId %q is already used at %s", id, first)
		} else {
			l.opIDs[id] = ptr + "/operationId"
		}
	}

	opPS, opOK := l.resolveParams(ptr+"/parameters", op["parameters"], false, true)
	l.uniqueParams(ptr+"/parameters", opPS)
	o.known = itemOK && opOK

	// "the new definition will override it but can never remove it".
	overridden := map[string]bool{}
	for _, p := range opPS {
		overridden[p.key()] = true
	}
	for _, p := range itemPS {
		if !overridden[p.key()] {
			o.ps = append(o.ps, p)
		}
	}
	o.ps = append(o.ps, opPS...)
	l.qs(o.ps, true)

	if cbs := obj(op["callbacks"]); cbs != nil {
		for _, k := range sortedKeys(cbs) {
			if cb := obj(cbs[k]); cb != nil {
				l.callback(ptr+"/callbacks/"+escape(k), cb)
			}
		}
	}
	return o
}

// uniqueParams: "The list MUST NOT include duplicated parameters. A unique
// parameter is defined by a combination of a name and location." (Path Item
// Object parameters, Operation Object parameters.)
func (l *semLinter) uniqueParams(ptr string, ps []param) {
	seen := map[string]bool{}
	for _, p := range ps {
		if seen[p.key()] {
			l.add(p.ptr, "duplicate parameter (name %q, in %q) in %s", p.name, p.in, ptr)
			continue
		}
		seen[p.key()] = true
	}
}

// qs reports the querystring rules on a parameter list. With skipItemOnly
// (an operation's effective list) conflicts wholly inside the path item are
// skipped, since the path item pass reports them.
func (l *semLinter) qs(ps []param, skipItemOnly bool) {
	var firstQS *param
	for i, p := range ps {
		if p.in != "querystring" {
			continue
		}
		if firstQS != nil && !(skipItemOnly && firstQS.fromItem && p.fromItem) {
			l.add(p.ptr, "querystring parameter %q: an in: querystring parameter MUST NOT appear more than once (also %q)", p.name, firstQS.name)
		}
		if firstQS == nil {
			firstQS = &ps[i]
		}
	}
	if firstQS == nil {
		return
	}
	for _, p := range ps {
		if p.in == "query" && !(skipItemOnly && firstQS.fromItem && p.fromItem) {
			l.add(p.ptr, "query parameter %q cannot be combined with the in: querystring parameter %q", p.name, firstQS.name)
		}
	}
}

// dataKeys are the keywords whose value is example data, not OAS structure;
// a "$ref" key inside is not a reference.
var dataKeys = map[string]bool{
	"example": true, "default": true, "const": true, "enum": true,
	"value": true, "dataValue": true, "serializedValue": true,
}

// nameMaps are the keywords whose value maps user-chosen names to objects,
// so a child named like a dataKeys entry is a name, not data.
var nameMaps = map[string]bool{
	"properties": true, "patternProperties": true, "$defs": true, "definitions": true,
	"dependentSchemas": true, "schemas": true, "parameters": true, "responses": true,
	"examples": true, "requestBodies": true, "headers": true, "securitySchemes": true,
	"links": true, "callbacks": true, "pathItems": true, "mediaTypes": true,
	"content": true, "paths": true, "webhooks": true, "variables": true,
	"encoding": true, "scopes": true,
}

// refs walks the whole document and checks every local reference.
//
// Reference Object, $ref: "The reference identifier. This MUST be in the
// form of a URI." Schema Object $ref (JSON Schema 2020-12, 8.2.3.1): "The
// value of the "$ref" keyword MUST be a string which is a URI-reference",
// resolved against the base URI; a fragment that starts with "/" is a JSON
// Pointer (RFC 6901, via 8.2.3.1 and RFC 3986 section 3.5), so a reference
// "#/components/links/x" to a missing value is unresolvable. Fragments
// without "/" (plain-name anchors), external and relative URIs are skipped.
func (l *semLinter) refs(v any, ptr string, inNames bool) {
	switch t := v.(type) {
	case map[string]any:
		for _, k := range sortedKeys(t) {
			c := t[k]
			p := ptr + "/" + escape(k)
			if !inNames {
				if strings.HasPrefix(k, "x-") || dataKeys[k] {
					continue
				}
				if s, ok := c.(string); ok && (k == "$ref" || k == "$dynamicRef") {
					l.checkRef(p, k, s)
					continue
				}
			}
			l.refs(c, p, !inNames && nameMaps[k])
		}
	case []any:
		for i, c := range t {
			l.refs(c, fmt.Sprintf("%s/%d", ptr, i), false)
		}
	}
}

func (l *semLinter) checkRef(ptr, kw, ref string) {
	frag, ok := strings.CutPrefix(ref, "#")
	if !ok {
		return
	}
	if dec, err := url.PathUnescape(frag); err == nil {
		frag = dec
	} else {
		l.add(ptr, "%s %q has an invalid percent-encoding", kw, ref)
		return
	}
	if frag == "" || frag[0] != '/' {
		return // whole document, or a plain-name anchor
	}
	var cur any = l.root
	for seg := range strings.SplitSeq(frag[1:], "/") {
		seg = strings.NewReplacer("~1", "/", "~0", "~").Replace(seg)
		switch t := cur.(type) {
		case map[string]any:
			next, found := t[seg]
			if !found {
				l.add(ptr, "%s %q does not resolve: no %q", kw, ref, seg)
				return
			}
			cur = next
		case []any:
			i, ok := arrayIndex(seg)
			if !ok || i >= len(t) {
				l.add(ptr, "%s %q does not resolve: no index %q", kw, ref, seg)
				return
			}
			cur = t[i]
		default:
			l.add(ptr, "%s %q does not resolve: %q is under a scalar", kw, ref, seg)
			return
		}
	}
}

// arrayIndex parses an RFC 6901 array index: digits without a leading zero.
func arrayIndex(s string) (int, bool) {
	if s == "" || (len(s) > 1 && s[0] == '0') {
		return 0, false
	}
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' || n > 1<<24 {
			return 0, false
		}
		n = n*10 + int(c-'0')
	}
	return n, true
}
