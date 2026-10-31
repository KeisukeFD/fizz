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
}
