package sdk

import "google.golang.org/protobuf/types/known/structpb"

type Call struct {
	ExportName string
	MethodName string
	Args       []any
	RawArgs    []*structpb.Value
}

func (c *Call) Len() int {
	if c == nil {
		return 0
	}
	return len(c.Args)
}

func (c *Call) Value(index int) (any, error) {
	if c == nil {
		return nil, argIndexError(index, 0)
	}
	if index < 0 || index >= len(c.Args) {
		return nil, argIndexError(index, len(c.Args))
	}
	return c.Args[index], nil
}

func (c *Call) String(index int) (string, error) {
	value, err := c.Value(index)
	if err != nil {
		return "", err
	}
	s, ok := value.(string)
	if !ok {
		return "", argTypeError(index, "string", value)
	}
	return s, nil
}

func (c *Call) StringDefault(index int, fallback string) string {
	value, err := c.String(index)
	if err != nil {
		return fallback
	}
	if value == "" {
		return fallback
	}
	return value
}

func (c *Call) Float64(index int) (float64, error) {
	value, err := c.Value(index)
	if err != nil {
		return 0, err
	}
	number, ok := value.(float64)
	if !ok {
		return 0, argTypeError(index, "float64", value)
	}
	return number, nil
}

func (c *Call) Bool(index int) (bool, error) {
	value, err := c.Value(index)
	if err != nil {
		return false, err
	}
	b, ok := value.(bool)
	if !ok {
		return false, argTypeError(index, "bool", value)
	}
	return b, nil
}

func (c *Call) Map(index int) (map[string]any, error) {
	value, err := c.Value(index)
	if err != nil {
		return nil, err
	}
	m, ok := value.(map[string]any)
	if !ok {
		return nil, argTypeError(index, "map[string]any", value)
	}
	return m, nil
}

func (c *Call) Slice(index int) ([]any, error) {
	value, err := c.Value(index)
	if err != nil {
		return nil, err
	}
	v, ok := value.([]any)
	if !ok {
		return nil, argTypeError(index, "[]any", value)
	}
	return v, nil
}
