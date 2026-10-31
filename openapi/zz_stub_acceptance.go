//go:build acceptance

// Stub declarations for symbols whose tasks are still open (ADR 0001 §5, §6).
// Each declaration is deleted by the task named in its FixedBy line.

package openapi

import "reflect"

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
