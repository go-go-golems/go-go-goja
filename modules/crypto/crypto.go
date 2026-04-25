package cryptomod

import (
	"crypto/md5" //nolint:gosec // Node compatibility for createHash("md5").
	"crypto/rand"
	"crypto/sha1" //nolint:gosec // Node compatibility for createHash("sha1").
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"hash"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/buffer"
	"github.com/go-go-golems/go-go-goja/modules"
	"github.com/go-go-golems/go-go-goja/pkg/tsgen/spec"
	"github.com/google/uuid"
)

type m struct{}

var _ modules.NativeModule = (*m)(nil)
var _ modules.TypeScriptDeclarer = (*m)(nil)

func (m) Name() string { return "crypto" }
func (m) Doc() string {
	return `The crypto module provides randomUUID, randomBytes, and basic createHash support.`
}
func (m) TypeScriptModule() *spec.Module {
	return &spec.Module{Name: "crypto", Functions: []spec.Function{
		{Name: "randomUUID", Returns: spec.String()},
		{Name: "randomBytes", Params: []spec.Param{{Name: "size", Type: spec.Number()}}, Returns: spec.Named("Buffer")},
		{Name: "createHash", Params: []spec.Param{{Name: "algorithm", Type: spec.String()}}, Returns: spec.Unknown()},
	}}
}

func (mod m) Loader(vm *goja.Runtime, moduleObj *goja.Object) {
	exports := moduleObj.Get("exports").(*goja.Object)
	modules.SetExport(exports, mod.Name(), "randomUUID", func() string { return uuid.NewString() })
	modules.SetExport(exports, mod.Name(), "randomBytes", func(size int) goja.Value {
		if size < 0 {
			panic(vm.NewTypeError("randomBytes size must be >= 0"))
		}
		b := make([]byte, size)
		if _, err := rand.Read(b); err != nil {
			panic(vm.NewGoError(err))
		}
		return buffer.WrapBytes(vm, b)
	})
	modules.SetExport(exports, mod.Name(), "createHash", func(algorithm string) *goja.Object {
		h, err := newHash(algorithm)
		if err != nil {
			panic(vm.NewGoError(err))
		}
		obj := vm.NewObject()
		_ = obj.Set("update", func(call goja.FunctionCall) goja.Value {
			data := buffer.DecodeBytes(vm, call.Argument(0), call.Argument(1))
			_, _ = h.Write(data)
			return obj
		})
		_ = obj.Set("digest", func(call goja.FunctionCall) goja.Value {
			sum := h.Sum(nil)
			enc := call.Argument(0)
			if goja.IsUndefined(enc) || goja.IsNull(enc) {
				return buffer.WrapBytes(vm, sum)
			}
			switch enc.String() {
			case "hex":
				return vm.ToValue(hex.EncodeToString(sum))
			case "base64":
				return vm.ToValue(base64.StdEncoding.EncodeToString(sum))
			default:
				panic(vm.NewTypeError("unsupported digest encoding %q", enc.String()))
			}
		})
		return obj
	})
}

func newHash(algorithm string) (hash.Hash, error) {
	switch algorithm {
	case "sha1":
		return sha1.New(), nil
	case "sha256":
		return sha256.New(), nil
	case "sha512":
		return sha512.New(), nil
	case "md5":
		return md5.New(), nil
	default:
		return nil, fmt.Errorf("unsupported hash algorithm %q", algorithm)
	}
}

func init() { modules.Register(&m{}) }
