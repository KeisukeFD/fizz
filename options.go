package fizz

import (
	"fmt"

	"github.com/wI2L/fizz/openapi"
)

// OperationOption represents an option-pattern function
// used to add informations to an operation.
type OperationOption func(*openapi.OperationInfo)

// StatusDescription sets the default status description of the operation.
func StatusDescription(desc string) func(*openapi.OperationInfo) {
	return func(o *openapi.OperationInfo) {
		o.StatusDescription = desc
	}
}

// Summary adds a summary to an operation.
func Summary(summary string) func(*openapi.OperationInfo) {
	return func(o *openapi.OperationInfo) {
		o.Summary = summary
	}
}

// Summaryf adds a summary to an operation according
// to a format specifier.
func Summaryf(format string, a ...any) func(*openapi.OperationInfo) {
	return func(o *openapi.OperationInfo) {
		o.Summary = fmt.Sprintf(format, a...)
	}
}

// Description adds a description to an operation.
func Description(desc string) func(*openapi.OperationInfo) {
	return func(o *openapi.OperationInfo) {
		o.Description = desc
	}
}

// Descriptionf adds a description to an operation
// according to a format specifier.
func Descriptionf(format string, a ...any) func(*openapi.OperationInfo) {
	return func(o *openapi.OperationInfo) {
		o.Description = fmt.Sprintf(format, a...)
	}
}

// ID overrides the operation ID.
func ID(id string) func(*openapi.OperationInfo) {
	return func(o *openapi.OperationInfo) {
		o.ID = id
	}
}

// Deprecated marks the operation as deprecated.
func Deprecated(deprecated bool) func(*openapi.OperationInfo) {
	return func(o *openapi.OperationInfo) {
		o.Deprecated = deprecated
	}
}

// Response adds an additional response to the operation.
func Response(statusCode, desc string, model any, headers []*openapi.ResponseHeader, example any) func(*openapi.OperationInfo) {
	return func(o *openapi.OperationInfo) {
		o.Responses = append(o.Responses, &openapi.OperationResponse{
			Code:        statusCode,
			Description: desc,
			Model:       model,
			Headers:     headers,
			Example:     example,
		})
	}
}

// ResponseWithExamples is a variant of Response that accept many examples.
func ResponseWithExamples(statusCode, desc string, model any, headers []*openapi.ResponseHeader, examples map[string]any) func(*openapi.OperationInfo) {
	return func(o *openapi.OperationInfo) {
		o.Responses = append(o.Responses, &openapi.OperationResponse{
			Code:        statusCode,
			Description: desc,
			Model:       model,
			Headers:     headers,
			Examples:    examples,
		})
	}
}

// DefaultResponseExamples adds additional examples to the default response.
func DefaultResponseExamples(examples map[string]any) func(*openapi.OperationInfo) {
	return func(o *openapi.OperationInfo) {
		o.DefaultResponseExamples = examples
	}
}

// Header adds a header to the operation.
func Header(name, desc string, model any) func(*openapi.OperationInfo) {
	return func(o *openapi.OperationInfo) {
		o.Headers = append(o.Headers, &openapi.ResponseHeader{
			Name:        name,
			Description: desc,
			Model:       model,
		})
	}
}

// InputModel overrides the binding model of the operation.
func InputModel(model any) func(*openapi.OperationInfo) {
	return func(o *openapi.OperationInfo) {
		o.InputModel = model
	}
}

// XCodeSample adds a code sample to the operation.
func XCodeSample(cs *openapi.XCodeSample) func(*openapi.OperationInfo) {
	return func(o *openapi.OperationInfo) {
		o.XCodeSamples = append(o.XCodeSamples, cs)
	}
}

// Overrides top-level security requirement for this operation.
// Note that this function can be used more than once to add several requirements.
func Security(security *openapi.SecurityRequirement) func(*openapi.OperationInfo) {
	return func(o *openapi.OperationInfo) {
		o.Security = append(o.Security, security)
	}
}

// Add an empty security requirement to this operation to make other security requirements optional.
func WithOptionalSecurity() func(*openapi.OperationInfo) {
	return func(o *openapi.OperationInfo) {
		var emptyRequirement openapi.SecurityRequirement = make(openapi.SecurityRequirement)
		o.Security = append(o.Security, &emptyRequirement)
	}
}

// Remove any top-level security requirements for this operation.
func WithoutSecurity() func(*openapi.OperationInfo) {
	return func(o *openapi.OperationInfo) {
		o.Security = []*openapi.SecurityRequirement{}
	}
}

// XInternal marks the operation as internal.
func XInternal() func(*openapi.OperationInfo) {
	return func(o *openapi.OperationInfo) {
		o.XInternal = true
	}
}

// ContentOption describes one media type of a request body: see Content.
type ContentOption func(*contentSpec)

// contentSpec is the state a ContentOption fills in.
type contentSpec struct {
	mediaType string
	model     any
	encodings []*openapi.EncodingInfo
}

// EncodingOption sets the encoding of a property of a Content.
type EncodingOption func(*encodingSpec)

// encodingSpec is the state an EncodingOption fills in.
type encodingSpec struct {
	property    string
	contentType string
}

// RequestBody adjusts the request body of the operation, over what the
// input type and the tags give (and under the OperationHook). desc is
// its description; required overrides the inferred required flag, in
// both directions. If contents are given they replace the inferred
// content entirely; a media type given twice is an error and the first
// is kept. Without contents the inferred content is kept, and it is an
// error if there is none. A request body is an error on GET and HEAD
// operations; DELETE may have one.
func RequestBody(desc string, required bool, contents ...ContentOption) OperationOption {
	return func(o *openapi.OperationInfo) {
		rb := &openapi.RequestBodyInfo{Description: desc, Required: required}
		for _, c := range contents {
			if c == nil {
				continue
			}
			var s contentSpec
			c(&s)
			rb.Contents = append(rb.Contents, &openapi.ContentInfo{
				MediaType: s.mediaType,
				Model:     s.model,
				Encodings: s.encodings,
			})
		}
		o.RequestBody = rb
	}
}

// Content describes the media type of a request body, whose schema is
// the one of model, as for any other type: a named struct is a
// component. A nil model gives a media type without a schema.
func Content(mediaType string, model any, encs ...EncodingOption) ContentOption {
	return func(s *contentSpec) {
		s.mediaType = mediaType
		s.model = model
		for _, e := range encs {
			if e == nil {
				continue
			}
			var es encodingSpec
			e(&es)
			s.encodings = append(s.encodings, &openapi.EncodingInfo{
				Property:    es.property,
				ContentType: es.contentType,
			})
		}
	}
}

// Encoding sets the content type of a property of the schema of a
// Content (the "encoding" of a media type). A property that the schema
// does not have is reported as a warning.
func Encoding(property, contentType string) EncodingOption {
	return func(s *encodingSpec) {
		s.property = property
		s.contentType = contentType
	}
}

// Parameter adds p to the parameters of the operation, or merges it into
// the inferred one that has the same name and location ("in"). It acts
// over what the input type, the tags and the ParameterHook give, and under
// the OperationHook.
//
// The merge is field by field: each non-zero field of p replaces the
// inferred one (a non-nil pointer, slice or map is non-zero), so that
// a field p leaves out keeps its inferred value, the schema included.
// Schema and Content are replaced whole, not merged; Extensions are
// merged key by key. A bool that is false cannot unset a true one.
// Options on the same parameter apply in order, the last one winning.
// A parameter with no inferred match is appended.
//
// A nil p is ignored, and p is copied: it may be reused. A parameter
// without a name or a location, a path parameter that is not in the
// path, and a merge into a $ref parameter are recorded errors; the
// operation is still added, without that parameter.
func Parameter(p *openapi.Parameter) OperationOption {
	if p == nil {
		return func(*openapi.OperationInfo) {}
	}
	c, err := p.Clone()
	if err != nil {
		// Keep the original: the generator copies it again and records
		// the error against the operation.
		c = p
	}
	return func(o *openapi.OperationInfo) {
		o.Parameters = append(o.Parameters, c)
	}
}

// Tags adds tags to the operation, after the tag of its group and
// without duplicates: a tag that is already there, and an empty one,
// are ignored. The tags of an operation are not declared at the root of
// the document; use Generator.SetTag for that.
func Tags(tags ...string) OperationOption {
	return func(o *openapi.OperationInfo) {
		o.Tags = append(o.Tags, tags...)
	}
}

// ExternalDocs sets the external documentation of the operation. An
// empty url is a recorded error and nothing is emitted.
func ExternalDocs(url, desc string) OperationOption {
	return func(o *openapi.OperationInfo) {
		o.ExternalDocs = &openapi.ExternalDocs{URL: url, Description: desc}
	}
}

// Extension sets the "x-" specification extension key of the operation
// to value, the last option of a key winning. An invalid key
// (openapi.ErrInvalidExtensionKey, openapi.ErrReservedExtension), a
// value that is not JSON-encodable, and a key that a typed option owns
// ("x-codeSamples" is XCodeSample's, "x-internal" is XInternal's) are
// recorded errors, and the extension is not emitted.
func Extension(key string, value any) OperationOption {
	return func(o *openapi.OperationInfo) {
		o.Extensions = append(o.Extensions, &openapi.ExtensionInfo{Key: key, Value: value})
	}
}

// CustomizeOperation adjusts the operation through fn, which runs on a
// copy of it once every other option has been applied (whatever the
// order of the options), and before the OperationHook. Several
// functions run in the order they are given. If fn returns an error its
// changes are discarded and the error is recorded with the operation ID;
// the next function still runs. The operation ID cannot be changed: a
// new one is reverted and a warning is recorded. A nil fn is ignored.
func CustomizeOperation(fn func(*openapi.Operation) error) OperationOption {
	return func(o *openapi.OperationInfo) {
		if fn != nil {
			o.Customizers = append(o.Customizers, fn)
		}
	}
}

// ResponseOption sets a part of a response added by ResponseWith.
type ResponseOption func(*responseSpec)

// responseSpec is the state a ResponseOption fills in.
type responseSpec struct {
	model      any
	headers    []*openapi.ResponseHeader
	example    any
	examples   map[string]any
	contents   []*openapi.ResponseContent
	extensions []*openapi.ExtensionInfo
}

// ResponseWith adds a response to the operation, built the same way as
// the one of Response: same naming of the schemas, default media type
// and nullability. A code already used by the operation is an error.
//
// The model, the example and the examples go under the default media
// type; WithContent adds other media types. WithExample together with
// WithExamples, and a media type given twice, are recorded errors (the
// examples, and the first media type, are kept).
func ResponseWith(code, desc string, opts ...ResponseOption) OperationOption {
	return func(o *openapi.OperationInfo) {
		var s responseSpec
		for _, opt := range opts {
			if opt != nil {
				opt(&s)
			}
		}
		o.Responses = append(o.Responses, &openapi.OperationResponse{
			Code:        code,
			Description: desc,
			Model:       s.model,
			Headers:     s.headers,
			Example:     s.example,
			Examples:    s.examples,
			Contents:    s.contents,
			Extensions:  s.extensions,
			// ResponseWith keeps the examples of a conflict and goes on.
			KeepExamples: true,
		})
	}
}

// WithModel sets the type of the body of the response, under the default
// media type.
func WithModel(model any) ResponseOption {
	return func(s *responseSpec) { s.model = model }
}

// WithHeaders adds headers to the response.
func WithHeaders(headers ...*openapi.ResponseHeader) ResponseOption {
	return func(s *responseSpec) { s.headers = append(s.headers, headers...) }
}

// WithExample sets the example of the response, under the default media
// type. It excludes WithExamples.
func WithExample(example any) ResponseOption {
	return func(s *responseSpec) { s.example = example }
}

// WithExamples sets the named examples of the response, under the
// default media type. It excludes WithExample.
func WithExamples(examples map[string]any) ResponseOption {
	return func(s *responseSpec) { s.examples = examples }
}

// WithContent adds a media type to the response, whose schema is the one
// of model. A nil model gives a media type without a schema.
func WithContent(mediaType string, model any) ResponseOption {
	return func(s *responseSpec) {
		s.contents = append(s.contents, &openapi.ResponseContent{MediaType: mediaType, Model: model})
	}
}

// WithResponseExtension sets the "x-" extension key of the response, with
// the rules of Extension.
func WithResponseExtension(key string, value any) ResponseOption {
	return func(s *responseSpec) {
		s.extensions = append(s.extensions, &openapi.ExtensionInfo{Key: key, Value: value})
	}
}
