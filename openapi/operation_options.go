package openapi

import (
	"errors"
	"fmt"
	"reflect"
)

// opTypedExtensions are the extension keys that Operation models as
// typed fields; the typed option wins, so an extension cannot set them.
var opTypedExtensions = []string{"x-codeSamples", "x-internal"}

// setExtensionValue validates key and value and stores a copy of the
// value in *m, the last writer of a key winning. typed lists the keys
// that the object models as fields.
func setExtensionValue(m *map[string]any, key string, value any, typed ...string) error {
	var scratch map[string]any
	if err := setExtension(&scratch, key, nil, typed...); err != nil {
		return err
	}
	if err := jsonEncodable(value); err != nil {
		return fmt.Errorf("extension %q is not JSON-encodable: %w", key, err)
	}
	return setExtension(m, key, copyAny(value))
}

// applyOperationOptions applies the ExternalDocs and Extensions of
// info to op. Invalid ones are recorded and left out.
func (g *Generator) applyOperationOptions(op *Operation, info *OperationInfo) {
	if d := info.ExternalDocs; d != nil {
		if d.URL == "" {
			g.error(errors.New("ExternalDocs: the URL is empty"))
		} else if c, err := deepClone(d); err != nil {
			g.error(fmt.Errorf("ExternalDocs: %w", err))
		} else {
			op.ExternalDocs = c
		}
	}
	for _, e := range info.Extensions {
		if e == nil {
			continue
		}
		if err := setExtensionValue(&op.Extensions, e.Key, e.Value, opTypedExtensions...); err != nil {
			g.error(err)
		}
	}
}

// addOperationResponse adds the response resp to op: the Model under the
// default media type, then the Contents, then the Extensions. A code
// that is already used, or an invalid one, is returned as an error.
// With KeepExamples, both an Example and Examples is recorded as an
// error and the examples are kept.
func (g *Generator) addOperationResponse(op *Operation, resp *OperationResponse) error {
	example := resp.Example
	if resp.KeepExamples && example != nil && len(resp.Examples) > 0 {
		g.error(fmt.Errorf("response %s: 'example' and 'examples' are mutually exclusive, the example is dropped", resp.Code))
		example = nil
	}
	if err := g.setOperationResponse(op, "", reflect.TypeOf(resp.Model), resp.Code, g.mediaType(),
		resp.Description, resp.Headers, example, resp.Examples); err != nil {
		return err
	}
	r := op.Responses[resp.Code].Value
	for _, c := range resp.Contents {
		if c == nil {
			continue
		}
		if _, ok := r.Content[c.MediaType]; ok {
			g.error(fmt.Errorf("response %s: media type %q is given twice, the first is kept", resp.Code, c.MediaType))
			continue
		}
		r.Content[c.MediaType] = &MediaTypeOrRef{Value: &MediaType{Schema: g.newSchemaFromModel(reflect.TypeOf(c.Model))}}
	}
	for _, e := range resp.Extensions {
		if e == nil {
			continue
		}
		if err := setExtensionValue(&r.Extensions, e.Key, e.Value); err != nil {
			g.error(fmt.Errorf("response %s: %w", resp.Code, err))
		}
	}
	return nil
}
