package jsverbs

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"
	tree_sitter_javascript "github.com/tree-sitter/tree-sitter-javascript/bindings/go"
)

func ScanDir(root string, opts ...ScanOptions) (*Registry, error) {
	options := DefaultScanOptions()
	if len(opts) > 0 {
		options = opts[0]
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
		if !supportsExtension(filePath, options.Extensions) {
			return nil
		}
		relPath, err := filepath.Rel(absRoot, filePath)
		if err != nil {
			return fmt.Errorf("relpath %s: %w", filePath, err)
		}
		source, err := rootHandle.ReadFile(relPath)
		if err != nil {
			return fmt.Errorf("read %s: %w", filePath, err)
		}
		inputs = append(inputs, sourceInput{
			AbsPath:    filePath,
			RelPath:    filepath.ToSlash(relPath),
			ModulePath: modulePathFromRelative(relPath),
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
	root = strings.TrimSpace(root)
	if root == "" {
		root = "."
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
		if !supportsExtension(filePath, options.Extensions) {
			return nil
		}
		source, err := fs.ReadFile(fsys, filePath)
		if err != nil {
			return fmt.Errorf("read %s: %w", filePath, err)
		}
		relPath, err := filepath.Rel(root, filePath)
		if err != nil {
			return fmt.Errorf("relpath %s: %w", filePath, err)
		}
		inputs = append(inputs, sourceInput{
			RelPath:    filepath.ToSlash(relPath),
			ModulePath: modulePathFromRelative(relPath),
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

	inputs := make([]sourceInput, 0, len(files))
	for _, file := range files {
		modulePath, err := normalizeModulePath(file.Path)
		if err != nil {
			return nil, err
		}
		inputs = append(inputs, sourceInput{
			RelPath:    strings.TrimPrefix(modulePath, "/"),
			ModulePath: modulePath,
			Source:     append([]byte(nil), file.Source...),
		})
	}

	return scanInputs("", inputs, options)
}

type sourceInput struct {
	AbsPath    string
	RelPath    string
	ModulePath string
	Source     []byte
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
		file, err := scanInput(input, options, registry)
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
		if strings.EqualFold(ext, candidate) {
			return true
		}
	}
	return false
}

func shouldSkipDir(name string) bool {
	return name == "node_modules" || strings.HasPrefix(name, ".")
}

func scanInput(input sourceInput, options ScanOptions, registry *Registry) (*FileSpec, error) {
	parser := tree_sitter.NewParser()
	defer parser.Close()
	lang := tree_sitter.NewLanguage(tree_sitter_javascript.Language())
	if err := parser.SetLanguage(lang); err != nil {
		return nil, fmt.Errorf("set javascript language: %w", err)
	}

	tree := parser.Parse(input.Source, nil)
	if tree == nil {
		return nil, fmt.Errorf("parse %s: nil tree", input.RelPath)
	}
	defer tree.Close()

	file := &FileSpec{
		AbsPath:        input.AbsPath,
		RelPath:        filepath.ToSlash(input.RelPath),
		ModulePath:     input.ModulePath,
		Source:         append([]byte(nil), input.Source...),
		Functions:      []*FunctionSpec{},
		functionByName: map[string]*FunctionSpec{},
		SectionOrder:   []string{},
		Sections:       map[string]*SectionSpec{},
		VerbMeta:       map[string]*VerbSpec{},
		Docs:           map[string]string{},
	}

	extractor := &extractor{
		src:      input.Source,
		relPath:  file.RelPath,
		options:  options,
		file:     file,
		registry: registry,
	}
	extractor.extract(tree.RootNode())
	return file, nil
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

		if !r.options.IncludePublicFunctions {
			continue
		}
		for _, fn := range file.Functions {
			if strings.HasPrefix(fn.Name, "_") {
				continue
			}
			if _, ok := file.VerbMeta[fn.Name]; ok {
				continue
			}
			verb := &VerbSpec{
				FunctionName: fn.Name,
				Fields:       map[string]*FieldSpec{},
				OutputMode:   OutputModeGlaze,
			}
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
	src      []byte
	relPath  string
	options  ScanOptions
	file     *FileSpec
	registry *Registry
}

func (e *extractor) extract(root *tree_sitter.Node) {
	for i := uint(0); i < root.ChildCount(); i++ {
		e.processTopLevel(root.Child(i))
	}
}

func (e *extractor) processTopLevel(node *tree_sitter.Node) {
	if node == nil {
		return
	}
	switch node.Kind() {
	case "expression_statement":
		e.processTopLevel(node.Child(0))
	case "call_expression":
		e.handleCallExpression(node)
	case "function_declaration":
		e.handleFunctionDeclaration(node)
	case "lexical_declaration", "variable_declaration", "export_statement":
		for i := uint(0); i < node.ChildCount(); i++ {
			e.processTopLevel(node.Child(i))
		}
	case "variable_declarator":
		e.handleVariableDeclarator(node)
	}
}

func (e *extractor) handleFunctionDeclaration(node *tree_sitter.Node) {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil || nameNode.Kind() != "identifier" {
		return
	}
	e.addFunction(e.nodeText(nameNode), extractParameters(node, e.nodeText))
}

func (e *extractor) handleVariableDeclarator(node *tree_sitter.Node) {
	nameNode := node.ChildByFieldName("name")
	valueNode := node.ChildByFieldName("value")
	if nameNode == nil || valueNode == nil || nameNode.Kind() != "identifier" {
		return
	}
	switch valueNode.Kind() {
	case "arrow_function", "function":
		e.addFunction(e.nodeText(nameNode), extractParameters(valueNode, e.nodeText))
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
	fn := &FunctionSpec{Name: name, Params: params}
	e.file.Functions = append(e.file.Functions, fn)
	e.file.functionByName[name] = fn
}

func (e *extractor) handleCallExpression(node *tree_sitter.Node) {
	fnNode := node.ChildByFieldName("function")
	if fnNode == nil {
		return
	}
	fnName := e.nodeText(fnNode)
	if fnName == "doc" {
		e.handleDocTemplate(node)
		return
	}
	argsNode := node.ChildByFieldName("arguments")
	if argsNode == nil {
		return
	}
	switch fnName {
	case "__package__":
		e.handlePackage(argsNode)
	case "__section__":
		e.handleSection(argsNode)
	case "__verb__":
		e.handleVerb(argsNode)
	}
}

func (e *extractor) handlePackage(argsNode *tree_sitter.Node) {
	objectArg := e.firstObjectArg(argsNode)
	if objectArg == nil {
		e.errorf("", "__package__ requires an object argument")
		return
	}
	value, err := e.parseLiteralNode(objectArg)
	if err != nil {
		e.errorf("", "invalid __package__ metadata: %v", err)
		return
	}
	data, ok := value.(map[string]interface{})
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

func (e *extractor) handleSection(argsNode *tree_sitter.Node) {
	name, objectNode := e.namedObjectArgs(argsNode)
	if objectNode == nil {
		e.errorf(name, "__section__ requires a name and object metadata")
		return
	}
	value, err := e.parseLiteralNode(objectNode)
	if err != nil {
		e.errorf(name, "invalid __section__ metadata: %v", err)
		return
	}
	data, ok := value.(map[string]interface{})
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

func (e *extractor) handleVerb(argsNode *tree_sitter.Node) {
	name, objectNode := e.namedObjectArgs(argsNode)
	if objectNode == nil {
		e.errorf(name, "__verb__ requires a function name and object metadata")
		return
	}
	value, err := e.parseLiteralNode(objectNode)
	if err != nil {
		e.errorf(name, "invalid __verb__ metadata: %v", err)
		return
	}
	data, ok := value.(map[string]interface{})
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

func (e *extractor) handleDocTemplate(node *tree_sitter.Node) {
	var templateNode *tree_sitter.Node
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child.Kind() == "template_string" {
			templateNode = child
			break
		}
	}
	if templateNode == nil {
		return
	}
	frontmatter, body := splitFrontmatter(e.templateRawText(templateNode))
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

func (e *extractor) namedObjectArgs(argsNode *tree_sitter.Node) (string, *tree_sitter.Node) {
	args := e.collectArgs(argsNode)
	switch len(args) {
	case 1:
		if args[0].Kind() == "object" {
			return "", args[0]
		}
	case 2:
		nameValue, err := e.parseLiteralNode(args[0])
		if err == nil {
			if name, ok := nameValue.(string); ok {
				return name, args[1]
			}
		}
	}
	return "", nil
}

func (e *extractor) collectArgs(argsNode *tree_sitter.Node) []*tree_sitter.Node {
	ret := []*tree_sitter.Node{}
	for i := uint(0); i < argsNode.ChildCount(); i++ {
		child := argsNode.Child(i)
		switch child.Kind() {
		case ",", "(", ")", "comment":
			continue
		default:
			ret = append(ret, child)
		}
	}
	return ret
}

func (e *extractor) firstObjectArg(argsNode *tree_sitter.Node) *tree_sitter.Node {
	for _, arg := range e.collectArgs(argsNode) {
		if arg.Kind() == "object" {
			return arg
		}
	}
	return nil
}

func (e *extractor) nodeText(node *tree_sitter.Node) string {
	if node == nil {
		return ""
	}
	start := node.StartByte()
	end := node.EndByte()
	srcLen := uint(len(e.src))
	if end < start || start > srcLen || end > srcLen {
		return ""
	}
	return string(e.src[start:end])
}

func (e *extractor) templateRawText(tmplNode *tree_sitter.Node) string {
	text := e.nodeText(tmplNode)
	text = strings.TrimPrefix(text, "`")
	text = strings.TrimSuffix(text, "`")
	return text
}

func (e *extractor) parseLiteralNode(node *tree_sitter.Node) (interface{}, error) {
	if node == nil {
		return nil, fmt.Errorf("literal node is nil")
	}
	switch node.Kind() {
	case "object":
		return e.parseObjectLiteral(node)
	case "array":
		return e.parseArrayLiteral(node)
	case "string":
		return decodeStringLiteral(e.nodeText(node))
	case "template_string":
		return e.parseTemplateLiteral(node)
	case "number":
		return strconv.ParseFloat(e.nodeText(node), 64)
	case "true":
		return true, nil
	case "false":
		return false, nil
	case "null":
		return nil, nil
	default:
		return nil, fmt.Errorf("unsupported metadata literal %q", node.Kind())
	}
}

func (e *extractor) parseObjectLiteral(node *tree_sitter.Node) (map[string]interface{}, error) {
	ret := map[string]interface{}{}
	for i := uint(0); i < node.NamedChildCount(); i++ {
		child := node.NamedChild(i)
		switch child.Kind() {
		case "pair":
			keyNode := child.ChildByFieldName("key")
			valueNode := child.ChildByFieldName("value")
			if keyNode == nil || valueNode == nil {
				return nil, fmt.Errorf("object pair missing key or value")
			}
			key, err := e.parseObjectKey(keyNode)
			if err != nil {
				return nil, err
			}
			value, err := e.parseLiteralNode(valueNode)
			if err != nil {
				return nil, fmt.Errorf("field %q: %w", key, err)
			}
			ret[key] = value
		case "comment":
			continue
		default:
			return nil, fmt.Errorf("unsupported object element %q", child.Kind())
		}
	}
	return ret, nil
}

func (e *extractor) parseArrayLiteral(node *tree_sitter.Node) ([]interface{}, error) {
	ret := []interface{}{}
	for i := uint(0); i < node.NamedChildCount(); i++ {
		child := node.NamedChild(i)
		if child.Kind() == "comment" {
			continue
		}
		value, err := e.parseLiteralNode(child)
		if err != nil {
			return nil, err
		}
		ret = append(ret, value)
	}
	return ret, nil
}

func (e *extractor) parseObjectKey(node *tree_sitter.Node) (string, error) {
	switch node.Kind() {
	case "property_identifier", "identifier":
		return e.nodeText(node), nil
	case "string", "template_string":
		value, err := e.parseLiteralNode(node)
		if err != nil {
			return "", err
		}
		s, ok := value.(string)
		if !ok {
			return "", fmt.Errorf("object key must resolve to string")
		}
		return s, nil
	default:
		return "", fmt.Errorf("unsupported object key %q", node.Kind())
	}
}

func (e *extractor) parseTemplateLiteral(node *tree_sitter.Node) (string, error) {
	for i := uint(0); i < node.NamedChildCount(); i++ {
		child := node.NamedChild(i)
		if child.Kind() != "string_fragment" {
			return "", fmt.Errorf("template metadata strings cannot contain substitutions")
		}
	}
	return e.templateRawText(node), nil
}

func extractParameters(node *tree_sitter.Node, textFn func(*tree_sitter.Node) string) []ParameterSpec {
	if node == nil {
		return nil
	}
	if paramsNode := node.ChildByFieldName("parameters"); paramsNode != nil {
		return collectParameterNodes(paramsNode, textFn)
	}
	if paramNode := node.ChildByFieldName("parameter"); paramNode != nil {
		return []ParameterSpec{parseParameterNode(paramNode, textFn)}
	}
	return nil
}

func collectParameterNodes(paramsNode *tree_sitter.Node, textFn func(*tree_sitter.Node) string) []ParameterSpec {
	params := []ParameterSpec{}
	for i := uint(0); i < paramsNode.ChildCount(); i++ {
		child := paramsNode.Child(i)
		switch child.Kind() {
		case ",", "(", ")", "comment":
			continue
		default:
			params = append(params, parseParameterNode(child, textFn))
		}
	}
	return params
}

func parseParameterNode(node *tree_sitter.Node, textFn func(*tree_sitter.Node) string) ParameterSpec {
	if node == nil {
		return ParameterSpec{}
	}
	switch node.Kind() {
	case "identifier":
		return ParameterSpec{Name: textFn(node), Kind: ParameterIdentifier}
	case "rest_pattern":
		argument := node.ChildByFieldName("argument")
		if argument == nil && node.NamedChildCount() > 0 {
			argument = node.NamedChild(0)
		}
		param := parseParameterNode(argument, textFn)
		param.Rest = true
		return param
	case "assignment_pattern":
		left := node.ChildByFieldName("left")
		if left == nil && node.NamedChildCount() > 0 {
			left = node.NamedChild(0)
		}
		return parseParameterNode(left, textFn)
	case "object_pattern":
		return ParameterSpec{Name: textFn(node), Kind: ParameterObject}
	case "array_pattern":
		return ParameterSpec{Name: textFn(node), Kind: ParameterArray}
	default:
		return ParameterSpec{Name: textFn(node), Kind: ParameterUnknown}
	}
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

func decodeStringLiteral(raw string) (string, error) {
	if len(raw) < 2 {
		return "", fmt.Errorf("invalid string literal")
	}
	switch raw[0] {
	case '"':
		value, err := strconv.Unquote(raw)
		if err != nil {
			return "", fmt.Errorf("decode string literal: %w", err)
		}
		return value, nil
	case '\'':
		return decodeSingleQuotedLiteral(raw)
	default:
		return "", fmt.Errorf("unsupported string literal %q", raw)
	}
}

func decodeSingleQuotedLiteral(raw string) (string, error) {
	if len(raw) < 2 || raw[0] != '\'' || raw[len(raw)-1] != '\'' {
		return "", fmt.Errorf("invalid single-quoted literal")
	}
	var sb strings.Builder
	for i := 1; i < len(raw)-1; i++ {
		ch := raw[i]
		if ch != '\\' {
			sb.WriteByte(ch)
			continue
		}
		i++
		if i >= len(raw)-1 {
			return "", fmt.Errorf("unterminated escape sequence")
		}
		switch raw[i] {
		case '\\', '\'', '"', '`':
			sb.WriteByte(raw[i])
		case 'n':
			sb.WriteByte('\n')
		case 'r':
			sb.WriteByte('\r')
		case 't':
			sb.WriteByte('\t')
		default:
			return "", fmt.Errorf("unsupported escape sequence \\%c", raw[i])
		}
	}
	return sb.String(), nil
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
