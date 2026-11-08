package jsverbs

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

func ScanDir(root string, opts ...ScanOptions) (*Registry, error) {
	options := DefaultScanOptions()
	if len(opts) > 0 {
		options = opts[0]
	}
	if err := validateScanPatterns(options); err != nil {
		return nil, err
	}

	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve root: %w", err)
	}
	rootHandle, err := os.OpenRoot(absRoot)
	if err != nil {
		return nil, fmt.Errorf("open root %s: %w", absRoot, err)
	}
	defer func() {
		_ = rootHandle.Close()
	}()

	inputs := []sourceInput{}
	err = filepath.WalkDir(absRoot, func(filePath string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		name := d.Name()
		if d.IsDir() {
			if shouldSkipDir(name) {
				if filePath == absRoot {
					return nil
				}
				return filepath.SkipDir
			}
			return nil
		}
		relPath, err := filepath.Rel(absRoot, filePath)
		if err != nil {
			return fmt.Errorf("relpath %s: %w", filePath, err)
		}
		relPathSlash := filepath.ToSlash(relPath)
		matched, err := matchesScanPatterns(relPathSlash, options)
		if err != nil {
			return err
		}
		if !supportsExtension(relPathSlash, options.Extensions) || !matched {
			return nil
		}
		source, err := rootHandle.ReadFile(relPath)
		if err != nil {
			return fmt.Errorf("read %s: %w", filePath, err)
		}
		inputs = append(inputs, sourceInput{
			AbsPath:    filePath,
			RelPath:    relPathSlash,
			ModulePath: modulePathFromRelative(relPath),
			ResolveDir: filepath.Dir(filePath),
			Source:     source,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}

	return scanInputs(absRoot, inputs, options)
}

func ScanFS(fsys fs.FS, root string, opts ...ScanOptions) (*Registry, error) {
	options := DefaultScanOptions()
	if len(opts) > 0 {
		options = opts[0]
	}
	if err := validateScanPatterns(options); err != nil {
		return nil, err
	}
	root = strings.TrimSpace(root)
	if root == "" {
		root = "."
	}

	rootFS := fsys
	if root != "." {
		var err error
		rootFS, err = fs.Sub(fsys, root)
		if err != nil {
			return nil, fmt.Errorf("open fs root %s: %w", root, err)
		}
	}

	inputs := []sourceInput{}
	err := fs.WalkDir(fsys, root, func(filePath string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		name := d.Name()
		if d.IsDir() {
			if shouldSkipDir(name) {
				if filePath == root {
					return nil
				}
				return fs.SkipDir
			}
			return nil
		}
		relPath, err := filepath.Rel(root, filePath)
		if err != nil {
			return fmt.Errorf("relpath %s: %w", filePath, err)
		}
		relPathSlash := filepath.ToSlash(relPath)
		matched, err := matchesScanPatterns(relPathSlash, options)
		if err != nil {
			return err
		}
		if !supportsExtension(relPathSlash, options.Extensions) || !matched {
			return nil
		}
		source, err := fs.ReadFile(fsys, filePath)
		if err != nil {
			return fmt.Errorf("read %s: %w", filePath, err)
		}
		inputs = append(inputs, sourceInput{
			RelPath:    relPathSlash,
			ModulePath: modulePathFromRelative(relPath),
			RootFS:     rootFS,
			Source:     source,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}

	return scanInputs(root, inputs, options)
}

func ScanSource(filePath string, source string, opts ...ScanOptions) (*Registry, error) {
	return ScanSources([]SourceFile{{Path: filePath, Source: []byte(source)}}, opts...)
}

func ScanSources(files []SourceFile, opts ...ScanOptions) (*Registry, error) {
	options := DefaultScanOptions()
	if len(opts) > 0 {
		options = opts[0]
	}
	if err := validateScanPatterns(options); err != nil {
		return nil, err
	}

	inputs := make([]sourceInput, 0, len(files))
	for _, file := range files {
		modulePath, err := normalizeModulePath(file.Path)
		if err != nil {
			return nil, err
		}
		inputs = append(inputs, sourceInput{
			AbsPath:        file.AbsPath,
			RelPath:        strings.TrimPrefix(modulePath, "/"),
			ModulePath:     modulePath,
			ResolveDir:     file.ResolveDir,
			RootFS:         file.RootFS,
			Source:         append([]byte(nil), file.Source...),
			OriginalSource: append([]byte(nil), file.OriginalSource...),
			SourceLanguage: file.Language,
		})
	}

	return scanInputs("", inputs, options)
}

type sourceInput struct {
	AbsPath        string
	RelPath        string
	ModulePath     string
	ResolveDir     string
	RootFS         fs.FS
	Source         []byte
	OriginalSource []byte
	SourceLanguage string
}

func scanInputs(rootDir string, inputs []sourceInput, options ScanOptions) (*Registry, error) {
	registry := &Registry{
		RootDir:            rootDir,
		Files:              []*FileSpec{},
		Diagnostics:        []Diagnostic{},
		SharedSections:     map[string]*SectionSpec{},
		SharedSectionOrder: []string{},
		verbsByKey:         map[string]*VerbSpec{},
		filesByModule:      map[string]*FileSpec{},
		options:            options,
	}

	for _, input := range inputs {
		transformed, err := transformSourceInput(input, options)
		if err != nil {
			return registry, err
		}
		file, err := scanInput(transformed, options, registry)
		if err != nil {
			return registry, err
		}
		registry.Files = append(registry.Files, file)
		registry.filesByModule[file.ModulePath] = file
	}

	sort.Slice(registry.Files, func(i, j int) bool {
		return registry.Files[i].RelPath < registry.Files[j].RelPath
	})

	if err := registry.finalizeVerbs(); err != nil {
		return registry, err
	}

	if diagnostics := registry.ErrorDiagnostics(); len(diagnostics) > 0 && options.FailOnErrorDiagnostics {
		return registry, &ScanError{Diagnostics: diagnostics}
	}

	return registry, nil
}

func supportsExtension(filePath string, extensions []string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	for _, candidate := range extensions {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		if !strings.HasPrefix(candidate, ".") {
			candidate = "." + candidate
		}
		if strings.EqualFold(ext, candidate) {
			return true
		}
	}
	return false
}

func validateScanPatterns(options ScanOptions) error {
	if err := validateScanPatternSet("include", options.Include); err != nil {
		return err
	}
	return validateScanPatternSet("exclude", options.Exclude)
}

func validateScanPatternSet(kind string, patterns []string) error {
	for i, pattern := range patterns {
		pattern = normalizeScanPattern(pattern)
		if pattern == "" {
			continue
		}
		if !doublestar.ValidatePathPattern(pattern) {
			return fmt.Errorf("invalid jsverb %s glob at index %d: %q", kind, i, patterns[i])
		}
	}
	return nil
}

func matchesScanPatterns(relPath string, options ScanOptions) (bool, error) {
	relPath = filepath.ToSlash(strings.TrimPrefix(relPath, "./"))
	if len(options.Include) > 0 {
		matched, err := matchesAnyScanPattern(relPath, options.Include)
		if err != nil || !matched {
			return false, err
		}
	}
	excluded, err := matchesAnyScanPattern(relPath, options.Exclude)
	if err != nil || excluded {
		return false, err
	}
	return true, nil
}

func matchesAnyScanPattern(relPath string, patterns []string) (bool, error) {
	for _, pattern := range patterns {
		pattern = normalizeScanPattern(pattern)
		if pattern == "" {
			continue
		}
		matched, err := doublestar.PathMatch(pattern, relPath)
		if err != nil {
			return false, fmt.Errorf("match jsverb glob %q against %q: %w", pattern, relPath, err)
		}
		if matched {
			return true, nil
		}
	}
	return false, nil
}

func normalizeScanPattern(pattern string) string {
	pattern = filepath.ToSlash(strings.TrimSpace(pattern))
	return strings.TrimPrefix(pattern, "./")
}

func shouldSkipDir(name string) bool {
	return name == "node_modules" || strings.HasPrefix(name, ".")
}

func transformSourceInput(input sourceInput, options ScanOptions) (sourceInput, error) {
	if options.SourceTransform == nil {
		if len(input.OriginalSource) == 0 {
			input.OriginalSource = append([]byte(nil), input.Source...)
		}
		return input, nil
	}
	path := input.RelPath
	if input.AbsPath != "" {
		path = input.AbsPath
	}
	transformed, err := options.SourceTransform(SourceFile{
		Path:           path,
		AbsPath:        input.AbsPath,
		ResolveDir:     input.ResolveDir,
		RootFS:         input.RootFS,
		Source:         append([]byte(nil), input.Source...),
		OriginalSource: append([]byte(nil), input.OriginalSource...),
		Language:       input.SourceLanguage,
	})
	if err != nil {
		return input, err
	}
	if len(transformed.OriginalSource) == 0 {
		transformed.OriginalSource = append([]byte(nil), input.Source...)
	}
	input.Source = append([]byte(nil), transformed.Source...)
	input.OriginalSource = append([]byte(nil), transformed.OriginalSource...)
	input.SourceLanguage = transformed.Language
	if transformed.ResolveDir != "" {
		input.ResolveDir = transformed.ResolveDir
	}
	if transformed.RootFS != nil {
		input.RootFS = transformed.RootFS
	}
	return input, nil
}

func scanInput(input sourceInput, options ScanOptions, registry *Registry) (*FileSpec, error) {
	file := &FileSpec{
		AbsPath:        input.AbsPath,
		RelPath:        filepath.ToSlash(input.RelPath),
		ModulePath:     input.ModulePath,
		Source:         append([]byte(nil), input.Source...),
		OriginalSource: append([]byte(nil), input.OriginalSource...),
		SourceLanguage: input.SourceLanguage,
		ResolveDir:     input.ResolveDir,
		RootFS:         input.RootFS,
		Functions:      []*FunctionSpec{},
		functionByName: map[string]*FunctionSpec{},
		SectionOrder:   []string{},
		Sections:       map[string]*SectionSpec{},
		VerbMeta:       map[string]*VerbSpec{},
		Docs:           map[string]string{},
	}

	parser := options.Parser
	if parser == nil {
		parser = DefaultSourceParser()
	}
	parsed, err := parser.ParseSource(ParseInput{RelPath: file.RelPath, Source: input.Source})
	if err != nil {
		registry.Diagnostics = append(registry.Diagnostics, parseFailureDiagnostic(file.RelPath, input.Source, err))
		return file, nil
	}

	extractor := &extractor{
		relPath:  file.RelPath,
		file:     file,
		registry: registry,
	}
	for _, item := range parsed.Items {
		extractor.apply(item)
	}
	return file, nil
}

// parseFailureDiagnostic reports a source the parser could not read. A file
// that mentions a metadata sentinel was meant to declare verbs, so its failure
// is an error; any other file (a helper, a bundled asset) only warns.
func parseFailureDiagnostic(relPath string, source []byte, err error) Diagnostic {
	severity := DiagnosticSeverityWarning
	text := string(source)
	for _, sentinel := range []string{PackageSentinel, SectionSentinel, VerbSentinel} {
		if strings.Contains(text, sentinel) {
			severity = DiagnosticSeverityError
			break
		}
	}
	return Diagnostic{
		Severity: severity,
		Path:     relPath,
		Message: fmt.Sprintf("cannot parse source: %v (the default parser reads JavaScript as goja runs it: "+
			"no import/export; compile to CommonJS, or scan with pkg/jsverbs/treesitter)", err),
	}
}

func (r *Registry) finalizeVerbs() error {
	for _, file := range r.Files {
		for _, fn := range file.Functions {
			if doc, ok := file.Docs[fn.Name]; ok && strings.TrimSpace(fn.Doc) == "" {
				fn.Doc = strings.TrimSpace(doc)
			}
		}

		explicitNames := make([]string, 0, len(file.VerbMeta))
		for name := range file.VerbMeta {
			explicitNames = append(explicitNames, name)
		}
		sort.Strings(explicitNames)

		for _, functionName := range explicitNames {
			verb := file.VerbMeta[functionName]
			if err := r.finalizeVerb(file, verb); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *Registry) finalizeVerb(file *FileSpec, verb *VerbSpec) error {
	fn, ok := file.functionByName[verb.FunctionName]
	if !ok {
		return fmt.Errorf("%s references unknown function %q", file.RelPath, verb.FunctionName)
	}

	verb.File = file
	verb.Params = append([]ParameterSpec{}, fn.Params...)
	if verb.Fields == nil {
		verb.Fields = map[string]*FieldSpec{}
	}
	if verb.OutputMode == "" {
		verb.OutputMode = OutputModeGlaze
	}
	if verb.Name == "" {
		verb.Name = cleanCommandWord(fn.Name)
	}
	if verb.Name == "" {
		return fmt.Errorf("%s function %q resolved to empty command name", file.RelPath, fn.Name)
	}
	if len(verb.Parents) == 0 {
		verb.Parents = defaultParentsForFile(file)
	} else {
		verb.Parents = dedupeStrings(verb.Parents)
	}
	if strings.TrimSpace(verb.Long) == "" && strings.TrimSpace(fn.Doc) != "" {
		verb.Long = strings.TrimSpace(fn.Doc)
	}
	if strings.TrimSpace(verb.Short) == "" {
		verb.Short = fmt.Sprintf("Run %s from %s", fn.Name, file.RelPath)
	}
	verb.Tags = dedupeStrings(verb.Tags)
	verb.UseSections = dedupeStrings(verb.UseSections)

	fullPath := verb.FullPath()
	if _, exists := r.verbsByKey[fullPath]; exists {
		return fmt.Errorf("duplicate js verb path %q (%s)", fullPath, verb.SourceRef())
	}
	r.verbs = append(r.verbs, verb)
	r.verbsByKey[fullPath] = verb
	return nil
}

func defaultParentsForFile(file *FileSpec) []string {
	parents := append([]string{}, file.Package.Parents...)
	dir := path.Dir(file.RelPath)
	if dir != "." {
		for _, part := range strings.Split(dir, "/") {
			part = cleanCommandWord(part)
			if part != "" {
				parents = append(parents, part)
			}
		}
	}
	group := file.Package.Name
	if strings.TrimSpace(group) == "" {
		base := strings.TrimSuffix(path.Base(file.RelPath), path.Ext(file.RelPath))
		if base != "index" || len(parents) == 0 {
			group = base
		}
	}
	if group != "" {
		parents = append(parents, group)
	}
	return dedupeStrings(parents)
}

type extractor struct {
	relPath  string
	file     *FileSpec
	registry *Registry
}

func (e *extractor) apply(item ParsedItem) {
	switch item.Kind {
	case ParsedItemFunction:
		e.addFunction(item.Name, item.Params)
	case ParsedItemCall:
		switch item.Name {
		case PackageSentinel:
			e.handlePackage(item.Args)
		case SectionSentinel:
			e.handleSection(item.Args)
		case VerbSentinel:
			e.handleVerb(item.Args)
		}
	case ParsedItemDoc:
		e.handleDocTemplate(item.Text)
	}
}

func (e *extractor) addFunction(name string, params []ParameterSpec) {
	name = strings.TrimSpace(name)
	if name == "" {
		return
	}
	if _, ok := e.file.functionByName[name]; ok {
		return
	}
	fn := &FunctionSpec{Name: name, Params: append([]ParameterSpec{}, params...)}
	e.file.Functions = append(e.file.Functions, fn)
	e.file.functionByName[name] = fn
}

func (e *extractor) handlePackage(args []ParsedArg) {
	var objectArg *ParsedArg
	for i := range args {
		if args[i].Object {
			objectArg = &args[i]
			break
		}
	}
	if objectArg == nil {
		e.errorf("", "__package__ requires an object argument")
		return
	}
	if objectArg.Err != nil {
		e.errorf("", "invalid __package__ metadata: %v", objectArg.Err)
		return
	}
	data, ok := objectArg.Value.(map[string]interface{})
	if !ok {
		e.errorf("", "__package__ metadata must be an object")
		return
	}
	e.file.Package = PackageSpec{
		Name:    stringValue(data["name"]),
		Short:   stringValue(data["short"]),
		Long:    stringValue(data["long"]),
		Parents: stringSlice(data["parents"]),
		Tags:    stringSlice(data["tags"]),
	}
}

func (e *extractor) handleSection(args []ParsedArg) {
	name, objectArg := namedObjectArgs(args)
	if objectArg == nil {
		e.errorf(name, "__section__ requires a name and object metadata")
		return
	}
	if objectArg.Err != nil {
		e.errorf(name, "invalid __section__ metadata: %v", objectArg.Err)
		return
	}
	data, ok := objectArg.Value.(map[string]interface{})
	if !ok {
		e.errorf(name, "__section__ metadata must be an object")
		return
	}
	if name == "" {
		name = stringValue(data["name"])
	}
	slug := cleanCommandWord(name)
	if slug == "" {
		e.errorf(name, "__section__ resolved to empty slug")
		return
	}
	section := &SectionSpec{
		Slug:        slug,
		Title:       stringFirst(data["title"], data["name"]),
		Description: stringFirst(data["description"], data["help"]),
		Fields:      parseFieldMap(data),
	}
	if section.Title == "" {
		section.Title = slug
	}
	if _, ok := e.file.Sections[slug]; !ok {
		e.file.SectionOrder = append(e.file.SectionOrder, slug)
	}
	e.file.Sections[slug] = section
}

func (e *extractor) handleVerb(args []ParsedArg) {
	name, objectArg := namedObjectArgs(args)
	if objectArg == nil {
		e.errorf(name, "__verb__ requires a function name and object metadata")
		return
	}
	if objectArg.Err != nil {
		e.errorf(name, "invalid __verb__ metadata: %v", objectArg.Err)
		return
	}
	data, ok := objectArg.Value.(map[string]interface{})
	if !ok {
		e.errorf(name, "__verb__ metadata must be an object")
		return
	}
	if name == "" {
		name = stringFirst(data["function"], data["name"])
	}
	functionName := strings.TrimSpace(name)
	if functionName == "" {
		e.errorf(name, "__verb__ resolved to empty function name")
		return
	}

	verb := &VerbSpec{
		FunctionName: functionName,
		Name:         cleanCommandWord(stringValue(data["command"])),
		Short:        stringValue(data["short"]),
		Long:         stringValue(data["long"]),
		OutputMode:   normalizeOutputMode(stringFirst(data["output"], data["outputMode"], data["mode"])),
		Parents:      stringSlice(data["parents"]),
		Tags:         stringSlice(data["tags"]),
		UseSections:  stringSlice(firstNonNil(data["sections"], data["useSections"])),
		Fields:       parseFieldMap(data),
	}
	if verb.Name == "" {
		verb.Name = cleanCommandWord(stringValue(data["name"]))
	}
	e.file.VerbMeta[functionName] = verb
}

func (e *extractor) handleDocTemplate(raw string) {
	frontmatter, body := splitFrontmatter(raw)
	prose := strings.TrimSpace(body)
	if prose == "" {
		return
	}
	meta := parseFrontmatter(frontmatter)
	target := strings.TrimSpace(firstString(meta["verb"], meta["symbol"]))
	if target == "" {
		return
	}
	e.file.Docs[target] = prose
}

// namedObjectArgs reads the (name, object) or (object) argument forms of
// __section__ and __verb__. It returns a nil argument when neither form fits.
func namedObjectArgs(args []ParsedArg) (string, *ParsedArg) {
	switch len(args) {
	case 1:
		if args[0].Object {
			return "", &args[0]
		}
	case 2:
		if args[0].Err == nil {
			if name, ok := args[0].Value.(string); ok {
				return name, &args[1]
			}
		}
	}
	return "", nil
}

func parseFieldMap(data map[string]interface{}) map[string]*FieldSpec {
	rawFields, _ := firstMap(data["fields"], data["params"])
	if len(rawFields) == 0 {
		return map[string]*FieldSpec{}
	}
	ret := map[string]*FieldSpec{}
	for name, raw := range rawFields {
		fieldMap, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		field := &FieldSpec{
			Name:     name,
			Type:     stringValue(fieldMap["type"]),
			Help:     stringFirst(fieldMap["help"], fieldMap["description"]),
			Short:    stringValue(fieldMap["short"]),
			Bind:     stringValue(fieldMap["bind"]),
			Section:  cleanCommandWord(stringValue(fieldMap["section"])),
			Default:  fieldMap["default"],
			Choices:  stringSlice(fieldMap["choices"]),
			Required: boolValue(fieldMap["required"]),
			Argument: boolFirst(fieldMap["argument"], fieldMap["arg"]),
		}
		ret[name] = field
	}
	return ret
}

func splitFrontmatter(text string) (string, string) {
	text = strings.TrimSpace(text)
	if !strings.HasPrefix(text, "---") {
		return "", text
	}
	rest := text[3:]
	end := strings.Index(rest, "\n---")
	if end == -1 {
		return "", text
	}
	return strings.TrimSpace(rest[:end]), strings.TrimSpace(rest[end+4:])
}

func parseFrontmatter(fm string) map[string]string {
	ret := map[string]string{}
	for _, line := range strings.Split(fm, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.Index(line, ":")
		if idx == -1 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		value := strings.TrimSpace(line[idx+1:])
		ret[key] = value
	}
	return ret
}

func firstMap(values ...interface{}) (map[string]interface{}, bool) {
	for _, value := range values {
		if m, ok := value.(map[string]interface{}); ok {
			return m, true
		}
	}
	return nil, false
}

func firstNonNil(values ...interface{}) interface{} {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}

func firstString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func stringFirst(values ...interface{}) string {
	for _, value := range values {
		if s := stringValue(value); s != "" {
			return s
		}
	}
	return ""
}

func stringValue(value interface{}) string {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	default:
		return ""
	}
}

func stringSlice(value interface{}) []string {
	switch v := value.(type) {
	case []interface{}:
		ret := make([]string, 0, len(v))
		for _, item := range v {
			if s := stringValue(item); s != "" {
				ret = append(ret, s)
			}
		}
		return ret
	case []string:
		ret := make([]string, 0, len(v))
		for _, item := range v {
			if s := stringValue(item); s != "" {
				ret = append(ret, s)
			}
		}
		return ret
	default:
		return nil
	}
}

func boolValue(value interface{}) bool {
	b, _ := value.(bool)
	return b
}

func boolFirst(values ...interface{}) bool {
	for _, value := range values {
		if b, ok := value.(bool); ok {
			return b
		}
	}
	return false
}

func normalizeOutputMode(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "glaze", "table", "structured":
		return OutputModeGlaze
	case "text", "raw", "writer", "plain":
		return OutputModeText
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}

func modulePathFromRelative(relPath string) string {
	return "/" + strings.TrimPrefix(filepath.ToSlash(relPath), "/")
}

func normalizeModulePath(filePath string) (string, error) {
	cleaned := path.Clean("/" + strings.TrimSpace(filepath.ToSlash(filePath)))
	if cleaned == "/" || cleaned == "." {
		return "", fmt.Errorf("module path is empty")
	}
	return cleaned, nil
}

func (e *extractor) errorf(symbol, format string, args ...interface{}) {
	if e.registry == nil {
		return
	}
	e.registry.Diagnostics = append(e.registry.Diagnostics, Diagnostic{
		Severity: DiagnosticSeverityError,
		Path:     e.relPath,
		Symbol:   strings.TrimSpace(symbol),
		Message:  fmt.Sprintf(format, args...),
	})
}
