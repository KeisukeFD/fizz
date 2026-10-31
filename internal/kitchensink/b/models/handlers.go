package models

import "github.com/gin-gonic/gin"

// Get is a handler whose default operation ID collides with
// ../../a/models.Get.
func Get(*gin.Context) error { return nil }
