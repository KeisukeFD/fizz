package openapi

// OpenAPI represents the root document object of
// an OpenAPI document.
type OpenAPI struct {
	OpenAPI           string                 `json:"openapi"`
	Self              string                 `json:"$self,omitzero"`
	Info              *Info                  `json:"info"`
	JSONSchemaDialect string                 `json:"jsonSchemaDialect,omitzero"`
	Servers           []*Server              `json:"servers,omitempty"`
	Paths             Paths                  `json:"paths,omitzero"`
	Webhooks          map[string]*PathItem   `json:"webhooks,omitempty"`
	Components        *Components            `json:"components,omitzero"`
	Tags              []*Tag                 `json:"tags,omitempty"`
	Security          []*SecurityRequirement `json:"security,omitempty"`
	ExternalDocs      *ExternalDocs          `json:"externalDocs,omitzero"`
	XTagGroups        []*XTagGroup           `json:"x-tagGroups,omitempty"`

	// Extensions holds the "x-" specification extensions. Keys are
	// emitted sorted after the typed members. Use SetExtension to
	// validate keys.
	Extensions map[string]any `json:",embed"`
}

// Components holds a set of reusable objects for different
// aspects of the specification.
type Components struct {
	Schemas         map[string]*Schema              `json:"schemas,omitempty"`
	Responses       map[string]*ResponseOrRef       `json:"responses,omitempty"`
	Parameters      map[string]*ParameterOrRef      `json:"parameters,omitempty"`
	Examples        map[string]*ExampleOrRef        `json:"examples,omitempty"`
	RequestBodies   map[string]*RequestBodyOrRef    `json:"requestBodies,omitempty"`
	Headers         map[string]*HeaderOrRef         `json:"headers,omitempty"`
	SecuritySchemes map[string]*SecuritySchemeOrRef `json:"securitySchemes,omitempty"`
	Links           map[string]*LinkOrRef           `json:"links,omitempty"`
	Callbacks       map[string]*CallbackOrRef       `json:"callbacks,omitempty"`
	PathItems       map[string]*PathItem            `json:"pathItems,omitempty"`
	MediaTypes      map[string]*MediaTypeOrRef      `json:"mediaTypes,omitempty"`

	// Extensions holds the "x-" specification extensions. Keys are
	// emitted sorted after the typed members. Use SetExtension to
	// validate keys.
	Extensions map[string]any `json:",embed"`
}

// Info represents the metadata of an API.
type Info struct {
	Title          string   `json:"title"`
	Summary        string   `json:"summary,omitempty"`
	Description    string   `json:"description,omitempty"`
	TermsOfService string   `json:"termsOfService,omitempty"`
	Contact        *Contact `json:"contact,omitzero"`
	License        *License `json:"license,omitzero"`
	Version        string   `json:"version"`
	XLogo          *XLogo   `json:"x-logo,omitzero"`

	// Extensions holds the "x-" specification extensions. Keys are
	// emitted sorted after the typed members. Use SetExtension to
	// validate keys.
	Extensions map[string]any `json:",embed"`
}

// Contact represents the the contact informations
// exposed for an API.
type Contact struct {
	Name  string `json:"name,omitempty"`
	URL   string `json:"url,omitempty"`
	Email string `json:"email,omitempty"`

	// Extensions holds the "x-" specification extensions. Keys are
	// emitted sorted after the typed members. Use SetExtension to
	// validate keys.
	Extensions map[string]any `json:",embed"`
}

// License represents the license informations
// exposed for an API.
type License struct {
	Name       string `json:"name"`
	Identifier string `json:"identifier,omitempty"`
	URL        string `json:"url,omitempty"`

	// Extensions holds the "x-" specification extensions. Keys are
	// emitted sorted after the typed members. Use SetExtension to
	// validate keys.
	Extensions map[string]any `json:",embed"`
}

// Server represents a server.
type Server struct {
	URL         string                     `json:"url"`
	Name        string                     `json:"name,omitempty"`
	Description string                     `json:"description,omitempty"`
	Variables   map[string]*ServerVariable `json:"variables,omitempty"`

	// Extensions holds the "x-" specification extensions. Keys are
	// emitted sorted after the typed members. Use SetExtension to
	// validate keys.
	Extensions map[string]any `json:",embed"`
}

// ServerVariable represents a server variable for server
// URL template substitution.
type ServerVariable struct {
	Enum        []string `json:"enum,omitempty"`
	Default     string   `json:"default"`
	Description string   `json:"description,omitempty"`

	// Extensions holds the "x-" specification extensions. Keys are
	// emitted sorted after the typed members. Use SetExtension to
	// validate keys.
	Extensions map[string]any `json:",embed"`
}

// Paths represents the relative paths to the individual
// endpoints and their operations.
type Paths map[string]*PathItem

// PathItem describes the operations available on a single
// API path.
type PathItem struct {
	Ref         string            `json:"$ref,omitempty"`
	Summary     string            `json:"summary,omitempty"`
	Description string            `json:"description,omitempty"`
	GET         *Operation        `json:"get,omitzero"`
	PUT         *Operation        `json:"put,omitzero"`
	POST        *Operation        `json:"post,omitzero"`
	DELETE      *Operation        `json:"delete,omitzero"`
	OPTIONS     *Operation        `json:"options,omitzero"`
	HEAD        *Operation        `json:"head,omitzero"`
	PATCH       *Operation        `json:"patch,omitzero"`
	TRACE       *Operation        `json:"trace,omitzero"`
	Servers     []*Server         `json:"servers,omitempty"`
	Parameters  []*ParameterOrRef `json:"parameters,omitempty"`
	Query       *Operation        `json:"query,omitzero"`

	// AdditionalOperations holds operations for HTTP methods
	// that have no dedicated field, keyed by method name.
	AdditionalOperations map[string]*Operation `json:"additionalOperations,omitempty"`

	// Extensions holds the "x-" specification extensions. Keys are
	// emitted sorted after the typed members. Use SetExtension to
	// validate keys.
	Extensions map[string]any `json:",embed"`
}

// Tag represents the metadata of a single tag.
type Tag struct {
	Name         string        `json:"name"`
	Summary      string        `json:"summary,omitempty"`
	Description  string        `json:"description,omitempty"`
	ExternalDocs *ExternalDocs `json:"externalDocs,omitzero"`
	Parent       string        `json:"parent,omitempty"`
	Kind         string        `json:"kind,omitempty"`

	// Extensions holds the "x-" specification extensions. Keys are
	// emitted sorted after the typed members. Use SetExtension to
	// validate keys.
	Extensions map[string]any `json:",embed"`
}
