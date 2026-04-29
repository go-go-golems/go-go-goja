package javascript

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/dop251/goja"
	"github.com/go-go-golems/bobatea/pkg/repl"
	ggjengine "github.com/go-go-golems/go-go-goja/engine"
	"github.com/go-go-golems/go-go-goja/pkg/docaccess"
	docaccessruntime "github.com/go-go-golems/go-go-goja/pkg/docaccess/runtime"
	"github.com/go-go-golems/go-go-goja/pkg/hashiplugin/host"
	"github.com/go-go-golems/go-go-goja/pkg/jsparse"
	"github.com/pkg/errors"
)

var (
	helpBarSymbolSignatures = map[string]string{
		"console":        "console: object (log, error, warn, info, debug, table)",
		"console.log":    "console.log(...args): void",
		"console.error":  "console.error(...args): void",
		"console.warn":   "console.warn(...args): void",
		"console.info":   "console.info(...args): void",
		"console.debug":  "console.debug(...args): void",
		"console.table":  "console.table(data, columns?): void",
		"Math":           "Math: object (numeric helpers)",
		"Math.max":       "Math.max(...values): number",
		"Math.min":       "Math.min(...values): number",
		"Math.random":    "Math.random(): number",
		"Math.floor":     "Math.floor(value): number",
		"Math.ceil":      "Math.ceil(value): number",
		"Math.round":     "Math.round(value): number",
		"JSON":           "JSON: object (parse, stringify)",
		"JSON.parse":     "JSON.parse(text): any",
		"JSON.stringify": "JSON.stringify(value): string",
		"fs":             "fs: module alias (file system APIs)",
		"fs.readFile":    "fs.readFile(path, [options], callback): void",
		"fs.writeFile":   "fs.writeFile(path, data, [options], callback): void",
		"fs.existsSync":  "fs.existsSync(path): bool",
		"fs.mkdirSync":   "fs.mkdirSync(path, [options]): string | undefined",
		"path":           "path: module alias (path utilities)",
		"path.join":      "path.join(...parts): string",
		"path.resolve":   "path.resolve(...parts): string",
		"path.dirname":   "path.dirname(path): string",
		"path.basename":  "path.basename(path): string",
		"path.extname":   "path.extname(path): string",
		"url":            "url: module alias (URL utilities)",
		"url.parse":      "url.parse(input): URLRecord",
		"url.URL":        "url.URL(input): URL",
	}
)

// Evaluator implements the Bobatea-oriented JavaScript REPL surface.
//
// This type remains the TUI/help/completion-facing evaluator stack. It is not
// the same subsystem as replsession, which owns durable session lifecycle and
// cell-by-cell kernel behavior.
type Evaluator struct {
	runtime            *goja.Runtime
	ownedRuntime       *ggjengine.Runtime
	runtimeMu          sync.Mutex
	tsParser           *jsparse.TSParser
	tsMu               sync.Mutex
	runtimeDeclaredMu  sync.RWMutex
	runtimeDeclaredIDs map[string]jsparse.CompletionCandidate
	docHub             *docaccess.Hub
	docsResolver       *docsResolver
	assistance         *Assistance
	config             Config
}

// Config holds configuration for the JavaScript evaluator
type Config struct {
	EnableModules      bool
	EnableConsoleLog   bool
	EnableNodeModules  bool
	PluginDirectories  []string
	PluginAllowModules []string
	PluginReporter     *host.ReportCollector
	HelpSources        []docaccessruntime.HelpSource
	JSDocSources       []docaccessruntime.JSDocSource
	RuntimeRegistrars  []ggjengine.RuntimeModuleRegistrar
	CustomModules      map[string]interface{}
	// Runtime, when set, reuses an existing VM instead of creating a new one.
	Runtime *goja.Runtime
}

// DefaultConfig returns a default configuration for JavaScript evaluation
func DefaultConfig() Config {
	return Config{
		EnableModules:      true,
		EnableConsoleLog:   true,
		EnableNodeModules:  true,
		PluginDirectories:  nil,
		PluginAllowModules: nil,
		HelpSources:        nil,
		JSDocSources:       nil,
		RuntimeRegistrars:  nil,
		CustomModules:      make(map[string]interface{}),
		Runtime:            nil,
	}
}

// New creates a new JavaScript evaluator with the given configuration
func New(config Config) (*Evaluator, error) {
	var runtime *goja.Runtime
	var ownedRuntime *ggjengine.Runtime

	if config.Runtime != nil {
		runtime = config.Runtime
	} else if config.EnableModules {
		// Create runtime with module support using explicit engine composition.
		builder := ggjengine.NewBuilder().
			UseModuleMiddleware(ggjengine.MiddlewareSafe())
		if len(config.PluginDirectories) > 0 {
			builder = builder.WithRuntimeModuleRegistrars(host.NewRegistrar(host.Config{
				Directories:  config.PluginDirectories,
				AllowModules: config.PluginAllowModules,
				Report:       config.PluginReporter,
			}))
		}
		if len(config.PluginDirectories) > 0 || len(config.HelpSources) > 0 || len(config.JSDocSources) > 0 {
			builder = builder.WithRuntimeModuleRegistrars(docaccessruntime.NewRegistrar(docaccessruntime.Config{
				HelpSources:  append([]docaccessruntime.HelpSource(nil), config.HelpSources...),
				JSDocSources: append([]docaccessruntime.JSDocSource(nil), config.JSDocSources...),
			}))
		}
		if len(config.RuntimeRegistrars) > 0 {
			builder = builder.WithRuntimeModuleRegistrars(config.RuntimeRegistrars...)
		}
		factory, err := builder.Build()
		if err != nil {
			return nil, errors.Wrap(err, "failed to build runtime factory")
		}
		ownedRuntime, err = factory.NewRuntime(context.Background())
		if err != nil {
			return nil, errors.Wrap(err, "failed to create runtime")
		}
		runtime = ownedRuntime.VM
	} else {
		// Create basic runtime without modules
		runtime = goja.New()
	}

	evaluator := &Evaluator{
		runtime:            runtime,
		ownedRuntime:       ownedRuntime,
		runtimeDeclaredIDs: map[string]jsparse.CompletionCandidate{},
		config:             config,
	}
	evaluator.docHub = docHubFromRuntime(ownedRuntime)
	evaluator.docsResolver = newDocsResolver(ownedRuntime)
	closeOwnedRuntimeOnError := func() {
		if evaluator.ownedRuntime != nil {
			_ = evaluator.ownedRuntime.Close(context.Background())
		}
	}
	if parser, parserErr := jsparse.NewTSParser(); parserErr == nil {
		evaluator.tsParser = parser
	}

	// Set up console.log override if enabled
	if config.EnableConsoleLog {
		if err := evaluator.setupConsole(); err != nil {
			closeOwnedRuntimeOnError()
			return nil, errors.Wrap(err, "failed to setup console")
		}
	}

	// Register custom modules if provided
	for name, module := range config.CustomModules {
		if err := evaluator.registerModule(name, module); err != nil {
			closeOwnedRuntimeOnError()
			return nil, errors.Wrapf(err, "failed to register custom module %s", name)
		}
	}

	evaluator.assistance = NewAssistance(AssistanceConfig{
		TSParser: evaluator.tsParser,
		TSMu:     &evaluator.tsMu,
		WithRuntime: func(ctx context.Context, fn func(*goja.Runtime, *docaccess.Hub) error) error {
			evaluator.runtimeMu.Lock()
			defer evaluator.runtimeMu.Unlock()
			return fn(evaluator.runtime, evaluator.docHub)
		},
		BindingHints: func(context.Context) ([]jsparse.CompletionCandidate, error) {
			return evaluator.runtimeIdentifierHints(), nil
		},
	})

	return evaluator, nil
}

// NewWithDefaults creates a new JavaScript evaluator with default configuration
func NewWithDefaults() (*Evaluator, error) {
	return New(DefaultConfig())
}

// setupConsole overrides console.log to provide clean REPL output
func (e *Evaluator) setupConsole() error {
	consoleObj := e.runtime.NewObject()

	// Override console.log to write directly without timestamps
	err := consoleObj.Set("log", func(call goja.FunctionCall) goja.Value {
		var args []interface{}
		for _, arg := range call.Arguments {
			args = append(args, arg.Export())
		}
		fmt.Println(args...)
		return goja.Undefined()
	})
	if err != nil {
		return errors.Wrap(err, "failed to set console.log")
	}

	// Add other console methods
	err = consoleObj.Set("error", func(call goja.FunctionCall) goja.Value {
		var args []interface{}
		for _, arg := range call.Arguments {
			args = append(args, arg.Export())
		}
		fmt.Printf("Error: %v\n", args...)
		return goja.Undefined()
	})
	if err != nil {
		return errors.Wrap(err, "failed to set console.error")
	}

	err = consoleObj.Set("warn", func(call goja.FunctionCall) goja.Value {
		var args []interface{}
		for _, arg := range call.Arguments {
			args = append(args, arg.Export())
		}
		fmt.Printf("Warning: %v\n", args...)
		return goja.Undefined()
	})
	if err != nil {
		return errors.Wrap(err, "failed to set console.warn")
	}

	return e.runtime.Set("console", consoleObj)
}

// registerModule registers a custom module with the runtime
func (e *Evaluator) registerModule(name string, module interface{}) error {
	return e.runtime.Set(name, module)
}

// Evaluate executes the given JavaScript code and returns the result
func (e *Evaluator) Evaluate(ctx context.Context, code string) (string, error) {
	// Check context for cancellation
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	code, wrappedAwait := wrapTopLevelAwaitExpression(code)

	e.runtimeMu.Lock()
	defer e.runtimeMu.Unlock()

	var (
		result goja.Value
		err    error
	)
	if e.ownedRuntime != nil {
		result, err = e.runOwned(ctx, code)
	} else {
		result, err = e.runtime.RunString(code)
	}
	if err != nil {
		return "", errors.Wrap(err, "JavaScript execution failed")
	}

	output, err := e.stringifyResult(ctx, result)
	if err != nil {
		return "", errors.Wrap(err, "JavaScript execution failed")
	}
	if wrappedAwait && output == "undefined" {
		output = ""
	}
	e.observeRuntimeDeclarations(code)

	return output, nil
}

func (e *Evaluator) runOwned(ctx context.Context, code string) (goja.Value, error) {
	ret, err := e.ownedRuntime.Owner.Call(ctx, "javascript.evaluate", func(_ context.Context, vm *goja.Runtime) (any, error) {
		return vm.RunString(code)
	})
	if err != nil {
		return nil, err
	}
	value, ok := ret.(goja.Value)
	if !ok && ret == nil {
		return nil, nil
	}
	if !ok {
		return nil, errors.Errorf("unexpected evaluation result type %T", ret)
	}
	return value, nil
}

func (e *Evaluator) stringifyResult(ctx context.Context, result goja.Value) (string, error) {
	if result == nil || goja.IsUndefined(result) {
		return "undefined", nil
	}

	if promise, ok := result.Export().(*goja.Promise); ok {
		return e.waitForPromise(ctx, promise)
	}

	return result.String(), nil
}

func (e *Evaluator) waitForPromise(ctx context.Context, promise *goja.Promise) (string, error) {
	if promise == nil {
		return "undefined", nil
	}
	if e.ownedRuntime == nil {
		return promiseString(promise)
	}

	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		ret, err := e.ownedRuntime.Owner.Call(ctx, "javascript.promise-state", func(_ context.Context, vm *goja.Runtime) (any, error) {
			return promiseSnapshot{
				State:  promise.State(),
				Result: promise.Result(),
			}, nil
		})
		if err != nil {
			return "", err
		}
		snapshot, ok := ret.(promiseSnapshot)
		if !ok {
			return "", errors.Errorf("unexpected promise snapshot type %T", ret)
		}
		switch snapshot.State {
		case goja.PromiseStatePending:
			time.Sleep(5 * time.Millisecond)
			continue
		case goja.PromiseStateRejected:
			return "", errors.Errorf("Promise rejected: %s", valueString(snapshot.Result))
		case goja.PromiseStateFulfilled:
			return valueString(snapshot.Result), nil
		}
	}
}

type promiseSnapshot struct {
	State  goja.PromiseState
	Result goja.Value
}

func promiseString(promise *goja.Promise) (string, error) {
	switch promise.State() {
	case goja.PromiseStatePending:
		return "Promise { <pending> }", nil
	case goja.PromiseStateRejected:
		return "", errors.Errorf("Promise rejected: %s", valueString(promise.Result()))
	case goja.PromiseStateFulfilled:
		return valueString(promise.Result()), nil
	}

	return "Promise { <pending> }", nil
}

func valueString(value goja.Value) string {
	if value == nil || goja.IsUndefined(value) || goja.IsNull(value) {
		return "undefined"
	}
	return value.String()
}

func wrapTopLevelAwaitExpression(code string) (string, bool) {
	trimmed := strings.TrimSpace(code)
	if strings.HasPrefix(trimmed, "await ") {
		return "(async () => { return " + trimmed + "; })()", true
	}
	return code, false
}

// EvaluateStream adapts Evaluate to the streaming interface used by the timeline-based REPL.
func (e *Evaluator) EvaluateStream(ctx context.Context, code string, emit func(repl.Event)) error {
	out, err := e.Evaluate(ctx, code)
	if err != nil {
		emit(repl.Event{Kind: repl.EventResultMarkdown, Props: map[string]any{"markdown": fmt.Sprintf("Error: %v", err)}})
		return nil
	}
	emit(repl.Event{Kind: repl.EventResultMarkdown, Props: map[string]any{"markdown": out}})
	return nil
}

// CompleteInput resolves JavaScript completions using jsparse CST + resolver primitives.
func (e *Evaluator) CompleteInput(ctx context.Context, req repl.CompletionRequest) (repl.CompletionResult, error) {
	return e.assistance.CompleteInput(ctx, req)
}

// GetHelpBar resolves contextual one-line symbol help for the JS REPL input.
func (e *Evaluator) GetHelpBar(ctx context.Context, req repl.HelpBarRequest) (repl.HelpBarPayload, error) {
	return e.assistance.GetHelpBar(ctx, req)
}

// GetHelpDrawer resolves rich contextual help for the JS REPL input.
func (e *Evaluator) GetHelpDrawer(ctx context.Context, req repl.HelpDrawerRequest) (repl.HelpDrawerDocument, error) {
	return e.assistance.GetHelpDrawer(ctx, req)
}

// GetPrompt returns the prompt string for JavaScript evaluation
func (e *Evaluator) GetPrompt() string {
	return "js>"
}

// GetName returns the name of this evaluator
func (e *Evaluator) GetName() string {
	return "JavaScript"
}

// SupportsMultiline returns true since JavaScript supports multiline input
func (e *Evaluator) SupportsMultiline() bool {
	return true
}

// GetFileExtension returns the file extension for external editor
func (e *Evaluator) GetFileExtension() string {
	return ".js"
}

// GetRuntime returns the underlying Goja runtime (for advanced usage)
func (e *Evaluator) GetRuntime() *goja.Runtime {
	return e.runtime
}

// SetVariable sets a variable in the JavaScript runtime
func (e *Evaluator) SetVariable(name string, value interface{}) error {
	e.runtimeMu.Lock()
	defer e.runtimeMu.Unlock()
	return e.runtime.Set(name, value)
}

// GetVariable gets a variable from the JavaScript runtime
func (e *Evaluator) GetVariable(name string) (interface{}, error) {
	e.runtimeMu.Lock()
	defer e.runtimeMu.Unlock()
	val := e.runtime.Get(name)
	if val == nil {
		return nil, fmt.Errorf("variable %s not found", name)
	}
	return val.Export(), nil
}

// LoadScript loads and executes a JavaScript file
func (e *Evaluator) LoadScript(ctx context.Context, filename string, content string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	e.runtimeMu.Lock()
	_, err := e.runtime.RunString(content)
	e.runtimeMu.Unlock()
	if err != nil {
		return errors.Wrapf(err, "failed to load script %s", filename)
	}

	return nil
}

// Reset resets the JavaScript runtime to a clean state
func (e *Evaluator) Reset() error {
	// Create a new runtime with the same configuration
	newEvaluator, err := New(e.config)
	if err != nil {
		return errors.Wrap(err, "failed to reset JavaScript evaluator")
	}

	var oldOwnedRuntime *ggjengine.Runtime
	e.runtimeMu.Lock()
	oldOwnedRuntime = e.ownedRuntime
	e.runtime = newEvaluator.runtime
	e.ownedRuntime = newEvaluator.ownedRuntime
	e.tsParser = newEvaluator.tsParser
	e.docHub = newEvaluator.docHub
	e.docsResolver = newEvaluator.docsResolver
	e.assistance = newEvaluator.assistance
	e.runtimeMu.Unlock()

	if oldOwnedRuntime != nil {
		_ = oldOwnedRuntime.Close(context.Background())
	}

	e.runtimeDeclaredMu.Lock()
	e.runtimeDeclaredIDs = map[string]jsparse.CompletionCandidate{}
	e.runtimeDeclaredMu.Unlock()
	return nil
}

// Close releases owned runtime resources when this evaluator created its own runtime.
// It is a no-op when evaluator reuses an externally provided runtime.
func (e *Evaluator) Close() error {
	e.runtimeMu.Lock()
	ownedRuntime := e.ownedRuntime
	e.ownedRuntime = nil
	e.runtimeMu.Unlock()

	if ownedRuntime != nil {
		if err := ownedRuntime.Close(context.Background()); err != nil {
			return errors.Wrap(err, "failed to close owned runtime")
		}
	}
	return nil
}

// GetConfig returns the current configuration
func (e *Evaluator) GetConfig() Config {
	return e.config
}

// UpdateConfig updates the evaluator configuration
func (e *Evaluator) UpdateConfig(config Config) error {
	e.config = config

	// Re-setup console if needed
	if config.EnableConsoleLog {
		if err := e.setupConsole(); err != nil {
			return errors.Wrap(err, "failed to re-setup console")
		}
	}

	// Re-register custom modules
	for name, module := range config.CustomModules {
		if err := e.registerModule(name, module); err != nil {
			return errors.Wrapf(err, "failed to re-register custom module %s", name)
		}
	}

	return nil
}

func clampCursor(cursor, upperBound int) int {
	if cursor < 0 {
		return 0
	}
	if cursor > upperBound {
		return upperBound
	}
	return cursor
}

func byteOffsetToRowCol(input string, cursor int) (int, int) {
	cursor = clampCursor(cursor, len(input))
	row, col := 0, 0
	for i := 0; i < cursor; i++ {
		if input[i] == '\n' {
			row++
			col = 0
			continue
		}
		col++
	}
	return row, col
}

func (e *Evaluator) observeRuntimeDeclarations(code string) {
	candidates := jsparse.ExtractTopLevelBindingCandidates(code)
	if len(candidates) == 0 {
		return
	}
	e.runtimeDeclaredMu.Lock()
	defer e.runtimeDeclaredMu.Unlock()
	for _, candidate := range candidates {
		if strings.TrimSpace(candidate.Label) == "" {
			continue
		}
		e.runtimeDeclaredIDs[candidate.Label] = candidate
	}
}

// RecordDeclarations updates completion/runtime identifier hints from source text
// without evaluating the code.
func (e *Evaluator) RecordDeclarations(code string) {
	e.observeRuntimeDeclarations(code)
}

func (e *Evaluator) runtimeIdentifierHints() []jsparse.CompletionCandidate {
	e.runtimeDeclaredMu.RLock()
	defer e.runtimeDeclaredMu.RUnlock()
	if len(e.runtimeDeclaredIDs) == 0 {
		return nil
	}
	out := make([]jsparse.CompletionCandidate, 0, len(e.runtimeDeclaredIDs))
	for _, candidate := range e.runtimeDeclaredIDs {
		out = append(out, candidate)
	}
	return out
}

func tokenAtCursor(input string, cursor int) (string, int, int) {
	cursor = clampCursor(cursor, len(input))
	start := cursor
	for start > 0 && isTokenByte(input[start-1]) {
		start--
	}
	end := cursor
	for end < len(input) && isTokenByte(input[end]) {
		end++
	}
	return input[start:end], start, end
}

func isTokenByte(b byte) bool {
	return (b >= 'a' && b <= 'z') ||
		(b >= 'A' && b <= 'Z') ||
		(b >= '0' && b <= '9') ||
		b == '_' ||
		b == '$' ||
		b == '.'
}

func normalizeCandidateDetail(detail string) string {
	detail = strings.TrimSpace(detail)
	if detail == "" {
		return "symbol"
	}
	return detail
}

func docEntrySummary(entry *docaccess.Entry) string {
	if entry == nil {
		return ""
	}
	summary := strings.TrimSpace(entry.Summary)
	if summary == "" {
		return entry.Title
	}
	return fmt.Sprintf("%s - %s", entry.Title, summary)
}

func makeHelpBarPayload(text, kind string) repl.HelpBarPayload {
	return repl.HelpBarPayload{
		Show:     true,
		Text:     text,
		Kind:     kind,
		Severity: "info",
	}
}

func describeHelpDrawerTrigger(trigger repl.HelpDrawerTrigger) string {
	switch trigger {
	case repl.HelpDrawerTriggerToggleOpen:
		return "trigger: toggle-open"
	case repl.HelpDrawerTriggerManualRefresh:
		return "trigger: manual-refresh"
	case repl.HelpDrawerTriggerTyping:
		return "trigger: typing"
	default:
		return "trigger: unknown"
	}
}

func describeCompletionKind(kind jsparse.CompletionKind) string {
	switch kind {
	case jsparse.CompletionIdentifier:
		return "identifier"
	case jsparse.CompletionProperty:
		return "property"
	case jsparse.CompletionArgument:
		return "argument"
	case jsparse.CompletionNone:
		return "none"
	default:
		return "unknown"
	}
}

func clipForDrawer(s string, limit int) string {
	if limit <= 0 || len(s) <= limit {
		return s
	}
	return s[:limit] + "\n// ...truncated"
}

// IsValidCode checks if the given code is syntactically valid JavaScript
func (e *Evaluator) IsValidCode(code string) bool {
	// Try to run the code in a temporary runtime to check syntax
	tempRuntime := goja.New()
	_, err := tempRuntime.RunString(code)
	return err == nil
}

// GetAvailableModules returns a list of available modules
func (e *Evaluator) GetAvailableModules() []string {
	modules := make([]string, 0)

	// Add custom modules
	for name := range e.config.CustomModules {
		modules = append(modules, name)
	}

	// Add standard modules if enabled
	if e.config.EnableModules {
		// These are typical modules available through go-go-goja
		standardModules := []string{
			"database",
			"http",
			"fs",
			"path",
			"url",
		}
		modules = append(modules, standardModules...)
	}

	return modules
}

// GetHelpText returns help text for JavaScript evaluation
func (e *Evaluator) GetHelpText() string {
	var help strings.Builder

	help.WriteString("JavaScript REPL - Powered by Goja\n\n")
	help.WriteString("Available features:\n")
	help.WriteString("- Full ES5/ES6 JavaScript support\n")
	help.WriteString("- Multiline input support\n")
	help.WriteString("- Variable persistence across evaluations\n")

	if e.config.EnableConsoleLog {
		help.WriteString("- Console logging (console.log, console.error, console.warn)\n")
	}

	if e.config.EnableModules {
		help.WriteString("- Module system support (require())\n")
		modules := e.GetAvailableModules()
		if len(modules) > 0 {
			help.WriteString("- Available modules: ")
			help.WriteString(strings.Join(modules, ", "))
			help.WriteString("\n")
		}
	}

	help.WriteString("\nExamples:\n")
	help.WriteString("  let x = 42;\n")
	help.WriteString("  console.log('Hello, World!');\n")
	help.WriteString("  function greet(name) { return 'Hello, ' + name; }\n")

	if e.config.EnableModules {
		help.WriteString("  const db = require('database');\n")
	}

	return help.String()
}

// Compile the interface implementation
var _ repl.Evaluator = (*Evaluator)(nil)
