package openapi

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/loopfz/gadgeto/tonic"
)

// newSchemaFromStructField returns a new Schema builded
// from the field's type and its tags.
func (g *Generator) newSchemaFromStructField(sf reflect.StructField, required bool, fname string, parent reflect.Type) *SchemaOrRef {
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

	if schema.Type == "array" && schema.Items != nil {
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
		} else {
			schema.Example = parsed
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

// flattenStructSchema recursively flatten the embedded
// fields of the struct type t to the given schema.
func (g *Generator) flattenStructSchema(t, parent reflect.Type, schema *Schema) *Schema {
	for f := range t.Fields() {
		ft := f.Type

		// Dereference pointer.
		if ft.Kind() == reflect.Pointer {
			ft = ft.Elem()
		}
		isUnexported := f.PkgPath != ""
		mediaTag := mediaTags[tonic.MediaType()]
		_, hasTag := f.Tag.Lookup(mediaTag)

		if f.Anonymous && !hasTag {
			if isUnexported && ft.Kind() != reflect.Struct {
				// Ignore embedded fields of unexported non-struct types.
				continue
			}
			// Do not ignore embedded fields of unexported struct
			// types since they may have exported fields.
			if ft.Kind() == reflect.Struct {
				// However, if the type of the embedded field is the same
				// as the topmost parent, skip it to avoid an infinite
				// recursive loop.
				if ft == parent {
					g.error(&FieldError{
						Message:  "recursive embedding",
						Name:     f.Name,
						TypeName: g.typeName(parent),
						Type:     parent,
						Parent:   parent,
					})
				} else {
					schema = g.flattenStructSchema(ft, parent, schema)
				}
			}
			continue
		} else if isUnexported {
			// Ignore unexported non-embedded fields.
			continue
		}
		fname := fieldNameFromTag(f, mediaTag)
		if fname == "" {
			// Field has no name, skip it.
			continue
		}
		var required bool
		// The required property of a field is not part of its
		// own schema but specified in the parent schema.
		if fname != "" && g.isStructFieldRequired(f) {
			required = true
			schema.Required = append(schema.Required, fname)
			sort.Strings(schema.Required)
		}
		sfs := g.newSchemaFromStructField(f, required, fname, t)
		if sfs != nil {
			schema.Properties[fname] = sfs
		}
	}
	return schema
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

// fieldTagName returns the name of a struct field
// extracted from a serialization tag using its name.
func fieldNameFromTag(sf reflect.StructField, tagName string) string {
	v, ok := sf.Tag.Lookup(tagName)
	if !ok {
		return sf.Name
	}
	parts := strings.Split(strings.TrimSpace(v), ",")

	// Split return a one item slice if
	// the input string is empty, thus we
	// don't check the length.
	name := parts[0]
	if name == "" {
		return sf.Name
	}
	if name == "-" {
		return ""
	}
	return name
}
