package openapi

import (
	"reflect"
	"strings"
)

// typeName returns the component name of a type. It returns "" for
// unnamed types, whose schema is inlined.
//
// Precedence: an explicit override (OverrideTypeName), the Typer
// interface, the SchemaNamer of the generator, then the default name:
// the package name and the type name in upper camel case, joined
// without separator. Whatever the source, the result is sanitized to
// ^[A-Za-z0-9._-]+$.
func (g *Generator) typeName(t reflect.Type) string {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.PkgPath() == "" {
		// Predeclared or unnamed type, return an empty
		// name, the schema will be inlined in the spec.
		return ""
	}
	if name, ok := g.typeNames[t]; ok {
		return sanitizeName(name)
	}
	// Use a type assertion on a new instance of t to check
	// if it implements the Typer interface.
	v := reflect.New(t)
	if v.CanInterface() {
		if tn, ok := v.Interface().(Typer); ok {
			return sanitizeName(tn.TypeName())
		}
	}
	if g.schemaNamer != nil {
		if name := g.schemaNamer(t); name != "" {
			return sanitizeName(name)
		}
	}
	return sanitizeName(g.defaultTypeName(t))
}

// defaultTypeName returns the package name and type name of t in upper
// camel case, or the type name alone if full names are disabled. The
// package is omitted for the main package.
func (g *Generator) defaultTypeName(t reflect.Type) string {
	pkg, _, _ := strings.Cut(t.String(), ".")
	if pkg == "main" {
		pkg = ""
	}
	typ := upperFirst(genericName(t.Name(), packageNames(t)))
	if !g.fullNames {
		return typ
	}
	return upperFirst(pkg) + typ
}

// upperFirst upper-cases the first byte of s if it is an ASCII letter.
func upperFirst(s string) string {
	if s == "" || s[0] < 'a' || s[0] > 'z' {
		return s
	}
	return string(s[0]-'a'+'A') + s[1:]
}

// sanitizeName replaces every run of characters outside
// [A-Za-z0-9._-] with a single underscore.
func sanitizeName(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	bad := false
	for _, r := range s {
		if r < 0x80 && (r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '.' || r == '_' || r == '-') {
			b.WriteRune(r)
			bad = false
			continue
		}
		if !bad {
			b.WriteByte('_')
			bad = true
		}
	}
	return b.String()
}

// genericName rewrites the name of an instantiated generic type,
// "Page[github.com/x/models.Item]", as "Page_models.Item". Type
// arguments are joined with "_". A name without type arguments is
// returned as is.
func genericName(name string, pkgs map[string]string) string {
	i := strings.IndexByte(name, '[')
	if i < 0 || !strings.HasSuffix(name, "]") {
		return name
	}
	parts := []string{name[:i]}
	for _, a := range splitTop(name[i+1 : len(name)-1]) {
		parts = append(parts, typeArgName(a, pkgs))
	}
	return strings.Join(parts, "_")
}

// typeArgName renders one type argument of a generic type name.
func typeArgName(a string, pkgs map[string]string) string {
	switch {
	case strings.HasPrefix(a, "*"):
		return typeArgName(a[1:], pkgs)
	case strings.HasPrefix(a, "[]"):
		return "List_" + typeArgName(a[2:], pkgs)
	case strings.HasPrefix(a, "["):
		// Array: [N]T.
		if end := matchBracket(a, 0); end > 0 {
			return "Array_" + typeArgName(a[end+1:], pkgs)
		}
	case strings.HasPrefix(a, "map["):
		if end := matchBracket(a, 3); end > 0 {
			return "Map_" + typeArgName(a[4:end], pkgs) + "_" + typeArgName(a[end+1:], pkgs)
		}
	}
	// Named type, possibly generic itself. Only the base carries a
	// package path, replaced by the package name when it is known,
	// else by its last segment.
	base, rest := a, ""
	if i := strings.IndexByte(a, '['); i >= 0 {
		base, rest = a[:i], a[i:]
	}
	j := strings.LastIndexByte(base, '/') + 1
	if dot := strings.IndexByte(base[j:], '.'); dot >= 0 {
		if name, ok := pkgs[base[:j+dot]]; ok {
			base = name + base[j+dot:]
		} else {
			base = base[j:]
		}
	} else {
		base = base[j:]
	}
	if rest != "" {
		return genericName(base+rest, pkgs)
	}
	return base
}

// packageNames maps the import path of every package reachable through
// the fields and elements of t to its package name. reflect renders the
// type arguments of a generic type with import paths, whereas names use
// the package name, which can differ from the last path segment.
func packageNames(t reflect.Type) map[string]string {
	if !strings.Contains(t.Name(), "[") {
		return nil
	}
	m := make(map[string]string)
	seen := make(map[reflect.Type]bool)
	var walk func(reflect.Type)
	walk = func(t reflect.Type) {
		if seen[t] {
			return
		}
		seen[t] = true
		if p := t.PkgPath(); p != "" {
			if name, _, ok := strings.Cut(t.String(), "."); ok {
				m[p] = name
			}
		}
		switch t.Kind() {
		case reflect.Pointer, reflect.Slice, reflect.Array:
			walk(t.Elem())
		case reflect.Map:
			walk(t.Key())
			walk(t.Elem())
		case reflect.Struct:
			for i := range t.NumField() {
				walk(t.Field(i).Type)
			}
		}
	}
	walk(t)
	return m
}

// matchBracket returns the index of the ']' closing the '[' at s[open],
// or -1.
func matchBracket(s string, open int) int {
	depth := 0
	for i := open; i < len(s); i++ {
		switch s[i] {
		case '[':
			depth++
		case ']':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

// splitTop splits s at the commas that are not nested in brackets or
// parentheses.
func splitTop(s string) []string {
	var out []string
	depth, start := 0, 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '[', '(', '{':
			depth++
		case ']', ')', '}':
			depth--
		case ',':
			if depth == 0 {
				out = append(out, s[start:i])
				start = i + 1
			}
		}
	}
	return append(out, s[start:])
}
