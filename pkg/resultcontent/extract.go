package resultcontent

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/dop251/goja"
	"github.com/pkg/errors"
)

var (
	providerType    = reflect.TypeOf((*Provider)(nil)).Elem()
	plainObjectType = reflect.TypeOf(map[string]interface{}{})
	plainArrayType  = reflect.TypeOf([]interface{}{})
	proxyType       = reflect.TypeOf(goja.Proxy{})
	timeType        = reflect.TypeOf(time.Time{})
	promiseType     = reflect.TypeOf((*goja.Promise)(nil))
)

// Extraction is the result of walking one settled value.
type Extraction struct {
	// Text renders the value with "[content N: placeholder]" in place of each
	// provider. N is 1-based and indexes Providers.
	Text string
	// Providers are the distinct providers found, in walk order.
	Providers []Provider
	// Truncated reports that a depth, property or text limit cut the walk.
	Truncated bool
}

// ProviderOf reports whether value is a Go-backed object whose exported value
// implements Provider. It inspects the export type first and never calls
// JavaScript, so it is safe for proxies and thenables. It must run on the
// runtime owner goroutine.
func ProviderOf(value goja.Value) (Provider, bool) {
	obj, ok := value.(*goja.Object)
	if !ok || obj == nil {
		return nil, false
	}
	t := obj.ExportType()
	if t == nil || !t.Implements(providerType) {
		return nil, false
	}
	p, ok := obj.Export().(Provider)
	return p, ok && p != nil
}

// Placeholder returns the text that stands for provider number n (1-based).
func Placeholder(n int, p Provider) string {
	desc := Describe(p)
	if desc == "" {
		return fmt.Sprintf("[content %d]", n)
	}
	return fmt.Sprintf("[content %d: %s]", n, desc)
}

// Describe returns the provider's Describer placeholder, or "" when it has
// none (or the Describer panicked). It calls Go code only.
func Describe(p Provider) string {
	d, ok := p.(Describer)
	if !ok {
		return ""
	}
	desc := ""
	func() {
		defer func() { _ = recover() }()
		desc = d.ResultPlaceholder()
	}()
	return strings.TrimSpace(desc)
}

// Extract walks a settled value and collects content providers. It must run
// on the runtime owner goroutine.
//
// The walk descends only into arrays and plain objects, and into Go maps with
// string keys and Go slices/arrays exposed by native modules (including typed
// ones such as map[string]*Image or []*Image; typed map keys render sorted,
// byte slices render as "[N bytes]"). It never reads or calls "then", never awaits, never
// invokes getters (accessor properties render as "[Getter]"), and treats
// proxies, functions, promises and other exotic objects as opaque leaves. It
// is bounded by limits.MaxDepth, limits.MaxProperties and limits.MaxTextChars
// and is cycle-safe; the same provider reached twice keeps one number.
func Extract(vm *goja.Runtime, value goja.Value, limits Limits) (Extraction, error) {
	if vm == nil {
		return Extraction{}, errors.New("resultcontent: runtime is nil")
	}
	w := &walker{
		vm:          vm,
		limits:      limits.Normalize(),
		providerIdx: map[any]int{},
		inProgress:  map[any]struct{}{},
	}
	w.getDesc = lookupGetOwnPropertyDescriptor(vm)

	var walkErr error
	func() {
		defer func() {
			if r := recover(); r != nil {
				walkErr = errors.Errorf("resultcontent: walk failed: %v", r)
			}
		}()
		if exc := vm.Try(func() { w.walk(value, 0) }); exc != nil {
			walkErr = errors.Wrap(exc, "resultcontent: walk failed")
		}
	}()
	if walkErr != nil {
		return Extraction{}, walkErr
	}
	return Extraction{Text: w.text(), Providers: w.providers, Truncated: w.truncated}, nil
}

type walker struct {
	vm          *goja.Runtime
	limits      Limits
	getDesc     goja.Callable
	b           strings.Builder
	textFull    bool
	props       int
	stopped     bool
	truncated   bool
	providers   []Provider
	providerIdx map[any]int
	inProgress  map[any]struct{}
}

func lookupGetOwnPropertyDescriptor(vm *goja.Runtime) goja.Callable {
	objectCtor, ok := vm.GlobalObject().Get("Object").(*goja.Object)
	if !ok || objectCtor == nil {
		return nil
	}
	fn, ok := goja.AssertFunction(objectCtor.Get("getOwnPropertyDescriptor"))
	if !ok {
		return nil
	}
	return fn
}

func (w *walker) text() string {
	s := w.b.String()
	if w.textFull {
		return s + "…"
	}
	return s
}

func (w *walker) write(s string) {
	if w.textFull {
		return
	}
	remaining := w.limits.MaxTextChars - w.b.Len()
	if len(s) > remaining {
		if remaining > 0 {
			w.b.WriteString(truncateUTF8(s, remaining))
		}
		w.textFull = true
		w.truncated = true
		return
	}
	w.b.WriteString(s)
}

// countProperty consumes one unit of the property budget. It returns false
// when the budget is exhausted, after which the walk stops descending.
func (w *walker) countProperty() bool {
	if w.stopped {
		return false
	}
	w.props++
	if w.props > w.limits.MaxProperties {
		w.stopped = true
		w.truncated = true
		return false
	}
	return true
}

func (w *walker) walk(value goja.Value, depth int) {
	if value == nil || goja.IsUndefined(value) {
		w.write("undefined")
		return
	}
	if goja.IsNull(value) {
		w.write("null")
		return
	}
	obj, ok := value.(*goja.Object)
	if !ok {
		w.writePrimitive(value)
		return
	}
	if p, ok := ProviderOf(obj); ok {
		w.write(Placeholder(w.addProvider(obj, p), p))
		return
	}
	if _, isFn := goja.AssertFunction(obj); isFn {
		w.write("[Function]")
		return
	}
	t := obj.ExportType()
	switch t {
	case proxyType:
		w.write("[Proxy]")
		return
	case promiseType:
		w.write("[Promise]")
		return
	}
	class := obj.ClassName()
	switch {
	case class == "Array" && t == plainArrayType:
		w.container(obj, nil, depth, w.walkArray)
	case class == "Object" && t == plainObjectType:
		w.container(obj, goBackedIdentity(obj, t), depth, w.walkObject)
	case isByteSlice(t):
		w.writeBytes(obj)
	case class == "Array" && t != nil && (t.Kind() == reflect.Slice || t.Kind() == reflect.Array):
		// Typed Go slice/array exposed by a native module, e.g. []*Image.
		w.container(obj, goBackedIdentity(obj, t), depth, w.walkArray)
	case t != nil && t != plainObjectType && t.Kind() == reflect.Map && t.Key().Kind() == reflect.String:
		// Typed Go map exposed by a native module, e.g. map[string]*Image.
		w.container(obj, goBackedIdentity(obj, t), depth, w.walkGoMap)
	case t == timeType:
		if tm, ok := obj.Export().(time.Time); ok {
			w.write(tm.UTC().Format(time.RFC3339Nano))
			return
		}
		w.write("[Date]")
	case class == "Error":
		w.writeError(obj)
	default:
		w.write("[" + class + "]")
	}
}

// container walks obj unless it is already being walked further up. Go-backed
// maps and slices are re-wrapped in a new *goja.Object on every property read,
// so they are tracked by goID (type + backing pointer) instead of the wrapper.
func (w *walker) container(obj *goja.Object, goID any, depth int, fn func(*goja.Object, int)) {
	var id any = obj
	if goID != nil {
		id = goID
	}
	if _, cyclic := w.inProgress[id]; cyclic {
		w.write("[Circular]")
		return
	}
	if depth >= w.limits.MaxDepth || w.stopped {
		w.truncated = true
		w.write("…")
		return
	}
	w.inProgress[id] = struct{}{}
	defer delete(w.inProgress, id)
	fn(obj, depth)
}

// goBackedIdentity returns the backing-pointer identity of a Go map or slice
// behind obj, or nil for plain JavaScript objects and arrays. Export on a
// Go-backed object returns the wrapped value without running JavaScript.
func goBackedIdentity(obj *goja.Object, t reflect.Type) any {
	if t == nil || t == plainArrayType {
		return nil
	}
	if t == plainObjectType {
		// Plain JS objects export as map[string]interface{} too and cannot be
		// told apart from a Go map[string]any without Export, which would read
		// every property of a JS object. They are tracked by their wrapper;
		// cycles through untyped Go maps are bounded by MaxDepth instead.
		return nil
	}
	id, ok := containerIdentity(reflect.ValueOf(obj.Export()))
	if !ok {
		return nil
	}
	return id
}

func isByteSlice(t reflect.Type) bool {
	return t != nil && (t.Kind() == reflect.Slice || t.Kind() == reflect.Array) && t.Elem().Kind() == reflect.Uint8
}

func (w *walker) writeBytes(obj *goja.Object) {
	rv := reflect.ValueOf(obj.Export())
	if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array {
		w.write(fmt.Sprintf("[%d bytes]", rv.Len()))
		return
	}
	w.write("[bytes]")
}

// walkGoMap renders a typed Go map with sorted keys (Go map order is random).
func (w *walker) walkGoMap(obj *goja.Object, depth int) {
	keys := obj.Keys()
	sort.Strings(keys)
	w.writeObjectKeys(obj, keys, depth)
}

func (w *walker) walkArray(obj *goja.Object, depth int) {
	length := obj.Get("length").ToInteger()
	w.write("[")
	for i := int64(0); i < length; i++ {
		if i > 0 {
			w.write(", ")
		}
		if !w.countProperty() {
			w.write("…")
			break
		}
		w.walkProperty(obj, strconv.FormatInt(i, 10), depth)
	}
	w.write("]")
}

func (w *walker) walkObject(obj *goja.Object, depth int) {
	w.writeObjectKeys(obj, obj.Keys(), depth)
}

func (w *walker) writeObjectKeys(obj *goja.Object, keys []string, depth int) {
	w.write("{")
	first := true
	for _, key := range keys {
		if key == "then" {
			continue
		}
		if !first {
			w.write(", ")
		}
		first = false
		if !w.countProperty() {
			w.write("…")
			break
		}
		w.write(formatKey(key))
		w.write(": ")
		w.walkProperty(obj, key, depth)
	}
	w.write("}")
}

// walkProperty reads one own property through its descriptor so accessors
// are reported instead of invoked.
func (w *walker) walkProperty(obj *goja.Object, key string, depth int) {
	if w.getDesc == nil {
		w.write("[Unknown]")
		return
	}
	descValue, err := w.getDesc(goja.Undefined(), obj, w.vm.ToValue(key))
	if err != nil {
		w.write("[Unreadable]")
		return
	}
	desc, ok := descValue.(*goja.Object)
	if !ok || desc == nil {
		w.write("undefined")
		return
	}
	if isAccessorDescriptor(desc) {
		w.write("[Getter]")
		return
	}
	w.walk(desc.Get("value"), depth+1)
}

func isAccessorDescriptor(desc *goja.Object) bool {
	for _, name := range []string{"get", "set"} {
		v := desc.Get(name)
		if v != nil && !goja.IsUndefined(v) {
			return true
		}
	}
	return false
}

func (w *walker) writeError(obj *goja.Object) {
	if w.getDesc != nil {
		if descValue, err := w.getDesc(goja.Undefined(), obj, w.vm.ToValue("message")); err == nil {
			if desc, ok := descValue.(*goja.Object); ok && desc != nil && !isAccessorDescriptor(desc) {
				if msg := desc.Get("value"); msg != nil && !goja.IsUndefined(msg) {
					if _, isObj := msg.(*goja.Object); !isObj {
						w.write("[Error: " + msg.String() + "]")
						return
					}
				}
			}
		}
	}
	w.write("[Error]")
}

func (w *walker) writePrimitive(value goja.Value) {
	switch exported := value.Export().(type) {
	case string:
		w.write(strconv.Quote(exported))
	default:
		w.write(value.String())
	}
}

// addProvider numbers p. Pointer-like providers are deduplicated by address
// (so the same *Image reached through two freshly wrapped Go slice elements
// keeps one number); value providers by their *goja.Object wrapper, so two
// equal values returned by two calls stay two items.
func (w *walker) addProvider(obj *goja.Object, p Provider) int {
	var key any = obj
	if k, ok := addressKey(p); ok {
		key = k
	}
	if idx, ok := w.providerIdx[key]; ok {
		return idx
	}
	w.providers = append(w.providers, p)
	idx := len(w.providers)
	w.providerIdx[key] = idx
	return idx
}

func formatKey(key string) string {
	if isIdentifier(key) {
		return key
	}
	return strconv.Quote(key)
}

func isIdentifier(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		switch {
		case r == '_' || r == '$':
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z':
		case i > 0 && r >= '0' && r <= '9':
		default:
			return false
		}
	}
	return true
}

func truncateUTF8(s string, n int) string {
	if n >= len(s) {
		return s
	}
	for n > 0 && n < len(s) && (s[n]&0xC0) == 0x80 {
		n--
	}
	return s[:n]
}
