package openapi

import (
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"regexp"
	"sort"
	"strconv"

	"github.com/gofrs/uuid"
	"github.com/loopfz/gadgeto/tonic"
)

const (
	version              = "3.0.1"
	anyMediaType         = "*/*"
	formatTag            = "format"
	deprecatedTag        = "deprecated"
	descriptionTag       = "description"
	componentsSchemaPath = "#/components/schemas/"
)

var (
	paramsInPathRe = regexp.MustCompile(`\{(.*?)\}`)
	ginPathParamRe = regexp.MustCompile(`\/:([^\/]*)`)
)

// mediaTags maps media types to well-known
// struct tags used for marshaling.
var mediaTags = map[string]string{
	"application/json": "json",
	"application/xml":  "xml",
}

// Generator is an OpenAPI 3 generator.
type Generator struct {
	api           *OpenAPI
	config        *SpecGenConfig
	schemaTypes   map[reflect.Type]struct{}
	typeNames     map[reflect.Type]string
	dataTypes     map[reflect.Type]*OverridedDataType
	operationsIDS map[string]struct{}
	errors        []error
	fullNames     bool
	sortParams    bool
	sortTags      bool
}

// NewGenerator returns a new OpenAPI generator.
func NewGenerator(conf *SpecGenConfig) (*Generator, error) {
	if conf == nil {
		return nil, errors.New("missing config")
	}
	components := &Components{
		Schemas:    make(map[string]*SchemaOrRef),
		Responses:  make(map[string]*ResponseOrRef),
		Parameters: make(map[string]*ParameterOrRef),
		Headers:    make(map[string]*HeaderOrRef),
	}
	return &Generator{
		config: conf,
		api: &OpenAPI{
			OpenAPI:    version,
			Info:       &Info{},
			Paths:      make(Paths),
			Components: components,
		},
		schemaTypes:   make(map[reflect.Type]struct{}),
		typeNames:     make(map[reflect.Type]string),
		dataTypes:     make(map[reflect.Type]*OverridedDataType),
		operationsIDS: make(map[string]struct{}),
		fullNames:     true,
		sortParams:    true,
		sortTags:      true,
	}, nil
}

// SpecGenConfig represents the configuration
// of the spec generator.
type SpecGenConfig struct {
	// Name of the tag used by the validator.v8
	// package. This is used by the spec generator
	// to determine if a field is required.
	ValidatorTag      string
	PathLocationTag   string
	QueryLocationTag  string
	HeaderLocationTag string
	EnumTag           string
	DefaultTag        string
}

// SetInfo uses the given OpenAPI info for the
// current specification.
func (g *Generator) SetInfo(info *Info) {
	g.api.Info = info
}

// SetServers sets the server list for the
// current specification.
func (g *Generator) SetServers(servers []*Server) {
	g.api.Servers = servers
}

// SetSecurityRequirement sets the security options for the
// current specification.
func (g *Generator) SetSecurityRequirement(security []*SecurityRequirement) {
	g.api.Security = security
}

// SetSecuritySchemes sets the security schemes that can be used
// inside the operations of the specification.
func (g *Generator) SetSecuritySchemes(security map[string]*SecuritySchemeOrRef) {
	g.api.Components.SecuritySchemes = security
}

// API returns a copy of the internal OpenAPI object.
func (g *Generator) API() *OpenAPI {
	cpy := *g.api
	return &cpy
}

// Errors returns the errors thar occurred during
// the generation of the specification.
func (g *Generator) Errors() []error {
	return g.errors
}

func (g *Generator) error(err error) {
	g.errors = append(g.errors, err)
}

// UseFullSchemaNames defines whether the generator should generates
// a full name for the components using the package name of the type
// as a prefix.
// Omitting the package part of the name increases the risks of conflicts.
// It is the responsibility of the developper to ensure that unique type
// names are used across all the packages of the application.
// Default to true.
func (g *Generator) UseFullSchemaNames(b bool) {
	g.fullNames = b
}

// SetSortParams controls whether the generator should
// sort the parameters of an operation by location and
// name in ascending order.
func (g *Generator) SetSortParams(b bool) {
	g.sortParams = b
}

// SetSortTags controls whether the generator should
// sort the global tags sections.
func (g *Generator) SetSortTags(b bool) {
	g.sortTags = b
}

// OverrideTypeName registers a custom name for a
// type that will override the default generation
// and have precedence over types that implements
// the Typer interface.
func (g *Generator) OverrideTypeName(t reflect.Type, name string) error {
	if name == "" {
		return errors.New("type name is empty")
	}
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if _, ok := g.typeNames[t]; ok {
		return errors.New("type name already overrided")
	}
	g.typeNames[t] = name

	return nil
}

// OverrideDataType registers a custom schema type and
// format for the given type that will overrided the
// default generation.
func (g *Generator) OverrideDataType(t reflect.Type, typ, format string) error {
	if typ == "" {
		return errors.New("type is mandatory")
	}
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if _, ok := g.dataTypes[t]; ok {
		return errors.New("data type already overrided")
	}
	g.dataTypes[t] = &OverridedDataType{
		format: format,
		typ:    typ,
	}
	return nil
}

func (g *Generator) datatype(t reflect.Type) DataType {
	if dt, ok := g.dataTypes[t]; ok {
		return dt
	}
	return DataTypeFromType(t)
}

// AddTag adds a new tag to the OpenAPI specification.
// If a tag already exists with the same name, it is
// overwritten.
func (g *Generator) AddTag(name, desc string) {
	if name == "" {
		return
	}
	// Search for an existing tag with the same name,
	// and update its description before returning
	// if one is found.
	for _, tag := range g.api.Tags {
		if tag != nil {
			if tag.Name == name {
				tag.Description = desc
				return
			}
		}
	}
	// Add a new tag to the spec.
	g.api.Tags = append(g.api.Tags, &Tag{
		Name:        name,
		Description: desc,
	})
	if g.sortTags {
		sort.SliceStable(g.api.Tags, func(i, j int) bool {
			if g.api.Tags[i] != nil && g.api.Tags[j] != nil {
				return g.api.Tags[i].Name < g.api.Tags[j].Name
			}
			return false
		})
	}
}

// AddOperation add a new operation to the OpenAPI specification
// using the method and path of the route and the tonic
// handler informations.
func (g *Generator) AddOperation(path, method, tag string, in, out reflect.Type, info *OperationInfo) (*Operation, error) {
	op := &Operation{
		ID: uuid.Must(uuid.NewV4()).String(),
	}
	path = rewritePath(path)

	if info != nil {
		// Ensure that the provided operation ID is unique.
		if _, ok := g.operationsIDS[info.ID]; ok {
			return nil, fmt.Errorf("ID %s is already used by another operation", info.ID)
		}
		g.operationsIDS[info.ID] = struct{}{}
	}
	// If a PathItem does not exists for this
	// path, create a new one.
	item, ok := g.api.Paths[path]
	if !ok {
		item = new(PathItem)
		g.api.Paths[path] = item
	}
	// Create a new operation and set it
	// to the according method of the PathItem.
	if info != nil {
		op.ID = info.ID
		op.Summary = info.Summary
		op.Description = info.Description
		op.Deprecated = info.Deprecated
		op.Responses = make(Responses)
		op.XCodeSamples = info.XCodeSamples
		op.Security = info.Security
		op.XInternal = info.XInternal
	}
	if tag != "" {
		op.Tags = append(op.Tags, tag)
	}
	// Operations with methods GET/HEAD/DELETE cannot have a body.
	// Non parameters fields will be ignored.
	allowBody := method != http.MethodGet &&
		method != http.MethodHead &&
		method != http.MethodDelete

	if in != nil {
		if in.Kind() == reflect.Pointer {
			in = in.Elem()
		}
		if in.Kind() != reflect.Struct {
			return nil, errors.New("input type is not a struct")
		}
		if err := g.setOperationParams(op, in, in, allowBody, path); err != nil {
			return nil, err
		}
	}
	// Generate the default response from the tonic
	// handler return type. If the handler has no output
	// type, the response won't have a schema.
	if err := g.setOperationResponse(op, out, strconv.Itoa(info.StatusCode), tonic.MediaType(), info.StatusDescription, info.Headers, nil, info.DefaultResponseExamples); err != nil {
		return nil, err
	}
	// Generate additional responses from the operation
	// informations.
	for _, resp := range info.Responses {
		if resp != nil {
			if err := g.setOperationResponse(op,
				reflect.TypeOf(resp.Model),
				resp.Code,
				tonic.MediaType(),
				resp.Description,
				resp.Headers,
				resp.Example,
				resp.Examples,
			); err != nil {
				return nil, err
			}
		}
	}
	setOperationBymethod(item, op, method)

	return op, nil
}

// rewritePath converts a Gin operation path that use
// colons and asterisks to declare path parameters, to
// an OpenAPI representation that use curly braces.
func rewritePath(path string) string {
	return ginPathParamRe.ReplaceAllString(path, "/{$1}")
}

// setOperationBymethod sets the operation op to the appropriate
// field of item according to the given method.
func setOperationBymethod(item *PathItem, op *Operation, method string) {
	switch method {
	case "GET":
		item.GET = op
	case "PUT":
		item.PUT = op
	case "POST":
		item.POST = op
	case "PATCH":
		item.PATCH = op
	case "HEAD":
		item.HEAD = op
	case "OPTIONS":
		item.OPTIONS = op
	case "TRACE":
		item.TRACE = op
	case "DELETE":
		item.DELETE = op
	}
}
