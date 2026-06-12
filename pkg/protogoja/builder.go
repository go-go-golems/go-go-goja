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

const hiddenBuilderRefKey = "__go_go_goja_proto_builder_ref"

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
		return fmt.Errorf("protogoja: %s[]: %w", field.FullName(), err)
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
		return fmt.Errorf("protogoja: %s[%s]: %w", field.FullName(), mapKeyDisplay(key), err)
	}
	b.msg.ProtoReflect().Mutable(field).Map().Set(mapKey, mapValue)
	return nil
}

// Delete removes one entry from a map field. Deleting a missing key is a no-op,
// matching protobuf map Clear(key) semantics.
func (b *BuilderRef) Delete(vm *goja.Runtime, field protoreflect.FieldDescriptor, key goja.Value) error {
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
	b.msg.ProtoReflect().Mutable(field).Map().Clear(mapKey)
	return nil
}

// Has reports whether field is present on the builder message. For explicit
// presence fields, including proto2 scalars, proto3 optional scalars, messages,
// and oneof alternatives, this reports protobuf presence. For implicit proto3
// scalar fields, protobuf reflection reports presence when the value is non-zero.
func (b *BuilderRef) Has(field protoreflect.FieldDescriptor) (bool, error) {
	if err := b.validateField(field); err != nil {
		return false, err
	}
	return b.msg.ProtoReflect().Has(field), nil
}

// Clear clears field from the builder message.
func (b *BuilderRef) Clear(field protoreflect.FieldDescriptor) error {
	if err := b.validateField(field); err != nil {
		return err
	}
	b.msg.ProtoReflect().Clear(field)
	return nil
}

// ClearOneof clears whichever field is currently selected for oneof. Clearing an
// unset oneof is a no-op.
func (b *BuilderRef) ClearOneof(oneof protoreflect.OneofDescriptor) error {
	if err := b.validateOneof(oneof); err != nil {
		return err
	}
	selected := b.msg.ProtoReflect().WhichOneof(oneof)
	if selected != nil {
		b.msg.ProtoReflect().Clear(selected)
	}
	return nil
}

// WhichOneof returns the currently selected field for oneof, or nil when no
// field in the oneof is set.
func (b *BuilderRef) WhichOneof(oneof protoreflect.OneofDescriptor) (protoreflect.FieldDescriptor, error) {
	if err := b.validateOneof(oneof); err != nil {
		return nil, err
	}
	return b.msg.ProtoReflect().WhichOneof(oneof), nil
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

// AttachBuilderRef attaches a hidden, non-enumerable builder reference to obj.
// Generated fluent-builder modules use this to associate JavaScript builder
// objects with their Go-owned mutable protobuf state.
func AttachBuilderRef(vm *goja.Runtime, obj *goja.Object, ref *BuilderRef) error {
	if vm == nil {
		return fmt.Errorf("protogoja: nil runtime")
	}
	if obj == nil {
		return fmt.Errorf("protogoja: nil object")
	}
	if ref == nil || ref.msg == nil || ref.desc == nil {
		return fmt.Errorf("protogoja: nil builder reference")
	}
	value := vm.ToValue(ref)
	if err := obj.Set(hiddenBuilderRefKey, value); err != nil {
		return fmt.Errorf("protogoja: attach hidden builder ref: %w", err)
	}
	return obj.DefineDataProperty(
		hiddenBuilderRefKey,
		value,
		goja.FLAG_FALSE, // writable
		goja.FLAG_FALSE, // enumerable
		goja.FLAG_FALSE, // configurable
	)
}

// BuilderRefFromValue extracts the hidden builder reference from a JavaScript
// builder object created by generated protobuf builder modules. The returned
// reference is mutable and should only be used by trusted generated module code
// or runtime conversion helpers.
func BuilderRefFromValue(value goja.Value) (*BuilderRef, bool) {
	if value == nil || goja.IsUndefined(value) || goja.IsNull(value) {
		return nil, false
	}
	obj, ok := value.(*goja.Object)
	if !ok || obj == nil {
		return nil, false
	}
	raw := obj.Get(hiddenBuilderRefKey)
	if raw == nil || goja.IsUndefined(raw) || goja.IsNull(raw) {
		return nil, false
	}
	ref, ok := raw.Export().(*BuilderRef)
	return ref, ok && ref != nil && ref.msg != nil && ref.desc != nil
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

func (b *BuilderRef) validateOneof(oneof protoreflect.OneofDescriptor) error {
	if b == nil || b.msg == nil || b.desc == nil {
		return fmt.Errorf("protogoja: nil builder")
	}
	if oneof == nil {
		return fmt.Errorf("protogoja: nil oneof descriptor")
	}
	parent, ok := oneof.Parent().(protoreflect.MessageDescriptor)
	if !ok || parent.FullName() != b.desc.FullName() {
		return fmt.Errorf("protogoja: oneof %s does not belong to %s", oneof.FullName(), b.desc.FullName())
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
	for i, item := range items {
		converted, err := valueForField(vm, field, item)
		if err != nil {
			return fmt.Errorf("protogoja: %s[%d]: %w", field.FullName(), i, err)
		}
		list.Append(converted)
	}
	return nil
}

func (b *BuilderRef) setMap(vm *goja.Runtime, field protoreflect.FieldDescriptor, value goja.Value) error {
	entries, err := mapEntries(vm, field, value)
	if err != nil {
		return err
	}
	pbMap := b.msg.ProtoReflect().Mutable(field).Map()
	pbMap.Range(func(key protoreflect.MapKey, _ protoreflect.Value) bool {
		pbMap.Clear(key)
		return true
	})
	for _, entry := range entries {
		pbMap.Set(entry.key, entry.value)
	}
	return nil
}

type mapEntry struct {
	key   protoreflect.MapKey
	value protoreflect.Value
}

func mapEntries(vm *goja.Runtime, field protoreflect.FieldDescriptor, value goja.Value) ([]mapEntry, error) {
	if vm == nil {
		return nil, fmt.Errorf("protogoja: nil runtime")
	}
	if value == nil || goja.IsUndefined(value) || goja.IsNull(value) {
		return nil, fmt.Errorf("protogoja: %s expects an object or Map", field.FullName())
	}
	if entries, ok, err := jsMapEntries(vm, field, value); ok || err != nil {
		return entries, err
	}
	return objectMapEntries(vm, field, value)
}

func objectMapEntries(vm *goja.Runtime, field protoreflect.FieldDescriptor, value goja.Value) ([]mapEntry, error) {
	obj := value.ToObject(vm)
	out := make([]mapEntry, 0, len(obj.Keys()))
	for _, rawKey := range obj.Keys() {
		key, err := mapKeyForField(vm, field.MapKey(), vm.ToValue(rawKey))
		if err != nil {
			return nil, fmt.Errorf("protogoja: %s key %q: %w", field.FullName(), rawKey, err)
		}
		converted, err := valueForField(vm, field.MapValue(), obj.Get(rawKey))
		if err != nil {
			return nil, fmt.Errorf("protogoja: %s[%q]: %w", field.FullName(), rawKey, err)
		}
		out = append(out, mapEntry{key: key, value: converted})
	}
	return out, nil
}

func jsMapEntries(vm *goja.Runtime, field protoreflect.FieldDescriptor, value goja.Value) ([]mapEntry, bool, error) {
	obj := value.ToObject(vm)
	entriesFn, ok := goja.AssertFunction(obj.Get("entries"))
	if !ok || goja.IsUndefined(obj.Get("size")) {
		return nil, false, nil
	}
	iterator, err := entriesFn(obj)
	if err != nil {
		return nil, true, fmt.Errorf("protogoja: %s read Map entries: %w", field.FullName(), err)
	}
	arrayCtor := vm.Get("Array").ToObject(vm)
	fromFn, ok := goja.AssertFunction(arrayCtor.Get("from"))
	if !ok {
		return nil, true, fmt.Errorf("protogoja: Array.from is not available")
	}
	pairsValue, err := fromFn(arrayCtor, iterator)
	if err != nil {
		return nil, true, fmt.Errorf("protogoja: %s materialize Map entries: %w", field.FullName(), err)
	}
	pairs, err := arrayElements(vm, field.FullName(), pairsValue)
	if err != nil {
		return nil, true, err
	}
	out := make([]mapEntry, 0, len(pairs))
	for i, pairValue := range pairs {
		pair, err := arrayElements(vm, field.FullName(), pairValue)
		if err != nil {
			return nil, true, fmt.Errorf("protogoja: %s Map entry %d: %w", field.FullName(), i, err)
		}
		if len(pair) != 2 {
			return nil, true, fmt.Errorf("protogoja: %s Map entry %d expected [key, value], got length %d", field.FullName(), i, len(pair))
		}
		key, err := mapKeyForField(vm, field.MapKey(), pair[0])
		if err != nil {
			return nil, true, fmt.Errorf("protogoja: %s Map entry %d key: %w", field.FullName(), i, err)
		}
		converted, err := valueForField(vm, field.MapValue(), pair[1])
		if err != nil {
			return nil, true, fmt.Errorf("protogoja: %s[%s]: %w", field.FullName(), mapKeyDisplay(pair[0]), err)
		}
		out = append(out, mapEntry{key: key, value: converted})
	}
	return out, true, nil
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
		msg, ok := messageForMessageField(value)
		if !ok {
			return protoreflect.Value{}, expectedFieldError(field, string(field.Message().FullName())+" ProtoMessage or builder", value)
		}
		if msg.ProtoReflect().Descriptor().FullName() != field.Message().FullName() {
			return protoreflect.Value{}, fmt.Errorf("protogoja: %s expected %s ProtoMessage or builder, got %s", field.FullName(), field.Message().FullName(), msg.ProtoReflect().Descriptor().FullName())
		}
		return protoreflect.ValueOfMessage(msg.ProtoReflect()), nil
	default:
		return protoreflect.Value{}, fmt.Errorf("protogoja: %s unsupported field kind %s", field.FullName(), field.Kind())
	}
}

func messageForMessageField(value goja.Value) (proto.Message, bool) {
	if msg, ok := MessageFromValue(value); ok {
		return msg, true
	}
	builder, ok := BuilderRefFromValue(value)
	if !ok {
		return nil, false
	}
	msg := builder.Build()
	return msg, msg != nil
}

func mapKeyForField(vm *goja.Runtime, field protoreflect.FieldDescriptor, value goja.Value) (protoreflect.MapKey, error) {
	converted, err := valueForField(vm, field, value)
	if err != nil {
		return protoreflect.MapKey{}, err
	}
	return converted.MapKey(), nil
}

func mapKeyDisplay(value goja.Value) string {
	if value == nil || goja.IsUndefined(value) || goja.IsNull(value) {
		return "<nil>"
	}
	if s, ok := value.Export().(string); ok {
		return strconv.Quote(s)
	}
	return fmt.Sprint(value.Export())
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
