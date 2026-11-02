package openapi_test

import (
	"testing"

	"github.com/KeisukeFD/fizz/openapi"
)

// TestDirectJSONMethods calls the encoding/json (v1) methods of every spec
// object directly: encoding/json may route through the v2 methods instead.
func TestDirectJSONMethods(t *testing.T) {
	check := func(name string, b []byte, err error) {
		t.Helper()
		if err != nil || len(b) == 0 || b[0] != 0x7b {
			t.Errorf("%s: marshal = %s, %v", name, b, err)
		}
	}
	{
		b, err := openapi.OpenAPI{}.MarshalJSON()
		check("OpenAPI", b, err)
		var v openapi.OpenAPI
		if err := v.UnmarshalJSON(b); err != nil {
			t.Errorf("OpenAPI: unmarshal: %v", err)
		}
		if err := v.UnmarshalJSON([]byte("[]")); err == nil {
			t.Errorf("OpenAPI: expected error decoding an array")
		}
	}
	{
		b, err := openapi.Info{}.MarshalJSON()
		check("Info", b, err)
		var v openapi.Info
		if err := v.UnmarshalJSON(b); err != nil {
			t.Errorf("Info: unmarshal: %v", err)
		}
		if err := v.UnmarshalJSON([]byte("[]")); err == nil {
			t.Errorf("Info: expected error decoding an array")
		}
	}
	{
		b, err := openapi.Contact{}.MarshalJSON()
		check("Contact", b, err)
		var v openapi.Contact
		if err := v.UnmarshalJSON(b); err != nil {
			t.Errorf("Contact: unmarshal: %v", err)
		}
		if err := v.UnmarshalJSON([]byte("[]")); err == nil {
			t.Errorf("Contact: expected error decoding an array")
		}
	}
	{
		b, err := openapi.License{}.MarshalJSON()
		check("License", b, err)
		var v openapi.License
		if err := v.UnmarshalJSON(b); err != nil {
			t.Errorf("License: unmarshal: %v", err)
		}
		if err := v.UnmarshalJSON([]byte("[]")); err == nil {
			t.Errorf("License: expected error decoding an array")
		}
	}
	{
		b, err := openapi.Server{}.MarshalJSON()
		check("Server", b, err)
		var v openapi.Server
		if err := v.UnmarshalJSON(b); err != nil {
			t.Errorf("Server: unmarshal: %v", err)
		}
		if err := v.UnmarshalJSON([]byte("[]")); err == nil {
			t.Errorf("Server: expected error decoding an array")
		}
	}
	{
		b, err := openapi.ServerVariable{}.MarshalJSON()
		check("ServerVariable", b, err)
		var v openapi.ServerVariable
		if err := v.UnmarshalJSON(b); err != nil {
			t.Errorf("ServerVariable: unmarshal: %v", err)
		}
		if err := v.UnmarshalJSON([]byte("[]")); err == nil {
			t.Errorf("ServerVariable: expected error decoding an array")
		}
	}
	{
		b, err := openapi.Components{}.MarshalJSON()
		check("Components", b, err)
		var v openapi.Components
		if err := v.UnmarshalJSON(b); err != nil {
			t.Errorf("Components: unmarshal: %v", err)
		}
		if err := v.UnmarshalJSON([]byte("[]")); err == nil {
			t.Errorf("Components: expected error decoding an array")
		}
	}
	{
		b, err := openapi.PathItem{}.MarshalJSON()
		check("PathItem", b, err)
		var v openapi.PathItem
		if err := v.UnmarshalJSON(b); err != nil {
			t.Errorf("PathItem: unmarshal: %v", err)
		}
		if err := v.UnmarshalJSON([]byte("[]")); err == nil {
			t.Errorf("PathItem: expected error decoding an array")
		}
	}
	{
		b, err := openapi.Operation{}.MarshalJSON()
		check("Operation", b, err)
		var v openapi.Operation
		if err := v.UnmarshalJSON(b); err != nil {
			t.Errorf("Operation: unmarshal: %v", err)
		}
		if err := v.UnmarshalJSON([]byte("[]")); err == nil {
			t.Errorf("Operation: expected error decoding an array")
		}
	}
	{
		b, err := openapi.ExternalDocs{}.MarshalJSON()
		check("ExternalDocs", b, err)
		var v openapi.ExternalDocs
		if err := v.UnmarshalJSON(b); err != nil {
			t.Errorf("ExternalDocs: unmarshal: %v", err)
		}
		if err := v.UnmarshalJSON([]byte("[]")); err == nil {
			t.Errorf("ExternalDocs: expected error decoding an array")
		}
	}
	{
		b, err := openapi.Parameter{}.MarshalJSON()
		check("Parameter", b, err)
		var v openapi.Parameter
		if err := v.UnmarshalJSON(b); err != nil {
			t.Errorf("Parameter: unmarshal: %v", err)
		}
		if err := v.UnmarshalJSON([]byte("[]")); err == nil {
			t.Errorf("Parameter: expected error decoding an array")
		}
	}
	{
		b, err := openapi.RequestBody{}.MarshalJSON()
		check("RequestBody", b, err)
		var v openapi.RequestBody
		if err := v.UnmarshalJSON(b); err != nil {
			t.Errorf("RequestBody: unmarshal: %v", err)
		}
		if err := v.UnmarshalJSON([]byte("[]")); err == nil {
			t.Errorf("RequestBody: expected error decoding an array")
		}
	}
	{
		b, err := openapi.MediaType{}.MarshalJSON()
		check("MediaType", b, err)
		var v openapi.MediaType
		if err := v.UnmarshalJSON(b); err != nil {
			t.Errorf("MediaType: unmarshal: %v", err)
		}
		if err := v.UnmarshalJSON([]byte("[]")); err == nil {
			t.Errorf("MediaType: expected error decoding an array")
		}
	}
	{
		b, err := openapi.Encoding{}.MarshalJSON()
		check("Encoding", b, err)
		var v openapi.Encoding
		if err := v.UnmarshalJSON(b); err != nil {
			t.Errorf("Encoding: unmarshal: %v", err)
		}
		if err := v.UnmarshalJSON([]byte("[]")); err == nil {
			t.Errorf("Encoding: expected error decoding an array")
		}
	}
	{
		b, err := openapi.Response{}.MarshalJSON()
		check("Response", b, err)
		var v openapi.Response
		if err := v.UnmarshalJSON(b); err != nil {
			t.Errorf("Response: unmarshal: %v", err)
		}
		if err := v.UnmarshalJSON([]byte("[]")); err == nil {
			t.Errorf("Response: expected error decoding an array")
		}
	}
	{
		b, err := openapi.Header{}.MarshalJSON()
		check("Header", b, err)
		var v openapi.Header
		if err := v.UnmarshalJSON(b); err != nil {
			t.Errorf("Header: unmarshal: %v", err)
		}
		if err := v.UnmarshalJSON([]byte("[]")); err == nil {
			t.Errorf("Header: expected error decoding an array")
		}
	}
	{
		b, err := openapi.Example{}.MarshalJSON()
		check("Example", b, err)
		var v openapi.Example
		if err := v.UnmarshalJSON(b); err != nil {
			t.Errorf("Example: unmarshal: %v", err)
		}
		if err := v.UnmarshalJSON([]byte("[]")); err == nil {
			t.Errorf("Example: expected error decoding an array")
		}
	}
	{
		b, err := openapi.Link{}.MarshalJSON()
		check("Link", b, err)
		var v openapi.Link
		if err := v.UnmarshalJSON(b); err != nil {
			t.Errorf("Link: unmarshal: %v", err)
		}
		if err := v.UnmarshalJSON([]byte("[]")); err == nil {
			t.Errorf("Link: expected error decoding an array")
		}
	}
	{
		b, err := openapi.Tag{}.MarshalJSON()
		check("Tag", b, err)
		var v openapi.Tag
		if err := v.UnmarshalJSON(b); err != nil {
			t.Errorf("Tag: unmarshal: %v", err)
		}
		if err := v.UnmarshalJSON([]byte("[]")); err == nil {
			t.Errorf("Tag: expected error decoding an array")
		}
	}
	{
		b, err := openapi.Discriminator{}.MarshalJSON()
		check("Discriminator", b, err)
		var v openapi.Discriminator
		if err := v.UnmarshalJSON(b); err != nil {
			t.Errorf("Discriminator: unmarshal: %v", err)
		}
		if err := v.UnmarshalJSON([]byte("[]")); err == nil {
			t.Errorf("Discriminator: expected error decoding an array")
		}
	}
	{
		b, err := openapi.XML{}.MarshalJSON()
		check("XML", b, err)
		var v openapi.XML
		if err := v.UnmarshalJSON(b); err != nil {
			t.Errorf("XML: unmarshal: %v", err)
		}
		if err := v.UnmarshalJSON([]byte("[]")); err == nil {
			t.Errorf("XML: expected error decoding an array")
		}
	}
	{
		b, err := openapi.SecurityScheme{}.MarshalJSON()
		check("SecurityScheme", b, err)
		var v openapi.SecurityScheme
		if err := v.UnmarshalJSON(b); err != nil {
			t.Errorf("SecurityScheme: unmarshal: %v", err)
		}
		if err := v.UnmarshalJSON([]byte("[]")); err == nil {
			t.Errorf("SecurityScheme: expected error decoding an array")
		}
	}
	{
		b, err := openapi.OAuthFlows{}.MarshalJSON()
		check("OAuthFlows", b, err)
		var v openapi.OAuthFlows
		if err := v.UnmarshalJSON(b); err != nil {
			t.Errorf("OAuthFlows: unmarshal: %v", err)
		}
		if err := v.UnmarshalJSON([]byte("[]")); err == nil {
			t.Errorf("OAuthFlows: expected error decoding an array")
		}
	}
	{
		b, err := openapi.OAuthFlow{}.MarshalJSON()
		check("OAuthFlow", b, err)
		var v openapi.OAuthFlow
		if err := v.UnmarshalJSON(b); err != nil {
			t.Errorf("OAuthFlow: unmarshal: %v", err)
		}
		if err := v.UnmarshalJSON([]byte("[]")); err == nil {
			t.Errorf("OAuthFlow: expected error decoding an array")
		}
	}
}
