package fizz

import (
	"regexp"
	"strings"
)

var closureElem = regexp.MustCompile(`^(func\d+|gowrap\d+|\d+)?$`)

// handlerBaseName derives the short name of a handler from the name
// reported by tonic (runtime.FuncForPC). It returns the package name
// and the function name, both possibly empty:
//
//	github.com/x/pkg.Handler            -> pkg, Handler
//	github.com/x/pkg.(*T).Method-fm     -> pkg, Method
//	github.com/x/pkg.Handler[...]       -> pkg, Handler
//	github.com/x/pkg.outer.func1.2      -> pkg, outer
func handlerBaseName(full string) (pkg, name string) {
	full = strings.TrimSuffix(full, "-fm")
	full = stripBrackets(full)
	if i := strings.LastIndex(full, "/"); i >= 0 {
		full = full[i+1:]
	}
	pkg, rest, ok := strings.Cut(full, ".")
	if !ok {
		return "", pkg
	}
	elems := strings.Split(rest, ".")
	for len(elems) > 0 && closureElem.MatchString(elems[len(elems)-1]) {
		elems = elems[:len(elems)-1]
	}
	if len(elems) == 0 {
		return pkg, ""
	}
	return pkg, elems[len(elems)-1]
}

// stripBrackets removes every balanced [...] group from s.
func stripBrackets(s string) string {
	var b strings.Builder
	depth := 0
	for _, r := range s {
		switch {
		case r == '[':
			depth++
		case r == ']' && depth > 0:
			depth--
		case depth == 0:
			b.WriteRune(r)
		}
	}
	return b.String()
}

var pathSegReplacer = strings.NewReplacer("{", "", "}", "", ":", "", "*", "")

// methodPathID builds an operation ID from the method and the OpenAPI
// path: GET /users/{id} -> get_users_id.
func methodPathID(method, path string) string {
	parts := []string{strings.ToLower(method)}
	for _, seg := range strings.Split(path, "/") {
		seg = pathSegReplacer.Replace(seg)
		if seg != "" {
			parts = append(parts, seg)
		}
	}
	return strings.Join(parts, "_")
}

// defaultOperationID picks the ID of an operation that has no explicit
// one. The candidates are tried in order and the first one that is not
// taken wins: the namer result (or the bare function name), then
// "pkg.Name" (no namer), then method_path. handlerName is the full
// runtime name of the handler. The package qualifier of the second
// candidate is the last import path segment: IDs must not depend on
// source files or build flags.
func defaultOperationID(method, path, handlerName string, namer func(method, path, handlerName string) string, taken func(string) bool) string {
	var candidates []string
	if namer != nil {
		if id := namer(method, path, handlerName); id != "" {
			candidates = append(candidates, id)
		}
	}
	if len(candidates) == 0 {
		if pkg, name := handlerBaseName(handlerName); name != "" {
			candidates = append(candidates, name)
			if pkg != "" {
				candidates = append(candidates, pkg+"."+name)
			}
		}
	}
	for _, id := range candidates {
		if !taken(id) {
			return id
		}
	}
	return methodPathID(method, path)
}
