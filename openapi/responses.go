package openapi

import (
	"fmt"
	"net/http"
	"reflect"
	"strconv"
)

// defaultResponseDescription returns the description of a response
// whose description the user left empty: "Default response" for
// "default", "<code> response" for a range such as "4XX", the standard
// status text for a known code and "Status <n>" otherwise.
func defaultResponseDescription(code string) string {
	if code == "default" {
		return "Default response"
	}
	if isResponseCodeRange(code) {
		return code + " response"
	}
	if ci, err := strconv.Atoi(code); err == nil {
		if text := http.StatusText(ci); text != "" {
			return text
		}
	}
	return "Status " + code
}

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
// return the type t (where, if not empty, names the operation in errors
// that are not wrapped in an *OperationError) with the given media type and status code.
func (g *Generator) setOperationResponse(op *Operation, where string, t reflect.Type, code, mt, desc string, headers []*ResponseHeader, example any, examples map[string]any) error {
	var prefix string
	if where != "" {
		prefix = where + ": "
	}
	if _, ok := op.Responses[code]; ok {
		// A response already exists for this code.
		return fmt.Errorf("response with code %s already exists", code)
	}
	if example != nil && len(examples) > 0 {
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
		}
	}
	if desc == "" {
		desc = defaultResponseDescription(code)
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
			g.error(fmt.Errorf("%sresponse %s: example dropped, it is not JSON-encodable: %w", prefix, code, err))
			example = nil
		}
	}
	var castedExamples map[string]*ExampleOrRef
	if len(examples) > 0 {
		castedExamples = make(map[string]*ExampleOrRef)
		dropped := false
		for name, val := range examples {
			if err := jsonEncodable(val); err != nil {
				g.error(fmt.Errorf("%sresponse %s: example %q dropped, it is not JSON-encodable: %w", prefix, code, name, err))
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
			// Header examples that do not encode as JSON are reported
			// and dropped, like those of the body.
			hExample := h.Example
			if hExample != nil {
				if err := jsonEncodable(hExample); err != nil {
					g.error(fmt.Errorf("%sresponse %s: header %q: example dropped, it is not JSON-encodable: %w", prefix, code, h.Name, err))
					hExample = nil
				}
			}
			hExamples := h.Examples
			if len(hExamples) > 0 {
				hExamples = make(map[string]*ExampleOrRef, len(h.Examples))
				for name, e := range h.Examples {
					if err := jsonEncodable(e); err != nil {
						g.error(fmt.Errorf("%sresponse %s: header %q: example %q dropped, it is not JSON-encodable: %w", prefix, code, h.Name, name, err))
						continue
					}
					hExamples[name] = e
				}
				if len(hExamples) == 0 {
					hExamples = nil
				}
			}
			r.Headers[h.Name] = &HeaderOrRef{Value: &Header{
				Description: h.Description,
				Required:    h.Required,
				Deprecated:  h.Deprecated,
				Schema:      sor,
				Example:     hExample,
				Examples:    hExamples,
			}}
		}
	}
	op.Responses[code] = &ResponseOrRef{Value: r}

	return nil
}
