package openapi

import "encoding/json"

// Reference is a simple object to allow referencing
// other components in the specification, internally and
// externally.
type Reference struct {
	Ref string `json:"$ref" yaml:"$ref"`
}

// Parameter describes a single operation parameter.
type Parameter struct {
	Name            string  `json:"name" yaml:"name"`
	In              string  `json:"in" yaml:"in"`
	Description     string  `json:"description,omitempty" yaml:"description,omitempty"`
	Required        bool    `json:"required,omitempty" yaml:"required,omitempty"`
	Deprecated      bool    `json:"deprecated,omitempty" yaml:"deprecated,omitempty"`
	AllowEmptyValue bool    `json:"allowEmptyValue,omitempty" yaml:"allowEmptyValue,omitempty"`
	Schema          *Schema `json:"schema,omitempty" yaml:"schema,omitempty"`
	Style           string  `json:"style,omitempty" yaml:"style,omitempty"`
	Explode         bool    `json:"explode,omitempty" yaml:"explode,omitempty"`
}

// ParameterOrRef represents a Parameter that can be inlined
// or referenced in the API description.
type ParameterOrRef struct {
	*Parameter
	*Reference
}

// MarshalYAML implements yaml.Marshaler for ParameterOrRef.
func (por *ParameterOrRef) MarshalYAML() (any, error) {
	if por.Parameter != nil {
		return por.Parameter, nil
	}
	return por.Reference, nil
}

// RequestBody represents a request body.
type RequestBody struct {
	Description string                `json:"description,omitempty" yaml:"description,omitempty"`
	Content     map[string]*MediaType `json:"content" yaml:"content"`
	Required    bool                  `json:"required,omitempty" yaml:"required,omitempty"`
}

// Operation describes an API operation on a path.
type Operation struct {
	Tags         []string               `json:"tags,omitempty" yaml:"tags,omitempty"`
	Summary      string                 `json:"summary,omitempty" yaml:"summary,omitempty"`
	Description  string                 `json:"description,omitempty" yaml:"description,omitempty"`
	ID           string                 `json:"operationId,omitempty" yaml:"operationId,omitempty"`
	Parameters   []*ParameterOrRef      `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	RequestBody  *RequestBody           `json:"requestBody,omitempty" yaml:"requestBody,omitempty"`
	Responses    Responses              `json:"responses,omitempty" yaml:"responses,omitempty"`
	Deprecated   bool                   `json:"deprecated,omitempty" yaml:"deprecated,omitempty"`
	Servers      []*Server              `json:"servers,omitempty" yaml:"servers,omitempty"`
	Security     []*SecurityRequirement `json:"security" yaml:"security"`
	XCodeSamples []*XCodeSample         `json:"x-codeSamples,omitempty" yaml:"x-codeSamples,omitempty"`
	XInternal    bool                   `json:"x-internal,omitempty" yaml:"x-internal,omitempty"`
}

// A workaround for missing omitnil functionality.
// Explicitely omit the Security field from marshaling when it is nil, but not when empty.
type operationNilOmitted struct {
	Tags         []string          `json:"tags,omitempty" yaml:"tags,omitempty"`
	Summary      string            `json:"summary,omitempty" yaml:"summary,omitempty"`
	Description  string            `json:"description,omitempty" yaml:"description,omitempty"`
	ID           string            `json:"operationId,omitempty" yaml:"operationId,omitempty"`
	Parameters   []*ParameterOrRef `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	RequestBody  *RequestBody      `json:"requestBody,omitempty" yaml:"requestBody,omitempty"`
	Responses    Responses         `json:"responses,omitempty" yaml:"responses,omitempty"`
	Deprecated   bool              `json:"deprecated,omitempty" yaml:"deprecated,omitempty"`
	Servers      []*Server         `json:"servers,omitempty" yaml:"servers,omitempty"`
	XCodeSamples []*XCodeSample    `json:"x-codeSamples,omitempty" yaml:"x-codeSamples,omitempty"`
	XInternal    bool              `json:"x-internal,omitempty" yaml:"x-internal,omitempty"`
}

// MarshalYAML implements yaml.Marshaler for Operation.
// Needed to marshall empty but non-null SecurityRequirements.
func (o *Operation) MarshalYAML() (any, error) {
	if o.Security == nil {
		return omitOperationNilFields(o), nil
	}
	return o, nil
}

// MarshalJSON excludes empty but non-null SecurityRequirements.
func (o *Operation) MarshalJSON() ([]byte, error) {
	if o.Security == nil {
		return json.Marshal(omitOperationNilFields(o))
	}
	return json.Marshal(*o)
}

func omitOperationNilFields(o *Operation) *operationNilOmitted {
	return &operationNilOmitted{
		Tags:         o.Tags,
		Summary:      o.Summary,
		Description:  o.Description,
		ID:           o.ID,
		Parameters:   o.Parameters,
		RequestBody:  o.RequestBody,
		Responses:    o.Responses,
		Deprecated:   o.Deprecated,
		Servers:      o.Servers,
		XCodeSamples: o.XCodeSamples,
		XInternal:    o.XInternal,
	}
}

// Responses represents a container for the expected responses
// of an opration. It maps a HTTP response code to the expected
// response.
type Responses map[string]*ResponseOrRef

// ResponseOrRef represents a Response that can be inlined
// or referenced in the API description.
type ResponseOrRef struct {
	*Response
	*Reference
}

// MarshalYAML implements yaml.Marshaler for ResponseOrRef.
func (ror *ResponseOrRef) MarshalYAML() (any, error) {
	if ror.Response != nil {
		return ror.Response, nil
	}
	return ror.Reference, nil
}

// Response describes a single response from an API.
type Response struct {
	Description string                     `json:"description,omitempty" yaml:"description,omitempty"`
	Headers     map[string]*HeaderOrRef    `json:"headers,omitempty" yaml:"headers,omitempty"`
	Content     map[string]*MediaTypeOrRef `json:"content,omitempty" yaml:"content,omitempty"`
}

// HeaderOrRef represents a Header that can be inlined
// or referenced in the API description.
type HeaderOrRef struct {
	*Header
	*Reference
}

// MarshalYAML implements yaml.Marshaler for HeaderOrRef.
func (hor *HeaderOrRef) MarshalYAML() (any, error) {
	if hor.Header != nil {
		return hor.Header, nil
	}
	return hor.Reference, nil
}

// Header represents an HTTP header.
type Header struct {
	Description     string  `json:"description,omitempty" yaml:"description,omitempty"`
	Required        bool    `json:"required,omitempty" yaml:"required,omitempty"`
	Deprecated      bool    `json:"deprecated,omitempty" yaml:"deprecated,omitempty"`
	AllowEmptyValue bool    `json:"allowEmptyValue,omitempty" yaml:"allowEmptyValue,omitempty"`
	Schema          *Schema `json:"schema,omitempty" yaml:"schema,omitempty"`
}

// MediaTypeOrRef represents a MediaType that can be inlined
// or referenced in the API description.
type MediaTypeOrRef struct {
	*MediaType
	*Reference
}

// MarshalYAML implements yaml.Marshaler for MediaTypeOrRef.
func (mtor *MediaTypeOrRef) MarshalYAML() (any, error) {
	if mtor.MediaType != nil {
		return mtor.MediaType, nil
	}
	return mtor.Reference, nil
}

// MediaType represents the type of a media.
type MediaType struct {
	Schema   *Schema                  `json:"schema" yaml:"schema"`
	Example  any                      `json:"example,omitempty" yaml:"example,omitempty"`
	Examples map[string]*ExampleOrRef `json:"examples,omitempty" yaml:"examples,omitempty"`
	Encoding map[string]*Encoding     `json:"encoding,omitempty" yaml:"encoding,omitempty"`
}

// ExampleOrRef represents an Example that can be inlined
// or referenced in the API description.
type ExampleOrRef struct {
	*Example
	*Reference
}

// MarshalYAML implements yaml.Marshaler for ExampleOrRef.
func (eor *ExampleOrRef) MarshalYAML() (any, error) {
	if eor.Example != nil {
		return eor.Example, nil
	}
	return eor.Reference, nil
}

// Example represents the example of a media type.
type Example struct {
	Summary       string `json:"summary,omitempty" yaml:"summary,omitempty"`
	Description   string `json:"description,omitempty" yaml:"description,omitempty"`
	Value         any    `json:"value,omitempty" yaml:"value,omitempty"`
	ExternalValue string `json:"externalValue,omitempty" yaml:"externalValue,omitempty"`
}

// Encoding represents a single encoding definition
// applied to a single schema property.
type Encoding struct {
	ContentType   string                  `json:"contentType,omitempty" yaml:"contentType,omitempty"`
	Headers       map[string]*HeaderOrRef `json:"headers,omitempty" yaml:"headers,omitempty"`
	Style         string                  `json:"style,omitempty" yaml:"style,omitempty"`
	Explode       bool                    `json:"explode,omitempty" yaml:"explode,omitempty"`
	AllowReserved bool                    `json:"allowReserved,omitempty" yaml:"allowReserved,omitempty"`
}
