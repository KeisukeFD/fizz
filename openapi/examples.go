package openapi

import (
	"fmt"
	"reflect"
	"strconv"
)

// parseExampleValue is used to transform the string representation of the example value to the correct type.
func parseExampleValue(t reflect.Type, value string) (any, error) {
	return parseExample(t, value, false)
}

// parseExample is parseExampleValue for a body field (param false) or a
// parameter (param true), which differ for time.Duration (D16).
func parseExample(t reflect.Type, value string, param bool) (any, error) {
	// If the type implements Exampler use the ParseExample method to create the example
	i, ok := reflect.New(t).Interface().(Exampler)
	if ok {
		return i.ParseExample(value)
	}

	if t.AssignableTo(tofDuration) {
		return durationNanoseconds(value, param)
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
		return parseFloat(value, t.Bits())
	case reflect.Pointer:
		return parseExample(t.Elem(), value, param)
	case reflect.Struct:
		return nil, fmt.Errorf("type %s does not implement Exampler", t.String())
	default:
		return nil, fmt.Errorf("unsuported type: %s", t.String())
	}
}
