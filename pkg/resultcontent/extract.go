package resultcontent

import (
	"fmt"
	"reflect"
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
	desc := describe(p)
	if desc == "" {
		return fmt.Sprintf("[content %d]", n)
	}
	return fmt.Sprintf("[content %d: %s]", n, desc)
}

func describe(p Provider) string {
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
// The walk descends only into arrays and plain objects (including Go maps
// exposed as objects). It never reads or calls "then", never awaits, never
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
		providerIdx: map[*goja.Object]int{},
		inProgress:  map[*goja.Object]struct{}{},
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
	providerIdx map[*goja.Object]int
	inProgress  map[*goja.Object]struct{}
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
		w.container(obj, depth, w.walkArray)
	case class == "Object" && t == plainObjectType:
		w.container(obj, depth, w.walkObject)
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

func (w *walker) container(obj *goja.Object, depth int, fn func(*goja.Object, int)) {
	if _, cyclic := w.inProgress[obj]; cyclic {
		w.write("[Circular]")
		return
	}
	if depth >= w.limits.MaxDepth || w.stopped {
		w.truncated = true
		w.write("…")
		return
	}
	w.inProgress[obj] = struct{}{}
	defer delete(w.inProgress, obj)
	fn(obj, depth)
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
	keys := obj.Keys()
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

func (w *walker) addProvider(obj *goja.Object, p Provider) int {
	if idx, ok := w.providerIdx[obj]; ok {
		return idx
	}
	w.providers = append(w.providers, p)
	idx := len(w.providers)
	w.providerIdx[obj] = idx
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
