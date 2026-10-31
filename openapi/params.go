package openapi

import (
	"cmp"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"slices"
	"sort"
	"strconv"
	"strings"
)

// setOperationParams adds the fields of the struct type t
// to the given operation.
func (g *Generator) setOperationParams(op *Operation, t, parent reflect.Type, allowBody bool) error {
	if t.Kind() != reflect.Struct {
		return errors.New("input type is not a struct")
	}
	if err := g.buildParams(op, t, allowBody); err != nil {
		return err
	}
	// Input fields that are neither path- nor query-bound
	// have been extracted into the operation's RequestBody.
	if op.RequestBody != nil {
		g.nameRequestBody(op, t)
	}
	return nil
}

// applyRequestBody applies the RequestBody option rb to op, whose
// inferred request body, if any, is already built (unless rb has
// contents, which replace it: it is then not inferred at all). Problems
// are recorded and leave the operation without the part that is invalid.
func (g *Generator) applyRequestBody(op *Operation, method string, rb *RequestBodyInfo) {
	if rb == nil {
		return
	}
	if method == http.MethodGet || method == http.MethodHead {
		g.error(fmt.Errorf("RequestBody: %s operations cannot have a request body", method))
		return
	}
	if len(rb.Contents) == 0 {
		if op.RequestBody == nil {
			g.error(errors.New("RequestBody: the operation has no request body to describe"))
			return
		}
	} else {
		op.RequestBody = &RequestBody{Content: g.contentsFromInfo(rb.Contents)}
	}
	op.RequestBody.Description = rb.Description
	op.RequestBody.Required = rb.Required
}

// contentsFromInfo builds the content map of a request body. A media
// type given twice keeps its first definition and records an error.
func (g *Generator) contentsFromInfo(contents []*ContentInfo) map[string]*MediaType {
	m := make(map[string]*MediaType, len(contents))
	for _, c := range contents {
		if c == nil {
			continue
		}
		if c.MediaType == "" {
			g.error(errors.New("RequestBody: a content has no media type"))
			continue
		}
		if _, dup := m[c.MediaType]; dup {
			g.error(fmt.Errorf("RequestBody: media type %q is given more than once: the first is kept", c.MediaType))
			continue
		}
		mt := &MediaType{}
		if c.Model != nil {
			mt.Schema = g.newSchemaFromModel(reflect.TypeOf(c.Model))
		}
		for _, e := range c.Encodings {
			if e == nil {
				continue
			}
			if !g.hasProperty(mt.Schema, e.Property) {
				g.warn(fmt.Sprintf("RequestBody: encoding of %q: the schema of %q has no such property", e.Property, c.MediaType))
			}
			if mt.Encoding == nil {
				mt.Encoding = make(map[string]*Encoding)
			}
			mt.Encoding[e.Property] = &Encoding{ContentType: e.ContentType}
		}
		m[c.MediaType] = mt
	}
	return m
}

// hasProperty reports whether the schema s, resolving a reference to a
// component, has the property name. A composed schema is not inspected
// and is assumed to have it.
func (g *Generator) hasProperty(s *Schema, name string) bool {
	if s != nil && s.Ref != "" {
		s = g.api.Components.Schemas[strings.TrimPrefix(s.Ref, componentsSchemaPath)]
	}
	if s == nil {
		return false
	}
	if len(s.AllOf)+len(s.AnyOf)+len(s.OneOf) > 0 {
		return true
	}
	_, ok := s.Properties[name]
	return ok
}

// wildcardNote is appended to the description of a path parameter that
// holds a gin catch-all ("*name") segment.
const wildcardNote = "The value may contain '/'."

// finishPathParams completes the path parameters of op once the input
// has been processed. A parameter declared in path ("{name}") but
// absent from the input (D10) is synthesized as a required string and
// a warning is recorded. Parameters named in wildcards also get the
// wildcardNote. The parameters are then sorted, if sorting is enabled.
func (g *Generator) finishPathParams(op *Operation, path string, wildcards []string) {
	// A path parameter that matches no template expression makes the
	// document invalid (OAS 3.2): drop it and record an error.
	op.Parameters = slices.DeleteFunc(op.Parameters, func(pr *ParameterOrRef) bool {
		v := pr.Value
		if v == nil || v.In != "path" || pathHasParam(path, v.Name) {
			return false
		}
		g.error(fmt.Errorf("path parameter %q is not in the path template %q: dropped", v.Name, path))
		return true
	})
	// Parameters is omitted when nil but written as [] when empty, so
	// an emptied list goes back to nil.
	if len(op.Parameters) == 0 {
		op.Parameters = nil
	}
	for _, m := range paramsInPathRe.FindAllStringSubmatch(path, -1) {
		name := m[1]
		var p *Parameter
		for _, pr := range op.Parameters {
			if v := pr.Value; v != nil && v.In == "path" && v.Name == name {
				p = v
				break
			}
		}
		if p == nil {
			p = &Parameter{
				Name:     name,
				In:       "path",
				Required: true,
				Schema:   &Schema{Type: Types("string")},
			}
			op.Parameters = append(op.Parameters, &ParameterOrRef{Value: p})
			g.warn(fmt.Sprintf("path parameter %q is declared in the path but missing from the input: synthesized as a required string", name))
		}
		if slices.Contains(wildcards, name) {
			p.Description = strings.TrimSpace(p.Description + " " + wildcardNote)
		}
	}
	// Sort operations parameters by location and name
	// in ascending order.
	if g.sortParams {
		slices.SortStableFunc(op.Parameters, g.compareParams)
	}
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
			n := len(op.Parameters)
			err := g.walkParams(op, et, active)
			delete(active, et)
			if err != nil {
				return err
			}
			// reflect cannot allocate an unexported embedded pointer, so
			// the binder leaves its fields alone: they are not documented.
			if sf.Type.Kind() == reflect.Pointer && !sf.IsExported() && len(op.Parameters) > n {
				op.Parameters = op.Parameters[:n]
				if n == 0 {
					op.Parameters = nil
				}
				g.error(&FieldError{
					Message:  "parameters promoted through an unexported embedded pointer cannot be bound; embed the struct by value or export the type",
					Name:     sf.Name,
					Field:    sf.Name,
					TypeName: g.typeName(t),
					Type:     t,
				})
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
				Field:             sf.Name,
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
	// A reference that does not resolve sorts last.
	if a == nil || b == nil {
		return cmp.Compare(boolInt(a == nil), boolInt(b == nil))
	}
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
	// A field whose type has no schema was reported: it is left out,
	// as flattenStructSchema does, rather than emitted as null.
	sfs, skipped := g.newSchemaFromField(f, required, t)
	if skipped {
		dropRequired(schema, f.name)
	} else if sfs != nil {
		schema.Properties[f.name] = sfs
	}
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
	fieldErr := func(msg, name string, typ reflect.Type, cause error) *FieldError {
		return &FieldError{
			Message:           msg,
			Name:              name,
			Field:             field.Name,
			TypeName:          g.typeName(t),
			Type:              typ,
			ParameterLocation: location,
			Parent:            t,
			Err:               cause,
		}
	}
	rawTag, _ := g.locationTag(field, location)
	name, tagRequired, tagDefault, err := ParseParamTag(rawTag)
	if err != nil {
		// A malformed tag is recorded and the field is skipped: it
		// must not fail the operation.
		g.error(fieldErr("invalid parameter tag", rawTag, field.Type, err))
		return nil, nil
	}
	// A name of "-" or no name is not a parameter.
	if name == "" || name == "-" {
		return nil, nil
	}
	if g.dropReservedHeader(location, name) {
		return nil, nil
	}
	// Cookies are scalar-only: no style or explode is emitted.
	if location == cookieLocation && !g.isScalar(field.Type) {
		g.error(fieldErr("cookie parameter must be a scalar type", name, field.Type, nil))
		return nil, nil
	}
	// Outside deepObject tonic binds each value with bindStringValue: a
	// map or a struct that is not text-bindable has no representation.
	if location != cookieLocation && !g.isDeepObject(field, location) && !g.isBindable(field.Type) {
		g.error(fieldErr(fmt.Sprintf("%s parameter must be a scalar type or a slice of scalars (a map or struct is only supported as a %s query parameter)", location, styleDeepObject), name, field.Type, nil))
		return nil, nil
	}
	deep := g.isDeepObject(field, location)
	if deep {
		// Tonic would bind the required and default options and tag
		// (handler.go:208-216), so they are errors. Required-ness comes
		// only from validate:"required".
		switch {
		case tagRequired:
			g.error(fieldErr(fmt.Sprintf("invalid %q tag: a %s parameter cannot have the \"required\" option, use the validate tag", BinderTags.Query, styleDeepObject), name, field.Type, nil))
			return nil, nil
		case tagDefault != "" || field.Tag.Get(BinderTags.Default) != "":
			g.error(fieldErr(fmt.Sprintf("a %s parameter cannot have a default value", styleDeepObject), name, field.Type, nil))
			return nil, nil
		}
	}
	required := tagRequired || g.isStructFieldRequired(field)

	// Path parameters are always required.
	isPath := location == BinderTags.Path
	if isPath {
		required = true
	}
	// The default option is the deprecated spelling of the default
	// tag. As in tonic's extractors, it wins over the tag.
	if tagDefault != "" {
		field.Tag = reflect.StructTag(BinderTags.Default + ":" + strconv.Quote(tagDefault) + " " + string(field.Tag))
	}
	// D15: a default makes a query or header parameter optional.
	// The warning is recorded with the field's schema.
	hasDefault := field.Tag.Get(BinderTags.Default) != ""
	schemaRequired := required && !isPath
	if schemaRequired && hasDefault {
		required = false
	}
	dep, _ := tagLookup(field, g.tags.Deprecated)
	desc, _ := tagLookup(field, g.tags.Description)
	// Consider invalid values as false.
	deprecated, _ := strconv.ParseBool(dep)

	p := &Parameter{
		Name:        name,
		In:          location,
		Description: desc,
		Required:    required,
		Deprecated:  deprecated,
	}
	if deep {
		var err error
		if p.Schema, err = g.deepObjectSchema(field.Type); err != nil {
			g.error(fieldErr("invalid parameter type", name, field.Type, err))
			return nil, nil
		}
	} else {
		p.Schema = g.fieldSchema(field, schemaRequired, name, t, valueForm{param: true})
	}
	if p.Schema == nil {
		g.error(fieldErr("unsupported parameter type", name, field.Type, nil))
		return nil, nil
	}
	// Style.
	if location == BinderTags.Query {
		if field.Type.Kind() == reflect.Slice || field.Type.Kind() == reflect.Array {
			p.Explode = new(true) // default
			p.Style = "form"      // default in spec, but make it obvious
			if v, ok := field.Tag.Lookup(BinderTags.Explode); ok {
				if explode, err := strconv.ParseBool(v); err != nil {
					// tonic ignores an invalid value: the default stays.
					g.error(fieldErr("invalid explode value", name, field.Type, err))
				} else if !explode {
					p.Explode = new(false)
				}
			}
		}
	}
	g.applyStyleTags(p, field, location, fieldErr)
	if g.parameterHook(t.Field(idx), p) {
		return nil, nil
	}
	return p, nil
}

// paramStyles lists the OAS 3.2 styles a parameter location accepts.
var paramStyles = map[string][]string{
	"query":        {"form", "spaceDelimited", "pipeDelimited", "deepObject"},
	"header":       {"simple"},
	"path":         {"simple", "label", "matrix"},
	cookieLocation: {"form", "cookie"},
}

// applyStyleTags applies the style and allowReserved tags of field to
// p. An invalid value is recorded and dropped.
func (g *Generator) applyStyleTags(p *Parameter, field reflect.StructField, location string, fieldErr func(msg, name string, typ reflect.Type, cause error) *FieldError) {
	if v, ok := tagLookup(field, g.tags.Style); ok && v != "" {
		allowed := paramStyles[location]
		switch {
		case !slices.Contains(allowed, v):
			g.error(fieldErr(fmt.Sprintf("invalid %q tag: style %q is not allowed in %s (want one of %s)",
				g.tags.Style, v, location, strings.Join(allowed, ", ")), p.Name, field.Type, nil))
		case v == "deepObject":
			// The object schema was built with the parameter
			// (deepObjectSchema). Only the exploded form is defined.
			p.Style = v
			p.Explode = new(true)
			if ev, ok := field.Tag.Lookup(BinderTags.Explode); ok {
				if b, err := strconv.ParseBool(ev); err == nil && !b {
					g.warn(fmt.Sprintf("parameter %q: explode=false is undefined for style %s, explode stays true", p.Name, styleDeepObject))
				}
			}
		case (v == "spaceDelimited" || v == "pipeDelimited") && !isArrayLike(field.Type):
			g.error(fieldErr(fmt.Sprintf("invalid %q tag: style %s needs a slice or array", g.tags.Style, v), p.Name, field.Type, nil))
		default:
			p.Style = v
			// An exploded delimited array is serialized as with
			// form: only the unexploded one uses the delimiter.
			if v == "spaceDelimited" || v == "pipeDelimited" {
				if _, set := field.Tag.Lookup(BinderTags.Explode); !set {
					p.Explode = new(false)
				}
			}
		}
	}
	if v, ok := tagLookup(field, g.tags.AllowReserved); ok && v != "" {
		b, err := strconv.ParseBool(v)
		switch {
		case err != nil:
			g.error(fieldErr(fmt.Sprintf("invalid %q tag: %q is not a boolean", g.tags.AllowReserved, v), p.Name, field.Type, err))
		case b && location != BinderTags.Query:
			g.error(fieldErr(fmt.Sprintf("invalid %q tag: allowed on query parameters only, not %s", g.tags.AllowReserved, location), p.Name, field.Type, nil))
		default:
			p.AllowReserved = b
		}
	}
}

// paramLocation parses the tags of the struct field to extract
// the location of an operation parameter.
func (g *Generator) paramLocation(f reflect.StructField, in reflect.Type) (string, error) {
	var c, p int

	// Count the number of keys that represents
	// a parameter location from the tag of the
	// struct field.
	parameterLocations := []string{
		BinderTags.Path,
		BinderTags.Query,
		BinderTags.Header,
		cookieLocation,
	}
	for i, n := range parameterLocations {
		if _, ok := g.locationTag(f, n); ok {
			c++
			// save name position to extract
			// the value of the unique key.
			p = i
		}
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
			Field:    f.Name,
			TypeName: g.typeName(in),
			Type:     in,
		}
	}
	return parameterLocations[p], nil
}

// resolveParameter returns either the inlined parameter
// in p or the one referenced in the API components.
func (g *Generator) resolveParameter(p *ParameterOrRef) *Parameter {
	if p.Ref != nil {
		return g.componentParameter(p.Ref.Ref)
	}
	return p.Value
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// cookieLocation is the "in" value of cookie parameters. The name of
// the struct tag that selects it is configurable: TagNames.Cookie.
const cookieLocation = "cookie"

// locationTag returns the value of the tag that selects the parameter
// location. The cookie tag is looked up under its configured name, and
// is absent when that name is disabled.
func (g *Generator) locationTag(sf reflect.StructField, location string) (string, bool) {
	if location == cookieLocation {
		return tagLookup(sf, g.tags.Cookie)
	}
	return sf.Tag.Lookup(location)
}

// isScalar reports whether t, dereferenced, is described by a schema
// of type string, number, integer or boolean. Byte slices, though
// described as strings, are not scalars.
func (g *Generator) isScalar(t reflect.Type) bool {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	switch t.Kind() {
	case reflect.Slice, reflect.Array, reflect.Map:
		return false
	}
	dt := g.datatype(t)
	if dt == TypeComplex {
		return false
	}
	switch dt.Type() {
	case "string", "number", "integer", "boolean":
		return true
	}
	return false
}

// reservedHeaders are the header names that OAS 3.2 says a header
// parameter SHALL NOT use: they are described by other objects.
var reservedHeaders = []string{"Accept", "Content-Type", "Authorization"}

// dropReservedHeader records a warning and reports true when the header
// parameter name is Accept, Content-Type or Authorization (compared
// case-insensitively). The parameter must not be emitted: the request
// content, the media types and the security schemes describe them.
func (g *Generator) dropReservedHeader(location, name string) bool {
	if location != InHeader {
		return false
	}
	for _, h := range reservedHeaders {
		if !strings.EqualFold(name, h) {
			continue
		}
		hint := "the media types of the request and response content describe it"
		if h == "Authorization" {
			hint = "use a security scheme instead"
		}
		g.warn(fmt.Sprintf("header parameter %q is reserved by OpenAPI and ignored: dropped; %s", name, hint))
		return true
	}
	return false
}

// isBindable reports whether tonic can bind a non-deepObject parameter of
// type t: a scalar (including types that are described as a string, such
// as time.Time or an encoding.TextUnmarshaler) or a slice or array of
// scalars. A map, a plain struct or a nested slice cannot be bound.
func (g *Generator) isBindable(t reflect.Type) bool {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if g.datatype(t) != TypeComplex {
		return true
	}
	if t.Kind() != reflect.Slice && t.Kind() != reflect.Array {
		return false
	}
	e := t.Elem()
	for e.Kind() == reflect.Pointer {
		e = e.Elem()
	}
	return g.datatype(e) != TypeComplex
}

// requestBodyName returns the component name of the synthesized request
// body of operation opID with input type t: the RequestBodyNamer result,
// or upperFirst(opID)+"Input", sanitized.
func (g *Generator) requestBodyName(opID string, t reflect.Type) string {
	if g.bodyNamer != nil {
		if name := sanitizeName(g.bodyNamer(opID, t)); name != "" {
			return name
		}
	}
	return sanitizeName(upperFirst(opID) + "Input")
}

// nameRequestBody moves the schema of the operation's request body to a
// component and replaces it with a reference.
//
// The required flag is set if any body field is required. When the
// input has no parameters and is, or only embeds, a single named struct,
// the body references that struct's component. Otherwise the body is
// registered as a component named by the RequestBodyNamer. The name is
// claimed in the registry for the input type t: if another type owns it,
// a *CollisionError is recorded and the schema stays inline.
func (g *Generator) nameRequestBody(op *Operation, t reflect.Type) {
	mt := g.mediaType()
	if mt == "" {
		mt = anyMediaType
	}
	media := op.RequestBody.Content[mt]
	if media == nil || media.Schema == nil {
		return
	}
	sch := media.Schema
	if len(sch.Required) > 0 {
		op.RequestBody.Required = true
	}
	if len(op.Parameters) == 0 {
		if st, ok := g.singleBodyStruct(t); ok {
			if ref := g.newSchemaFromType(st); ref != nil {
				media.Schema = ref
				return
			}
		}
	}
	name := g.requestBodyName(op.ID, t)
	kept, mine := g.registry.claim(name, t)
	if !mine {
		g.error(&CollisionError{Name: name, Kept: kept, Rejected: t})
		return
	}
	if _, exists := g.api.Components.Schemas[name]; !exists {
		g.api.Components.Schemas[name] = sch
	}
	media.Schema = &Schema{Ref: componentsSchemaPath + name}
}

// singleBodyStruct returns the named struct type that is the whole body
// of input type t: the type embedded, without a json name, as its only
// field. All the fields of that type must be body fields.
func (g *Generator) singleBodyStruct(t reflect.Type) (reflect.Type, bool) {
	if t.NumField() == 1 {
		sf := t.Field(0)
		et := sf.Type
		if et.Kind() == reflect.Pointer {
			et = et.Elem()
		}
		if sf.Anonymous && et.Kind() == reflect.Struct &&
			g.typeName(et) != "" && jsonName(sf) == "" {
			if n, ok := g.bodyFields(et); ok && n > 0 {
				return et, true
			}
			return nil, false
		}
	}
	return nil, false
}

// jsonName returns the name part of the json tag of sf.
func jsonName(sf reflect.StructField) string {
	name, _, _ := strings.Cut(sf.Tag.Get("json"), ",")
	return name
}

// bodyFields returns the number of fields of t selected for the body,
// and whether they are all body fields, with no location tag and no
// binding:"-".
func (g *Generator) bodyFields(t reflect.Type) (int, bool) {
	n := 0
	for f := range g.fieldsOf(t, mediaTags[g.mediaType()]) {
		loc, err := g.paramLocation(f.sf, t)
		if err != nil || loc != "" || f.sf.Tag.Get("binding") == "-" {
			return 0, false
		}
		n++
	}
	return n, true
}
