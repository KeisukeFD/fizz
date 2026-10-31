package openapi

import (
	"fmt"
	"net/http"
	"reflect"
	"strconv"
)

func isResponseCodeRange(code string) bool {
	if len(code) != 3 {
		return false
	}
	// First char must be 1, 2, 3, 4 or 5.
	pre := code[0]
	if pre < 49 || pre > 53 {
		return false
	}
	// Last two chars are wildcard letter X.
	if code[1] != 'X' || code[2] != 'X' {
		return false
	}
	return true
}

// setOperationResponse adds a response to the operation that
// return the type t (where names the operation in errors) with the given media type and status code.
func (g *Generator) setOperationResponse(op *Operation, where string, t reflect.Type, code, mt, desc string, headers []*ResponseHeader, example any, examples map[string]any) error {
	if _, ok := op.Responses[code]; ok {
		// A response already exists for this code.
		return fmt.Errorf("response with code %s already exists", code)
	}
	if example != nil && examples != nil {
		// Cannot set both 'example' and 'examples' values
		return fmt.Errorf("'example' and 'examples' are mutually exclusive")
	}

	// Check that the response code is valid per the spec:
	// https://github.com/OAI/OpenAPI-Specification/blob/master/versions/3.0.2.md#patterned-fields-1
	if code != "default" {
		if !isResponseCodeRange(code) { // ignore ranges
			// Convert code to number and check that it is
			// between 100 and 599.
			ci, err := strconv.Atoi(code)
			if err != nil {
				return fmt.Errorf("invalid response code: %s", err)
			}
			if ci < 100 || ci > 599 {
				return fmt.Errorf("response code out of range: %s", code)
			}
			if desc == "" {
				desc = http.StatusText(ci)
			}
		}
	}
	r := &Response{
		Description: desc,
		Content:     make(map[string]*MediaTypeOrRef),
		Headers:     make(map[string]*HeaderOrRef),
	}

	// Examples that do not encode as JSON are reported and dropped, so
	// that the specification can still be served.
	if example != nil {
		if err := jsonEncodable(example); err != nil {
			g.error(fmt.Errorf("%s: response %s: example dropped, it is not JSON-encodable: %w", where, code, err))
			example = nil
		}
	}
	var castedExamples map[string]*ExampleOrRef
	if examples != nil {
		castedExamples = make(map[string]*ExampleOrRef)
		dropped := false
		for name, val := range examples {
			if err := jsonEncodable(val); err != nil {
				g.error(fmt.Errorf("%s: response %s: example %q dropped, it is not JSON-encodable: %w", where, code, name, err))
				dropped = true
				continue
			}
			castedExamples[name] = &ExampleOrRef{Value: &Example{Value: val}}
		}
		if dropped && len(castedExamples) == 0 {
			castedExamples = nil
		}
	}

	// The response may have no content type specified,
	// in which case we don't assign a schema.
	schema := g.newSchemaFromModel(t)

	if schema != nil || example != nil || castedExamples != nil {
		r.Content[mt] = &MediaTypeOrRef{Value: &MediaType{
			Schema:   schema,
			Example:  example,
			Examples: castedExamples,
		}}
	}
	// Assign headers.
	for _, h := range headers {
		if h != nil {
			var sor *Schema
			if h.Model == nil {
				// default to string if no type is given.
				sor = &Schema{Type: Types("string")}
			} else {
				sor = g.newSchemaFromModel(reflect.TypeOf(h.Model))
			}
			r.Headers[h.Name] = &HeaderOrRef{Value: &Header{
				Description: h.Description,
				Schema:      sor,
			}}
		}
	}
	op.Responses[code] = &ResponseOrRef{Value: r}

	return nil
}
