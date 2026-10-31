package openapi

import (
	"fmt"
	"reflect"
)

// FieldError is the error returned when an
// error related to a field occurs.
type FieldError struct {
	Name              string
	TypeName          string
	Type              reflect.Type
	Message           string
	ParameterLocation string
	Parent            reflect.Type
}

// Error implements the builtin error interface for FieldError.
func (fe *FieldError) Error() string {
	return fmt.Sprintf("%s: field=%s, type=%s", fe.Message, fe.Name, fe.TypeName)
}

// TypeError is the error returned when the generator
// encounters an unknow or unsupported type.
type TypeError struct {
	Message string
	Type    reflect.Type
}

// Error implements the builtin error interface for TypeError.
func (te *TypeError) Error() string {
	return fmt.Sprintf("%s: type=%s, kind=%s", te.Message, te.Type, te.Type.Kind())
}

// Warning is a non-fatal diagnostic recorded in Generator.Errors. The
// document is still generated. Method, Path and ID identify the
// operation being added when the warning arose; they are empty for
// warnings that do not belong to an operation.
type Warning struct {
	Method, Path, ID, Msg string
}

// Error implements the builtin error interface for Warning.
func (w *Warning) Error() string {
	var where string
	if w.Method != "" || w.Path != "" {
		where = w.Method + " " + w.Path
	}
	if w.ID != "" {
		if where != "" {
			where += " "
		}
		where += "(" + w.ID + ")"
	}
	if where == "" {
		return "warning: " + w.Msg
	}
	return "warning: " + where + ": " + w.Msg
}

// IsWarning reports that w is a warning rather than a failure.
func (w *Warning) IsWarning() bool { return true }
