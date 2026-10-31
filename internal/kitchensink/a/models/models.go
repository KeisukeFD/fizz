// Package models is a seed model package. A sibling package with the
// same name lives under ../../b/models so that cases can exercise
// schema name collisions.
package models

// Item is a minimal model shared with b/models.Item by name.
type Item struct {
	ID   int    `json:"id"`
	Name string `json:"name" validate:"required"`
}
