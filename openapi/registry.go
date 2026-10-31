package openapi

import (
	"fmt"
	"reflect"
)

// CollisionError reports that two distinct Go types map to the same
// component name. The first type registered (Kept) owns the name; the
// schema of the second (Rejected) is inlined at its use sites.
type CollisionError struct {
	Name     string
	Kept     reflect.Type
	Rejected reflect.Type
}

// Error implements the error interface.
func (e *CollisionError) Error() string {
	return fmt.Sprintf("component name %q is used by both %s and %s: %s keeps it, %s is inlined",
		e.Name, e.Kept, e.Rejected, e.Kept, e.Rejected)
}

type collisionKey struct {
	name     string
	rejected reflect.Type
}

// registry maps component schema names to the Go type that owns them.
// The zero value is ready to use.
type registry struct {
	names    map[string]reflect.Type
	reported map[collisionKey]bool
}

// owner returns the type that owns name, if any.
func (r *registry) owner(name string) (reflect.Type, bool) {
	t, ok := r.names[name]
	return t, ok
}

// claim records name for t. It returns the type that owns name and
// whether that is t.
func (r *registry) claim(name string, t reflect.Type) (reflect.Type, bool) {
	if cur, ok := r.names[name]; ok {
		return cur, cur == t
	}
	if r.names == nil {
		r.names = make(map[string]reflect.Type)
	}
	r.names[name] = t
	return t, true
}

// componentName returns the component name of the dereferenced named
// type t and registers it. It returns ok=false when the type is unnamed
// or the name is owned by another type: a *CollisionError is then
// recorded (once) and the caller must inline the schema of t.
func (g *Generator) componentName(t reflect.Type) (name string, ok bool) {
	name = g.typeName(t)
	if name == "" {
		return "", false
	}
	kept, mine := g.registry.claim(name, t)
	if mine {
		return name, true
	}
	k := collisionKey{name, t}
	if g.registry.reported == nil {
		g.registry.reported = make(map[collisionKey]bool)
	}
	if !g.registry.reported[k] {
		g.registry.reported[k] = true
		g.error(&CollisionError{Name: name, Kept: kept, Rejected: t})
	}
	return "", false
}
