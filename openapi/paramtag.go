package openapi

import (
	"fmt"
	"strings"
)

// BinderTagNames holds the struct tag names the generator reads
// to locate and describe request parameters.
type BinderTagNames struct {
	Path, Query, Header, Enum, Default, Validate, Explode string
}

// BinderTags are the binder tag names. They are fixed to the tag
// names of the tonic binder and are not configurable.
var BinderTags = BinderTagNames{
	Path:     "path",
	Query:    "query",
	Header:   "header",
	Enum:     "enum",
	Default:  "default",
	Validate: "validate",
	Explode:  "explode",
}

// ParseParamTag parses a parameter struct tag value such as
// "name,required,default=5" and returns the parameter name, whether
// it is required and its default value. Unknown options are an error.
func ParseParamTag(tag string) (name string, required bool, defaultValue string, err error) {
	parts := strings.Split(tag, ",")
	name, options := parts[0], parts[1:]
	for _, o := range options {
		o = strings.TrimSpace(o)
		switch {
		case o == "required":
			required = true
		case strings.HasPrefix(o, "default="):
			defaultValue = strings.TrimPrefix(o, "default=")
		default:
			return "", false, "", fmt.Errorf("malformed tag for param '%s': unknown option '%s'", name, o)
		}
	}
	return name, required, defaultValue, nil
}
