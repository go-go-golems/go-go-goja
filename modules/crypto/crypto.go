package cryptomod

import (
	"crypto/md5" // #nosec G501 -- Node compatibility for createHash("md5"); not used for internal security.
	"crypto/rand"
	"crypto/sha1" // #nosec G505 -- Node compatibility for createHash("sha1"); not used for internal security.
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

type m struct{ name string }

var _ modules.NativeModule = (*m)(nil)
var _ modules.TypeScriptDeclarer = (*m)(nil)

func (m m) Name() string {
	if m.name != "" {
		return m.name
	}
	return "crypto"
}
func (m m) Doc() string {
	return `The crypto module provides randomUUID, randomBytes, and basic createHash support.`
}
func (m m) TypeScriptModule() *spec.Module {
	return &spec.Module{Name: m.Name(), Functions: []spec.Function{
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
		return sha1.New(), nil // #nosec G401 -- Node compatibility for caller-requested createHash("sha1").
	case "sha256":
		return sha256.New(), nil
	case "sha512":
		return sha512.New(), nil
	case "md5":
		return md5.New(), nil // #nosec G401 -- Node compatibility for caller-requested createHash("md5").
	default:
		return nil, fmt.Errorf("unsupported hash algorithm %q", algorithm)
	}
}

func init() {
	modules.Register(&m{name: "crypto"})
	modules.Register(&m{name: "node:crypto"})
}
