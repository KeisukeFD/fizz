package openapi

import (
	"reflect"
	"strings"
)

// newSchemaFromType creates a new OpenAPI schema from
// the given reflect type.
func (g *Generator) newSchemaFromType(t reflect.Type) *SchemaOrRef {
	if t == nil {
		return nil
	}
	var nullable bool

	// Dereference pointer.
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
		nullable = true
	} else if t.Implements(tofNullable) {
		i, ok := reflect.New(t).Interface().(Nullable)
		if ok {
			nullable = i.Nullable()
		}
	}
	dt := g.datatype(t)

	if dt == TypeUnsupported {
		g.error(&TypeError{
			Message: "unsupported type",
			Type:    t,
		})
		return nil
	}
	if dt == TypeComplex {
		switch t.Kind() {
		case reflect.Slice, reflect.Array, reflect.Map:
			return g.buildSchemaRecursive(t)
		case reflect.Struct:
			return g.newSchemaFromStruct(t)
		}
	}
	if dt == TypeAny {
		return &SchemaOrRef{
			Schema: &Schema{
				Nullable:    true,
				Description: "Value of any type, including null",
			},
		}
	}
	schema := &Schema{
		Type:     dt.Type(),
		Format:   dt.Format(),
		Nullable: nullable,
	}
	return &SchemaOrRef{Schema: schema}
}

// buildSchemaRecursive recursively decomposes the complex
// type t into subsequent schemas.
func (g *Generator) buildSchemaRecursive(t reflect.Type) *SchemaOrRef {
	schema := &Schema{}

	switch t.Kind() {
	case reflect.Pointer:
		return g.buildSchemaRecursive(t.Elem())
	case reflect.Struct:
		return g.newSchemaFromStruct(t)
	case reflect.Map:
		// Map type is considered as a type "object"
		// and should declare underlying items type
		// in additional properties field.
		schema.Type = "object"

		// JSON Schema allow only strings as object key.
		if t.Key().Kind() != reflect.String {
			g.error(&TypeError{
				Message: "encountered type Map with keys of unsupported type",
				Type:    t,
			})
			return nil
		}
		schema.AdditionalProperties = g.buildSchemaRecursive(t.Elem())
	case reflect.Slice, reflect.Array:
		// Slice/Array types are considered as a type
		// "array" and should declare underlying items
		// type in items field.
		schema.Type = "array"

		// Go arrays have fixed size.
		if t.Kind() == reflect.Array {
			schema.MinItems = t.Len()
			schema.MaxItems = t.Len()
		}
		schema.Items = g.buildSchemaRecursive(t.Elem())
	default:
		dt := g.datatype(t)
		schema.Type, schema.Format = dt.Type(), dt.Format()
	}
	return &SchemaOrRef{Schema: schema}
}

// structSchema returns an OpenAPI schema that describe
// the Go struct represented by the type t.
func (g *Generator) newSchemaFromStruct(t reflect.Type) *SchemaOrRef {
	if t.Kind() != reflect.Struct {
		return nil
	}
	name := g.typeName(t)

	// If the type of the field has already been registered,
	// skip the schema generation to avoid a recursive loop.
	// We're not returning directly a reference from the components,
	// because there is no guarantee the generation is complete yet.
	if _, ok := g.schemaTypes[t]; ok {
		return &SchemaOrRef{Reference: &Reference{
			Ref: componentsSchemaPath + name,
		}}
	}
	schema := &Schema{
		Type:       "object",
		Properties: make(map[string]*SchemaOrRef),
	}
	// Register the type once before diving into
	// the recursive hole if it has a name. Anonymous
	// struct are all considered unique.
	if name != "" {
		g.schemaTypes[t] = struct{}{}
	}
	schema = g.flattenStructSchema(t, t, schema)

	sor := &SchemaOrRef{Schema: schema}

	// Register the schema within the speccomponents and return a
	// relative reference. Unnamed types, like anonymous structs,
	// will always be inlined in the specification.
	if name != "" {
		g.api.Components.Schemas[name] = sor

		return &SchemaOrRef{Reference: &Reference{
			Ref: componentsSchemaPath + name,
		}}
	}
	// Return an inlined schema for types with no name.
	return sor
}

// resolveSchema returns either the inlined schema
// in s or the one referenced in the API components.
func (g *Generator) resolveSchema(s *SchemaOrRef) *Schema {
	if s.Schema != nil && s.Reference == nil {
		return s.Schema
	}
	if s.Reference != nil {
		parts := strings.Split(s.Reference.Ref, "/")
		if len(parts) == 4 {
			if parts[0] == "#" && // relative ref
				parts[1] == "components" &&
				parts[2] == "schemas" &&
				parts[3] != "" {
				ref, ok := g.api.Components.Schemas[parts[3]]
				if ok && ref != nil {
					return ref.Schema
				}
				return nil
			}
		}
	}
	return nil
}
