package openapi

import (
	"encoding/json/jsontext"
	"maps"
	"slices"
)

// SecuritySchemeOrRef represents a SecurityScheme that can be inlined
// or referenced in the API description.
type SecuritySchemeOrRef = OrRef[SecurityScheme]

// SecurityScheme represents a security scheme that can be used by an operation.
type SecurityScheme struct {
	Type              string      `json:"type,omitempty"`
	Scheme            string      `json:"scheme,omitempty"`
	BearerFormat      string      `json:"bearerFormat,omitempty"`
	Description       string      `json:"description,omitempty"`
	Deprecated        bool        `json:"deprecated,omitzero"`
	In                string      `json:"in,omitempty"`
	Name              string      `json:"name,omitempty"`
	OpenIDConnectURL  string      `json:"openIdConnectUrl,omitempty"`
	Flows             *OAuthFlows `json:"flows,omitzero"`
	OAuth2MetadataURL string      `json:"oauth2MetadataUrl,omitempty"`

	// Extensions holds the "x-" specification extensions. Keys are
	// emitted sorted after the typed members. Use SetExtension to
	// validate keys.
	Extensions map[string]any `json:",embed"`
}

// OAuthFlows represents all the supported OAuth flows.
type OAuthFlows struct {
	Implicit            *OAuthFlow `json:"implicit,omitzero"`
	Password            *OAuthFlow `json:"password,omitzero"`
	ClientCredentials   *OAuthFlow `json:"clientCredentials,omitzero"`
	AuthorizationCode   *OAuthFlow `json:"authorizationCode,omitzero"`
	DeviceAuthorization *OAuthFlow `json:"deviceAuthorization,omitzero"`

	// Extensions holds the "x-" specification extensions. Keys are
	// emitted sorted after the typed members. Use SetExtension to
	// validate keys.
	Extensions map[string]any `json:",embed"`
}

// OAuthFlow represents an OAuth security scheme.
type OAuthFlow struct {
	AuthorizationURL       string `json:"authorizationUrl,omitempty"`
	DeviceAuthorizationURL string `json:"deviceAuthorizationUrl,omitempty"`
	TokenURL               string `json:"tokenUrl,omitempty"`
	RefreshURL             string `json:"refreshUrl,omitempty"`
	// Scopes is REQUIRED by the specification and MAY be empty: a
	// nil map is emitted as {}.
	Scopes map[string]string `json:"scopes"`

	// Extensions holds the "x-" specification extensions. Keys are
	// emitted sorted after the typed members. Use SetExtension to
	// validate keys.
	Extensions map[string]any `json:",embed"`
}

// SecurityRequirement represents the security object in the API specification.
type SecurityRequirement map[string][]string

// MarshalJSONTo implements jsonv2.MarshalerTo for SecurityRequirement.
// A nil scope list is emitted as [], as the specification requires, and
// the names are sorted.
func (r SecurityRequirement) MarshalJSONTo(enc *jsontext.Encoder) error {
	if r == nil {
		return enc.WriteToken(jsontext.Null)
	}
	if err := enc.WriteToken(jsontext.BeginObject); err != nil {
		return err
	}
	for _, name := range slices.Sorted(maps.Keys(r)) {
		if err := enc.WriteToken(jsontext.String(name)); err != nil {
			return err
		}
		scopes := r[name]
		if scopes == nil {
			scopes = []string{}
		}
		if err := encodeValue(enc, scopes); err != nil {
			return err
		}
	}
	return enc.WriteToken(jsontext.EndObject)
}

// MarshalJSON implements json.Marshaler for SecurityRequirement.
func (r SecurityRequirement) MarshalJSON() ([]byte, error) { return marshalJSON(r) }
