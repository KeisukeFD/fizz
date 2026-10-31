// Package models is the second of two same-named fixture packages.
package models

import "github.com/gin-gonic/gin"

// Item is shared by name with the x package.
type Item struct {
	SKU string `json:"sku"`
}

// Get is a handler whose default operation ID collides with x.Get.
func Get(c *gin.Context) error { return nil }
