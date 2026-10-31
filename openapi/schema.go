package openapi

import (
	"reflect"
	"strings"
)

// newSchemaFromType creates a new OpenAPI schema from
// the given reflect type.
func (g *Generator) newSchemaFromType(t reflect.Type) *Schema {
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
		return &Schema{}
	}
	schema := &Schema{
		Type:   typeSetOf(dt.Type()),
		Format: dt.Format(),
	}
	if dt == TypeByte {
		schema.Format = ""
		schema.ContentEncoding = "base64"
	}
	if nullable {
		schema.MakeNullable()
	}
	return schema
}

// buildSchemaRecursive recursively decomposes the complex
// type t into subsequent schemas.
func (g *Generator) buildSchemaRecursive(t reflect.Type) *Schema {
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
		schema.Type = Types("object")

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
		schema.Type = Types("array")

		// Go arrays have fixed size.
		if t.Kind() == reflect.Array {
			schema.MinItems = u64(uint64(t.Len()))
			schema.MaxItems = u64(uint64(t.Len()))
		}
		schema.Items = g.buildSchemaRecursive(t.Elem())
	default:
		dt := g.datatype(t)
		schema.Type, schema.Format = typeSetOf(dt.Type()), dt.Format()
	}
	return schema
}

// structSchema returns an OpenAPI schema that describe
// the Go struct represented by the type t.
func (g *Generator) newSchemaFromStruct(t reflect.Type) *Schema {
	if t.Kind() != reflect.Struct {
		return nil
	}
	name := g.typeName(t)

	// If the type of the field has already been registered,
	// skip the schema generation to avoid a recursive loop.
	// We're not returning directly a reference from the components,
	// because there is no guarantee the generation is complete yet.
	if _, ok := g.schemaTypes[t]; ok {
		return &Schema{Ref: componentsSchemaPath + name}
	}
	schema := &Schema{
		Type:       Types("object"),
		Properties: make(map[string]*Schema),
	}
	// Register the type once before diving into
	// the recursive hole if it has a name. Anonymous
	// struct are all considered unique.
	if name != "" {
		g.schemaTypes[t] = struct{}{}
	}
	schema = g.flattenStructSchema(t, t, schema)

	sor := schema

	// Register the schema within the speccomponents and return a
	// relative reference. Unnamed types, like anonymous structs,
	// will always be inlined in the specification.
	if name != "" {
		g.api.Components.Schemas[name] = sor

		return &Schema{Ref: componentsSchemaPath + name}
	}
	// Return an inlined schema for types with no name.
	return sor
}

// resolveSchema returns either the inlined schema
// in s or the one referenced in the API components.
func (g *Generator) resolveSchema(s *Schema) *Schema {
	if s.Ref == "" {
		return s
	}
	{
		parts := strings.Split(s.Ref, "/")
		if len(parts) == 4 {
			if parts[0] == "#" && // relative ref
				parts[1] == "components" &&
				parts[2] == "schemas" &&
				parts[3] != "" {
				ref, ok := g.api.Components.Schemas[parts[3]]
				if ok && ref != nil {
					return ref
				}
				return nil
			}
		}
	}
	return nil
}

// typeSetOf returns a TypeSet holding t, or nil if t is empty.
func typeSetOf(t string) TypeSet {
	if t == "" {
		return nil
	}
	return Types(t)
}
