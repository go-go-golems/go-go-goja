package protogoja

import (
	"encoding/base64"
	"fmt"
	"math"
	"strconv"

	"github.com/dop251/goja"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// BuilderRef owns mutable protobuf message state while generated fluent builder
// methods set fields. Build returns a clone so callers receive stable
// ProtoMessage values rather than mutable builder internals.
type BuilderRef struct {
	msg  proto.Message
	desc protoreflect.MessageDescriptor
}

// NewBuilder creates a mutable builder around a clone of msg.
func NewBuilder(msg proto.Message) (*BuilderRef, error) {
	if msg == nil {
		return nil, fmt.Errorf("protogoja: nil proto message")
	}
	cloned := proto.Clone(msg)
	return &BuilderRef{msg: cloned, desc: cloned.ProtoReflect().Descriptor()}, nil
}

// Descriptor returns the protobuf descriptor for the builder message.
func (b *BuilderRef) Descriptor() protoreflect.MessageDescriptor {
	if b == nil {
		return nil
	}
	return b.desc
}

// Set replaces a singular, repeated, or map field value.
func (b *BuilderRef) Set(vm *goja.Runtime, field protoreflect.FieldDescriptor, value goja.Value) error {
	if err := b.validateField(field); err != nil {
		return err
	}
	msg := b.msg.ProtoReflect()
	if field.IsMap() {
		return b.setMap(vm, field, value)
	}
	if field.IsList() {
		return b.setList(vm, field, value)
	}
	converted, err := valueForField(vm, field, value)
	if err != nil {
		return err
	}
	msg.Set(field, converted)
	return nil
}

// Add appends one value to a repeated field.
func (b *BuilderRef) Add(vm *goja.Runtime, field protoreflect.FieldDescriptor, value goja.Value) error {
	if err := b.validateField(field); err != nil {
		return err
	}
	if !field.IsList() || field.IsMap() {
		return fmt.Errorf("protogoja: %s is not a repeated field", field.FullName())
	}
	converted, err := valueForField(vm, field, value)
	if err != nil {
		return err
	}
	b.msg.ProtoReflect().Mutable(field).List().Append(converted)
	return nil
}

// Put inserts or replaces one map entry.
func (b *BuilderRef) Put(vm *goja.Runtime, field protoreflect.FieldDescriptor, key, value goja.Value) error {
	if err := b.validateField(field); err != nil {
		return err
	}
	if !field.IsMap() {
		return fmt.Errorf("protogoja: %s is not a map field", field.FullName())
	}
	mapKey, err := mapKeyForField(vm, field.MapKey(), key)
	if err != nil {
		return err
	}
	mapValue, err := valueForField(vm, field.MapValue(), value)
	if err != nil {
		return err
	}
	b.msg.ProtoReflect().Mutable(field).Map().Set(mapKey, mapValue)
	return nil
}

// Clear clears field from the builder message.
func (b *BuilderRef) Clear(field protoreflect.FieldDescriptor) error {
	if err := b.validateField(field); err != nil {
		return err
	}
	b.msg.ProtoReflect().Clear(field)
	return nil
}

// Build returns a clone of the current builder state.
func (b *BuilderRef) Build() proto.Message {
	if b == nil || b.msg == nil {
		return nil
	}
	return proto.Clone(b.msg)
}

// Clone returns an independent builder with a clone of the current state.
func (b *BuilderRef) Clone() (*BuilderRef, error) {
	if b == nil || b.msg == nil {
		return nil, fmt.Errorf("protogoja: nil builder")
	}
	return NewBuilder(b.msg)
}

func (b *BuilderRef) validateField(field protoreflect.FieldDescriptor) error {
	if b == nil || b.msg == nil || b.desc == nil {
		return fmt.Errorf("protogoja: nil builder")
	}
	if field == nil {
		return fmt.Errorf("protogoja: nil field descriptor")
	}
	if field.ContainingMessage().FullName() != b.desc.FullName() {
		return fmt.Errorf("protogoja: field %s does not belong to %s", field.FullName(), b.desc.FullName())
	}
	return nil
}

func (b *BuilderRef) setList(vm *goja.Runtime, field protoreflect.FieldDescriptor, value goja.Value) error {
	items, err := arrayElements(vm, field.FullName(), value)
	if err != nil {
		return err
	}
	list := b.msg.ProtoReflect().Mutable(field).List()
	list.Truncate(0)
	for _, item := range items {
		converted, err := valueForField(vm, field, item)
		if err != nil {
			return err
		}
		list.Append(converted)
	}
	return nil
}

func (b *BuilderRef) setMap(vm *goja.Runtime, field protoreflect.FieldDescriptor, value goja.Value) error {
	if vm == nil {
		return fmt.Errorf("protogoja: nil runtime")
	}
	if value == nil || goja.IsUndefined(value) || goja.IsNull(value) {
		return fmt.Errorf("protogoja: %s expects an object", field.FullName())
	}
	obj := value.ToObject(vm)
	pbMap := b.msg.ProtoReflect().Mutable(field).Map()
	pbMap.Range(func(key protoreflect.MapKey, _ protoreflect.Value) bool {
		pbMap.Clear(key)
		return true
	})
	for _, rawKey := range obj.Keys() {
		key, err := mapKeyForField(vm, field.MapKey(), vm.ToValue(rawKey))
		if err != nil {
			return err
		}
		converted, err := valueForField(vm, field.MapValue(), obj.Get(rawKey))
		if err != nil {
			return err
		}
		pbMap.Set(key, converted)
	}
	return nil
}

func arrayElements(vm *goja.Runtime, name protoreflect.FullName, value goja.Value) ([]goja.Value, error) {
	if vm == nil {
		return nil, fmt.Errorf("protogoja: nil runtime")
	}
	if value == nil || goja.IsUndefined(value) || goja.IsNull(value) {
		return nil, fmt.Errorf("protogoja: %s expects an array", name)
	}
	obj := value.ToObject(vm)
	lengthValue := obj.Get("length")
	if lengthValue == nil || goja.IsUndefined(lengthValue) || goja.IsNull(lengthValue) {
		return nil, fmt.Errorf("protogoja: %s expects an array-like value", name)
	}
	length := lengthValue.ToInteger()
	if length < 0 || length > math.MaxInt32 {
		return nil, fmt.Errorf("protogoja: %s invalid array length %d", name, length)
	}
	out := make([]goja.Value, 0, int(length))
	for i := int64(0); i < length; i++ {
		out = append(out, obj.Get(strconv.FormatInt(i, 10)))
	}
	return out, nil
}

func valueForField(vm *goja.Runtime, field protoreflect.FieldDescriptor, value goja.Value) (protoreflect.Value, error) {
	if field == nil {
		return protoreflect.Value{}, fmt.Errorf("protogoja: nil field descriptor")
	}
	if value == nil || goja.IsUndefined(value) || goja.IsNull(value) {
		return protoreflect.Value{}, fmt.Errorf("protogoja: %s cannot be null or undefined", field.FullName())
	}
	switch field.Kind() {
	case protoreflect.BoolKind:
		v, ok := value.Export().(bool)
		if !ok {
			return protoreflect.Value{}, expectedFieldError(field, "boolean", value)
		}
		return protoreflect.ValueOfBool(v), nil
	case protoreflect.EnumKind:
		number, err := enumNumberForField(field, value)
		if err != nil {
			return protoreflect.Value{}, err
		}
		return protoreflect.ValueOfEnum(number), nil
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		v, err := int64ForField(field, value)
		if err != nil {
			return protoreflect.Value{}, err
		}
		if v < math.MinInt32 || v > math.MaxInt32 {
			return protoreflect.Value{}, fmt.Errorf("protogoja: %s value %d outside int32 range", field.FullName(), v)
		}
		return protoreflect.ValueOfInt32(int32(v)), nil
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		v, err := int64ForField(field, value)
		if err != nil {
			return protoreflect.Value{}, err
		}
		return protoreflect.ValueOfInt64(v), nil
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		v, err := uint64ForField(field, value)
		if err != nil {
			return protoreflect.Value{}, err
		}
		if v > math.MaxUint32 {
			return protoreflect.Value{}, fmt.Errorf("protogoja: %s value %d outside uint32 range", field.FullName(), v)
		}
		return protoreflect.ValueOfUint32(uint32(v)), nil
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		v, err := uint64ForField(field, value)
		if err != nil {
			return protoreflect.Value{}, err
		}
		return protoreflect.ValueOfUint64(v), nil
	case protoreflect.FloatKind:
		v, err := float64ForField(field, value)
		if err != nil {
			return protoreflect.Value{}, err
		}
		return protoreflect.ValueOfFloat32(float32(v)), nil
	case protoreflect.DoubleKind:
		v, err := float64ForField(field, value)
		if err != nil {
			return protoreflect.Value{}, err
		}
		return protoreflect.ValueOfFloat64(v), nil
	case protoreflect.StringKind:
		v, ok := value.Export().(string)
		if !ok {
			return protoreflect.Value{}, expectedFieldError(field, "string", value)
		}
		return protoreflect.ValueOfString(v), nil
	case protoreflect.BytesKind:
		bytes, err := bytesForField(field, value)
		if err != nil {
			return protoreflect.Value{}, err
		}
		return protoreflect.ValueOfBytes(bytes), nil
	case protoreflect.MessageKind, protoreflect.GroupKind:
		msg, ok := MessageFromValue(value)
		if !ok {
			return protoreflect.Value{}, expectedFieldError(field, string(field.Message().FullName())+" ProtoMessage", value)
		}
		if msg.ProtoReflect().Descriptor().FullName() != field.Message().FullName() {
			return protoreflect.Value{}, fmt.Errorf("protogoja: %s expected %s ProtoMessage, got %s", field.FullName(), field.Message().FullName(), msg.ProtoReflect().Descriptor().FullName())
		}
		return protoreflect.ValueOfMessage(msg.ProtoReflect()), nil
	default:
		return protoreflect.Value{}, fmt.Errorf("protogoja: %s unsupported field kind %s", field.FullName(), field.Kind())
	}
}

func mapKeyForField(vm *goja.Runtime, field protoreflect.FieldDescriptor, value goja.Value) (protoreflect.MapKey, error) {
	converted, err := valueForField(vm, field, value)
	if err != nil {
		return protoreflect.MapKey{}, err
	}
	return converted.MapKey(), nil
}

func enumNumberForField(field protoreflect.FieldDescriptor, value goja.Value) (protoreflect.EnumNumber, error) {
	if name, ok := value.Export().(string); ok {
		enumValue := field.Enum().Values().ByName(protoreflect.Name(name))
		if enumValue == nil {
			return 0, fmt.Errorf("protogoja: %s unknown enum name %q for %s", field.FullName(), name, field.Enum().FullName())
		}
		return enumValue.Number(), nil
	}
	number, err := int64ForField(field, value)
	if err != nil {
		return 0, err
	}
	if number < math.MinInt32 || number > math.MaxInt32 {
		return 0, fmt.Errorf("protogoja: %s enum number %d outside int32 range", field.FullName(), number)
	}
	if field.Enum().Values().ByNumber(protoreflect.EnumNumber(number)) == nil {
		return 0, fmt.Errorf("protogoja: %s unknown enum number %d for %s", field.FullName(), number, field.Enum().FullName())
	}
	return protoreflect.EnumNumber(number), nil
}

func int64ForField(field protoreflect.FieldDescriptor, value goja.Value) (int64, error) {
	switch v := value.Export().(type) {
	case int:
		return int64(v), nil
	case int8:
		return int64(v), nil
	case int16:
		return int64(v), nil
	case int32:
		return int64(v), nil
	case int64:
		return v, nil
	case uint:
		if uint64(v) > math.MaxInt64 {
			return 0, fmt.Errorf("protogoja: %s value %d outside int64 range", field.FullName(), v)
		}
		return int64(v), nil
	case uint8:
		return int64(v), nil
	case uint16:
		return int64(v), nil
	case uint32:
		return int64(v), nil
	case uint64:
		if v > math.MaxInt64 {
			return 0, fmt.Errorf("protogoja: %s value %d outside int64 range", field.FullName(), v)
		}
		return int64(v), nil
	case float32:
		return checkedInteger(field, float64(v))
	case float64:
		return checkedInteger(field, v)
	case string:
		parsed, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("protogoja: %s parse int64 %q: %w", field.FullName(), v, err)
		}
		return parsed, nil
	default:
		return 0, expectedFieldError(field, "integer number or base-10 string", value)
	}
}

func uint64ForField(field protoreflect.FieldDescriptor, value goja.Value) (uint64, error) {
	switch v := value.Export().(type) {
	case int:
		if v < 0 {
			return 0, fmt.Errorf("protogoja: %s value %d outside uint64 range", field.FullName(), v)
		}
		return uint64(v), nil
	case int8:
		if v < 0 {
			return 0, fmt.Errorf("protogoja: %s value %d outside uint64 range", field.FullName(), v)
		}
		return uint64(v), nil
	case int16:
		if v < 0 {
			return 0, fmt.Errorf("protogoja: %s value %d outside uint64 range", field.FullName(), v)
		}
		return uint64(v), nil
	case int32:
		if v < 0 {
			return 0, fmt.Errorf("protogoja: %s value %d outside uint64 range", field.FullName(), v)
		}
		return uint64(v), nil
	case int64:
		if v < 0 {
			return 0, fmt.Errorf("protogoja: %s value %d outside uint64 range", field.FullName(), v)
		}
		return uint64(v), nil
	case uint:
		return uint64(v), nil
	case uint8:
		return uint64(v), nil
	case uint16:
		return uint64(v), nil
	case uint32:
		return uint64(v), nil
	case uint64:
		return v, nil
	case float32:
		integer, err := checkedInteger(field, float64(v))
		if err != nil {
			return 0, err
		}
		if integer < 0 {
			return 0, fmt.Errorf("protogoja: %s value %d outside uint64 range", field.FullName(), integer)
		}
		return uint64(integer), nil
	case float64:
		integer, err := checkedInteger(field, v)
		if err != nil {
			return 0, err
		}
		if integer < 0 {
			return 0, fmt.Errorf("protogoja: %s value %d outside uint64 range", field.FullName(), integer)
		}
		return uint64(integer), nil
	case string:
		parsed, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("protogoja: %s parse uint64 %q: %w", field.FullName(), v, err)
		}
		return parsed, nil
	default:
		return 0, expectedFieldError(field, "unsigned integer number or base-10 string", value)
	}
}

func float64ForField(field protoreflect.FieldDescriptor, value goja.Value) (float64, error) {
	switch v := value.Export().(type) {
	case float32:
		return float64(v), nil
	case float64:
		return v, nil
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
	default:
		return 0, expectedFieldError(field, "number", value)
	}
}

func checkedInteger(field protoreflect.FieldDescriptor, value float64) (int64, error) {
	if math.IsNaN(value) || math.IsInf(value, 0) || math.Trunc(value) != value {
		return 0, fmt.Errorf("protogoja: %s expected integer, got %v", field.FullName(), value)
	}
	if value < float64(math.MinInt64) || value > float64(math.MaxInt64) {
		return 0, fmt.Errorf("protogoja: %s value %v outside int64 range", field.FullName(), value)
	}
	if math.Abs(value) > float64(1<<53-1) {
		return 0, fmt.Errorf("protogoja: %s number %.0f outside JavaScript safe integer range; pass a string", field.FullName(), value)
	}
	return int64(value), nil
}

func bytesForField(field protoreflect.FieldDescriptor, value goja.Value) ([]byte, error) {
	switch v := value.Export().(type) {
	case []byte:
		return append([]byte(nil), v...), nil
	case string:
		decoded, err := base64.StdEncoding.DecodeString(v)
		if err != nil {
			return nil, fmt.Errorf("protogoja: %s expected base64 bytes string: %w", field.FullName(), err)
		}
		return decoded, nil
	default:
		return nil, expectedFieldError(field, "Uint8Array, []byte, or base64 string", value)
	}
}

func expectedFieldError(field protoreflect.FieldDescriptor, expected string, value goja.Value) error {
	actual := "<nil>"
	if value != nil {
		actual = fmt.Sprintf("%T", value.Export())
	}
	return fmt.Errorf("protogoja: %s expected %s, got %s", field.FullName(), expected, actual)
}
