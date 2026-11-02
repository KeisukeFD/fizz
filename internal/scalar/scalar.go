// Package scalar holds the one notion of a scalar shared by bind, which
// decodes parameters, and openapi, which describes them: the two must agree
// on which types decode from one string.
package scalar

import (
	"encoding"
	"reflect"
)

var textUnmarshaler = reflect.TypeFor[encoding.TextUnmarshaler]()

// Text reports whether a pointer to t implements encoding.TextUnmarshaler.
func Text(t reflect.Type) bool {
	return reflect.PointerTo(t).Implements(textUnmarshaler)
}

// Is reports whether t can be decoded from one string: a string, a bool, a
// number, or a type whose pointer implements encoding.TextUnmarshaler.
func Is(t reflect.Type) bool {
	if Text(t) {
		return true
	}
	switch t.Kind() {
	case reflect.String, reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	}
	return false
}

// IsValue reports whether t is a scalar or a slice of scalars.
func IsValue(t reflect.Type) bool {
	return Is(t) || (t.Kind() == reflect.Slice && Is(t.Elem()))
}
