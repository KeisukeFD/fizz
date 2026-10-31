package openapi

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

type rboUpload struct {
	File string `json:"file"`
	Note string `json:"note"`
}

func rboAdd(g *Generator, method string, in reflect.Type, rb *RequestBodyInfo) (*Operation, error) {
	return g.AddOperation("/p", method, "", in, nil, &OperationInfo{ID: "rbo" + method, StatusCode: 200, RequestBody: rb})
}

// rboErrors returns the recorded errors that are not warnings.
func rboErrors(g *Generator) (errs, warns []error) {
	for _, err := range g.Errors() {
		var w *Warning
		if errors.As(err, &w) {
			warns = append(warns, err)
		} else {
			errs = append(errs, err)
		}
	}
	return
}

func TestRequestBodyOptionOverrides(t *testing.T) {
	g := gen(t)
	op, err := rboAdd(g, "POST", reflect.TypeFor[rbOpt](), &RequestBodyInfo{Description: "Create", Required: true})
	if err != nil {
		t.Fatal(err)
	}
	rb := op.RequestBody
	if rb == nil || rb.Description != "Create" || !rb.Required {
		t.Fatalf("requestBody = %+v", rb)
	}
	if rbRef(op) == "" {
		t.Error("the inferred content must be kept")
	}

	g = gen(t)
	op, err = rboAdd(g, "POST", reflect.TypeFor[rbTwo](), &RequestBodyInfo{})
	if err != nil {
		t.Fatal(err)
	}
	if op.RequestBody.Required || op.RequestBody.Description != "" {
		t.Errorf("requestBody = %+v, want required and description cleared", op.RequestBody)
	}
	if errs, _ := rboErrors(g); len(errs) != 0 {
		t.Errorf("errors = %v", errs)
	}
}

func TestRequestBodyOptionContentReplaces(t *testing.T) {
	g := gen(t)
	op, err := rboAdd(g, "POST", reflect.TypeFor[rbTwo](), &RequestBodyInfo{Contents: []*ContentInfo{
		{MediaType: "application/xml", Model: rbPet{}},
		{MediaType: "text/plain"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	c := op.RequestBody.Content
	if len(c) != 2 || c["application/json"] != nil {
		t.Fatalf("content keys = %v", c)
	}
	if ref := c["application/xml"].Schema.Ref; !strings.HasSuffix(ref, "RbPet") {
		t.Errorf("xml schema ref = %q", ref)
	}
	if mt := c["text/plain"]; mt == nil || mt.Schema != nil {
		t.Errorf("text/plain = %+v, want a media type without schema", mt)
	}
	if _, ok := g.api.Components.Schemas[sanitizeName(g.typeName(reflect.TypeFor[rbPet]()))]; !ok {
		t.Errorf("rbPet not registered: %v", g.api.Components.Schemas)
	}
}

func TestRequestBodyOptionContentWithoutInput(t *testing.T) {
	g := gen(t)
	op, err := rboAdd(g, "POST", nil, &RequestBodyInfo{Required: true, Contents: []*ContentInfo{{MediaType: "application/xml", Model: &rbPet{}}}})
	if err != nil {
		t.Fatal(err)
	}
	if op.RequestBody == nil || !op.RequestBody.Required || op.RequestBody.Content["application/xml"].Schema.Ref == "" {
		t.Fatalf("requestBody = %+v", op.RequestBody)
	}
}

func TestRequestBodyOptionEncoding(t *testing.T) {
	g := gen(t)
	op, err := rboAdd(g, "POST", nil, &RequestBodyInfo{Contents: []*ContentInfo{{
		MediaType: "multipart/form-data",
		Model:     rboUpload{},
		Encodings: []*EncodingInfo{{Property: "file", ContentType: "image/png"}},
	}}})
	if err != nil {
		t.Fatal(err)
	}
	enc := op.RequestBody.Content["multipart/form-data"].Encoding
	if enc["file"] == nil || enc["file"].ContentType != "image/png" {
		t.Fatalf("encoding = %+v", enc)
	}
	if errs, warns := rboErrors(g); len(errs)+len(warns) != 0 {
		t.Errorf("errors = %v %v", errs, warns)
	}

	// An unknown property is a warning, and is still emitted.
	g = gen(t)
	op, err = rboAdd(g, "POST", nil, &RequestBodyInfo{Contents: []*ContentInfo{{
		MediaType: "multipart/form-data",
		Model:     rboUpload{},
		Encodings: []*EncodingInfo{{Property: "nope", ContentType: "image/png"}},
	}}})
	if err != nil {
		t.Fatal(err)
	}
	errs, warns := rboErrors(g)
	if len(errs) != 0 || len(warns) != 1 || !strings.Contains(warns[0].Error(), "nope") {
		t.Errorf("errs = %v, warns = %v", errs, warns)
	}
}

func TestRequestBodyOptionDuplicateMediaType(t *testing.T) {
	g := gen(t)
	op, err := rboAdd(g, "POST", nil, &RequestBodyInfo{Contents: []*ContentInfo{
		{MediaType: "application/xml", Model: rbPet{}},
		{MediaType: "application/xml", Model: rbOpt{}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if ref := op.RequestBody.Content["application/xml"].Schema.Ref; !strings.HasSuffix(ref, "RbPet") {
		t.Errorf("ref = %q, want the first kept", ref)
	}
	errs, _ := rboErrors(g)
	if len(errs) != 1 || !strings.Contains(errs[0].Error(), "application/xml") {
		t.Errorf("errors = %v", errs)
	}
}

func TestRequestBodyOptionNothingToDescribe(t *testing.T) {
	g := gen(t)
	op, err := rboAdd(g, "POST", nil, &RequestBodyInfo{Description: "x"})
	if err != nil {
		t.Fatal(err)
	}
	if op.RequestBody != nil {
		t.Errorf("requestBody = %+v, want none", op.RequestBody)
	}
	errs, _ := rboErrors(g)
	var oe *OperationError
	if len(errs) != 1 || !errors.As(errs[0], &oe) {
		t.Fatalf("errors = %v", errs)
	}
}

func TestRequestBodyOptionMethods(t *testing.T) {
	for _, m := range []string{"GET", "HEAD"} {
		t.Run(m, func(t *testing.T) {
			g := gen(t)
			op, err := rboAdd(g, m, nil, &RequestBodyInfo{Contents: []*ContentInfo{{MediaType: "application/json", Model: rbPet{}}}})
			if err != nil {
				t.Fatal(err)
			}
			if op.RequestBody != nil {
				t.Errorf("requestBody = %+v, want none", op.RequestBody)
			}
			errs, _ := rboErrors(g)
			var oe *OperationError
			if len(errs) != 1 || !errors.As(errs[0], &oe) || oe.Method != m || oe.ID != "rbo"+m {
				t.Fatalf("errors = %v", errs)
			}
			if len(g.api.Components.Schemas) != 0 {
				t.Errorf("schemas registered: %v", g.api.Components.Schemas)
			}
		})
	}
	g := gen(t)
	op, err := rboAdd(g, "DELETE", nil, &RequestBodyInfo{Contents: []*ContentInfo{{MediaType: "application/json", Model: rbPet{}}}})
	if err != nil || op.RequestBody == nil {
		t.Fatalf("DELETE: op = %+v, err = %v", op, err)
	}
}

func TestRequestBodyOptionRunsBeforeOperationHook(t *testing.T) {
	g := gen(t)
	var seen *RequestBody
	g.SetHooks(Hooks{Operation: func(_ *OperationContext, op *Operation) error {
		seen = op.RequestBody
		return nil
	}})
	if _, err := rboAdd(g, "POST", reflect.TypeFor[rbOpt](), &RequestBodyInfo{Description: "d"}); err != nil {
		t.Fatal(err)
	}
	if seen == nil || seen.Description != "d" {
		t.Errorf("hook saw %+v", seen)
	}
}
