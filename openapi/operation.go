package openapi

// OperationInfo represents the informations of an operation
// that will be used when generating the OpenAPI specification.
type OperationInfo struct {
	ID                      string
	StatusCode              int
	StatusDescription       string
	Headers                 []*ResponseHeader
	Summary                 string
	Description             string
	Deprecated              bool
	InputModel              any
	Responses               []*OperationResponse
	Security                []*SecurityRequirement
	XCodeSamples            []*XCodeSample
	XInternal               bool
	DefaultResponseExamples map[string]any
	// RequestBody, if set, adjusts the request body of the operation:
	// see the RequestBody option of the fizz package.
	RequestBody *RequestBodyInfo
	// Parameters are merged into the parameters of the operation, in
	// order, by name and location: see the Parameter option of the
	// fizz package.
	Parameters []*Parameter
	// Tags are added to the tag of the group of the operation, after
	// it and without duplicates, empty names left out: see the Tags
	// option of the fizz package.
	Tags []string
	// ExternalDocs sets the externalDocs of the operation. An empty
	// URL is an error: see the ExternalDocs option of the fizz package.
	ExternalDocs *ExternalDocs
	// Extensions are "x-" extensions of the operation, applied in order,
	// the last one of a key winning: see the Extension option of the fizz
	// package.
	Extensions []*ExtensionInfo
	// Customizers run on a copy of the operation once every other
	// option has been applied, in order, before the OperationHook: see
	// the CustomizeOperation option of the fizz package. A nil one is
	// ignored.
	Customizers []func(*Operation) error
	// WildcardParams names the path parameters of the operation that
	// come from a catch-all gin segment ("/*name"). The router sets it
	// when it translates the gin path; the path given to AddOperation
	// no longer tells them apart. Their description notes that the value
	// may contain a slash.
	WildcardParams []string
}

// RequestBodyInfo describes how the request body of an operation
// differs from the one inferred from its input type.
type RequestBodyInfo struct {
	// Description is the description of the request body.
	Description string
	// Required overrides the inferred required flag, both ways.
	Required bool
	// Contents, if not empty, replaces all the inferred content.
	Contents []*ContentInfo
}

// ContentInfo is one media type of a RequestBodyInfo.
type ContentInfo struct {
	MediaType string
	// Model gives the schema of the media type. A nil Model gives a
	// media type without a schema.
	Model     any
	Encodings []*EncodingInfo
}

// EncodingInfo sets the content type of a property of the schema of a
// media type, as the encoding object of the specification does.
type EncodingInfo struct {
	Property    string
	ContentType string
}

// ExtensionInfo is one "x-" specification extension to set on an
// operation or on a response; the generator validates the key and the
// value when it applies it.
type ExtensionInfo struct {
	Key   string
	Value any
}

// ResponseContent is one media type of an OperationResponse, in
// addition to the default media type of the Model.
type ResponseContent struct {
	MediaType string
	// Model gives the schema of the media type. A nil Model gives a
	// media type without a schema.
	Model any
}

// ResponseHeader represents a single header that
// may be returned with an operation response.
type ResponseHeader struct {
	Name        string
	Description string
	Model       any
	Required    bool
	Deprecated  bool
	Example     any
	Examples    map[string]*ExampleOrRef
}

// OperationResponse represents a single response of an
// API operation.
type OperationResponse struct {
	// The response code can be "default"
	// according to OAS3 spec.
	Code        string
	Description string
	Model       any
	Headers     []*ResponseHeader
	Example     any
	Examples    map[string]any
	// Contents are further media types of the response, after the one
	// of the Model (the default media type). A media type that the
	// response already has is an error and the first is kept.
	Contents []*ResponseContent
	// Extensions are "x-" extensions of the response, applied in order.
	Extensions []*ExtensionInfo
	// KeepExamples, when both Example and Examples are set, drops the
	// Example and records an error instead of failing the operation.
	KeepExamples bool
}
