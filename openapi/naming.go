package openapi

import (
	"reflect"
	"strings"
)

// typeName returns the unique name of a type, which is
// the concatenation of the package name and the name
// of the given type, transformed to CamelCase without
// a dot separator between the two parts.
func (g *Generator) typeName(t reflect.Type) string {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.PkgPath() == "" {
		// Predeclared or unnamed type, return an empty
		// name, the schema will be inlined in the spec.
		return ""
	}
	// If the name of the type was overidded,
	// use it in priority.
	if name, ok := g.typeNames[t]; ok {
		return name
	}
	// Create a new instance of t's type and use a
	// type assertion to check if it implements the
	// Typer interface.
	v := reflect.New(t)
	if v.CanInterface() {
		if tn, ok := v.Interface().(Typer); ok {
			return tn.TypeName()
		}
	}
	name := t.String() // package.name.
	sp := strings.Index(name, ".")
	pkg := name[:sp]

	// If the package is the main package, remove
	// the package part from the name.
	if pkg == "main" {
		pkg = ""
	}
	typ := name[sp+1:]

	if !g.fullNames {
		return strings.Title(typ)
	}
	return strings.Title(pkg) + strings.Title(typ)
}
