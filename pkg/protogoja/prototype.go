package protogoja

import (
	"fmt"

	"github.com/dop251/goja"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const hiddenMessagePrototypeKey = "__go_go_goja_proto_message_prototype"

// MessagePrototypeRef is the Go-owned schema/prototype token attached to
// generated message namespace objects. It lets consuming modules discover the
// protobuf type represented by a namespace without requiring a built message.
type MessagePrototypeRef struct {
	msg      proto.Message
	typeName protoreflect.FullName
}

// NewMessagePrototypeRef creates a prototype token for msg's protobuf message
// type. The message is cloned so callers cannot mutate the stored prototype.
func NewMessagePrototypeRef(msg proto.Message) (*MessagePrototypeRef, error) {
	if msg == nil {
		return nil, fmt.Errorf("protogoja: nil proto message")
	}
	cloned := proto.Clone(msg)
	return &MessagePrototypeRef{
		msg:      cloned,
		typeName: cloned.ProtoReflect().Descriptor().FullName(),
	}, nil
}

// TypeName returns the full protobuf message name represented by this token.
func (r *MessagePrototypeRef) TypeName() protoreflect.FullName {
	if r == nil {
		return ""
	}
	return r.typeName
}

// NewMessage returns a cloned zero/prototype message for this token.
func (r *MessagePrototypeRef) NewMessage() proto.Message {
	if r == nil || r.msg == nil {
		return nil
	}
	return proto.Clone(r.msg)
}

// AttachMessagePrototype attaches a hidden, non-enumerable schema/prototype
// token to a generated message namespace object.
func AttachMessagePrototype(vm *goja.Runtime, obj *goja.Object, msg proto.Message) error {
	if vm == nil {
		return fmt.Errorf("protogoja: nil runtime")
	}
	if obj == nil {
		return fmt.Errorf("protogoja: nil object")
	}
	ref, err := NewMessagePrototypeRef(msg)
	if err != nil {
		return err
	}
	value := vm.ToValue(ref)
	if err := obj.Set(hiddenMessagePrototypeKey, value); err != nil {
		return fmt.Errorf("protogoja: attach hidden message prototype: %w", err)
	}
	return obj.DefineDataProperty(
		hiddenMessagePrototypeKey,
		value,
		goja.FLAG_FALSE, // writable
		goja.FLAG_FALSE, // enumerable
		goja.FLAG_FALSE, // configurable
	)
}

// MessagePrototypeFromValue extracts a generated message namespace prototype
// token. The returned reference must be treated as immutable by callers.
func MessagePrototypeFromValue(value goja.Value) (*MessagePrototypeRef, bool) {
	if value == nil || goja.IsUndefined(value) || goja.IsNull(value) {
		return nil, false
	}
	obj, ok := value.(*goja.Object)
	if !ok || obj == nil {
		return nil, false
	}
	raw := obj.Get(hiddenMessagePrototypeKey)
	if raw == nil || goja.IsUndefined(raw) || goja.IsNull(raw) {
		return nil, false
	}
	ref, ok := raw.Export().(*MessagePrototypeRef)
	return ref, ok && ref != nil && ref.msg != nil
}
