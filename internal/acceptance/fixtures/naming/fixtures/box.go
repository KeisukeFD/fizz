// Package fixtures holds a generic type, to check the component names
// of its instantiations.
package fixtures

// Box is a generic wrapper.
type Box[T any] struct {
	V T `json:"v"`
}
