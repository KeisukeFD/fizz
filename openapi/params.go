package openapi

import (
	"cmp"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"sort"
	"strconv"
	"strings"
)

// setOperationParams adds the fields of the struct type t
// to the given operation.
func (g *Generator) setOperationParams(op *Operation, t, parent reflect.Type, allowBody bool, path string) error {
	if t.Kind() != reflect.Struct {
		return errors.New("input type is not a struct")
	}
	if err := g.buildParamsRecursive(op, t, parent, allowBody); err != nil {
		return err
	}
	// Input fields that are neither path- nor query-bound
	// have been extracted into the operation's RequestBody
	// If the RequestBody is not nil, give it a name and
	// move it to the openapi spec's components/schemas section
	// Replace the RequestBody's schema with a reference
	// to the named schema in components/schemas
	if op.RequestBody != nil {
		mt := g.mediaType()
		if mt == "" {
			mt = anyMediaType
		}
		sch := op.RequestBody.Content[mt].Schema
		if sch != nil {
			name := strings.Title(op.ID) + "Input"
			g.api.Components.Schemas[name] = sch
			op.RequestBody.Content[mt].Schema = &Schema{Ref: componentsSchemaPath + name}
		}
	}
	// Extract all the path parameter names.
	matches := paramsInPathRe.FindAllStringSubmatch(path, -1)
	var pathParams []string
	for _, m := range matches {
		pathParams = append(pathParams, m[1])
	}
	// Check that all declared path parameters are
	// defined in the operation.
	for _, pp := range pathParams {
		has := false
		for _, param := range op.Parameters {
			if param.Value.In == "path" && param.Value.Name == pp {
				has = true
				break
			}
		}
		if !has {
			return fmt.Errorf("semantic error for path %s: declared path parameter %s needs to be defined at operation level", path, pp)
		}
	}
	// Sort operations parameters by location and name
	// in ascending order.
	if g.sortParams {
		slices.SortStableFunc(op.Parameters, g.compareParams)
	}
	return nil
}

func (g *Generator) buildParamsRecursive(op *Operation, t, parent reflect.Type, allowBody bool) error {
	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		sft := sf.Type

		// Dereference pointer.
		if sft.Kind() == reflect.Pointer {
			sft = sft.Elem()
		}
		isUnexported := sf.PkgPath != ""

		if sf.Anonymous {
			if isUnexported && sft.Kind() != reflect.Struct {
				// Ignore embedded fields of unexported non-struct types.
				continue
			}
			// Do not ignore embedded fields of unexported struct
			// types since they may have exported fields, and recursively
			// use its fields as operations params. This allow developers
			// to reuse input models using type composition.
			if sft.Kind() == reflect.Struct {
				// If the type of the embedded struct is the same as
				// the topmost parent, skip it to avoid an infinite
				// recursive loop.
				if sft == parent {
					g.error(&FieldError{
						Message:  "recursive embedding",
						Name:     sf.Name,
						TypeName: g.typeName(parent),
						Type:     parent,
					})
				} else if err := g.buildParamsRecursive(op, sft, parent, allowBody); err != nil {
					return err
				}
			}
			continue
		} else if isUnexported {
			// Ignore unexported non-embedded fields.
			continue
		}
		if err := g.addStructFieldToOperation(op, t, i, allowBody); err != nil {
			return err
		}
	}
	return nil
}

var locationsOrder = map[string]int{
	"path":   0,
	"query":  1,
	"header": 2,
	"cookie": 3,
}

// compareParams orders parameters by location, then by name.
func (g *Generator) compareParams(p1, p2 *ParameterOrRef) int {
	a, b := g.resolveParameter(p1), g.resolveParameter(p2)
	return cmp.Or(
		cmp.Compare(locationsOrder[a.In], locationsOrder[b.In]),
		cmp.Compare(a.Name, b.Name),
	)
}

// addStructFieldToOperation add the struct field of the type
// t at index idx to the operation op. A field will be considered
// as a parameter if it has a valid location tag key, or it will
// be treated as part of the request body.
func (g *Generator) addStructFieldToOperation(op *Operation, t reflect.Type, idx int, allowBody bool) error {
	sf := t.Field(idx)

	param, err := g.newParameterFromField(idx, t)
	if err != nil {
		return err
	}
	if param != nil {
		// Check if a parameter with same name/location
		// already exists.
		for _, p := range op.Parameters {
			if p != nil && (p.Value.Name == param.Name) && (p.Value.In == param.In) {
				g.error(&FieldError{
					Message:           "duplicate parameter",
					Name:              param.Name,
					TypeName:          g.typeName(t),
					Type:              t,
					ParameterLocation: param.In,
				})
				return nil
			}
		}
		op.Parameters = append(op.Parameters, &ParameterOrRef{
			Value: param,
		})
	} else {
		if !allowBody {
			return nil
		}
		// If binding is disabled for this field, don't
		// add it to the request body. This allow using
		// a model type as an operation input while also
		// omitting some fields that are computed by the
		// server.
		if sf.Tag.Get("binding") == "-" {
			return nil
		}
		// The field is not a parameter, add it to
		// the request body.
		if op.RequestBody == nil {
			op.RequestBody = &RequestBody{
				Content: make(map[string]*MediaType),
			}
		}
		// Select the corresponding media type for the
		// given field tag, or default to any type.
		mt := g.mediaType()
		if mt == "" {
			mt = anyMediaType
		}
		var schema *Schema

		// Create the media type if no fields
		// have been added yet.
		if _, ok := op.RequestBody.Content[mt]; !ok {
			schema = &Schema{
				Type:       Types("object"),
				Properties: make(map[string]*Schema),
			}
			op.RequestBody.Content[mt] = &MediaType{
				Schema: schema,
			}
		} else {
			schema = op.RequestBody.Content[mt].Schema
		}
		fname := fieldNameFromTag(sf, mediaTags[g.mediaType()])

		// Check if a field with the same name already exists.
		if _, ok := schema.Properties[fname]; ok {
			g.error(&FieldError{
				Message:           "duplicate request body parameter",
				Name:              fname,
				TypeName:          g.typeName(t),
				Type:              t,
				ParameterLocation: "body",
			})
			return nil
		}

		var required bool
		// The required property of a field is not part of its
		// own schema but specified in the parent schema.
		if fname != "" && g.isStructFieldRequired(sf) {
			required = true
			schema.Required = append(schema.Required, fname)
			sort.Strings(schema.Required)
		}
		sfs := g.newSchemaFromStructField(sf, required, fname, t)
		if schema != nil {
			schema.Properties[fname] = sfs
		}
	}
	return nil
}

// newParameterFromField create a new operation parameter
// from the struct field at index idx in type in. Only the
// parameters of type path, query, header or cookie are concerned.
func (g *Generator) newParameterFromField(idx int, t reflect.Type) (*Parameter, error) {
	field := t.Field(idx)

	location, err := g.paramLocation(field, t)
	if err != nil {
		return nil, err
	}
	// The parameter location is empty, return nil
	// to indicate that the field is not a parameter.
	if location == "" {
		return nil, nil
	}
	name, _, _, err := ParseParamTag(field.Tag.Get(location))
	if err != nil {
		return nil, err
	}
	required := g.isStructFieldRequired(field)

	// Path parameters are always required.
	if location == g.config.PathLocationTag {
		required = true
	}
	// Consider invalid values as false.
	deprecated, _ := strconv.ParseBool(field.Tag.Get(deprecatedTag))

	p := &Parameter{
		Name:        name,
		In:          location,
		Description: field.Tag.Get(descriptionTag),
		Required:    required,
		Deprecated:  deprecated,
		Schema:      g.newSchemaFromStructField(field, required, name, t),
	}
	// Style.
	if location == g.config.QueryLocationTag {
		if field.Type.Kind() == reflect.Slice || field.Type.Kind() == reflect.Array {
			p.Explode = new(true) // default
			p.Style = "form"      // default in spec, but make it obvious
			if t := field.Tag.Get(BinderTags.Explode); t != "" {
				if explode, err := strconv.ParseBool(t); err == nil && !explode { // ignore invalid values
					p.Explode = nil // explode=false is not emitted (kept until P2.6)
				}
			}
		}
	}
	return p, nil
}

// paramLocation parses the tags of the struct field to extract
// the location of an operation parameter.
func (g *Generator) paramLocation(f reflect.StructField, in reflect.Type) (string, error) {
	var c, p int

	has := func(name string, tag reflect.StructTag, i int) {
		if _, ok := tag.Lookup(name); ok {
			c++
			// save name position to extract
			// the value of the unique key.
			p = i
		}
	}
	// Count the number of keys that represents
	// a parameter location from the tag of the
	// struct field.
	parameterLocations := []string{
		g.config.PathLocationTag,
		g.config.QueryLocationTag,
		g.config.HeaderLocationTag,
	}
	for i, n := range parameterLocations {
		has(n, f.Tag, i)
	}
	if c == 0 {
		// This will be considered to be part
		// of the request body.
		return "", nil
	}
	if c > 1 {
		return "", &FieldError{
			Message:  "conflicting parameter location",
			Name:     f.Name,
			TypeName: g.typeName(in),
			Type:     in,
		}
	}
	return parameterLocations[p], nil
}

// resolveParameter returns either the inlined parameter
// in p or the one referenced in the API components.
func (g *Generator) resolveParameter(p *ParameterOrRef) *Parameter {
	// Parameters are always automatically inlined in the spec.
	if p.Value != nil && p.Ref == nil {
		return p.Value
	}
	return nil
}
