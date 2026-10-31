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
