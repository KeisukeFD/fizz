package openapi

import "reflect"

// TagNames holds the names of the struct tags that fizz owns. A zero
// field selects the default name, and "-" disables that tag. The
// binder tags (path, query, header, enum, default, validate, explode)
// are fixed to tonic's and are not configurable: see BinderTags.
type TagNames struct {
	// Description is the tag for descriptions. Default "description".
	Description string
	// Example is the tag for examples. Default "example".
	Example string
	// Format is the tag that overrides the format. Default "format".
	Format string
	// Deprecated is the tag that marks a field deprecated. Default "deprecated".
	Deprecated string
	// Title is the tag for titles. Default "title".
	Title string
	// Pattern is the tag for string patterns. Default "pattern".
	Pattern string
	// ReadOnly is the tag that marks a field read-only. Default "readOnly".
	ReadOnly string
	// WriteOnly is the tag that marks a field write-only. Default "writeOnly".
	WriteOnly string
	// Nullable is the tag that forces or removes nullability. Default "nullable".
	Nullable string
	// MultipleOf is the tag for numeric multiples. Default "multipleOf".
	MultipleOf string
	// AdditionalProperties is the tag for additionalProperties. Default "additionalProperties".
	AdditionalProperties string
	// Inline is the tag that inlines a named type. Default "inline".
	Inline string
	// Style is the tag for the parameter style. Default "style".
	Style string
	// AllowReserved is the tag for the parameter allowReserved flag. Default "allowReserved".
	AllowReserved string
	// Cookie is the tag that locates a cookie parameter. Default "cookie".
	Cookie string
}

// disabledTag is the TagNames value that disables a tag.
const disabledTag = "-"

// defaultTagNames are the tag names used for zero TagNames fields.
var defaultTagNames = TagNames{
	Description:          "description",
	Example:              "example",
	Format:               "format",
	Deprecated:           "deprecated",
	Title:                "title",
	Pattern:              "pattern",
	ReadOnly:             "readOnly",
	WriteOnly:            "writeOnly",
	Nullable:             "nullable",
	MultipleOf:           "multipleOf",
	AdditionalProperties: "additionalProperties",
	Inline:               "inline",
	Style:                "style",
	AllowReserved:        "allowReserved",
	Cookie:               "cookie",
}

// resolved returns n with every zero field replaced by its default.
func (n TagNames) resolved() TagNames {
	rv, dv := reflect.ValueOf(&n).Elem(), reflect.ValueOf(defaultTagNames)
	for i := range rv.NumField() {
		if rv.Field(i).String() == "" {
			rv.Field(i).Set(dv.Field(i))
		}
	}
	return n
}

// tagLookup looks name up in the tag of sf. It reports false when the
// tag is disabled.
func tagLookup(sf reflect.StructField, name string) (string, bool) {
	if name == disabledTag {
		return "", false
	}
	return sf.Tag.Lookup(name)
}

// SetTagNames sets the names of the fizz-owned struct tags. Zero
// fields keep the default name; "-" disables the tag.
func (g *Generator) SetTagNames(n TagNames) {
	g.tags = n.resolved()
}

// SetSchemaNamer sets the function that names component schemas from
// their type. Nil selects the default naming.
func (g *Generator) SetSchemaNamer(fn func(reflect.Type) string) {
	g.schemaNamer = fn
}
