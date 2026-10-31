package openapi

import (
	"fmt"
	"github.com/wI2L/fizz/openapi/internal/vtag"
	"reflect"
	"slices"
	"sort"
	"strconv"
	"strings"
)

// newSchemaFromStructField returns a new Schema builded
// from the field's type and its tags.
func (g *Generator) newSchemaFromStructField(sf reflect.StructField, required bool, fname string, parent reflect.Type) *Schema {
	return g.fieldSchema(sf, required, fname, parent, valueForm{})
}

// valueForm tells how the default, enum and example tags of a field are
// converted.
type valueForm struct {
	// param is set for a query, path or header parameter. D15 and D16
	// differ from the body fields there.
	param bool
	// quoted is set for a field with the ",string" option: its values
	// are the JSON strings that the field encodes as.
	quoted bool
}

// convert converts the string s of a tag of a field of type t.
func (vf valueForm) convert(s string, t reflect.Type) (any, error) {
	v, err := convertValue(s, t, vf.param)
	if err != nil || !vf.quoted {
		return v, err
	}
	return wireForm(v)
}

// fieldSchema is newSchemaFromStructField with the conversion form of
// the field's values. required is the requiredness declared by the tag.
func (g *Generator) fieldSchema(sf reflect.StructField, required bool, fname string, parent reflect.Type, vf valueForm) *Schema {
	sor := g.newSchemaFromType(sf.Type)
	if sor == nil {
		return nil
	}
	// The annotations of the field are use-site annotations: they
	// never touch the shared component a $ref points to. A reference
	// carries them as siblings, or in an allOf wrapper, an inline
	// schema is copied first.
	schema := g.annotationTarget(sor)

	// Default value.
	// See section 'Common Mistakes' at
	// https://swagger.io/docs/specification/describing-parameters/
	if d := sf.Tag.Get(BinderTags.Default); d != "" {
		switch {
		case required && !vf.param:
			// D15: tonic never applies defaults to body fields, so the
			// field stays required and the default is dropped.
			g.warn(fmt.Sprintf("field %q is required: its default value %q is dropped", fname, d))
		default:
			if required {
				// D15: tonic fills the default of a parameter before
				// validation, so the parameter is not required.
				g.warn(fmt.Sprintf("parameter %q has a default value: it is not required", fname))
			}
			if v, err := vf.convert(d, sf.Type); err != nil {
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
	// Must be applied to the underlying items schema if the
	// field is an array, instead of the field schema.
	if enum := g.enumFromStructField(sf, fname, parent, vf); len(enum) > 0 {
		switch {
		case isArrayLike(sf.Type) && slices.Contains(schema.Type, "array") && schema.Items != nil:
			// Copied items wrapper: the original may be shared.
			schema.Items = g.annotatedItems(schema.Items, enum)
		case isArrayLike(sf.Type) && (schema.Ref != "" || len(schema.AnyOf) > 0 || len(schema.AllOf) > 0):
			// The array is a component: constrain its elements here.
			schema.Items = &Schema{Enum: enum}
		default:
			schema.Enum = enum
		}
	}
	// Field description.
	if desc, ok := tagLookup(sf, g.tags.Description); ok {
		schema.Description = desc
	}
	// Deprecated.
	// Consider invalid values as false.
	dep, _ := tagLookup(sf, g.tags.Deprecated)
	schema.Deprecated, _ = strconv.ParseBool(dep)

	// Update schema fields related to the JSON Validation
	// spec based on the content of the validator tag.
	schema = g.updateSchemaValidation(schema, sf, fname, parent)

	// Allow overidding schema properties that were
	// auto inferred manually via tags.
	if t, ok := tagLookup(sf, g.tags.Format); ok {
		schema.Format = t
	}

	// Set example value from tag to schema
	ex, _ := tagLookup(sf, g.tags.Example)
	if e := strings.TrimSpace(ex); e != "" {
		if parsed, err := parseExample(sf.Type, e, vf.param); err != nil {
			g.error(&FieldError{
				Message:  fmt.Sprintf("could not parse the example value %q of field %q: %s", e, fname, err),
				Name:     fname,
				Type:     sf.Type,
				TypeName: g.typeName(sf.Type),
				Parent:   parent,
			})
		} else if parsed, err = quoteValue(parsed, vf); err != nil {
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

	return g.finishAnnotations(sor, schema)
}

// quoteValue returns v in its ",string" wire form if vf asks for it.
func quoteValue(v any, vf valueForm) (any, error) {
	if !vf.quoted {
		return v, nil
	}
	return wireForm(v)
}

// isArrayLike reports whether t, once dereferenced, is a slice or array
// that is described as a JSON array.
func isArrayLike(t reflect.Type) bool {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return (t.Kind() == reflect.Slice || t.Kind() == reflect.Array) && t.Elem().Kind() != reflect.Uint8
}

// annotationTarget returns the schema that receives the use-site
// annotations of a field whose schema is sor. It is never a component:
// sor is a reference or a nullable wrapper (annotated in place, as
// siblings), a reference annotated in the AllOfWrap style (a fresh
// allOf wrapper, dropped by finishAnnotations if left bare), or an
// inline schema (a copy).
func (g *Generator) annotationTarget(sor *Schema) *Schema {
	if sor.Ref != "" && g.annotationStyle == AllOfWrap {
		ref := *sor
		return &Schema{AllOf: []*Schema{&ref}}
	}
	if sor.Ref != "" || len(sor.AnyOf) > 0 {
		c := *sor
		return &c
	}
	return sor.clone()
}

// finishAnnotations returns the schema of the field: the annotated
// target, or sor itself if the target received nothing.
func (g *Generator) finishAnnotations(sor, target *Schema) *Schema {
	if len(target.AllOf) == 1 && target.AllOf[0].Ref != "" && sor.Ref != "" {
		bare := &Schema{AllOf: target.AllOf}
		if reflect.DeepEqual(target, bare) {
			return sor
		}
		return target
	}
	if sor.Ref != "" || len(sor.AnyOf) > 0 {
		if reflect.DeepEqual(target, sor) {
			return sor
		}
	}
	return target
}

// annotatedItems returns a copy of items carrying enum. A reference
// keeps the component untouched: the enum is its sibling, or the
// reference is wrapped in an allOf.
func (g *Generator) annotatedItems(items *Schema, enum []any) *Schema {
	if items.Ref != "" && g.annotationStyle == AllOfWrap {
		ref := *items
		return &Schema{AllOf: []*Schema{&ref}, Enum: enum}
	}
	c := items.clone()
	c.Enum = enum
	return c
}

func (g *Generator) enumFromStructField(sf reflect.StructField, fname string, parent reflect.Type, vf valueForm) []any {
	var enum []any

	etag := sf.Tag.Get(BinderTags.Enum)
	if etag != "" {
		values := strings.Split(etag, ",")
		sftype := sf.Type

		// Use underlying element type if it's an array/slice/pointer
		for sftype.Kind() == reflect.Pointer || sftype.Kind() == reflect.Slice || sftype.Kind() == reflect.Array {
			sftype = sftype.Elem()
		}
		for _, val := range values {
			val = strings.TrimSpace(val)
			if v, err := vf.convert(val, sftype); err != nil {
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
	sor := g.fieldSchema(f.sf, required, f.name, parent, valueForm{quoted: f.quoted})
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

// isStructFieldRequired returns whether a struct field is required,
// as read from its validate tag: the root level of the tag must have
// a "required" rule of its own (not an OR group, not under "dive",
// not "required_if" and the like) and must not be "omitempty".
//
// With "omitempty", validator v9 returns from the field as soon as
// the value is empty and skips every following rule, "required"
// included (validator.v9 validator.go, traverseField, case
// typeOmitEmpty). The field is then not required.
//
// A malformed tag is not required. The FieldError is recorded by
// updateSchemaValidation, not here: this function may be called
// several times for a field.
func (g *Generator) isStructFieldRequired(sf reflect.StructField) bool {
	ts := sf.Tag.Get(BinderTags.Validate)
	if ts == "" {
		return false
	}
	tag, err := vtag.Parse(ts)
	if err != nil || tag.Skip || tag.Root.OmitEmpty {
		return false
	}
	for _, grp := range tag.Root.Rules {
		if len(grp) == 1 && grp[0].Name == "required" {
			return true
		}
	}
	return false
}
