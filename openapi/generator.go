package openapi

import (
	"errors"
	"fmt"
	"maps"
	"net/http"
	"reflect"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
)

// Version is the OpenAPI Specification version of the generated documents.
const Version = "3.2.1"

const (
	anyMediaType         = "*/*"
	componentsSchemaPath = "#/components/schemas/"
)

var (
	paramsInPathRe = regexp.MustCompile(`\{(.*?)\}`)
)

// mediaTags maps media types to well-known
// struct tags used for marshaling.
var mediaTags = map[string]string{
	"application/json": "json",
	"application/xml":  "xml",
}

// Generator is an OpenAPI 3 generator.
//
// A Generator is safe for concurrent use. The methods that change the
// document or the way it is generated hold mu and bump gen; the
// readers (API, Errors, OperationIDUsed, Snapshot) take mu too, or
// read gen atomically. The fields below are guarded by mu.
type Generator struct {
	mu  sync.Mutex
	gen atomic.Uint64 // generation of the document, bumped by every mutator
	// cloneErrGen is 1 + the generation for which a clone failure has
	// been recorded in errors, 0 if none, so that it is recorded once.
	cloneErrGen uint64

	api           *OpenAPI
	config        *SpecGenConfig
	schemaTypes   map[reflect.Type]struct{}
	typeNames     map[reflect.Type]string
	dataTypes     map[reflect.Type]*OverridedDataType
	operationsIDS map[string]struct{}
	fieldCache    map[typeFieldsKey][]field
	stack         []reflect.Type        // named composites being built, outermost first
	promoted      map[reflect.Type]bool // named composites promoted to components
	errors        []error
	warned        map[reflect.Type]bool // types already warned about
	cur           *opContext            // operation being added, if any
	fullNames     bool
	sortParams    bool
	sortTags      bool

	pointersNullable bool
	annotationStyle  AnnotationStyle
	tags             TagNames
	schemaNamer      func(reflect.Type) string
	bodyNamer        func(opID string, t reflect.Type) string
	registry         registry
	hooks            Hooks
	inlining         map[reflect.Type]bool // collision losers being inlined
}

// NewGenerator returns a new OpenAPI generator.
func NewGenerator(conf *SpecGenConfig) (*Generator, error) {
	if conf == nil {
		return nil, errors.New("missing config")
	}
	if err := conf.checkBinderTags(); err != nil {
		return nil, err
	}
	components := &Components{
		Schemas:    make(map[string]*Schema),
		Responses:  make(map[string]*ResponseOrRef),
		Parameters: make(map[string]*ParameterOrRef),
		Headers:    make(map[string]*HeaderOrRef),
	}
	return &Generator{
		config: conf,
		api: &OpenAPI{
			OpenAPI:    Version,
			Info:       &Info{},
			Paths:      make(Paths),
			Components: components,
		},
		schemaTypes:   make(map[reflect.Type]struct{}),
		promoted:      make(map[reflect.Type]bool),
		warned:        make(map[reflect.Type]bool),
		typeNames:     make(map[reflect.Type]string),
		dataTypes:     make(map[reflect.Type]*OverridedDataType),
		operationsIDS: make(map[string]struct{}),
		fullNames:     true,
		sortParams:    true,
		sortTags:      true,

		pointersNullable: true,
		tags:             defaultTagNames,
	}, nil
}

// SpecGenConfig represents the configuration
// of the spec generator.
type SpecGenConfig struct {
	// Name of the tag used by the validator.v8
	// package. This is used by the spec generator
	// to determine if a field is required.
	//
	// Deprecated: binder tags are fixed to openapi.BinderTags (tonic's names); a non-empty different value is an error.
	ValidatorTag string
	// Deprecated: binder tags are fixed to openapi.BinderTags (tonic's names); a non-empty different value is an error.
	PathLocationTag string
	// Deprecated: binder tags are fixed to openapi.BinderTags (tonic's names); a non-empty different value is an error.
	QueryLocationTag string
	// Deprecated: binder tags are fixed to openapi.BinderTags (tonic's names); a non-empty different value is an error.
	HeaderLocationTag string
	// Deprecated: binder tags are fixed to openapi.BinderTags (tonic's names); a non-empty different value is an error.
	EnumTag string
	// Deprecated: binder tags are fixed to openapi.BinderTags (tonic's names); a non-empty different value is an error.
	DefaultTag string
	// DefaultMediaType is the media type of request and
	// response bodies. Empty means "application/json".
	DefaultMediaType string
}

// checkBinderTags reports the first binder tag field, in declaration
// order, that is neither empty nor equal to the fixed tonic name.
func (c *SpecGenConfig) checkBinderTags() error {
	for _, f := range []struct{ name, got, want string }{
		{"ValidatorTag", c.ValidatorTag, BinderTags.Validate},
		{"PathLocationTag", c.PathLocationTag, BinderTags.Path},
		{"QueryLocationTag", c.QueryLocationTag, BinderTags.Query},
		{"HeaderLocationTag", c.HeaderLocationTag, BinderTags.Header},
		{"EnumTag", c.EnumTag, BinderTags.Enum},
		{"DefaultTag", c.DefaultTag, BinderTags.Default},
	} {
		if f.got != "" && f.got != f.want {
			return fmt.Errorf("openapi: SpecGenConfig.%s = %q: binder tags are fixed to %q", f.name, f.got, f.want)
		}
	}
	return nil
}

// mediaType returns the media type of request and response bodies.
func (g *Generator) mediaType() string {
	if g.config.DefaultMediaType == "" {
		return "application/json"
	}
	return g.config.DefaultMediaType
}

// SetInfo uses the given OpenAPI info for the
// current specification.
func (g *Generator) SetInfo(info *Info) {
	defer g.mutate()()
	g.api.Info = info
}

// SetServers sets the server list for the
// current specification.
func (g *Generator) SetServers(servers []*Server) {
	defer g.mutate()()
	g.api.Servers = servers
}

// SetSecurityRequirement sets the security options for the
// current specification.
func (g *Generator) SetSecurityRequirement(security []*SecurityRequirement) {
	defer g.mutate()()
	g.api.Security = security
}

// SetSecuritySchemes sets the security schemes that can be used
// inside the operations of the specification. A scheme that has
// neither a $ref nor a value cannot be emitted: it is dropped and an
// error wrapping ErrNilOrRef that names the scheme is recorded.
func (g *Generator) SetSecuritySchemes(security map[string]*SecuritySchemeOrRef) {
	defer g.mutate()()
	valid := make(map[string]*SecuritySchemeOrRef, len(security))
	for _, name := range slices.Sorted(maps.Keys(security)) {
		s := security[name]
		if s == nil || (s.Ref == nil && s.Value == nil) {
			g.error(fmt.Errorf("security scheme %q: %w", name, ErrNilOrRef))
			continue
		}
		valid[name] = s
	}
	g.api.Components.SecuritySchemes = valid
}

// API returns a deep copy of the internal OpenAPI object: changing it
// affects neither the generator nor the served document. If the copy
// fails (see OpenAPI.Clone) API returns nil and records the error, once
// per generation of the document.
func (g *Generator) API() *OpenAPI {
	api, _, err := g.Snapshot()
	if err != nil {
		return nil
	}
	return api
}

// Generation returns the generation of the document: a counter that
// every change to the generator (the document or the way it is
// generated) increments. Recording an error does not change it.
func (g *Generator) Generation() uint64 { return g.gen.Load() }

// Snapshot returns a deep copy of the document taken under the lock of
// the generator, and the generation it was taken at. A failure to copy
// it is returned, and recorded in Errors once per generation.
func (g *Generator) Snapshot() (*OpenAPI, uint64, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.cloneLocked()
}

// cloneLocked is Snapshot for a caller that holds mu.
func (g *Generator) cloneLocked() (*OpenAPI, uint64, error) {
	gen := g.gen.Load()
	api, err := g.api.Clone()
	if err != nil {
		err = fmt.Errorf("copying the document: %w", err)
		if g.cloneErrGen != gen+1 {
			g.cloneErrGen = gen + 1
			g.error(err)
		}
		return nil, gen, err
	}
	return api, gen, nil
}

// mutate locks the generator for a change. The returned function bumps
// the generation and unlocks: defer g.mutate()().
func (g *Generator) mutate() func() {
	g.mu.Lock()
	return func() {
		g.gen.Add(1)
		g.mu.Unlock()
	}
}

// Errors returns a copy of the errors and warnings recorded during the
// generation of the specification, in registration order. While an
// operation is being added, the errors other than warnings are wrapped
// in an *OperationError that names it.
func (g *Generator) Errors() []error {
	g.mu.Lock()
	defer g.mu.Unlock()
	return slices.Clone(g.errors)
}

// RecordError appends err to the errors of the generator, as an error
// found by the caller (a router, a handler constructor) would be. It
// is not attached to any operation: wrap err in an *OperationError to
// do so. A nil err is ignored.
func (g *Generator) RecordError(err error) {
	if err != nil {
		g.mu.Lock()
		defer g.mu.Unlock()
		g.error(err)
	}
}

// error records err; the caller holds mu. While an operation is being added, an error that
// is neither a *Warning (which carries its own context) nor already an
// *OperationError is wrapped in one, so that every error identifies its
// operation and errors.As still finds the cause.
func (g *Generator) error(err error) {
	if g.cur != nil {
		switch err.(type) {
		case *Warning, *OperationError:
		default:
			err = &OperationError{Method: g.cur.method, Path: g.cur.path, ID: g.cur.id, Err: err}
		}
	}
	g.errors = append(g.errors, err)
}

// opContext identifies the operation being added to the generator.
type opContext struct{ method, path, id string }

// warn records a *Warning in the error list. It carries the method,
// path and ID of the operation being added, if any.
func (g *Generator) warn(msg string) {
	w := &Warning{Msg: msg}
	if g.cur != nil {
		w.Method, w.Path, w.ID = g.cur.method, g.cur.path, g.cur.id
	}
	g.error(w)
}

// jsonEncodable reports whether v encodes as JSON with the options of
// MarshalJSON. A document holding a value that does not (NaN, a
// channel, a function...) could not be served, so user examples are
// checked when they are registered.
func jsonEncodable(v any) error {
	_, err := marshalJSON(v)
	return err
}

// UseFullSchemaNames defines whether the generator should generates
// a full name for the components using the package name of the type
// as a prefix.
// Omitting the package part of the name increases the risks of conflicts.
// It is the responsibility of the developper to ensure that unique type
// names are used across all the packages of the application.
// Default to true.
func (g *Generator) UseFullSchemaNames(b bool) {
	defer g.mutate()()
	g.fullNames = b
}

// SetSortParams controls whether the generator should
// sort the parameters of an operation by location and
// name in ascending order.
func (g *Generator) SetSortParams(b bool) {
	defer g.mutate()()
	g.sortParams = b
}

// SetPointersAreNullable controls whether a pointer type makes its
// schema nullable. The Nullable interface applies regardless.
// Default to true.
func (g *Generator) SetPointersAreNullable(b bool) {
	defer g.mutate()()
	g.pointersNullable = b
}

// AnnotationStyle selects how use-site annotations (description,
// default, enum, deprecated, validation, ...) are attached to a
// schema that is a $ref.
type AnnotationStyle int

const (
	// SiblingAnnotations puts the annotations next to the $ref, as
	// OpenAPI 3.1+ allows. It is the default.
	SiblingAnnotations AnnotationStyle = iota
	// AllOfWrap emits {"allOf":[{"$ref":...}], <annotations>}
	// instead, for tools that ignore $ref siblings. A nullable
	// reference is an anyOf wrapper and keeps its annotations beside
	// the anyOf in both styles.
	AllOfWrap
)

// SetAnnotationStyle selects the style of use-site annotations on
// references. Default to SiblingAnnotations.
func (g *Generator) SetAnnotationStyle(s AnnotationStyle) {
	defer g.mutate()()
	g.annotationStyle = s
}

// SetSortTags controls whether the generator should
// sort the global tags sections.
func (g *Generator) SetSortTags(b bool) {
	defer g.mutate()()
	g.sortTags = b
}

// OverrideTypeName registers a custom name for a
// type that will override the default generation
// and have precedence over types that implements
// the Typer interface.
func (g *Generator) OverrideTypeName(t reflect.Type, name string) error {
	defer g.mutate()()
	if name == "" {
		return errors.New("type name is empty")
	}
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if _, ok := g.typeNames[t]; ok {
		return errors.New("type name already overrided")
	}
	if owner, ok := g.registry.owner(sanitizeName(name)); ok && owner != t {
		return fmt.Errorf("type name %q is already used by %s", name, owner)
	}
	g.typeNames[t] = name

	return nil
}

// OverrideDataType registers a custom schema type and
// format for the given type that will overrided the
// default generation.
func (g *Generator) OverrideDataType(t reflect.Type, typ, format string) error {
	defer g.mutate()()
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

// AddTag adds a tag to the OpenAPI specification. If a tag with the
// same name exists, an empty description leaves it unchanged, a
// description on a tag that has none fills it in, and a different
// non-empty description is rejected: the first one is kept and a
// *Warning naming the tag is recorded. SetTag replaces a tag outright.
func (g *Generator) AddTag(name, desc string) {
	defer g.mutate()()
	if name == "" {
		return
	}
	for _, tag := range g.api.Tags {
		if tag == nil || tag.Name != name {
			continue
		}
		switch {
		case desc == "" || tag.Description == desc:
		case tag.Description == "":
			tag.Description = desc
		default:
			g.warn(fmt.Sprintf("tag %q: description %q ignored, the first description %q is kept", name, desc, tag.Description))
		}
		return
	}
	g.appendTag(&Tag{Name: name, Description: desc})
}

// appendTag adds a new tag to the document, sorted by name if enabled;
// the caller holds mu.
func (g *Generator) appendTag(t *Tag) {
	g.api.Tags = append(g.api.Tags, t)
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
// using the method and OpenAPI-syntax path ("/a/{id}") of the
// route and the handler informations. A nil info is a zero OperationInfo.
//
// The operation is built first and committed (ID reserved, PathItem
// created, operation set) only if that succeeds. On failure an
// *OperationError is recorded in Errors and returned, and nothing the
// operation created remains in the generator: see snapshot.
// Problems limited to a field or a parameter are recorded and do not
// fail the operation.
func (g *Generator) AddOperation(path, method, tag string, in, out reflect.Type, info *OperationInfo) (*Operation, error) {
	if info == nil {
		info = &OperationInfo{}
	}
	method = strings.ToUpper(method)
	defer g.mutate()()
	g.cur = &opContext{method: method, path: path, id: info.ID}
	defer func() { g.cur = nil }()

	snap := g.snapshot()
	op, err := g.buildOperation(path, method, tag, in, out, info)
	if err != nil {
		snap.restore(g)
		err = &OperationError{Method: method, Path: path, ID: info.ID, Err: err}
		g.error(err)
		return nil, err
	}
	// Commit. Nothing below can fail.
	if info.ID != "" {
		g.operationsIDS[info.ID] = struct{}{}
	}
	item, ok := g.api.Paths[path]
	if !ok {
		item = new(PathItem)
		g.api.Paths[path] = item
	}
	setOperationBymethod(item, op, method)

	return op, nil
}

// OperationIDUsed reports whether id is already the ID of a documented
// operation. The router uses it to choose among the candidate default
// IDs of an operation.
func (g *Generator) OperationIDUsed(id string) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	_, ok := g.operationsIDS[id]
	return ok
}

// buildOperation builds the operation without touching the operation
// IDs or the paths. It still writes the component schemas, the type
// registry and their caches, which AddOperation rolls back on failure.
func (g *Generator) buildOperation(path, method, tag string, in, out reflect.Type, info *OperationInfo) (*Operation, error) {
	// Ensure that the provided operation ID is unique.
	if info.ID != "" {
		if _, ok := g.operationsIDS[info.ID]; ok {
			return nil, fmt.Errorf("ID %s is already used by another operation", info.ID)
		}
	}
	op := &Operation{
		ID:           info.ID,
		Summary:      info.Summary,
		Description:  info.Description,
		Deprecated:   info.Deprecated,
		Responses:    make(Responses),
		XCodeSamples: info.XCodeSamples,
		Security:     info.Security,
		XInternal:    info.XInternal,
	}
	if tag != "" {
		op.Tags = append(op.Tags, tag)
	}
	for _, t := range info.Tags {
		if t != "" && !slices.Contains(op.Tags, t) {
			op.Tags = append(op.Tags, t)
		}
	}
	// Operations with methods GET/HEAD cannot have a body (DELETE
	// can, D13). Non parameters fields will be ignored.
	allowBody := method != http.MethodGet &&
		method != http.MethodHead

	// The contents of the RequestBody option replace the inferred body,
	// which is then not built, so that it leaves no component behind.
	hasContents := info.RequestBody != nil && len(info.RequestBody.Contents) > 0

	if in != nil {
		if in.Kind() == reflect.Pointer {
			in = in.Elem()
		}
		if in.Kind() != reflect.Struct {
			return nil, errors.New("input type is not a struct")
		}
		if err := g.setOperationParams(op, in, in, allowBody && !hasContents); err != nil {
			return nil, err
		}
	}
	g.finishPathParams(op, path, info.WildcardParams)
	// Generate the default response from the tonic
	// handler return type. If the handler has no output
	// type, the response won't have a schema.
	if err := g.setOperationResponse(op, "", out, strconv.Itoa(info.StatusCode), g.mediaType(), info.StatusDescription, info.Headers, nil, info.DefaultResponseExamples); err != nil {
		return nil, err
	}
	// Generate additional responses from the operation
	// informations.
	for _, resp := range info.Responses {
		if resp != nil {
			if err := g.addOperationResponse(op, resp); err != nil {
				return nil, err
			}
		}
	}
	g.applyRequestBody(op, method, info.RequestBody)
	g.applyParameters(op, path, info.Parameters)
	g.applyOperationOptions(op, info)
	for _, fn := range info.Customizers {
		if fn != nil {
			op = g.customizeOperation("CustomizeOperation", op, fn)
		}
	}
	return g.operationHook(op, method, path), nil
}

// snapshot is the generator state that building an operation may
// extend: the component schemas and the registries that decide which
// schemas exist. It is shallow, since entries are only added or
// replaced, never mutated in place, once registered. The error list is
// deliberately not part of it: diagnostics recorded for a failed
// operation stay visible, and the maps that deduplicate them
// (warned, registry.reported) are therefore kept too. typeNames holds
// only the overrides of OverrideTypeName, which no operation writes.
type snapshot struct {
	schemas     map[string]*Schema
	names       map[string]reflect.Type
	schemaTypes map[reflect.Type]struct{}
	promoted    map[reflect.Type]bool
}

func (g *Generator) snapshot() snapshot {
	return snapshot{
		schemas:     maps.Clone(g.api.Components.Schemas),
		names:       maps.Clone(g.registry.names),
		schemaTypes: maps.Clone(g.schemaTypes),
		promoted:    maps.Clone(g.promoted),
	}
}

// restore puts the generator back in the state of s. The schema map is
// restored in place.
func (s snapshot) restore(g *Generator) {
	clear(g.api.Components.Schemas)
	maps.Copy(g.api.Components.Schemas, s.schemas)
	g.registry.names = s.names
	g.schemaTypes = s.schemaTypes
	g.promoted = s.promoted
}

// setOperationBymethod sets the operation op to the appropriate
// field of item according to the given method.
// The method is matched case-insensitively. A method without a field of
// its own is stored in AdditionalOperations, so no operation is dropped.
func setOperationBymethod(item *PathItem, op *Operation, method string) {
	method = strings.ToUpper(method)
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
	case "QUERY":
		item.Query = op
	default:
		if item.AdditionalOperations == nil {
			item.AdditionalOperations = make(map[string]*Operation)
		}
		item.AdditionalOperations[method] = op
	}
}

// SetRequestBodyNamer sets the function that names the component
// schema of a synthesized request body from the operation ID and the
// input type. Nil selects the default, the operation ID with its first
// letter upper-cased, followed by "Input".
func (g *Generator) SetRequestBodyNamer(fn func(opID string, t reflect.Type) string) {
	defer g.mutate()()
	g.bodyNamer = fn
}
