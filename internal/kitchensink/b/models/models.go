// Package models is a seed model package that deliberately shares its
// package name with ../../a/models, to test name collisions.
package models

// Item has the same name as a/models.Item but a different shape.
type Item struct {
	SKU   string  `json:"sku"`
	Price float64 `json:"price"`
}
