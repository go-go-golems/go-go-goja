package jsevents

import (
	"context"
	"fmt"
	"io/fs"
	"math"
	"os"
	pathpkg "path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/dop251/goja"
	"github.com/fsnotify/fsnotify"
	"github.com/go-go-golems/go-go-goja/engine"
)

// FSWatchOptions configures the opt-in JavaScript helper installed by
// FSWatchHelper.
type FSWatchOptions struct {
	// GlobalName is the JavaScript global object name. Defaults to "fswatch".
	GlobalName string
	// Root optionally restricts watched paths to a subtree. Relative JavaScript
	// paths are resolved against this root. Absolute paths must still remain
	// inside the root.
	Root string
	// AllowPath decides whether the normalized path may be watched. If nil, all
	// normalized paths are allowed.
	AllowPath func(path string) bool
	// AllowRecursive permits JavaScript callers to request recursive directory
	// watching with { recursive: true }. Recursive watching is disabled by
	// default because it can allocate one OS watch per directory.
	AllowRecursive bool
	// MaxDebounce optionally caps the JavaScript debounceMs option. Zero means no
	// explicit cap.
	MaxDebounce time.Duration
	// IgnorePath excludes host paths from recursive traversal and event delivery.
	// It receives normalized host paths.
	IgnorePath func(path string) bool
}

type fsWatchHelper struct {
	opts FSWatchOptions
}

type fsWatchCallOptions struct {
	Recursive bool
	Debounce  time.Duration
	Include   []string
	Exclude   []string
}

type fsWatchEventPayload struct {
	Source       string
	WatchPath    string
	Name         string
	RelativeName string
	Op           string
	Create       bool
	Write        bool
	Remove       bool
	Rename       bool
	Chmod        bool
	Recursive    bool
	Debounced    bool
	Count        int
}

func (p fsWatchEventPayload) ToValue(vm *goja.Runtime) goja.Value {
	obj := vm.NewObject()
	_ = obj.Set("source", p.Source)
	_ = obj.Set("watchPath", p.WatchPath)
	_ = obj.Set("name", p.Name)
	_ = obj.Set("relativeName", p.RelativeName)
	_ = obj.Set("op", p.Op)
	_ = obj.Set("create", p.Create)
	_ = obj.Set("write", p.Write)
	_ = obj.Set("remove", p.Remove)
	_ = obj.Set("rename", p.Rename)
	_ = obj.Set("chmod", p.Chmod)
	_ = obj.Set("recursive", p.Recursive)
	_ = obj.Set("debounced", p.Debounced)
	_ = obj.Set("count", p.Count)
	return obj
}

type fsWatchErrorPayload struct {
	Source  string
	Path    string
	Message string
}

func (p fsWatchErrorPayload) ToValue(vm *goja.Runtime) goja.Value {
	obj := vm.NewObject()
	_ = obj.Set("source", p.Source)
	_ = obj.Set("path", p.Path)
	_ = obj.Set("message", p.Message)
	return obj
}

type fsWatchConnection struct {
	Ref     *EmitterRef
	Path    string
	Options fsWatchCallOptions
}

func (c fsWatchConnection) ToValue(vm *goja.Runtime) goja.Value {
	obj := vm.NewObject()
	_ = obj.Set("id", c.Ref.ID())
	_ = obj.Set("path", c.Path)
	_ = obj.Set("recursive", c.Options.Recursive)
	_ = obj.Set("debounceMs", int64(c.Options.Debounce/time.Millisecond))
	_ = obj.Set("include", append([]string(nil), c.Options.Include...))
	_ = obj.Set("exclude", append([]string(nil), c.Options.Exclude...))
	_ = obj.Set("close", func() bool {
		return c.Ref.Close(context.Background()) == nil
	})
	return obj
}

type fsWatchGlobMatcher struct {
	include []string
	exclude []string
}

type pendingFSEvent struct {
	Event fsnotify.Event
	Count int
}

type fsWatchState struct {
	watchPath string
	opts      fsWatchCallOptions
	matcher   fsWatchGlobMatcher
	hostOpts  FSWatchOptions
	watcher   *fsnotify.Watcher
	ref       *EmitterRef

	mu           sync.Mutex
	watchedPaths map[string]struct{}

	debounceMu     sync.Mutex
	pending        map[string]pendingFSEvent
	timers         map[string]*time.Timer
	debounceClosed bool
}

// FSWatchHelper installs a JS-callable helper object with watch(path, emitter,
// options?). It does not create any filesystem watchers until JavaScript calls
// watch.
func FSWatchHelper(opts FSWatchOptions) engine.RuntimeInitializer {
	return &fsWatchHelper{opts: opts}
}

func (h *fsWatchHelper) ID() string { return "jsevents.fswatch-helper" }

func (h *fsWatchHelper) InitRuntime(ctx *engine.RuntimeContext) error {
	if ctx == nil || ctx.VM == nil {
		return fmt.Errorf("jsevents fswatch: incomplete runtime context")
	}
	managerValue, ok := ctx.Value(RuntimeValueKey)
	if !ok {
		return fmt.Errorf("jsevents fswatch: manager is not installed; add jsevents.Install() before FSWatchHelper")
	}
	manager, ok := managerValue.(*Manager)
	if !ok || manager == nil {
		return fmt.Errorf("jsevents fswatch: invalid manager value")
	}

	globalName := h.opts.GlobalName
	if globalName == "" {
		globalName = "fswatch"
	}

	obj := ctx.VM.NewObject()
	if err := obj.Set("watch", func(call goja.FunctionCall) goja.Value {
		path, err := normalizeWatchPath(call.Argument(0), h.opts)
		if err != nil {
			panic(ctx.VM.NewGoError(err))
		}
		callOpts, err := decodeFSWatchCallOptions(ctx.VM, call.Argument(2), h.opts)
		if err != nil {
			panic(ctx.VM.NewGoError(err))
		}

		ref, err := manager.AdoptEmitterOnOwner(call.Argument(1))
		if err != nil {
			panic(ctx.VM.NewGoError(err))
		}

		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			_ = ref.Close(context.Background())
			panic(ctx.VM.NewGoError(fmt.Errorf("fswatch: create watcher: %w", err)))
		}

		state := newFSWatchState(path, callOpts, h.opts, watcher, ref)
		if err := state.start(); err != nil {
			_ = watcher.Close()
			_ = ref.Close(context.Background())
			panic(ctx.VM.NewGoError(err))
		}

		watchCtx, cancel := context.WithCancel(ctx.Context)
		ref.SetCancel(func() {
			state.stopDebounceTimers()
			cancel()
		})
		go state.run(watchCtx)

		return fsWatchConnection{Ref: ref, Path: path, Options: callOpts}.ToValue(ctx.VM)
	}); err != nil {
		return err
	}

	return ctx.VM.Set(globalName, obj)
}

func normalizeWatchPath(value goja.Value, opts FSWatchOptions) (string, error) {
	if value == nil || goja.IsUndefined(value) || goja.IsNull(value) {
		return "", fmt.Errorf("fswatch: path is required")
	}
	raw := strings.TrimSpace(value.String())
	if raw == "" {
		return "", fmt.Errorf("fswatch: path is empty")
	}

	if opts.Root != "" {
		root, err := filepath.Abs(opts.Root)
		if err != nil {
			return "", fmt.Errorf("fswatch: resolve root: %w", err)
		}
		root = filepath.Clean(root)

		candidate := raw
		if filepath.IsAbs(candidate) {
			candidate = filepath.Clean(candidate)
		} else {
			candidate = filepath.Join(root, candidate)
		}
		candidate, err = filepath.Abs(candidate)
		if err != nil {
			return "", fmt.Errorf("fswatch: resolve path: %w", err)
		}
		candidate = filepath.Clean(candidate)

		rel, err := filepath.Rel(root, candidate)
		if err != nil {
			return "", fmt.Errorf("fswatch: compare path to root: %w", err)
		}
		if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return "", fmt.Errorf("fswatch: path %q escapes root %q", raw, root)
		}
		raw = candidate
	} else {
		raw = filepath.Clean(raw)
	}

	if opts.AllowPath != nil && !opts.AllowPath(raw) {
		return "", fmt.Errorf("fswatch: path %q is not allowed", raw)
	}
	return raw, nil
}

func decodeFSWatchCallOptions(vm *goja.Runtime, value goja.Value, hostOpts FSWatchOptions) (fsWatchCallOptions, error) {
	ret := fsWatchCallOptions{}
	if value == nil || goja.IsUndefined(value) || goja.IsNull(value) {
		return ret, nil
	}
	obj := value.ToObject(vm)

	recursive := obj.Get("recursive")
	if recursive != nil && !goja.IsUndefined(recursive) && !goja.IsNull(recursive) {
		ret.Recursive = recursive.ToBoolean()
	}
	if ret.Recursive && !hostOpts.AllowRecursive {
		return ret, fmt.Errorf("fswatch: recursive watches are not allowed")
	}

	debounce := obj.Get("debounceMs")
	if debounce != nil && !goja.IsUndefined(debounce) && !goja.IsNull(debounce) {
		ms := debounce.ToFloat()
		if math.IsNaN(ms) || math.IsInf(ms, 0) || ms < 0 {
			return ret, fmt.Errorf("fswatch: debounceMs must be a finite non-negative number")
		}
		ret.Debounce = time.Duration(ms) * time.Millisecond
		if hostOpts.MaxDebounce > 0 && ret.Debounce > hostOpts.MaxDebounce {
			return ret, fmt.Errorf("fswatch: debounceMs exceeds maximum of %dms", hostOpts.MaxDebounce/time.Millisecond)
		}
	}

	include, err := stringArrayOption(vm, obj.Get("include"), "include")
	if err != nil {
		return ret, err
	}
	exclude, err := stringArrayOption(vm, obj.Get("exclude"), "exclude")
	if err != nil {
		return ret, err
	}
	ret.Include = include
	ret.Exclude = exclude
	if err := validateGlobPatterns(ret.Include, "include"); err != nil {
		return ret, err
	}
	if err := validateGlobPatterns(ret.Exclude, "exclude"); err != nil {
		return ret, err
	}
	return ret, nil
}

func stringArrayOption(vm *goja.Runtime, value goja.Value, name string) ([]string, error) {
	if value == nil || goja.IsUndefined(value) || goja.IsNull(value) {
		return nil, nil
	}
	obj := value.ToObject(vm)
	lengthValue := obj.Get("length")
	if lengthValue == nil || goja.IsUndefined(lengthValue) || goja.IsNull(lengthValue) {
		return nil, fmt.Errorf("fswatch: %s must be an array of strings", name)
	}
	length := int(lengthValue.ToInteger())
	ret := make([]string, 0, length)
	for i := 0; i < length; i++ {
		item := obj.Get(fmt.Sprintf("%d", i))
		if item == nil || goja.IsUndefined(item) || goja.IsNull(item) {
			return nil, fmt.Errorf("fswatch: %s[%d] must be a string", name, i)
		}
		pattern := strings.TrimSpace(item.String())
		if pattern == "" {
			return nil, fmt.Errorf("fswatch: %s[%d] must not be empty", name, i)
		}
		ret = append(ret, filepath.ToSlash(pattern))
	}
	return ret, nil
}

func validateGlobPatterns(patterns []string, name string) error {
	for i, pattern := range patterns {
		for _, segment := range splitGlobPath(pattern) {
			if segment == "**" {
				continue
			}
			if _, err := pathpkg.Match(segment, ""); err != nil {
				return fmt.Errorf("fswatch: invalid %s glob at index %d: %w", name, i, err)
			}
		}
	}
	return nil
}

func newFSWatchState(watchPath string, opts fsWatchCallOptions, hostOpts FSWatchOptions, watcher *fsnotify.Watcher, ref *EmitterRef) *fsWatchState {
	return &fsWatchState{
		watchPath:    watchPath,
		opts:         opts,
		matcher:      fsWatchGlobMatcher{include: opts.Include, exclude: opts.Exclude},
		hostOpts:     hostOpts,
		watcher:      watcher,
		ref:          ref,
		watchedPaths: map[string]struct{}{},
		pending:      map[string]pendingFSEvent{},
		timers:       map[string]*time.Timer{},
	}
}

func (s *fsWatchState) start() error {
	info, err := os.Lstat(s.watchPath)
	if err != nil {
		return fmt.Errorf("fswatch: stat %q: %w", s.watchPath, err)
	}
	if s.hostOpts.IgnorePath != nil && s.hostOpts.IgnorePath(s.watchPath) {
		return fmt.Errorf("fswatch: path %q is ignored", s.watchPath)
	}
	if s.opts.Recursive && info.IsDir() {
		return s.addRecursive(s.watchPath)
	}
	return s.addWatchPath(s.watchPath)
}

func (s *fsWatchState) run(ctx context.Context) {
	defer func() {
		s.stopDebounceTimers()
		_ = s.watcher.Close()
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-s.watcher.Events:
			if !ok {
				_ = s.ref.Emit(context.Background(), "close")
				_ = s.ref.Close(context.Background())
				return
			}
			s.handleEvent(ctx, event)
		case err, ok := <-s.watcher.Errors:
			if !ok {
				_ = s.ref.Emit(context.Background(), "close")
				_ = s.ref.Close(context.Background())
				return
			}
			_ = s.emitError(ctx, err)
		}
	}
}

func (s *fsWatchState) handleEvent(ctx context.Context, event fsnotify.Event) {
	if s.opts.Recursive && event.Has(fsnotify.Create) {
		s.maybeAddCreatedDirectory(ctx, event.Name)
	}
	if s.opts.Recursive && (event.Has(fsnotify.Remove) || event.Has(fsnotify.Rename)) {
		s.removeWatchedPath(event.Name)
	}
	if !s.allowsEvent(event.Name) {
		return
	}
	s.dispatch(ctx, event)
}

func (s *fsWatchState) addRecursive(root string) error {
	return filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			return nil
		}
		if entry.Type()&fs.ModeSymlink != 0 {
			return filepath.SkipDir
		}
		if s.hostOpts.IgnorePath != nil && s.hostOpts.IgnorePath(path) {
			return filepath.SkipDir
		}
		rel := s.relativeName(path)
		if !s.matcher.ShouldDescend(rel) {
			return filepath.SkipDir
		}
		return s.addWatchPath(path)
	})
}

func (s *fsWatchState) addWatchPath(rawPath string) error {
	cleaned := filepath.Clean(rawPath)
	s.mu.Lock()
	if _, ok := s.watchedPaths[cleaned]; ok {
		s.mu.Unlock()
		return nil
	}
	s.mu.Unlock()

	if err := s.watcher.Add(cleaned); err != nil {
		return fmt.Errorf("fswatch: watch %q: %w", cleaned, err)
	}

	s.mu.Lock()
	s.watchedPaths[cleaned] = struct{}{}
	s.mu.Unlock()
	return nil
}

func (s *fsWatchState) maybeAddCreatedDirectory(ctx context.Context, rawPath string) {
	info, err := os.Lstat(rawPath)
	if err != nil || !info.IsDir() || info.Mode()&fs.ModeSymlink != 0 {
		return
	}
	if s.hostOpts.IgnorePath != nil && s.hostOpts.IgnorePath(rawPath) {
		return
	}
	if !s.matcher.ShouldDescend(s.relativeName(rawPath)) {
		return
	}
	if err := s.addRecursive(rawPath); err != nil {
		_ = s.emitError(ctx, err)
	}
}

func (s *fsWatchState) removeWatchedPath(rawPath string) {
	cleaned := filepath.Clean(rawPath)
	s.mu.Lock()
	_, ok := s.watchedPaths[cleaned]
	if ok {
		delete(s.watchedPaths, cleaned)
	}
	s.mu.Unlock()
	if ok {
		_ = s.watcher.Remove(cleaned)
	}
}

func (s *fsWatchState) allowsEvent(name string) bool {
	if s.hostOpts.IgnorePath != nil && s.hostOpts.IgnorePath(name) {
		return false
	}
	return s.matcher.Allows(s.relativeName(name))
}

func (s *fsWatchState) dispatch(ctx context.Context, event fsnotify.Event) {
	if s.opts.Debounce <= 0 {
		_ = s.emitEvent(ctx, event, 1, false)
		return
	}
	s.dispatchDebounced(event)
}

func (s *fsWatchState) dispatchDebounced(event fsnotify.Event) {
	key := filepath.Clean(event.Name)
	s.debounceMu.Lock()
	if s.debounceClosed {
		s.debounceMu.Unlock()
		return
	}
	pending, ok := s.pending[key]
	if ok {
		pending.Event.Op |= event.Op
		pending.Count++
	} else {
		pending = pendingFSEvent{Event: event, Count: 1}
	}
	s.pending[key] = pending
	if timer, ok := s.timers[key]; ok {
		timer.Stop()
	}
	s.timers[key] = time.AfterFunc(s.opts.Debounce, func() {
		s.flushDebounced(key)
	})
	s.debounceMu.Unlock()
}

func (s *fsWatchState) flushDebounced(key string) {
	s.debounceMu.Lock()
	if s.debounceClosed {
		s.debounceMu.Unlock()
		return
	}
	pending, ok := s.pending[key]
	if ok {
		delete(s.pending, key)
	}
	if timer, ok := s.timers[key]; ok {
		timer.Stop()
		delete(s.timers, key)
	}
	s.debounceMu.Unlock()
	if !ok {
		return
	}
	_ = s.emitEvent(context.Background(), pending.Event, pending.Count, true)
}

func (s *fsWatchState) stopDebounceTimers() {
	s.debounceMu.Lock()
	defer s.debounceMu.Unlock()
	s.debounceClosed = true
	for key, timer := range s.timers {
		timer.Stop()
		delete(s.timers, key)
	}
	for key := range s.pending {
		delete(s.pending, key)
	}
}

func (s *fsWatchState) emitEvent(ctx context.Context, event fsnotify.Event, count int, debounced bool) error {
	payload := s.eventPayload(event, count, debounced)
	return s.ref.EmitWithBuilder(ctx, "event", func(vm *goja.Runtime) ([]goja.Value, error) {
		return []goja.Value{payload.ToValue(vm)}, nil
	})
}

func (s *fsWatchState) emitError(ctx context.Context, err error) error {
	payload := fsWatchErrorPayload{
		Source:  "fsnotify",
		Path:    s.watchPath,
		Message: err.Error(),
	}
	return s.ref.EmitWithBuilder(ctx, "error", func(vm *goja.Runtime) ([]goja.Value, error) {
		return []goja.Value{payload.ToValue(vm)}, nil
	})
}

func (s *fsWatchState) eventPayload(event fsnotify.Event, count int, debounced bool) fsWatchEventPayload {
	return fsWatchEventPayload{
		Source:       "fsnotify",
		WatchPath:    s.watchPath,
		Name:         event.Name,
		RelativeName: s.relativeName(event.Name),
		Op:           event.Op.String(),
		Create:       event.Has(fsnotify.Create),
		Write:        event.Has(fsnotify.Write),
		Remove:       event.Has(fsnotify.Remove),
		Rename:       event.Has(fsnotify.Rename),
		Chmod:        event.Has(fsnotify.Chmod),
		Recursive:    s.opts.Recursive,
		Debounced:    debounced,
		Count:        count,
	}
}

func (s *fsWatchState) relativeName(name string) string {
	rel, err := filepath.Rel(s.watchPath, name)
	if err != nil || rel == "." {
		return ""
	}
	return filepath.ToSlash(rel)
}

func (m fsWatchGlobMatcher) Allows(rel string) bool {
	rel = normalizeGlobPath(rel)
	if len(m.include) > 0 {
		matched := false
		for _, pattern := range m.include {
			if matchGlob(pattern, rel) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	for _, pattern := range m.exclude {
		if matchGlob(pattern, rel) {
			return false
		}
	}
	return true
}

func (m fsWatchGlobMatcher) ShouldDescend(rel string) bool {
	rel = normalizeGlobPath(rel)
	for _, pattern := range m.exclude {
		if matchGlob(pattern, rel) {
			return false
		}
	}
	return true
}

func matchGlob(pattern, rel string) bool {
	patternSegments := splitGlobPath(pattern)
	relSegments := splitGlobPath(rel)
	return matchGlobSegments(patternSegments, relSegments)
}

func matchGlobSegments(patternSegments, relSegments []string) bool {
	if len(patternSegments) == 0 {
		return len(relSegments) == 0
	}
	if patternSegments[0] == "**" {
		if matchGlobSegments(patternSegments[1:], relSegments) {
			return true
		}
		for i := range relSegments {
			if matchGlobSegments(patternSegments[1:], relSegments[i+1:]) {
				return true
			}
		}
		return false
	}
	if len(relSegments) == 0 {
		return false
	}
	matched, err := pathpkg.Match(patternSegments[0], relSegments[0])
	if err != nil || !matched {
		return false
	}
	return matchGlobSegments(patternSegments[1:], relSegments[1:])
}

func splitGlobPath(value string) []string {
	value = normalizeGlobPath(value)
	if value == "" {
		return nil
	}
	parts := strings.Split(value, "/")
	ret := parts[:0]
	for _, part := range parts {
		if part != "" && part != "." {
			ret = append(ret, part)
		}
	}
	return ret
}

func normalizeGlobPath(value string) string {
	value = filepath.ToSlash(strings.TrimSpace(value))
	value = strings.TrimPrefix(value, "./")
	value = strings.Trim(value, "/")
	if value == "." {
		return ""
	}
	return value
}
