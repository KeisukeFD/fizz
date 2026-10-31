package openapi

import (
	"fmt"
	"reflect"
	"slices"
	"sort"
	"strconv"
	"strings"
)

// newSchemaFromStructField returns a new Schema builded
// from the field's type and its tags.
func (g *Generator) newSchemaFromStructField(sf reflect.StructField, required bool, fname string, parent reflect.Type) *Schema {
	sor := g.newSchemaFromType(sf.Type)
	if sor == nil {
		return nil
	}
	// Get the underlying schema, it may be a reference
	// to a component, and update its fields using the
	// informations in the struct field tags.
	schema := g.resolveSchema(sor)

	if schema == nil {
		return sor
	}
	// Default value.
	// See section 'Common Mistakes' at
	// https://swagger.io/docs/specification/describing-parameters/
	if d := sf.Tag.Get(g.config.DefaultTag); d != "" {
		if required {
			g.error(&FieldError{
				Message:  "field cannot be required and have a default value",
				Name:     fname,
				Type:     sf.Type,
				TypeName: g.typeName(sf.Type),
				Parent:   parent,
			})
		} else {
			if v, err := stringToType(d, sf.Type); err != nil {
				g.error(&FieldError{
					Message:  fmt.Sprintf("default value %s cannot be converted to field type: %s", d, err),
					Name:     fname,
					Type:     sf.Type,
					TypeName: g.typeName(sf.Type),
					Parent:   parent,
				})
			} else {
				schema.Default = v
			}
		}
	}
	// Enum.
	// Must be applied to underlying items schema if the
	// parameter is an array, instead of the parameter schema.
	enum := g.enumFromStructField(sf, fname, parent)

	if slices.Contains(schema.Type, "array") && schema.Items != nil {
		itemsSchema := g.resolveSchema(schema.Items)
		if itemsSchema != nil {
			itemsSchema.Enum = enum
		}
	} else {
		schema.Enum = enum
	}
	// Field description.
	if desc, ok := sf.Tag.Lookup(descriptionTag); ok {
		schema.Description = desc
	}
	// Deprecated.
	// Consider invalid values as false.
	schema.Deprecated, _ = strconv.ParseBool(sf.Tag.Get(deprecatedTag))

	// Update schema fields related to the JSON Validation
	// spec based on the content of the validator tag.
	schema = g.updateSchemaValidation(schema, sf)

	// Allow overidding schema properties that were
	// auto inferred manually via tags.
	if t, ok := sf.Tag.Lookup(formatTag); ok {
		schema.Format = t
	}

	// Set example value from tag to schema
	if e := strings.TrimSpace(sf.Tag.Get("example")); e != "" {
		if parsed, err := parseExampleValue(sf.Type, e); err != nil {
			g.error(&FieldError{
				Message:  fmt.Sprintf("could not parse the example value %q of field %q: %s", e, fname, err),
				Name:     fname,
				Type:     sf.Type,
				TypeName: g.typeName(sf.Type),
				Parent:   parent,
			})
		} else if err := jsonEncodable(parsed); err != nil {
			g.error(&FieldError{
				Message:  fmt.Sprintf("the example value %q of field %q is not JSON-encodable, dropped: %s", e, fname, err),
				Name:     fname,
				Type:     sf.Type,
				TypeName: g.typeName(sf.Type),
				Parent:   parent,
			})
		} else {
			schema.Examples = []any{parsed}
		}
	}

	return sor
}

func (g *Generator) enumFromStructField(sf reflect.StructField, fname string, parent reflect.Type) []any {
	var enum []any

	etag := sf.Tag.Get(g.config.EnumTag)
	if etag != "" {
		values := strings.Split(etag, ",")
		sftype := sf.Type

		// Use underlying element type if it's an array/slice/pointer
		for sftype.Kind() == reflect.Pointer || sftype.Kind() == reflect.Slice || sftype.Kind() == reflect.Array {
			sftype = sftype.Elem()
		}
		for _, val := range values {
			if v, err := stringToType(val, sftype); err != nil {
				g.error(&FieldError{
					Message:  fmt.Sprintf("enum value %s cannot be converted to field type: %s", val, err),
					Name:     fname,
					Type:     sf.Type,
					TypeName: g.typeName(sf.Type),
					Parent:   parent,
				})
			} else {
				enum = append(enum, v)
			}
		}
	}
	return enum
}

// flattenStructSchema adds the JSON members of the struct type t,
// as selected by typeFields (embedded structs promoted), to the
// given schema.
func (g *Generator) flattenStructSchema(t, parent reflect.Type, schema *Schema) *Schema {
	mediaTag := mediaTags[g.mediaType()]
	for f := range g.fieldsOf(t, mediaTag) {
		var required bool
		// The required property of a field is not part of its
		// own schema but specified in the parent schema.
		if g.isStructFieldRequired(f.sf) {
			required = true
			schema.Required = append(schema.Required, f.name)
			sort.Strings(schema.Required)
		}
		if sfs := g.newSchemaFromField(f, required, t); sfs != nil {
			schema.Properties[f.name] = sfs
		}
	}
	return schema
}

// newSchemaFromField returns the property schema of f. A field with
// the ",string" option is encoded as a JSON string by encoding/json,
// so its schema type becomes string.
func (g *Generator) newSchemaFromField(f field, required bool, parent reflect.Type) *Schema {
	sor := g.newSchemaFromStructField(f.sf, required, f.name, parent)
	if sor == nil || !f.quoted || sor.Ref != "" || len(sor.Type) == 0 {
		return sor
	}
	if slices.Contains(sor.Type, "null") {
		sor.Type = Types("string", "null")
	} else {
		sor.Type = Types("string")
	}
	sor.Format = ""
	return sor
}

// isStructFieldRequired returns whether a struct field
// is required. The information is read from the field
// tag 'binding'.
func (g *Generator) isStructFieldRequired(sf reflect.StructField) bool {
	if t, ok := sf.Tag.Lookup(g.config.ValidatorTag); ok {
		options := strings.SplitSeq(t, ",")
		for o := range options {
			// As soon as we see a 'dive' or 'keys'
			// options, the following options won't
			// apply to the given field.
			if o == "dive" || o == "keys" {
				return false
			}
			if o == "required" {
				return true
			}
		}
	}
	return false
}
