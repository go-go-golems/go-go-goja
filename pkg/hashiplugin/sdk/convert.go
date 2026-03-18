package sdk

import (
	"fmt"

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
	if value == nil {
		return structpb.NewNullValue(), nil
	}
	result, err := structpb.NewValue(value)
	if err != nil {
		return nil, fmt.Errorf("sdk encode result: %w", err)
	}
	return result, nil
}
