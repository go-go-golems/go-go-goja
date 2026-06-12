package protogoja

import (
	"encoding/json"
	"fmt"

	"github.com/dop251/goja"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const hiddenMessageRefKey = "__go_go_goja_proto_message_ref"

var marshalJSON = protojson.MarshalOptions{
	EmitUnpopulated: false,
	UseProtoNames:   false,
}

// MessageRef is the Go-owned reference attached to JavaScript ProtoMessage
// wrapper objects. It stores a cloned protobuf message so JavaScript code sees
// stable built-message values rather than mutable builder state.
type MessageRef struct {
	msg      proto.Message
	typeName protoreflect.FullName
}

// NewMessageRef clones msg into a stable reference suitable for attaching to a
// Goja object.
func NewMessageRef(msg proto.Message) (*MessageRef, error) {
	if msg == nil {
		return nil, fmt.Errorf("protogoja: nil proto message")
	}
	cloned := proto.Clone(msg)
	return &MessageRef{
		msg:      cloned,
		typeName: cloned.ProtoReflect().Descriptor().FullName(),
	}, nil
}

// TypeName returns the full protobuf message name carried by this reference.
func (r *MessageRef) TypeName() protoreflect.FullName {
	if r == nil {
		return ""
	}
	return r.typeName
}

// Message returns a clone of the referenced protobuf message.
func (r *MessageRef) Message() proto.Message {
	if r == nil || r.msg == nil {
		return nil
	}
	return proto.Clone(r.msg)
}

// ToValue wraps msg in a JavaScript object carrying a hidden MessageRef. The
// returned object exposes a small stable ProtoMessage API: typeName, toJSON(),
// clone(), and equals(other).
func ToValue(vm *goja.Runtime, msg proto.Message) (*goja.Object, error) {
	if vm == nil {
		return nil, fmt.Errorf("protogoja: nil runtime")
	}
	ref, err := NewMessageRef(msg)
	if err != nil {
		return nil, err
	}
	obj := vm.NewObject()
	if err := attachMessageRef(vm, obj, ref); err != nil {
		return nil, err
	}
	if err := defineReadOnly(obj, "typeName", vm.ToValue(string(ref.TypeName()))); err != nil {
		return nil, err
	}
	if err := obj.Set("toJSON", func() goja.Value {
		value, err := ref.toJSONValue(vm)
		if err != nil {
			panic(vm.NewGoError(err))
		}
		return value
	}); err != nil {
		return nil, fmt.Errorf("protogoja: define toJSON: %w", err)
	}
	if err := obj.Set("clone", func() goja.Value {
		cloned, err := ToValue(vm, ref.Message())
		if err != nil {
			panic(vm.NewGoError(err))
		}
		return cloned
	}); err != nil {
		return nil, fmt.Errorf("protogoja: define clone: %w", err)
	}
	if err := obj.Set("equals", func(other goja.Value) bool {
		otherMsg, ok := MessageFromValue(other)
		return ok && proto.Equal(ref.Message(), otherMsg)
	}); err != nil {
		return nil, fmt.Errorf("protogoja: define equals: %w", err)
	}
	return obj, nil
}

// MessageFromValue extracts a cloned protobuf message from a JavaScript value
// created by ToValue or generated protobuf builder modules.
func MessageFromValue(value goja.Value) (proto.Message, bool) {
	ref, ok := MessageRefFromValue(value)
	if !ok {
		return nil, false
	}
	msg := ref.Message()
	return msg, msg != nil
}

// MessageRefFromValue extracts the hidden message reference from a JavaScript
// object. The returned reference must be treated as immutable by callers.
func MessageRefFromValue(value goja.Value) (*MessageRef, bool) {
	if value == nil || goja.IsUndefined(value) || goja.IsNull(value) {
		return nil, false
	}
	obj, ok := value.(*goja.Object)
	if !ok || obj == nil {
		return nil, false
	}
	raw := obj.Get(hiddenMessageRefKey)
	if raw == nil || goja.IsUndefined(raw) || goja.IsNull(raw) {
		return nil, false
	}
	ref, ok := raw.Export().(*MessageRef)
	return ref, ok && ref != nil && ref.msg != nil
}

// MustMessageFromValue extracts a protobuf message or panics with a JavaScript
// TypeError when value is not a generated ProtoMessage object.
func MustMessageFromValue(vm *goja.Runtime, value goja.Value) proto.Message {
	msg, ok := MessageFromValue(value)
	if ok {
		return msg
	}
	if vm == nil {
		panic("protogoja: value is not a ProtoMessage")
	}
	panic(vm.NewTypeError("value is not a ProtoMessage"))
}

// TypeNameFromValue returns the full protobuf message name for value.
func TypeNameFromValue(value goja.Value) (protoreflect.FullName, bool) {
	ref, ok := MessageRefFromValue(value)
	if !ok {
		return "", false
	}
	return ref.TypeName(), true
}

// IsMessageValueOf reports whether value is a ProtoMessage object with the
// supplied full protobuf message name.
func IsMessageValueOf(value goja.Value, typeName protoreflect.FullName) bool {
	got, ok := TypeNameFromValue(value)
	return ok && got == typeName
}

func attachMessageRef(vm *goja.Runtime, obj *goja.Object, ref *MessageRef) error {
	if vm == nil {
		return fmt.Errorf("protogoja: nil runtime")
	}
	if obj == nil {
		return fmt.Errorf("protogoja: nil object")
	}
	if ref == nil || ref.msg == nil {
		return fmt.Errorf("protogoja: nil message reference")
	}
	value := vm.ToValue(ref)
	if err := obj.Set(hiddenMessageRefKey, value); err != nil {
		return fmt.Errorf("protogoja: attach hidden message ref: %w", err)
	}
	return obj.DefineDataProperty(
		hiddenMessageRefKey,
		value,
		goja.FLAG_FALSE, // writable
		goja.FLAG_FALSE, // enumerable
		goja.FLAG_FALSE, // configurable
	)
}

func defineReadOnly(obj *goja.Object, name string, value goja.Value) error {
	if obj == nil {
		return fmt.Errorf("protogoja: nil object")
	}
	return obj.DefineDataProperty(
		name,
		value,
		goja.FLAG_FALSE, // writable
		goja.FLAG_TRUE,  // enumerable
		goja.FLAG_FALSE, // configurable
	)
}

func (r *MessageRef) toJSONValue(vm *goja.Runtime) (goja.Value, error) {
	if vm == nil {
		return nil, fmt.Errorf("protogoja: nil runtime")
	}
	if r == nil || r.msg == nil {
		return nil, fmt.Errorf("protogoja: nil message reference")
	}
	data, err := marshalJSON.Marshal(r.msg)
	if err != nil {
		return nil, fmt.Errorf("protogoja: marshal %s to JSON: %w", r.typeName, err)
	}
	var decoded any
	if err := json.Unmarshal(data, &decoded); err != nil {
		return nil, fmt.Errorf("protogoja: decode %s JSON for Goja: %w", r.typeName, err)
	}
	return vm.ToValue(decoded), nil
}
