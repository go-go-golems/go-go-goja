package protogoja

import (
	"testing"

	"github.com/dop251/goja"
	contract "github.com/go-go-golems/go-go-goja/pkg/hashiplugin/contract"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestToValueRoundTripClonesMessage(t *testing.T) {
	vm := goja.New()
	original := &contract.ModuleManifest{ModuleName: "demo", Version: "v1"}

	value, err := ToValue(vm, original)
	require.NoError(t, err)

	original.ModuleName = "mutated-after-wrap"

	extracted, ok := MessageFromValue(value)
	require.True(t, ok)
	require.IsType(t, &contract.ModuleManifest{}, extracted)
	require.Equal(t, "demo", extracted.(*contract.ModuleManifest).GetModuleName())
	require.Equal(t, "v1", extracted.(*contract.ModuleManifest).GetVersion())

	extracted.(*contract.ModuleManifest).ModuleName = "mutated-after-extract"

	extractedAgain, ok := MessageFromValue(value)
	require.True(t, ok)
	require.Equal(t, "demo", extractedAgain.(*contract.ModuleManifest).GetModuleName())
}

func TestMessageValueMethodsFromJavaScript(t *testing.T) {
	vm := goja.New()
	value, err := ToValue(vm, &contract.ModuleManifest{ModuleName: "demo", Version: "v1"})
	require.NoError(t, err)
	require.NoError(t, vm.Set("msg", value))
	require.NoError(t, vm.Set("hiddenKey", hiddenMessageRefKey))

	_, err = vm.RunString(`
		if (msg.typeName !== "hashiplugin.contract.v1.ModuleManifest") {
			throw new Error("unexpected typeName: " + msg.typeName);
		}
		const json = msg.toJSON();
		if (json.moduleName !== "demo") {
			throw new Error("unexpected moduleName: " + json.moduleName);
		}
		if (json.version !== "v1") {
			throw new Error("unexpected version: " + json.version);
		}
		const cloned = msg.clone();
		if (cloned === msg) {
			throw new Error("clone returned the same object");
		}
		if (!msg.equals(cloned)) {
			throw new Error("message should equal clone");
		}
		if (!cloned.equals(msg)) {
			throw new Error("clone should equal message");
		}
		if (Object.keys(msg).includes(hiddenKey)) {
			throw new Error("hidden message ref must not be enumerable");
		}
		if (Object.prototype.propertyIsEnumerable.call(msg, hiddenKey)) {
			throw new Error("hidden message ref property must not be enumerable");
		}
	`)
	require.NoError(t, err)
}

func TestTypeNameHelpers(t *testing.T) {
	vm := goja.New()
	value, err := ToValue(vm, &contract.ModuleManifest{ModuleName: "demo"})
	require.NoError(t, err)

	typeName, ok := TypeNameFromValue(value)
	require.True(t, ok)
	require.Equal(t, protoreflect.FullName("hashiplugin.contract.v1.ModuleManifest"), typeName)
	require.True(t, IsMessageValueOf(value, "hashiplugin.contract.v1.ModuleManifest"))
	require.False(t, IsMessageValueOf(value, "hashiplugin.contract.v1.InvokeRequest"))

	_, ok = TypeNameFromValue(goja.Undefined())
	require.False(t, ok)
}

func TestEqualsRejectsNonMessagesAndDifferentMessages(t *testing.T) {
	vm := goja.New()
	manifest, err := ToValue(vm, &contract.ModuleManifest{ModuleName: "demo"})
	require.NoError(t, err)
	otherManifest, err := ToValue(vm, &contract.ModuleManifest{ModuleName: "other"})
	require.NoError(t, err)
	request, err := ToValue(vm, &contract.InvokeRequest{ExportName: "demo"})
	require.NoError(t, err)

	require.NoError(t, vm.Set("manifest", manifest))
	require.NoError(t, vm.Set("otherManifest", otherManifest))
	require.NoError(t, vm.Set("request", request))

	ret, err := vm.RunString(`
		manifest.equals({}) === false &&
		manifest.equals(otherManifest) === false &&
		manifest.equals(request) === false
	`)
	require.NoError(t, err)
	require.True(t, ret.ToBoolean())
}

func TestToValueRejectsInvalidInput(t *testing.T) {
	_, err := ToValue(nil, &contract.ModuleManifest{})
	require.ErrorContains(t, err, "nil runtime")

	vm := goja.New()
	_, err = ToValue(vm, nil)
	require.ErrorContains(t, err, "nil proto message")

	_, ok := MessageFromValue(vm.ToValue(map[string]any{"typeName": "fake"}))
	require.False(t, ok)
}

func TestMustMessageFromValuePanicsWithTypeError(t *testing.T) {
	vm := goja.New()
	require.Panics(t, func() {
		_ = MustMessageFromValue(vm, goja.Undefined())
	})
}

func TestMessageRefMessageReturnsClone(t *testing.T) {
	ref, err := NewMessageRef(&contract.ModuleManifest{ModuleName: "demo"})
	require.NoError(t, err)

	first := ref.Message().(*contract.ModuleManifest)
	second := ref.Message().(*contract.ModuleManifest)
	require.True(t, proto.Equal(first, second))
	first.ModuleName = "changed"
	require.Equal(t, "demo", second.GetModuleName())
	require.Equal(t, "demo", ref.Message().(*contract.ModuleManifest).GetModuleName())
}
