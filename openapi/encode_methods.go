package openapi

// This file bridges the model to encoding/json (v1). Each extensible
// type has the v2 pair (MarshalJSONTo, UnmarshalJSONFrom), which keeps the
// options of the caller, and the v1 pair (MarshalJSON, UnmarshalJSON),
// which delegates to v2 with the options of MarshalJSON. The alias
// types carry the fields but not the methods, which avoids recursion.

import (
	"encoding/json/jsontext"
)

type openAPIJSON OpenAPI

// MarshalJSONTo implements jsonv2.MarshalerTo for OpenAPI.
func (x OpenAPI) MarshalJSONTo(enc *jsontext.Encoder) error {
	return encodeValue(enc, openAPIJSON(x))
}

// UnmarshalJSONFrom implements jsonv2.UnmarshalerFrom for OpenAPI.
func (x *OpenAPI) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	return decodeValue(dec, (*openAPIJSON)(x))
}

// MarshalJSON implements json.Marshaler for OpenAPI.
func (x OpenAPI) MarshalJSON() ([]byte, error) { return marshalJSON(x) }

// UnmarshalJSON implements json.Unmarshaler for OpenAPI.
func (x *OpenAPI) UnmarshalJSON(b []byte) error { return unmarshalJSON(b, x) }

type infoJSON Info

// MarshalJSONTo implements jsonv2.MarshalerTo for Info.
func (x Info) MarshalJSONTo(enc *jsontext.Encoder) error {
	return encodeValue(enc, infoJSON(x))
}

// UnmarshalJSONFrom implements jsonv2.UnmarshalerFrom for Info.
func (x *Info) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	return decodeValue(dec, (*infoJSON)(x))
}

// MarshalJSON implements json.Marshaler for Info.
func (x Info) MarshalJSON() ([]byte, error) { return marshalJSON(x) }

// UnmarshalJSON implements json.Unmarshaler for Info.
func (x *Info) UnmarshalJSON(b []byte) error { return unmarshalJSON(b, x) }

type contactJSON Contact

// MarshalJSONTo implements jsonv2.MarshalerTo for Contact.
func (x Contact) MarshalJSONTo(enc *jsontext.Encoder) error {
	return encodeValue(enc, contactJSON(x))
}

// UnmarshalJSONFrom implements jsonv2.UnmarshalerFrom for Contact.
func (x *Contact) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	return decodeValue(dec, (*contactJSON)(x))
}

// MarshalJSON implements json.Marshaler for Contact.
func (x Contact) MarshalJSON() ([]byte, error) { return marshalJSON(x) }

// UnmarshalJSON implements json.Unmarshaler for Contact.
func (x *Contact) UnmarshalJSON(b []byte) error { return unmarshalJSON(b, x) }

type licenseJSON License

// MarshalJSONTo implements jsonv2.MarshalerTo for License.
func (x License) MarshalJSONTo(enc *jsontext.Encoder) error {
	return encodeValue(enc, licenseJSON(x))
}

// UnmarshalJSONFrom implements jsonv2.UnmarshalerFrom for License.
func (x *License) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	return decodeValue(dec, (*licenseJSON)(x))
}

// MarshalJSON implements json.Marshaler for License.
func (x License) MarshalJSON() ([]byte, error) { return marshalJSON(x) }

// UnmarshalJSON implements json.Unmarshaler for License.
func (x *License) UnmarshalJSON(b []byte) error { return unmarshalJSON(b, x) }

type serverJSON Server

// MarshalJSONTo implements jsonv2.MarshalerTo for Server.
func (x Server) MarshalJSONTo(enc *jsontext.Encoder) error {
	return encodeValue(enc, serverJSON(x))
}

// UnmarshalJSONFrom implements jsonv2.UnmarshalerFrom for Server.
func (x *Server) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	return decodeValue(dec, (*serverJSON)(x))
}

// MarshalJSON implements json.Marshaler for Server.
func (x Server) MarshalJSON() ([]byte, error) { return marshalJSON(x) }

// UnmarshalJSON implements json.Unmarshaler for Server.
func (x *Server) UnmarshalJSON(b []byte) error { return unmarshalJSON(b, x) }

type serverVariableJSON ServerVariable

// MarshalJSONTo implements jsonv2.MarshalerTo for ServerVariable.
func (x ServerVariable) MarshalJSONTo(enc *jsontext.Encoder) error {
	return encodeValue(enc, serverVariableJSON(x))
}

// UnmarshalJSONFrom implements jsonv2.UnmarshalerFrom for ServerVariable.
func (x *ServerVariable) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	return decodeValue(dec, (*serverVariableJSON)(x))
}

// MarshalJSON implements json.Marshaler for ServerVariable.
func (x ServerVariable) MarshalJSON() ([]byte, error) { return marshalJSON(x) }

// UnmarshalJSON implements json.Unmarshaler for ServerVariable.
func (x *ServerVariable) UnmarshalJSON(b []byte) error { return unmarshalJSON(b, x) }

type componentsJSON Components

// MarshalJSONTo implements jsonv2.MarshalerTo for Components.
func (x Components) MarshalJSONTo(enc *jsontext.Encoder) error {
	return encodeValue(enc, componentsJSON(x))
}

// UnmarshalJSONFrom implements jsonv2.UnmarshalerFrom for Components.
func (x *Components) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	return decodeValue(dec, (*componentsJSON)(x))
}

// MarshalJSON implements json.Marshaler for Components.
func (x Components) MarshalJSON() ([]byte, error) { return marshalJSON(x) }

// UnmarshalJSON implements json.Unmarshaler for Components.
func (x *Components) UnmarshalJSON(b []byte) error { return unmarshalJSON(b, x) }

type pathItemJSON PathItem

// MarshalJSONTo implements jsonv2.MarshalerTo for PathItem.
func (x PathItem) MarshalJSONTo(enc *jsontext.Encoder) error {
	return encodeValue(enc, pathItemJSON(x))
}

// UnmarshalJSONFrom implements jsonv2.UnmarshalerFrom for PathItem.
func (x *PathItem) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	return decodeValue(dec, (*pathItemJSON)(x))
}

// MarshalJSON implements json.Marshaler for PathItem.
func (x PathItem) MarshalJSON() ([]byte, error) { return marshalJSON(x) }

// UnmarshalJSON implements json.Unmarshaler for PathItem.
func (x *PathItem) UnmarshalJSON(b []byte) error { return unmarshalJSON(b, x) }

type operationJSON Operation

// MarshalJSONTo implements jsonv2.MarshalerTo for Operation.
func (x Operation) MarshalJSONTo(enc *jsontext.Encoder) error {
	return encodeValue(enc, operationJSON(x))
}

// UnmarshalJSONFrom implements jsonv2.UnmarshalerFrom for Operation.
func (x *Operation) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	return decodeValue(dec, (*operationJSON)(x))
}

// MarshalJSON implements json.Marshaler for Operation.
func (x Operation) MarshalJSON() ([]byte, error) { return marshalJSON(x) }

// UnmarshalJSON implements json.Unmarshaler for Operation.
func (x *Operation) UnmarshalJSON(b []byte) error { return unmarshalJSON(b, x) }

type externalDocsJSON ExternalDocs

// MarshalJSONTo implements jsonv2.MarshalerTo for ExternalDocs.
func (x ExternalDocs) MarshalJSONTo(enc *jsontext.Encoder) error {
	return encodeValue(enc, externalDocsJSON(x))
}

// UnmarshalJSONFrom implements jsonv2.UnmarshalerFrom for ExternalDocs.
func (x *ExternalDocs) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	return decodeValue(dec, (*externalDocsJSON)(x))
}

// MarshalJSON implements json.Marshaler for ExternalDocs.
func (x ExternalDocs) MarshalJSON() ([]byte, error) { return marshalJSON(x) }

// UnmarshalJSON implements json.Unmarshaler for ExternalDocs.
func (x *ExternalDocs) UnmarshalJSON(b []byte) error { return unmarshalJSON(b, x) }

type parameterJSON Parameter

// MarshalJSONTo implements jsonv2.MarshalerTo for Parameter.
func (x Parameter) MarshalJSONTo(enc *jsontext.Encoder) error {
	if x.requiredSet && !x.Required {
		return encodeValue(enc, struct {
			parameterJSON
			Required *bool `json:"required"`
		}{parameterJSON(x), new(false)})
	}
	return encodeValue(enc, parameterJSON(x))
}

// UnmarshalJSONFrom implements jsonv2.UnmarshalerFrom for Parameter.
func (x *Parameter) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	v, err := dec.ReadValue()
	if err != nil {
		return err
	}
	if err := unmarshalJSON(v, (*parameterJSON)(x)); err != nil {
		return err
	}
	x.requiredSet = hasMember(v, "required")
	return nil
}

// MarshalJSON implements json.Marshaler for Parameter.
func (x Parameter) MarshalJSON() ([]byte, error) { return marshalJSON(x) }

// UnmarshalJSON implements json.Unmarshaler for Parameter.
func (x *Parameter) UnmarshalJSON(b []byte) error { return unmarshalJSON(b, x) }

type requestBodyJSON RequestBody

// MarshalJSONTo implements jsonv2.MarshalerTo for RequestBody.
func (x RequestBody) MarshalJSONTo(enc *jsontext.Encoder) error {
	return encodeValue(enc, requestBodyJSON(x))
}

// UnmarshalJSONFrom implements jsonv2.UnmarshalerFrom for RequestBody.
func (x *RequestBody) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	return decodeValue(dec, (*requestBodyJSON)(x))
}

// MarshalJSON implements json.Marshaler for RequestBody.
func (x RequestBody) MarshalJSON() ([]byte, error) { return marshalJSON(x) }

// UnmarshalJSON implements json.Unmarshaler for RequestBody.
func (x *RequestBody) UnmarshalJSON(b []byte) error { return unmarshalJSON(b, x) }

type mediaTypeJSON MediaType

// MarshalJSONTo implements jsonv2.MarshalerTo for MediaType.
func (x MediaType) MarshalJSONTo(enc *jsontext.Encoder) error {
	return encodeValue(enc, mediaTypeJSON(x))
}

// UnmarshalJSONFrom implements jsonv2.UnmarshalerFrom for MediaType.
func (x *MediaType) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	return decodeValue(dec, (*mediaTypeJSON)(x))
}

// MarshalJSON implements json.Marshaler for MediaType.
func (x MediaType) MarshalJSON() ([]byte, error) { return marshalJSON(x) }

// UnmarshalJSON implements json.Unmarshaler for MediaType.
func (x *MediaType) UnmarshalJSON(b []byte) error { return unmarshalJSON(b, x) }

type encodingJSON Encoding

// MarshalJSONTo implements jsonv2.MarshalerTo for Encoding.
func (x Encoding) MarshalJSONTo(enc *jsontext.Encoder) error {
	return encodeValue(enc, encodingJSON(x))
}

// UnmarshalJSONFrom implements jsonv2.UnmarshalerFrom for Encoding.
func (x *Encoding) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	return decodeValue(dec, (*encodingJSON)(x))
}

// MarshalJSON implements json.Marshaler for Encoding.
func (x Encoding) MarshalJSON() ([]byte, error) { return marshalJSON(x) }

// UnmarshalJSON implements json.Unmarshaler for Encoding.
func (x *Encoding) UnmarshalJSON(b []byte) error { return unmarshalJSON(b, x) }

type responseJSON Response

// MarshalJSONTo implements jsonv2.MarshalerTo for Response.
func (x Response) MarshalJSONTo(enc *jsontext.Encoder) error {
	return encodeValue(enc, responseJSON(x))
}

// UnmarshalJSONFrom implements jsonv2.UnmarshalerFrom for Response.
func (x *Response) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	return decodeValue(dec, (*responseJSON)(x))
}

// MarshalJSON implements json.Marshaler for Response.
func (x Response) MarshalJSON() ([]byte, error) { return marshalJSON(x) }

// UnmarshalJSON implements json.Unmarshaler for Response.
func (x *Response) UnmarshalJSON(b []byte) error { return unmarshalJSON(b, x) }

type headerJSON Header

// MarshalJSONTo implements jsonv2.MarshalerTo for Header.
func (x Header) MarshalJSONTo(enc *jsontext.Encoder) error {
	if x.requiredSet && !x.Required {
		return encodeValue(enc, struct {
			headerJSON
			Required *bool `json:"required"`
		}{headerJSON(x), new(false)})
	}
	return encodeValue(enc, headerJSON(x))
}

// UnmarshalJSONFrom implements jsonv2.UnmarshalerFrom for Header.
func (x *Header) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	v, err := dec.ReadValue()
	if err != nil {
		return err
	}
	if err := unmarshalJSON(v, (*headerJSON)(x)); err != nil {
		return err
	}
	x.requiredSet = hasMember(v, "required")
	return nil
}

// MarshalJSON implements json.Marshaler for Header.
func (x Header) MarshalJSON() ([]byte, error) { return marshalJSON(x) }

// UnmarshalJSON implements json.Unmarshaler for Header.
func (x *Header) UnmarshalJSON(b []byte) error { return unmarshalJSON(b, x) }

type exampleJSON Example

// MarshalJSONTo implements jsonv2.MarshalerTo for Example.
func (x Example) MarshalJSONTo(enc *jsontext.Encoder) error {
	return encodeValue(enc, exampleJSON(x))
}

// UnmarshalJSONFrom implements jsonv2.UnmarshalerFrom for Example.
func (x *Example) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	return decodeValue(dec, (*exampleJSON)(x))
}

// MarshalJSON implements json.Marshaler for Example.
func (x Example) MarshalJSON() ([]byte, error) { return marshalJSON(x) }

// UnmarshalJSON implements json.Unmarshaler for Example.
func (x *Example) UnmarshalJSON(b []byte) error { return unmarshalJSON(b, x) }

type linkJSON Link

// MarshalJSONTo implements jsonv2.MarshalerTo for Link.
func (x Link) MarshalJSONTo(enc *jsontext.Encoder) error {
	return encodeValue(enc, linkJSON(x))
}

// UnmarshalJSONFrom implements jsonv2.UnmarshalerFrom for Link.
func (x *Link) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	return decodeValue(dec, (*linkJSON)(x))
}

// MarshalJSON implements json.Marshaler for Link.
func (x Link) MarshalJSON() ([]byte, error) { return marshalJSON(x) }

// UnmarshalJSON implements json.Unmarshaler for Link.
func (x *Link) UnmarshalJSON(b []byte) error { return unmarshalJSON(b, x) }

type tagJSON Tag

// MarshalJSONTo implements jsonv2.MarshalerTo for Tag.
func (x Tag) MarshalJSONTo(enc *jsontext.Encoder) error {
	return encodeValue(enc, tagJSON(x))
}

// UnmarshalJSONFrom implements jsonv2.UnmarshalerFrom for Tag.
func (x *Tag) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	return decodeValue(dec, (*tagJSON)(x))
}

// MarshalJSON implements json.Marshaler for Tag.
func (x Tag) MarshalJSON() ([]byte, error) { return marshalJSON(x) }

// UnmarshalJSON implements json.Unmarshaler for Tag.
func (x *Tag) UnmarshalJSON(b []byte) error { return unmarshalJSON(b, x) }

type discriminatorJSON Discriminator

// MarshalJSONTo implements jsonv2.MarshalerTo for Discriminator.
func (x Discriminator) MarshalJSONTo(enc *jsontext.Encoder) error {
	return encodeValue(enc, discriminatorJSON(x))
}

// UnmarshalJSONFrom implements jsonv2.UnmarshalerFrom for Discriminator.
func (x *Discriminator) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	return decodeValue(dec, (*discriminatorJSON)(x))
}

// MarshalJSON implements json.Marshaler for Discriminator.
func (x Discriminator) MarshalJSON() ([]byte, error) { return marshalJSON(x) }

// UnmarshalJSON implements json.Unmarshaler for Discriminator.
func (x *Discriminator) UnmarshalJSON(b []byte) error { return unmarshalJSON(b, x) }

type xmlJSON XML

// MarshalJSONTo implements jsonv2.MarshalerTo for XML.
func (x XML) MarshalJSONTo(enc *jsontext.Encoder) error {
	return encodeValue(enc, xmlJSON(x))
}

// UnmarshalJSONFrom implements jsonv2.UnmarshalerFrom for XML.
func (x *XML) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	return decodeValue(dec, (*xmlJSON)(x))
}

// MarshalJSON implements json.Marshaler for XML.
func (x XML) MarshalJSON() ([]byte, error) { return marshalJSON(x) }

// UnmarshalJSON implements json.Unmarshaler for XML.
func (x *XML) UnmarshalJSON(b []byte) error { return unmarshalJSON(b, x) }

type securitySchemeJSON SecurityScheme

// MarshalJSONTo implements jsonv2.MarshalerTo for SecurityScheme.
func (x SecurityScheme) MarshalJSONTo(enc *jsontext.Encoder) error {
	return encodeValue(enc, securitySchemeJSON(x))
}

// UnmarshalJSONFrom implements jsonv2.UnmarshalerFrom for SecurityScheme.
func (x *SecurityScheme) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	return decodeValue(dec, (*securitySchemeJSON)(x))
}

// MarshalJSON implements json.Marshaler for SecurityScheme.
func (x SecurityScheme) MarshalJSON() ([]byte, error) { return marshalJSON(x) }

// UnmarshalJSON implements json.Unmarshaler for SecurityScheme.
func (x *SecurityScheme) UnmarshalJSON(b []byte) error { return unmarshalJSON(b, x) }

type oAuthFlowsJSON OAuthFlows

// MarshalJSONTo implements jsonv2.MarshalerTo for OAuthFlows.
func (x OAuthFlows) MarshalJSONTo(enc *jsontext.Encoder) error {
	return encodeValue(enc, oAuthFlowsJSON(x))
}

// UnmarshalJSONFrom implements jsonv2.UnmarshalerFrom for OAuthFlows.
func (x *OAuthFlows) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	return decodeValue(dec, (*oAuthFlowsJSON)(x))
}

// MarshalJSON implements json.Marshaler for OAuthFlows.
func (x OAuthFlows) MarshalJSON() ([]byte, error) { return marshalJSON(x) }

// UnmarshalJSON implements json.Unmarshaler for OAuthFlows.
func (x *OAuthFlows) UnmarshalJSON(b []byte) error { return unmarshalJSON(b, x) }

type oAuthFlowJSON OAuthFlow

// MarshalJSONTo implements jsonv2.MarshalerTo for OAuthFlow.
// A nil Scopes is emitted as {}: the field is REQUIRED and MAY be empty.
func (x OAuthFlow) MarshalJSONTo(enc *jsontext.Encoder) error {
	if x.Scopes == nil {
		x.Scopes = map[string]string{}
	}
	return encodeValue(enc, oAuthFlowJSON(x))
}

// UnmarshalJSONFrom implements jsonv2.UnmarshalerFrom for OAuthFlow.
func (x *OAuthFlow) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	return decodeValue(dec, (*oAuthFlowJSON)(x))
}

// MarshalJSON implements json.Marshaler for OAuthFlow.
func (x OAuthFlow) MarshalJSON() ([]byte, error) { return marshalJSON(x) }

// UnmarshalJSON implements json.Unmarshaler for OAuthFlow.
func (x *OAuthFlow) UnmarshalJSON(b []byte) error { return unmarshalJSON(b, x) }
