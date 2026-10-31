package openapi

import (
	"encoding/json"
	"fmt"
	"github.com/wI2L/fizz/openapi/internal/vtag"
	"math"
	"reflect"
	"regexp"
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
	// The inline and nullable tags change the form of the schema
	// itself, before any annotation lands on it. Both work on copies.
	sor = g.useSiteForm(sor, sf, fname, parent)
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
	// The enum and const tags land after MakeNullable ran: they must
	// admit the null the schema accepts.
	if acceptsNull(schema) {
		schema.allowNullValue()
	}

	// Allow overidding schema properties that were
	// auto inferred manually via tags.
	if t, ok := tagLookup(sf, g.tags.Format); ok {
		schema.Format = t
	}

	// Keywords of the title, pattern, multipleOf, readOnly, writeOnly
	// and additionalProperties tags.
	g.applyTagKeywords(schema, sf, fname, parent)

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
		sfs, skipped := g.newSchemaFromField(f, required, t)
		if skipped {
			dropRequired(schema, f.name)
		} else if sfs != nil {
			schema.Properties[f.name] = sfs
		}
	}
	return schema
}

// newSchemaFromField returns the property schema of f. A field with
// the ",string" option is encoded as a JSON string by encoding/json,
// so its schema type becomes string.
// It runs the FieldHook last, and reports whether the hook skips the
// field.
func (g *Generator) newSchemaFromField(f field, required bool, parent reflect.Type) (s *Schema, skipped bool) {
	sor := g.fieldSchema(f.sf, required, f.name, parent, valueForm{quoted: f.quoted})
	if sor == nil {
		return nil, false
	}
	if f.quoted && sor.Ref == "" && len(sor.Type) > 0 {
		if slices.Contains(sor.Type, "null") {
			sor.Type = Types("string", "null")
		} else {
			sor.Type = Types("string")
		}
		sor.Format = ""
	}
	if g.fieldHook(f, parent, sor) {
		return nil, true
	}
	return sor, false
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

// tagError records a FieldError about the tag named tag of sf.
func (g *Generator) tagError(sf reflect.StructField, fname string, parent reflect.Type, tag, format string, args ...any) {
	g.error(&FieldError{
		Message:  fmt.Sprintf("invalid %q tag: ", tag) + fmt.Sprintf(format, args...),
		Name:     fname,
		Field:    sf.Name,
		Type:     sf.Type,
		TypeName: g.typeName(sf.Type),
		Parent:   parent,
	})
}

// boolTag reads the boolean tag named tag of sf. It reports false if
// the tag is absent, disabled, empty or not a boolean; the last case is
// recorded as a FieldError.
func (g *Generator) boolTag(sf reflect.StructField, fname string, parent reflect.Type, tag string) (value, ok bool) {
	v, found := tagLookup(sf, tag)
	if !found || v == "" {
		return false, false
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		g.tagError(sf, fname, parent, tag, "%q is not a boolean", v)
		return false, false
	}
	return b, true
}

// useSiteForm applies the inline (O12) and nullable (D6) tags of sf
// to sor. It returns sor itself, or a modified copy: the component
// behind a reference is never touched.
func (g *Generator) useSiteForm(sor *Schema, sf reflect.StructField, fname string, parent reflect.Type) *Schema {
	if b, ok := g.boolTag(sf, fname, parent, g.tags.Inline); ok && b {
		sor = g.inlineComponent(sor, sf, fname, parent)
	}
	if b, ok := g.boolTag(sf, fname, parent, g.tags.Nullable); ok {
		if b {
			sor = sor.clone()
			sor.MakeNullable()
		} else {
			sor = withoutNull(sor)
		}
	}
	return sor
}

// inlineComponent returns a copy of the component that sor references,
// to be emitted in place of the reference (O12). A nullable reference
// stays nullable. A schema that is already inline is returned as is.
func (g *Generator) inlineComponent(sor *Schema, sf reflect.StructField, fname string, parent reflect.Type) *Schema {
	t := sf.Type
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		g.tagError(sf, fname, parent, g.tags.Inline, "only a struct field can be inlined")
		return sor
	}
	ref, nullable := sor.Ref, false
	if ref == "" && len(sor.AnyOf) == 2 && sor.AnyOf[0] != nil && sor.AnyOf[0].Ref != "" && isNullSchema(sor.AnyOf[1]) {
		ref, nullable = sor.AnyOf[0].Ref, true
	}
	if ref == "" {
		return sor
	}
	comp, ok := g.api.Components.Schemas[strings.TrimPrefix(ref, componentsSchemaPath)]
	if !ok || comp == nil {
		g.tagError(sf, fname, parent, g.tags.Inline, "a recursive type cannot be inlined")
		return sor
	}
	c := copySchemaTree(comp)
	if nullable {
		c.MakeNullable()
	}
	return c
}

// isNullSchema reports whether s is {"type":"null"}.
func isNullSchema(s *Schema) bool {
	return s != nil && s.Ref == "" && slices.Equal(s.Type, Types("null"))
}

// acceptsNull reports whether s accepts null through its type set or
// a null branch of its anyOf.
func acceptsNull(s *Schema) bool {
	return slices.Contains(s.Type, "null") || slices.ContainsFunc(s.AnyOf, isNullSchema)
}

// withoutNull returns a copy of s that does not accept null: the
// inverse of Schema.MakeNullable. s is returned if it is not nullable.
func withoutNull(s *Schema) *Schema {
	if slices.Contains(s.Type, "null") {
		c := s.clone()
		c.Type = slices.DeleteFunc(slices.Clone(s.Type), func(t string) bool { return t == "null" })
		if slices.Contains(s.Enum, nil) {
			c.Enum = slices.DeleteFunc(slices.Clone(s.Enum), func(v any) bool { return v == nil })
		}
		return c
	}
	if len(s.Type) == 0 && len(s.AnyOf) == 2 && isNullSchema(s.AnyOf[1]) && s.AnyOf[0] != nil {
		first := s.AnyOf[0]
		c := s.clone()
		c.AnyOf, c.Ref = first.AnyOf, first.Ref
		return c
	}
	return s
}

// applyTagKeywords applies the title, pattern, multipleOf, readOnly,
// writeOnly and additionalProperties tags of sf to schema, a use-site
// target. An invalid value is recorded and dropped.
func (g *Generator) applyTagKeywords(schema *Schema, sf reflect.StructField, fname string, parent reflect.Type) {
	if v, ok := tagLookup(sf, g.tags.Title); ok && v != "" {
		schema.Title = v
	}
	if v, ok := tagLookup(sf, g.tags.Pattern); ok && v != "" {
		if _, err := regexp.Compile(v); err != nil {
			g.tagError(sf, fname, parent, g.tags.Pattern, "%q is not a valid regular expression: %s", v, err)
		} else {
			schema.Pattern = v
		}
	}
	if v, ok := tagLookup(sf, g.tags.MultipleOf); ok && v != "" {
		v = strings.TrimSpace(v)
		if f, err := strconv.ParseFloat(v, 64); err != nil || f <= 0 || math.IsInf(f, 0) || math.IsNaN(f) {
			g.tagError(sf, fname, parent, g.tags.MultipleOf, "%q is not a positive number", v)
		} else if json.Valid([]byte(v)) {
			schema.MultipleOf = json.Number(v)
		} else {
			// Go syntax that is not JSON ("1e", "+1", ".5", hex).
			schema.MultipleOf = json.Number(strconv.FormatFloat(f, 'g', -1, 64))
		}
	}
	ro, roSet := g.boolTag(sf, fname, parent, g.tags.ReadOnly)
	wo, woSet := g.boolTag(sf, fname, parent, g.tags.WriteOnly)
	switch {
	case ro && wo:
		g.warn(fmt.Sprintf("field %q is both %s and %s: both are dropped", fname, g.tags.ReadOnly, g.tags.WriteOnly))
	default:
		if roSet {
			schema.ReadOnly = ro
		}
		if woSet {
			schema.WriteOnly = wo
		}
	}
	if v, ok := tagLookup(sf, g.tags.AdditionalProperties); ok && v != "" {
		g.closeObject(schema, v, sf, fname, parent)
	}
}

// closeObject applies the additionalProperties tag (O11). Only "false"
// is supported. On a reference a sibling additionalProperties:false
// would reject every property, so unevaluatedProperties:false is used.
func (g *Generator) closeObject(schema *Schema, v string, sf reflect.StructField, fname string, parent reflect.Type) {
	tag := g.tags.AdditionalProperties
	if v != "false" {
		g.tagError(sf, fname, parent, tag, "%q is not supported, only \"false\"", v)
		return
	}
	no := &Schema{Bool: new(false)}
	ref := schema.Ref
	if ref == "" && len(schema.AllOf) == 1 && schema.AllOf[0] != nil {
		ref = schema.AllOf[0].Ref
	}
	if ref == "" && len(schema.AnyOf) > 0 && schema.AnyOf[0] != nil {
		ref = schema.AnyOf[0].Ref
	}
	if ref != "" {
		if comp, ok := g.api.Components.Schemas[strings.TrimPrefix(ref, componentsSchemaPath)]; ok && comp != nil &&
			len(comp.Type) > 0 && !slices.Contains(comp.Type, "object") {
			g.tagError(sf, fname, parent, tag, "the field is not an object")
			return
		}
		schema.UnevaluatedProperties = no
		return
	}
	switch {
	case !slices.Contains(schema.Type, "object"):
		g.tagError(sf, fname, parent, tag, "the field is not an object")
	case schema.AdditionalProperties != nil:
		g.tagError(sf, fname, parent, tag, "a map already declares additionalProperties")
	default:
		schema.AdditionalProperties = no
	}
}
