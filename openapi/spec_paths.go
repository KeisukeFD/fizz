package openapi

// Parameter locations.
const (
	InPath        = "path"
	InQuery       = "query"
	InHeader      = "header"
	InCookie      = "cookie"
	InQueryString = "querystring"
)

// Parameter describes a single operation parameter.
type Parameter struct {
	Name            string                   `json:"name"`
	In              string                   `json:"in"`
	Description     string                   `json:"description,omitempty"`
	Required        bool                     `json:"required,omitzero"`
	Deprecated      bool                     `json:"deprecated,omitzero"`
	AllowEmptyValue bool                     `json:"allowEmptyValue,omitzero"`
	Schema          *Schema                  `json:"schema,omitzero"`
	Style           string                   `json:"style,omitempty"`
	Explode         *bool                    `json:"explode,omitzero"`
	AllowReserved   bool                     `json:"allowReserved,omitzero"`
	Example         any                      `json:"example,omitzero"`
	Examples        map[string]*ExampleOrRef `json:"examples,omitempty"`
	Content         map[string]*MediaType    `json:"content,omitempty"`

	// Extensions holds the "x-" specification extensions. Keys are
	// emitted sorted after the typed members. Use SetExtension to
	// validate keys.
	Extensions map[string]any `json:",embed"`
}

// ParameterOrRef represents a Parameter that can be inlined
// or referenced in the API description.
type ParameterOrRef = OrRef[Parameter]

// RequestBodyOrRef represents a RequestBody that can be inlined
// or referenced in the API description.
type RequestBodyOrRef = OrRef[RequestBody]

// RequestBody represents a request body.
type RequestBody struct {
	Description string                `json:"description,omitempty"`
	Content     map[string]*MediaType `json:"content"`
	Required    bool                  `json:"required,omitzero"`

	// Extensions holds the "x-" specification extensions. Keys are
	// emitted sorted after the typed members. Use SetExtension to
	// validate keys.
	Extensions map[string]any `json:",embed"`
}

// Operation describes an API operation on a path.
type Operation struct {
	Tags         []string                  `json:"tags,omitempty"`
	Summary      string                    `json:"summary,omitempty"`
	Description  string                    `json:"description,omitempty"`
	ExternalDocs *ExternalDocs             `json:"externalDocs,omitempty"`
	ID           string                    `json:"operationId,omitempty"`
	Parameters   []*ParameterOrRef         `json:"parameters,omitempty"`
	RequestBody  *RequestBody              `json:"requestBody,omitzero"`
	Responses    Responses                 `json:"responses,omitempty"`
	Callbacks    map[string]*CallbackOrRef `json:"callbacks,omitempty"`
	Deprecated   bool                      `json:"deprecated,omitzero"`
	Servers      []*Server                 `json:"servers,omitempty"`
	Security     []*SecurityRequirement    `json:"security,omitzero"`
	XCodeSamples []*XCodeSample            `json:"x-codeSamples,omitempty"`
	XInternal    bool                      `json:"x-internal,omitzero"`

	// Extensions holds the "x-" specification extensions. Keys are
	// emitted sorted after the typed members. Use SetExtension to
	// validate keys.
	Extensions map[string]any `json:",embed"`
}

// Responses represents a container for the expected responses
// of an opration. It maps a HTTP response code to the expected
// response.
type Responses map[string]*ResponseOrRef

// ResponseOrRef represents a Response that can be inlined
// or referenced in the API description.
type ResponseOrRef = OrRef[Response]

// Response describes a single response from an API.
type Response struct {
	Summary     string                     `json:"summary,omitempty"`
	Description string                     `json:"description,omitempty"`
	Headers     map[string]*HeaderOrRef    `json:"headers,omitempty"`
	Content     map[string]*MediaTypeOrRef `json:"content,omitempty"`
	Links       map[string]*LinkOrRef      `json:"links,omitempty"`

	// Extensions holds the "x-" specification extensions. Keys are
	// emitted sorted after the typed members. Use SetExtension to
	// validate keys.
	Extensions map[string]any `json:",embed"`
}

// LinkOrRef represents a Link that can be inlined
// or referenced in the API description.
type LinkOrRef = OrRef[Link]

// Link represents a possible design-time link for a response.
type Link struct {
	OperationRef string         `json:"operationRef,omitempty"`
	OperationID  string         `json:"operationId,omitempty"`
	Parameters   map[string]any `json:"parameters,omitempty"`
	RequestBody  any            `json:"requestBody,omitzero"`
	Description  string         `json:"description,omitempty"`
	Server       *Server        `json:"server,omitzero"`

	// Extensions holds the "x-" specification extensions. Keys are
	// emitted sorted after the typed members. Use SetExtension to
	// validate keys.
	Extensions map[string]any `json:",embed"`
}

// CallbackOrRef represents a Callback that can be inlined
// or referenced in the API description.
type CallbackOrRef = OrRef[Callback]

// Callback maps a runtime expression to the path item describing
// the request that may be initiated by the API provider. Keys that
// start with "x-" are specification extensions, not expressions.
type Callback struct {
	Expressions map[string]*PathItem
	Extensions  map[string]any
}

// HeaderOrRef represents a Header that can be inlined
// or referenced in the API description.
type HeaderOrRef = OrRef[Header]

// Header represents an HTTP header.
type Header struct {
	Description string                   `json:"description,omitempty"`
	Required    bool                     `json:"required,omitzero"`
	Deprecated  bool                     `json:"deprecated,omitzero"`
	Schema      *Schema                  `json:"schema,omitzero"`
	Style       string                   `json:"style,omitempty"`
	Explode     *bool                    `json:"explode,omitzero"`
	Example     any                      `json:"example,omitzero"`
	Examples    map[string]*ExampleOrRef `json:"examples,omitempty"`
	Content     map[string]*MediaType    `json:"content,omitempty"`

	// Extensions holds the "x-" specification extensions. Keys are
	// emitted sorted after the typed members. Use SetExtension to
	// validate keys.
	Extensions map[string]any `json:",embed"`
}

// MediaTypeOrRef represents a MediaType that can be inlined
// or referenced in the API description.
type MediaTypeOrRef = OrRef[MediaType]

// MediaType represents the type of a media.
type MediaType struct {
	Schema         *Schema                  `json:"schema,omitzero"`
	ItemSchema     *Schema                  `json:"itemSchema,omitzero"`
	Example        any                      `json:"example,omitzero"`
	Examples       map[string]*ExampleOrRef `json:"examples,omitempty"`
	Encoding       map[string]*Encoding     `json:"encoding,omitempty"`
	PrefixEncoding []*Encoding              `json:"prefixEncoding,omitempty"`
	ItemEncoding   *Encoding                `json:"itemEncoding,omitzero"`

	// Extensions holds the "x-" specification extensions. Keys are
	// emitted sorted after the typed members. Use SetExtension to
	// validate keys.
	Extensions map[string]any `json:",embed"`
}

// ExampleOrRef represents an Example that can be inlined
// or referenced in the API description.
type ExampleOrRef = OrRef[Example]

// Example represents the example of a media type.
type Example struct {
	Summary         string `json:"summary,omitempty"`
	Description     string `json:"description,omitempty"`
	Value           any    `json:"value,omitzero"`
	DataValue       any    `json:"dataValue,omitzero"`
	SerializedValue string `json:"serializedValue,omitempty"`
	ExternalValue   string `json:"externalValue,omitempty"`

	// Extensions holds the "x-" specification extensions. Keys are
	// emitted sorted after the typed members. Use SetExtension to
	// validate keys.
	Extensions map[string]any `json:",embed"`
}

// Encoding represents a single encoding definition
// applied to a single schema property.
type Encoding struct {
	ContentType    string                  `json:"contentType,omitempty"`
	Headers        map[string]*HeaderOrRef `json:"headers,omitempty"`
	Style          string                  `json:"style,omitempty"`
	Explode        *bool                   `json:"explode,omitzero"`
	AllowReserved  bool                    `json:"allowReserved,omitzero"`
	Encoding       map[string]*Encoding    `json:"encoding,omitempty"`
	PrefixEncoding []*Encoding             `json:"prefixEncoding,omitempty"`
	ItemEncoding   *Encoding               `json:"itemEncoding,omitzero"`

	// Extensions holds the "x-" specification extensions. Keys are
	// emitted sorted after the typed members. Use SetExtension to
	// validate keys.
	Extensions map[string]any `json:",embed"`
}
