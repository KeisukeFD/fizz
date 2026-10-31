//go:build acceptance

// Stub declarations for symbols whose tasks are still open (ADR 0001 §5, §6).
// Each declaration is deleted by the task named in its FixedBy line.

package fizz

import (
	"errors"
	"github.com/loopfz/gadgeto/tonic"
	"github.com/wI2L/fizz/docsui"
	"github.com/wI2L/fizz/openapi"
)

// WithHooks is a stub.
// FixedBy: P6.4
func WithHooks(openapi.Hooks) Option {
	return func(*config) {}
}

// Document is a stub.
// FixedBy: P6.4
func (f *Fizz) Document() (*openapi.OpenAPI, error) {
	return nil, errors.New("stub: P6.4")
}

// EmbeddedSwaggerUI is a stub.
// FixedBy: P8.3
func EmbeddedSwaggerUI(opts ...docsui.Option) DocsUI {
	return DocsUI{}
}

// EmbeddedScalarUI is a stub.
// FixedBy: P8.3
func EmbeddedScalarUI(opts ...docsui.Option) DocsUI {
	return DocsUI{}
}

// BindHook is a stub.
// FixedBy: P7.2b
func BindHook(next tonic.BindHook) tonic.BindHook {
	return nil
}

// InstallBindHook is a stub.
// FixedBy: P7.2b
func InstallBindHook() {
}

// ContentOption is a stub type.
// FixedBy: P6.6a
type ContentOption func(*contentSpec)

// contentSpec is a stub type.
// FixedBy: P6.6a
type contentSpec struct{}

// EncodingOption is a stub type.
// FixedBy: P6.6a
type EncodingOption func(*encodingSpec)

// encodingSpec is a stub type.
// FixedBy: P6.6a
type encodingSpec struct{}

// RequestBody is a stub.
// FixedBy: P6.6a
func RequestBody(desc string, required bool, contents ...ContentOption) OperationOption {
	return func(*openapi.OperationInfo) {}
}

// Content is a stub.
// FixedBy: P6.6a
func Content(mediaType string, model any, encs ...EncodingOption) ContentOption {
	return func(*contentSpec) {}
}

// Encoding is a stub.
// FixedBy: P6.6a
func Encoding(property, contentType string) EncodingOption {
	return func(*encodingSpec) {}
}

// Parameter is a stub.
// FixedBy: P6.6b
func Parameter(p *openapi.Parameter) OperationOption {
	return func(*openapi.OperationInfo) {}
}

// Tags is a stub.
// FixedBy: P6.6c
func Tags(tags ...string) OperationOption {
	return func(*openapi.OperationInfo) {}
}

// ExternalDocs is a stub.
// FixedBy: P6.6c
func ExternalDocs(url, desc string) OperationOption {
	return func(*openapi.OperationInfo) {}
}

// Extension is a stub.
// FixedBy: P6.6c
func Extension(key string, value any) OperationOption {
	return func(*openapi.OperationInfo) {}
}

// CustomizeOperation is a stub.
// FixedBy: P6.6c
func CustomizeOperation(fn func(*openapi.Operation) error) OperationOption {
	return func(*openapi.OperationInfo) {}
}

// ResponseOption is a stub type.
// FixedBy: P6.6c
type ResponseOption func(*responseSpec)

// responseSpec is a stub type.
// FixedBy: P6.6c
type responseSpec struct{}

// ResponseWith is a stub.
// FixedBy: P6.6c
func ResponseWith(code, desc string, opts ...ResponseOption) OperationOption {
	return func(*openapi.OperationInfo) {}
}

// WithModel is a stub.
// FixedBy: P6.6c
func WithModel(any) ResponseOption {
	return func(*responseSpec) {}
}

// WithHeaders is a stub.
// FixedBy: P6.6c
func WithHeaders(...*openapi.ResponseHeader) ResponseOption {
	return func(*responseSpec) {}
}

// WithExample is a stub.
// FixedBy: P6.6c
func WithExample(any) ResponseOption {
	return func(*responseSpec) {}
}

// WithExamples is a stub.
// FixedBy: P6.6c
func WithExamples(map[string]any) ResponseOption {
	return func(*responseSpec) {}
}

// WithContent is a stub.
// FixedBy: P6.6c
func WithContent(mediaType string, model any) ResponseOption {
	return func(*responseSpec) {}
}

// WithResponseExtension is a stub.
// FixedBy: P6.6c
func WithResponseExtension(key string, value any) ResponseOption {
	return func(*responseSpec) {}
}
