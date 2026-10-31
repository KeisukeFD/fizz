package openapi

// XLogo represents the information about the x-logo extension.
// See: https://github.com/Redocly/redoc/blob/master/docs/redoc-vendor-extensions.md#x-logo
type XLogo struct {
	URL             string `json:"url,omitempty"`
	BackgroundColor string `json:"backgroundColor,omitempty"`
	AltText         string `json:"altText,omitempty"`
	Href            string `json:"href,omitempty"`
}

// XTagGroup represents the information about the x-tagGroups extension.
// See: https://github.com/Redocly/redoc/blob/master/docs/redoc-vendor-extensions.md#x-taggroups
type XTagGroup struct {
	Name string   `json:"name,omitempty"`
	Tags []string `json:"tags,omitempty"`
}

// XCodeSample represents the information about the x-codeSample extension.
// See: https://github.com/Redocly/redoc/blob/master/docs/redoc-vendor-extensions.md#x-codesamples
type XCodeSample struct {
	Lang   string `json:"lang,omitempty"`
	Label  string `json:"label,omitempty"`
	Source string `json:"source,omitempty"`
}

// SetExtension methods of the extensible objects. See setExtension.

// SetExtension sets the specification extension key to value. The key
// must start with "x-" followed by a name. It returns
// ErrReservedExtension for the reserved "x-oai-" and "x-oas-" prefixes,
// and an error wrapping ErrInvalidExtensionKey for any other invalid key
// or for a key that OpenAPI models as a typed field.
func (x *OpenAPI) SetExtension(key string, value any) error {
	return setExtension(&x.Extensions, key, value, "x-tagGroups")
}

// SetExtension sets the specification extension key to value. The key
// must start with "x-" followed by a name. It returns
// ErrReservedExtension for the reserved "x-oai-" and "x-oas-" prefixes,
// and an error wrapping ErrInvalidExtensionKey for any other invalid key
// or for a key that Info models as a typed field.
func (x *Info) SetExtension(key string, value any) error {
	return setExtension(&x.Extensions, key, value, "x-logo")
}

// SetExtension sets the specification extension key to value. The key
// must start with "x-" followed by a name. It returns
// ErrReservedExtension for the reserved "x-oai-" and "x-oas-" prefixes,
// and an error wrapping ErrInvalidExtensionKey for any other invalid key.
func (x *Contact) SetExtension(key string, value any) error {
	return setExtension(&x.Extensions, key, value)
}

// SetExtension sets the specification extension key to value. The key
// must start with "x-" followed by a name. It returns
// ErrReservedExtension for the reserved "x-oai-" and "x-oas-" prefixes,
// and an error wrapping ErrInvalidExtensionKey for any other invalid key.
func (x *License) SetExtension(key string, value any) error {
	return setExtension(&x.Extensions, key, value)
}

// SetExtension sets the specification extension key to value. The key
// must start with "x-" followed by a name. It returns
// ErrReservedExtension for the reserved "x-oai-" and "x-oas-" prefixes,
// and an error wrapping ErrInvalidExtensionKey for any other invalid key.
func (x *Server) SetExtension(key string, value any) error {
	return setExtension(&x.Extensions, key, value)
}

// SetExtension sets the specification extension key to value. The key
// must start with "x-" followed by a name. It returns
// ErrReservedExtension for the reserved "x-oai-" and "x-oas-" prefixes,
// and an error wrapping ErrInvalidExtensionKey for any other invalid key.
func (x *ServerVariable) SetExtension(key string, value any) error {
	return setExtension(&x.Extensions, key, value)
}

// SetExtension sets the specification extension key to value. The key
// must start with "x-" followed by a name. It returns
// ErrReservedExtension for the reserved "x-oai-" and "x-oas-" prefixes,
// and an error wrapping ErrInvalidExtensionKey for any other invalid key.
func (x *Components) SetExtension(key string, value any) error {
	return setExtension(&x.Extensions, key, value)
}

// SetExtension sets the specification extension key to value. The key
// must start with "x-" followed by a name. It returns
// ErrReservedExtension for the reserved "x-oai-" and "x-oas-" prefixes,
// and an error wrapping ErrInvalidExtensionKey for any other invalid key.
func (x *PathItem) SetExtension(key string, value any) error {
	return setExtension(&x.Extensions, key, value)
}

// SetExtension sets the specification extension key to value. The key
// must start with "x-" followed by a name. It returns
// ErrReservedExtension for the reserved "x-oai-" and "x-oas-" prefixes,
// and an error wrapping ErrInvalidExtensionKey for any other invalid key
// or for a key that Operation models as a typed field.
func (x *Operation) SetExtension(key string, value any) error {
	return setExtension(&x.Extensions, key, value, "x-codeSamples", "x-internal")
}

// SetExtension sets the specification extension key to value. The key
// must start with "x-" followed by a name. It returns
// ErrReservedExtension for the reserved "x-oai-" and "x-oas-" prefixes,
// and an error wrapping ErrInvalidExtensionKey for any other invalid key.
func (x *ExternalDocs) SetExtension(key string, value any) error {
	return setExtension(&x.Extensions, key, value)
}

// SetExtension sets the specification extension key to value. The key
// must start with "x-" followed by a name. It returns
// ErrReservedExtension for the reserved "x-oai-" and "x-oas-" prefixes,
// and an error wrapping ErrInvalidExtensionKey for any other invalid key.
func (x *Parameter) SetExtension(key string, value any) error {
	return setExtension(&x.Extensions, key, value)
}

// SetExtension sets the specification extension key to value. The key
// must start with "x-" followed by a name. It returns
// ErrReservedExtension for the reserved "x-oai-" and "x-oas-" prefixes,
// and an error wrapping ErrInvalidExtensionKey for any other invalid key.
func (x *RequestBody) SetExtension(key string, value any) error {
	return setExtension(&x.Extensions, key, value)
}

// SetExtension sets the specification extension key to value. The key
// must start with "x-" followed by a name. It returns
// ErrReservedExtension for the reserved "x-oai-" and "x-oas-" prefixes,
// and an error wrapping ErrInvalidExtensionKey for any other invalid key.
func (x *MediaType) SetExtension(key string, value any) error {
	return setExtension(&x.Extensions, key, value)
}

// SetExtension sets the specification extension key to value. The key
// must start with "x-" followed by a name. It returns
// ErrReservedExtension for the reserved "x-oai-" and "x-oas-" prefixes,
// and an error wrapping ErrInvalidExtensionKey for any other invalid key.
func (x *Encoding) SetExtension(key string, value any) error {
	return setExtension(&x.Extensions, key, value)
}

// SetExtension sets the specification extension key to value. The key
// must start with "x-" followed by a name. It returns
// ErrReservedExtension for the reserved "x-oai-" and "x-oas-" prefixes,
// and an error wrapping ErrInvalidExtensionKey for any other invalid key.
func (x *Response) SetExtension(key string, value any) error {
	return setExtension(&x.Extensions, key, value)
}

// SetExtension sets the specification extension key to value. The key
// must start with "x-" followed by a name. It returns
// ErrReservedExtension for the reserved "x-oai-" and "x-oas-" prefixes,
// and an error wrapping ErrInvalidExtensionKey for any other invalid key.
func (x *Header) SetExtension(key string, value any) error {
	return setExtension(&x.Extensions, key, value)
}

// SetExtension sets the specification extension key to value. The key
// must start with "x-" followed by a name. It returns
// ErrReservedExtension for the reserved "x-oai-" and "x-oas-" prefixes,
// and an error wrapping ErrInvalidExtensionKey for any other invalid key.
func (x *Example) SetExtension(key string, value any) error {
	return setExtension(&x.Extensions, key, value)
}

// SetExtension sets the specification extension key to value. The key
// must start with "x-" followed by a name. It returns
// ErrReservedExtension for the reserved "x-oai-" and "x-oas-" prefixes,
// and an error wrapping ErrInvalidExtensionKey for any other invalid key.
func (x *Link) SetExtension(key string, value any) error {
	return setExtension(&x.Extensions, key, value)
}

// SetExtension sets the specification extension key to value. The key
// must start with "x-" followed by a name. It returns
// ErrReservedExtension for the reserved "x-oai-" and "x-oas-" prefixes,
// and an error wrapping ErrInvalidExtensionKey for any other invalid key.
func (x *Tag) SetExtension(key string, value any) error {
	return setExtension(&x.Extensions, key, value)
}

// SetExtension sets the specification extension key to value. The key
// must start with "x-" followed by a name. It returns
// ErrReservedExtension for the reserved "x-oai-" and "x-oas-" prefixes,
// and an error wrapping ErrInvalidExtensionKey for any other invalid key.
func (x *Schema) SetExtension(key string, value any) error {
	return setExtension(&x.Extensions, key, value)
}

// SetExtension sets the specification extension key to value. The key
// must start with "x-" followed by a name. It returns
// ErrReservedExtension for the reserved "x-oai-" and "x-oas-" prefixes,
// and an error wrapping ErrInvalidExtensionKey for any other invalid key.
func (x *Discriminator) SetExtension(key string, value any) error {
	return setExtension(&x.Extensions, key, value)
}

// SetExtension sets the specification extension key to value. The key
// must start with "x-" followed by a name. It returns
// ErrReservedExtension for the reserved "x-oai-" and "x-oas-" prefixes,
// and an error wrapping ErrInvalidExtensionKey for any other invalid key.
func (x *XML) SetExtension(key string, value any) error {
	return setExtension(&x.Extensions, key, value)
}

// SetExtension sets the specification extension key to value. The key
// must start with "x-" followed by a name. It returns
// ErrReservedExtension for the reserved "x-oai-" and "x-oas-" prefixes,
// and an error wrapping ErrInvalidExtensionKey for any other invalid key.
func (x *SecurityScheme) SetExtension(key string, value any) error {
	return setExtension(&x.Extensions, key, value)
}

// SetExtension sets the specification extension key to value. The key
// must start with "x-" followed by a name. It returns
// ErrReservedExtension for the reserved "x-oai-" and "x-oas-" prefixes,
// and an error wrapping ErrInvalidExtensionKey for any other invalid key.
func (x *OAuthFlows) SetExtension(key string, value any) error {
	return setExtension(&x.Extensions, key, value)
}

// SetExtension sets the specification extension key to value. The key
// must start with "x-" followed by a name. It returns
// ErrReservedExtension for the reserved "x-oai-" and "x-oas-" prefixes,
// and an error wrapping ErrInvalidExtensionKey for any other invalid key.
func (x *OAuthFlow) SetExtension(key string, value any) error {
	return setExtension(&x.Extensions, key, value)
}
