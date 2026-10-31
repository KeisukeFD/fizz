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
	if err := g.buildParams(op, t, allowBody); err != nil {
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

// buildParams adds the parameters and the request body fields of
// the struct type t to the operation, in two passes.
//
// Parameters follow tonic's binding semantics (tonic/handler.go,
// bind): every anonymous struct field is descended into, tagged or
// not, and a field with a location tag (path, query, header) is a
// parameter named by that tag, whatever its json tag says.
//
// Body fields are the JSON members selected by typeFields that carry
// no location tag.
func (g *Generator) buildParams(op *Operation, t reflect.Type, allowBody bool) error {
	if err := g.walkParams(op, t, map[reflect.Type]bool{t: true}); err != nil {
		return err
	}
	if !allowBody {
		return nil
	}
	for f := range g.fieldsOf(t, mediaTags[g.mediaType()]) {
		loc, err := g.paramLocation(f.sf, t)
		if err != nil {
			return err
		}
		if loc != "" {
			continue
		}
		g.addBodyField(op, t, f)
	}
	return nil
}

// walkParams descends into the anonymous struct fields of t, as tonic
// does, and adds the location-tagged fields to the operation. active
// holds the struct types being walked, so that embedding cycles end.
func (g *Generator) walkParams(op *Operation, t reflect.Type, active map[reflect.Type]bool) error {
	for i := range t.NumField() {
		sf := t.Field(i)
		if sf.Anonymous {
			et := sf.Type
			if et.Kind() == reflect.Pointer {
				et = et.Elem()
			}
			// Embedded fields of unexported struct types may have
			// exported fields, so they are walked too.
			if et.Kind() != reflect.Struct || active[et] {
				continue
			}
			active[et] = true
			err := g.walkParams(op, et, active)
			delete(active, et)
			if err != nil {
				return err
			}
			continue
		}
		if !sf.IsExported() {
			continue
		}
		param, err := g.newParameterFromField(i, t)
		if err != nil {
			return err
		}
		if param == nil {
			continue
		}
		// Check if a parameter with same name/location
		// already exists.
		if slices.ContainsFunc(op.Parameters, func(p *ParameterOrRef) bool {
			return p != nil && p.Value.Name == param.Name && p.Value.In == param.In
		}) {
			g.error(&FieldError{
				Message:           "duplicate parameter",
				Name:              param.Name,
				TypeName:          g.typeName(t),
				Type:              t,
				ParameterLocation: param.In,
			})
			continue
		}
		op.Parameters = append(op.Parameters, &ParameterOrRef{Value: param})
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

// addBodyField adds the JSON member f of the root input type t to
// the operation's request body.
func (g *Generator) addBodyField(op *Operation, t reflect.Type, f field) {
	// If binding is disabled for this field, don't
	// add it to the request body. This allow using
	// a model type as an operation input while also
	// omitting some fields that are computed by the
	// server.
	if f.sf.Tag.Get("binding") == "-" {
		return
	}
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
	// Create the media type if no fields
	// have been added yet.
	if _, ok := op.RequestBody.Content[mt]; !ok {
		op.RequestBody.Content[mt] = &MediaType{
			Schema: &Schema{
				Type:       Types("object"),
				Properties: make(map[string]*Schema),
			},
		}
	}
	schema := op.RequestBody.Content[mt].Schema

	var required bool
	// The required property of a field is not part of its
	// own schema but specified in the parent schema.
	if g.isStructFieldRequired(f.sf) {
		required = true
		schema.Required = append(schema.Required, f.name)
		sort.Strings(schema.Required)
	}
	schema.Properties[f.name] = g.newSchemaFromField(f, required, t)
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
	isPath := location == g.config.PathLocationTag
	if isPath {
		required = true
	}
	// D15: a default makes a query or header parameter optional.
	// The warning is recorded with the field's schema.
	hasDefault := field.Tag.Get(g.config.DefaultTag) != ""
	schemaRequired := required && !isPath
	if schemaRequired && hasDefault {
		required = false
	}
	// Consider invalid values as false.
	deprecated, _ := strconv.ParseBool(field.Tag.Get(deprecatedTag))

	p := &Parameter{
		Name:        name,
		In:          location,
		Description: field.Tag.Get(descriptionTag),
		Required:    required,
		Deprecated:  deprecated,
		Schema:      g.fieldSchema(field, schemaRequired, name, t, valueForm{param: true}),
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
