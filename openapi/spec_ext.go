package openapi

// XLogo represents the information about the x-logo extension.
// See: https://github.com/Redocly/redoc/blob/master/docs/redoc-vendor-extensions.md#x-logo
type XLogo struct {
	URL             string `json:"url,omitempty" yaml:"url,omitempty"`
	BackgroundColor string `json:"backgroundColor,omitempty" yaml:"backgroundColor,omitempty"`
	AltText         string `json:"altText,omitempty" yaml:"altText,omitempty"`
	Href            string `json:"href,omitempty" yaml:"href,omitempty"`
}

// XTagGroup represents the information about the x-tagGroups extension.
// See: https://github.com/Redocly/redoc/blob/master/docs/redoc-vendor-extensions.md#x-taggroups
type XTagGroup struct {
	Name string   `json:"name,omitempty" yaml:"name,omitempty"`
	Tags []string `json:"tags,omitempty" yaml:"tags,omitempty"`
}

// XCodeSample represents the information about the x-codeSample extension.
// See: https://github.com/Redocly/redoc/blob/master/docs/redoc-vendor-extensions.md#x-codesamples
type XCodeSample struct {
	Lang   string `json:"lang,omitempty" yaml:"lang,omitempty"`
	Label  string `json:"label,omitempty" yaml:"label,omitempty"`
	Source string `json:"source,omitempty" yaml:"source,omitempty"`
}
