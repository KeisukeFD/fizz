// Package generic holds seed generic types for kitchen-sink cases.
package generic

// Page is a generic envelope.
type Page[T any] struct {
	Items []T `json:"items"`
	Total int `json:"total"`
}

// Pair has two type parameters.
type Pair[K comparable, V any] struct {
	Key   K `json:"key"`
	Value V `json:"value"`
}
