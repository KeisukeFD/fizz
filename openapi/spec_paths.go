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
	Name            string                   `json:"name" yaml:"name"`
	In              string                   `json:"in" yaml:"in"`
	Description     string                   `json:"description,omitempty" yaml:"description,omitempty"`
	Required        bool                     `json:"required,omitempty" yaml:"required,omitempty"`
	Deprecated      bool                     `json:"deprecated,omitempty" yaml:"deprecated,omitempty"`
	AllowEmptyValue bool                     `json:"allowEmptyValue,omitzero" yaml:"allowEmptyValue,omitempty"`
	Schema          *Schema                  `json:"schema,omitempty" yaml:"schema,omitempty"`
	Style           string                   `json:"style,omitempty" yaml:"style,omitempty"`
	Explode         *bool                    `json:"explode,omitempty" yaml:"explode,omitempty"`
	AllowReserved   bool                     `json:"allowReserved,omitempty" yaml:"allowReserved,omitempty"`
	Example         any                      `json:"example,omitempty" yaml:"example,omitempty"`
	Examples        map[string]*ExampleOrRef `json:"examples,omitempty" yaml:"examples,omitempty"`
	Content         map[string]*MediaType    `json:"content,omitempty" yaml:"content,omitempty"`

	// Extensions holds the "x-" specification extensions. Keys are
	// emitted sorted after the typed members. Use SetExtension to
	// validate keys.
	Extensions map[string]any `json:",embed" yaml:",inline"`
}

// ParameterOrRef represents a Parameter that can be inlined
// or referenced in the API description.
type ParameterOrRef = OrRef[Parameter]

// RequestBodyOrRef represents a RequestBody that can be inlined
// or referenced in the API description.
type RequestBodyOrRef = OrRef[RequestBody]

// RequestBody represents a request body.
type RequestBody struct {
	Description string                `json:"description,omitempty" yaml:"description,omitempty"`
	Content     map[string]*MediaType `json:"content" yaml:"content"`
	Required    bool                  `json:"required,omitempty" yaml:"required,omitempty"`

	// Extensions holds the "x-" specification extensions. Keys are
	// emitted sorted after the typed members. Use SetExtension to
	// validate keys.
	Extensions map[string]any `json:",embed" yaml:",inline"`
}

// Operation describes an API operation on a path.
type Operation struct {
	Tags         []string                  `json:"tags,omitempty" yaml:"tags,omitempty"`
	Summary      string                    `json:"summary,omitempty" yaml:"summary,omitempty"`
	Description  string                    `json:"description,omitempty" yaml:"description,omitempty"`
	ID           string                    `json:"operationId,omitempty" yaml:"operationId,omitempty"`
	Parameters   []*ParameterOrRef         `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	RequestBody  *RequestBody              `json:"requestBody,omitempty" yaml:"requestBody,omitempty"`
	Responses    Responses                 `json:"responses,omitempty" yaml:"responses,omitempty"`
	Callbacks    map[string]*CallbackOrRef `json:"callbacks,omitempty" yaml:"callbacks,omitempty"`
	Deprecated   bool                      `json:"deprecated,omitempty" yaml:"deprecated,omitempty"`
	Servers      []*Server                 `json:"servers,omitempty" yaml:"servers,omitempty"`
	Security     []*SecurityRequirement    `json:"security,omitzero" yaml:"-"`
	XCodeSamples []*XCodeSample            `json:"x-codeSamples,omitempty" yaml:"x-codeSamples,omitempty"`
	XInternal    bool                      `json:"x-internal,omitempty" yaml:"x-internal,omitempty"`

	// Extensions holds the "x-" specification extensions. Keys are
	// emitted sorted after the typed members. Use SetExtension to
	// validate keys.
	Extensions map[string]any `json:",embed" yaml:",inline"`
}

// MarshalYAML keeps an empty, non-nil Security (security: [], meaning
// "no authentication") in YAML output while omitting a nil one.
//
// Temporary: yaml.v2 has no omitzero support. Deleted by P2.5.
func (o *Operation) MarshalYAML() (any, error) {
	type plain Operation
	if o.Security == nil {
		return plain(*o), nil
	}
	return struct {
		plain    `yaml:",inline"`
		Security []*SecurityRequirement `yaml:"security"`
	}{plain(*o), o.Security}, nil
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
	Summary     string                     `json:"summary,omitempty" yaml:"summary,omitempty"`
	Description string                     `json:"description,omitempty" yaml:"description,omitempty"`
	Headers     map[string]*HeaderOrRef    `json:"headers,omitempty" yaml:"headers,omitempty"`
	Content     map[string]*MediaTypeOrRef `json:"content,omitempty" yaml:"content,omitempty"`
	Links       map[string]*LinkOrRef      `json:"links,omitempty" yaml:"links,omitempty"`

	// Extensions holds the "x-" specification extensions. Keys are
	// emitted sorted after the typed members. Use SetExtension to
	// validate keys.
	Extensions map[string]any `json:",embed" yaml:",inline"`
}

// LinkOrRef represents a Link that can be inlined
// or referenced in the API description.
type LinkOrRef = OrRef[Link]

// Link represents a possible design-time link for a response.
type Link struct {
	OperationRef string         `json:"operationRef,omitempty" yaml:"operationRef,omitempty"`
	OperationID  string         `json:"operationId,omitempty" yaml:"operationId,omitempty"`
	Parameters   map[string]any `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	RequestBody  any            `json:"requestBody,omitempty" yaml:"requestBody,omitempty"`
	Description  string         `json:"description,omitempty" yaml:"description,omitempty"`
	Server       *Server        `json:"server,omitempty" yaml:"server,omitempty"`

	// Extensions holds the "x-" specification extensions. Keys are
	// emitted sorted after the typed members. Use SetExtension to
	// validate keys.
	Extensions map[string]any `json:",embed" yaml:",inline"`
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
	Description string                   `json:"description,omitempty" yaml:"description,omitempty"`
	Required    bool                     `json:"required,omitempty" yaml:"required,omitempty"`
	Deprecated  bool                     `json:"deprecated,omitempty" yaml:"deprecated,omitempty"`
	Schema      *Schema                  `json:"schema,omitempty" yaml:"schema,omitempty"`
	Style       string                   `json:"style,omitempty" yaml:"style,omitempty"`
	Explode     *bool                    `json:"explode,omitempty" yaml:"explode,omitempty"`
	Example     any                      `json:"example,omitempty" yaml:"example,omitempty"`
	Examples    map[string]*ExampleOrRef `json:"examples,omitempty" yaml:"examples,omitempty"`
	Content     map[string]*MediaType    `json:"content,omitempty" yaml:"content,omitempty"`

	// Extensions holds the "x-" specification extensions. Keys are
	// emitted sorted after the typed members. Use SetExtension to
	// validate keys.
	Extensions map[string]any `json:",embed" yaml:",inline"`
}

// MediaTypeOrRef represents a MediaType that can be inlined
// or referenced in the API description.
type MediaTypeOrRef = OrRef[MediaType]

// MediaType represents the type of a media.
type MediaType struct {
	Schema         *Schema                  `json:"schema,omitempty" yaml:"schema,omitempty"`
	ItemSchema     *Schema                  `json:"itemSchema,omitempty" yaml:"itemSchema,omitempty"`
	Example        any                      `json:"example,omitempty" yaml:"example,omitempty"`
	Examples       map[string]*ExampleOrRef `json:"examples,omitempty" yaml:"examples,omitempty"`
	Encoding       map[string]*Encoding     `json:"encoding,omitempty" yaml:"encoding,omitempty"`
	PrefixEncoding []*Encoding              `json:"prefixEncoding,omitempty" yaml:"prefixEncoding,omitempty"`
	ItemEncoding   *Encoding                `json:"itemEncoding,omitempty" yaml:"itemEncoding,omitempty"`

	// Extensions holds the "x-" specification extensions. Keys are
	// emitted sorted after the typed members. Use SetExtension to
	// validate keys.
	Extensions map[string]any `json:",embed" yaml:",inline"`
}

// ExampleOrRef represents an Example that can be inlined
// or referenced in the API description.
type ExampleOrRef = OrRef[Example]

// Example represents the example of a media type.
type Example struct {
	Summary         string `json:"summary,omitempty" yaml:"summary,omitempty"`
	Description     string `json:"description,omitempty" yaml:"description,omitempty"`
	Value           any    `json:"value,omitempty" yaml:"value,omitempty"`
	DataValue       any    `json:"dataValue,omitempty" yaml:"dataValue,omitempty"`
	SerializedValue string `json:"serializedValue,omitempty" yaml:"serializedValue,omitempty"`
	ExternalValue   string `json:"externalValue,omitempty" yaml:"externalValue,omitempty"`

	// Extensions holds the "x-" specification extensions. Keys are
	// emitted sorted after the typed members. Use SetExtension to
	// validate keys.
	Extensions map[string]any `json:",embed" yaml:",inline"`
}

// Encoding represents a single encoding definition
// applied to a single schema property.
type Encoding struct {
	ContentType    string                  `json:"contentType,omitempty" yaml:"contentType,omitempty"`
	Headers        map[string]*HeaderOrRef `json:"headers,omitempty" yaml:"headers,omitempty"`
	Style          string                  `json:"style,omitempty" yaml:"style,omitempty"`
	Explode        *bool                   `json:"explode,omitempty" yaml:"explode,omitempty"`
	AllowReserved  bool                    `json:"allowReserved,omitempty" yaml:"allowReserved,omitempty"`
	Encoding       map[string]*Encoding    `json:"encoding,omitempty" yaml:"encoding,omitempty"`
	PrefixEncoding []*Encoding             `json:"prefixEncoding,omitempty" yaml:"prefixEncoding,omitempty"`
	ItemEncoding   *Encoding               `json:"itemEncoding,omitempty" yaml:"itemEncoding,omitempty"`

	// Extensions holds the "x-" specification extensions. Keys are
	// emitted sorted after the typed members. Use SetExtension to
	// validate keys.
	Extensions map[string]any `json:",embed" yaml:",inline"`
}
