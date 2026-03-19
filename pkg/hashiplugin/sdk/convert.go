package sdk

import (
	"fmt"
	"reflect"

	"google.golang.org/protobuf/types/known/structpb"
)

func decodeArgs(values []*structpb.Value) []any {
	if len(values) == 0 {
		return nil
	}
	out := make([]any, 0, len(values))
	for _, value := range values {
		if value == nil {
			out = append(out, nil)
			continue
		}
		out = append(out, value.AsInterface())
	}
	return out
}

func encodeResult(value any) (*structpb.Value, error) {
	if existing, ok := value.(*structpb.Value); ok {
		return existing, nil
	}
	normalized, err := normalizeResultValue(value)
	if err != nil {
		return nil, fmt.Errorf("sdk encode result: %w", err)
	}
	if normalized == nil {
		return structpb.NewNullValue(), nil
	}
	result, err := structpb.NewValue(normalized)
	if err != nil {
		return nil, fmt.Errorf("sdk encode result: %w", err)
	}
	return result, nil
}

func normalizeResultValue(value any) (any, error) {
	if value == nil {
		return nil, nil
	}

	switch v := value.(type) {
	case string, bool, float64:
		return v, nil
	case float32:
		return float64(v), nil
	case int:
		return float64(v), nil
	case int8:
		return float64(v), nil
	case int16:
		return float64(v), nil
	case int32:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case uint:
		return float64(v), nil
	case uint8:
		return float64(v), nil
	case uint16:
		return float64(v), nil
	case uint32:
		return float64(v), nil
	case uint64:
		return float64(v), nil
	case []any:
		out := make([]any, 0, len(v))
		for _, item := range v {
			normalized, err := normalizeResultValue(item)
			if err != nil {
				return nil, err
			}
			out = append(out, normalized)
		}
		return out, nil
	case map[string]any:
		out := make(map[string]any, len(v))
		for key, item := range v {
			normalized, err := normalizeResultValue(item)
			if err != nil {
				return nil, err
			}
			out[key] = normalized
		}
		return out, nil
	}

	rv := reflect.ValueOf(value)
	switch rv.Kind() {
	case reflect.Invalid:
		return nil, nil
	case reflect.Pointer:
		if rv.IsNil() {
			return nil, nil
		}
		return normalizeResultValue(rv.Elem().Interface())
	case reflect.Interface:
		if rv.IsNil() {
			return nil, nil
		}
		return normalizeResultValue(rv.Elem().Interface())
	case reflect.Slice, reflect.Array:
		out := make([]any, 0, rv.Len())
		for i := range rv.Len() {
			normalized, err := normalizeResultValue(rv.Index(i).Interface())
			if err != nil {
				return nil, err
			}
			out = append(out, normalized)
		}
		return out, nil
	case reflect.Map:
		if rv.Type().Key().Kind() != reflect.String {
			return nil, fmt.Errorf("unsupported map key type %s", rv.Type().Key())
		}
		out := make(map[string]any, rv.Len())
		iter := rv.MapRange()
		for iter.Next() {
			normalized, err := normalizeResultValue(iter.Value().Interface())
			if err != nil {
				return nil, err
			}
			out[iter.Key().String()] = normalized
		}
		return out, nil
	case reflect.Bool,
		reflect.Int,
		reflect.Int8,
		reflect.Int16,
		reflect.Int32,
		reflect.Int64,
		reflect.Uint,
		reflect.Uint8,
		reflect.Uint16,
		reflect.Uint32,
		reflect.Uint64,
		reflect.Uintptr,
		reflect.Float32,
		reflect.Float64,
		reflect.Complex64,
		reflect.Complex128,
		reflect.Chan,
		reflect.Func,
		reflect.String,
		reflect.Struct,
		reflect.UnsafePointer:
		return nil, fmt.Errorf("unsupported result type %T", value)
	}

	return nil, fmt.Errorf("unsupported result type %T", value)
}
