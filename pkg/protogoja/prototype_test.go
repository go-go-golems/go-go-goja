package protogoja

import (
	"testing"

	"github.com/dop251/goja"
	contract "github.com/go-go-golems/go-go-goja/pkg/hashiplugin/contract"
	"github.com/stretchr/testify/require"
)

func TestMessagePrototypeRefAttachmentAndExtraction(t *testing.T) {
	vm := goja.New()
	obj := vm.NewObject()
	require.NoError(t, AttachMessagePrototype(vm, obj, &contract.ModuleManifest{}))
	require.Empty(t, obj.Keys())

	ref, ok := MessagePrototypeFromValue(obj)
	require.True(t, ok)
	require.Equal(t, "hashiplugin.contract.v1.ModuleManifest", string(ref.TypeName()))

	msg := ref.NewMessage()
	manifest, ok := msg.(*contract.ModuleManifest)
	require.True(t, ok)
	manifest.ModuleName = "mutated"

	second := ref.NewMessage().(*contract.ModuleManifest)
	require.Empty(t, second.GetModuleName())
}

func TestMessagePrototypeRejectsInvalidValues(t *testing.T) {
	vm := goja.New()
	_, ok := MessagePrototypeFromValue(goja.Undefined())
	require.False(t, ok)
	_, ok = MessagePrototypeFromValue(vm.ToValue("not-a-namespace"))
	require.False(t, ok)

	err := AttachMessagePrototype(nil, vm.NewObject(), &contract.ModuleManifest{})
	require.ErrorContains(t, err, "nil runtime")
	err = AttachMessagePrototype(vm, nil, &contract.ModuleManifest{})
	require.ErrorContains(t, err, "nil object")
	err = AttachMessagePrototype(vm, vm.NewObject(), nil)
	require.ErrorContains(t, err, "nil proto message")
}
