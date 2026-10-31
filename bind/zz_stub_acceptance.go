//go:build acceptance

// Stub declarations for symbols whose tasks are still open (ADR 0001 §5, §6).
// Each declaration is deleted by the task named in its FixedBy line.

package bind

import (
	"errors"
	"net/http"
	"reflect"
)

// Tags is a stub type.
// FixedBy: P7.2a
type Tags struct {
	Query, Cookie, Style, Default string
}

// Plan is a stub type.
// FixedBy: P7.2a
type Plan struct{}

// PlanFor is a stub.
// FixedBy: P7.2a
func PlanFor(t reflect.Type, tags Tags) (*Plan, error) {
	return nil, errors.New("stub: P7.2a")
}

// QueryOption is a stub type.
// FixedBy: P7.2a
type QueryOption func(*queryConfig)

// queryConfig is a stub type.
// FixedBy: P7.2a
type queryConfig struct{}

// WithMaxKeys is a stub.
// FixedBy: P7.2a
func WithMaxKeys(n int) QueryOption {
	return func(*queryConfig) {}
}

// Query is a stub.
// FixedBy: P7.2a
func Query(r *http.Request, dst any, p *Plan, opts ...QueryOption) error {
	return errors.New("stub: P7.2a")
}

// ErrNestedKey is a stub.
// FixedBy: P7.2a
var ErrNestedKey = errors.New("stub: P7.2a")

// ErrTooManyKeys is a stub.
// FixedBy: P7.2a
var ErrTooManyKeys = errors.New("stub: P7.2a")

// Cookies is a stub.
// FixedBy: P7.2c
func Cookies(r *http.Request, dst any, p *Plan) error {
	return errors.New("stub: P7.2c")
}
