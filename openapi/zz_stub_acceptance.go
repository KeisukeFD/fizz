//go:build acceptance

// Stub declarations for symbols whose tasks are still open (ADR 0001 §5, §6).
// Each declaration is deleted by the task named in its FixedBy line.

package openapi

import (
	"errors"
	"reflect"
)

// Version is a stub.
// FixedBy: P2.6
const Version = ""

// Warning is a stub type.
// FixedBy: P3.3
type Warning struct {
	Method, Path, ID, Msg string
}

// Error is a stub.
// FixedBy: P3.3
func (w *Warning) Error() string {
	return ""
}

// IsWarning is a stub.
// FixedBy: P3.3
func (w *Warning) IsWarning() bool {
	return false
}

// AnnotationStyle is a stub type.
// FixedBy: P3.4
type AnnotationStyle int

// SiblingAnnotations is a stub.
// FixedBy: P3.4
const SiblingAnnotations AnnotationStyle = 0

// AllOfWrap is a stub.
// FixedBy: P3.4
const AllOfWrap AnnotationStyle = 1

// CollisionError is a stub type.
// FixedBy: P4.5
type CollisionError struct {
	Name     string
	Kept     reflect.Type
	Rejected reflect.Type
}

// Error is a stub.
// FixedBy: P4.5
func (e *CollisionError) Error() string {
	return ""
}

// OperationError is a stub type.
// FixedBy: P4.8
type OperationError struct {
	Method, Path, ID string
	Err              error
}

// Error is a stub.
// FixedBy: P4.8
func (e *OperationError) Error() string {
	return ""
}

// Unwrap is a stub.
// FixedBy: P4.8
func (e *OperationError) Unwrap() error {
	return nil
}

// Clone is a stub.
// FixedBy: P5.1
func (o *OpenAPI) Clone() (*OpenAPI, error) {
	return nil, errors.New("stub: P5.1")
}

// TagNames is a stub type.
// FixedBy: P6.1
type TagNames struct {
	Description, Example, Format, Deprecated, Title, Pattern, ReadOnly, WriteOnly, Nullable, MultipleOf, AdditionalProperties, Inline, Style, AllowReserved, Cookie string
}

// SchemaProvider is a stub type.
// FixedBy: P6.3
type SchemaProvider interface {
	OpenAPISchema() *Schema
}

// SchemaCustomizer is a stub type.
// FixedBy: P6.3
type SchemaCustomizer interface {
	CustomizeSchema(*Schema) error
}

// Enumer is a stub type.
// FixedBy: P6.3
type Enumer interface {
	Enum() []any
}

// SchemaContext is a stub type.
// FixedBy: P6.4
type SchemaContext struct {
	Type reflect.Type
	Name string
}

// FieldContext is a stub type.
// FixedBy: P6.4
type FieldContext struct {
	Parent      reflect.Type
	StructField reflect.StructField
	Name        string
}

// ParameterContext is a stub type.
// FixedBy: P6.4
type ParameterContext struct {
	OperationID, Method, Path, In string
	StructField                   reflect.StructField
}

// OperationContext is a stub type.
// FixedBy: P6.4
type OperationContext struct {
	OperationID, Method, Path string
}

// SchemaHook is a stub type.
// FixedBy: P6.4
type SchemaHook func(*SchemaContext, *Schema)

// FieldHook is a stub type.
// FixedBy: P6.4
type FieldHook func(*FieldContext, *Schema) (skip bool)

// ParameterHook is a stub type.
// FixedBy: P6.4
type ParameterHook func(*ParameterContext, *Parameter) (skip bool)

// OperationHook is a stub type.
// FixedBy: P6.4
type OperationHook func(*OperationContext, *Operation) error

// DocumentHook is a stub type.
// FixedBy: P6.4
type DocumentHook func(*OpenAPI) error

// Hooks is a stub type.
// FixedBy: P6.4
type Hooks struct {
	Schema    SchemaHook
	Field     FieldHook
	Parameter ParameterHook
	Operation OperationHook
	Document  DocumentHook
}

// SetHooks is a stub.
// FixedBy: P6.4
func (g *Generator) SetHooks(Hooks) {
}

// SetExtension is a stub.
// FixedBy: P6.5
func (g *Generator) SetExtension(key string, value any) error {
	return errors.New("stub: P6.5")
}

// RegisterSchema is a stub.
// FixedBy: P6.5
func (g *Generator) RegisterSchema(name string, s *Schema) error {
	return errors.New("stub: P6.5")
}

// RegisterParameter is a stub.
// FixedBy: P6.5
func (g *Generator) RegisterParameter(name string, p *Parameter) error {
	return errors.New("stub: P6.5")
}

// RegisterResponse is a stub.
// FixedBy: P6.5
func (g *Generator) RegisterResponse(name string, r *Response) error {
	return errors.New("stub: P6.5")
}

// RegisterExample is a stub.
// FixedBy: P6.5
func (g *Generator) RegisterExample(name string, e *Example) error {
	return errors.New("stub: P6.5")
}

// RegisterRequestBody is a stub.
// FixedBy: P6.5
func (g *Generator) RegisterRequestBody(name string, rb *RequestBody) error {
	return errors.New("stub: P6.5")
}

// RegisterHeader is a stub.
// FixedBy: P6.5
func (g *Generator) RegisterHeader(name string, h *Header) error {
	return errors.New("stub: P6.5")
}

// RegisterType is a stub.
// FixedBy: P6.5
func (g *Generator) RegisterType(t reflect.Type) error {
	return errors.New("stub: P6.5")
}

// SetTag is a stub.
// FixedBy: P6.5
func (g *Generator) SetTag(t *Tag) {
}

// SetTagGroups is a stub.
// FixedBy: P6.5
func (g *Generator) SetTagGroups(groups []*XTagGroup) {
}

// SetSelf is a stub.
// FixedBy: P6.5
func (g *Generator) SetSelf(uri string) {
}

// SetJSONSchemaDialect is a stub.
// FixedBy: P6.5
func (g *Generator) SetJSONSchemaDialect(uri string) {
}

// SetExternalDocs is a stub.
// FixedBy: P6.5
func (g *Generator) SetExternalDocs(d *ExternalDocs) {
}
