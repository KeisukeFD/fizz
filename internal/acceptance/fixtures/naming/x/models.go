// Package models is the first of two same-named fixture packages.
package models

import "github.com/gin-gonic/gin"

// Item is shared by name with the y package.
type Item struct {
	ID int `json:"id"`
}

// Get is a handler whose default operation ID collides with y.Get.
func Get(c *gin.Context) error { return nil }
