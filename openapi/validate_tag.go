package openapi

import (
	"reflect"
	"strconv"
	"strings"
)

// updateSchemaValidation fills the fields of the schema
// related to the JSON Schema Validation RFC based on the
// content of the validator tag.
// see https://godoc.org/gopkg.in/go-playground/validator.v8
func (g *Generator) updateSchemaValidation(schema *Schema, sf reflect.StructField) *Schema {
	ts := sf.Tag.Get(g.config.ValidatorTag)
	if ts == "" {
		return schema
	}
	ft := sf.Type
	if sf.Type.Kind() == reflect.Pointer {
		ft = sf.Type.Elem()
	}
	tags := strings.SplitSeq(ts, ",")

	for t := range tags {
		if t == "dive" || t == "keys" {
			break
		}
		if t == "email" {
			schema.Format = "email"
			break
		}
		// Tags can be joined together with an OR operator.
		parts := strings.SplitSeq(t, "|")

		for p := range parts {
			var k, v string
			// Split k/v pair using separator.
			before, after, ok := strings.Cut(p, "=")
			if !ok {
				k = p
			} else {
				k = before
				v = after
			}
			// Handle validators with value.
			switch k {
			case "len", "max", "min", "eq", "gt", "gte", "lt", "lte":
				n, err := strconv.Atoi(v)
				if err != nil {
					continue
				}
				switch k {
				case "len":
					setSchemaLen(schema, n, ft)
				case "max", "lte":
					setSchemaMax(schema, n, ft)
				case "min", "gte":
					setSchemaMin(schema, n, ft)
				case "lt":
					setSchemaMax(schema, n-1, ft)
				case "gt":
					setSchemaMin(schema, n+1, ft)
				case "eq":
					setSchemaEq(schema, n, ft)
				}
			}
		}
	}
	return schema
}
