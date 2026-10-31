package openapi

import (
	"fmt"
	"reflect"
	"strconv"
)

// parseExampleValue is used to transform the string representation of the example value to the correct type.
func parseExampleValue(t reflect.Type, value string) (any, error) {
	// If the type implements Exampler use the ParseExample method to create the example
	i, ok := reflect.New(t).Interface().(Exampler)
	if ok {
		return i.ParseExample(value)
	}

	switch t.Kind() {
	case reflect.Bool:
		return strconv.ParseBool(value)
	case reflect.String:
		return value, nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		i, err := strconv.ParseInt(value, 10, t.Bits())
		if err != nil {
			return nil, err
		}
		switch t.Bits() {
		case 8:
			return int8(i), nil
		case 16:
			return int16(i), nil
		case 32:
			return int32(i), nil
		default:
		}
		return i, nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		u, err := strconv.ParseUint(value, 10, t.Bits())
		if err != nil {
			return nil, err
		}
		switch t.Bits() {
		case 8:
			return uint8(u), nil
		case 16:
			return uint16(u), nil
		case 32:
			return uint32(u), nil
		default:
		}
		return u, nil
	case reflect.Float32, reflect.Float64:
		return strconv.ParseFloat(value, t.Bits())
	case reflect.Pointer:
		return parseExampleValue(t.Elem(), value)
	case reflect.Struct:
		return nil, fmt.Errorf("type %s does not implement Exampler", t.String())
	default:
		return nil, fmt.Errorf("unsuported type: %s", t.String())
	}
}
