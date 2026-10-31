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
	return g.schemaFor(t)
}

// schemaFor returns the schema of type t at a use site. It is the single
// recursion for top-level types, struct fields, slice/array elements and
// map values.
func (g *Generator) schemaFor(t reflect.Type) *Schema {
	var nullable bool

	// Dereference pointers.
	if t.Kind() == reflect.Pointer {
		for t.Kind() == reflect.Pointer {
			t = t.Elem()
		}
		nullable = true
	} else if t.Implements(tofNullable) {
		i, ok := reflect.New(t).Interface().(Nullable)
		if ok {
			nullable = i.Nullable()
		}
	}
	// Overrides, interfaces and known types are all
	// resolved by datatype.
	dt := g.datatype(t)

	switch dt {
	case TypeUnsupported:
		g.error(&TypeError{
			Message: "unsupported type",
			Type:    t,
		})
		return nil
	case TypeAny:
		return &Schema{}
	case TypeComplex:
		switch t.Kind() {
		case reflect.Struct:
			return g.newSchemaFromStruct(t)
		case reflect.Slice, reflect.Array, reflect.Map:
			return g.composite(t)
		}
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

// composite returns the schema of the slice, array or map type t.
// Named composites that form a cycle made purely of named composites
// are all promoted to components.
func (g *Generator) composite(t reflect.Type) *Schema {
	named := t.Name() != ""
	if named {
		ref := &Schema{Ref: componentsSchemaPath + g.typeName(t)}

		// Re-entered type: promote every composite of the cycle,
		// from t up to the top of the stack.
		for i, st := range g.stack {
			if st == t {
				for _, m := range g.stack[i:] {
					g.promoted[m] = true
				}
				return ref
			}
		}
		// Already promoted by an earlier cycle: its component exists.
		if g.promoted[t] {
			return ref
		}
		g.stack = append(g.stack, t)
		defer func() { g.stack = g.stack[:len(g.stack)-1] }()
	}
	schema := &Schema{}

	if t.Kind() == reflect.Map {
		// Map type is considered as a type "object"
		// and declares its value type in additionalProperties.
		schema.Type = Types("object")

		// JSON Schema allow only strings as object key.
		if t.Key().Kind() != reflect.String {
			g.error(&TypeError{
				Message: "encountered type Map with keys of unsupported type",
				Type:    t,
			})
			return nil
		}
		schema.AdditionalProperties = g.schemaFor(t.Elem())
	} else {
		// Slice/Array types are considered as a type
		// "array" and declare their element type in items.
		schema.Type = Types("array")

		// Go arrays have fixed size.
		if t.Kind() == reflect.Array {
			schema.MinItems = u64(uint64(t.Len()))
			schema.MaxItems = u64(uint64(t.Len()))
		}
		schema.Items = g.schemaFor(t.Elem())
	}
	if named && g.promoted[t] {
		name := g.typeName(t)
		g.api.Components.Schemas[name] = schema
		return &Schema{Ref: componentsSchemaPath + name}
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
	// A named struct is a component: it cuts any cycle with a $ref,
	// so cycles of named composites are scoped to the struct boundary.
	if name != "" {
		outer := g.stack
		g.stack = nil
		defer func() { g.stack = outer }()
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
