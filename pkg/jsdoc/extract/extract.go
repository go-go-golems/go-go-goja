// Package extract parses JavaScript files using tree-sitter and extracts
// documentation metadata from __package__, __doc__, __example__, and doc`...`
// sentinel patterns.
package extract

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	tree_sitter "github.com/tree-sitter/go-tree-sitter"
	tree_sitter_javascript "github.com/tree-sitter/tree-sitter-javascript/bindings/go"

	"github.com/go-go-golems/go-go-goja/pkg/jsdoc/model"
)

// ParseFile parses a single JS file and returns its extracted documentation.
func ParseFile(path string) (*model.FileDoc, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.Wrapf(err, "reading %s", path)
	}
	return ParseSource(path, src)
}

// ParseFSFile parses a single JS file from the provided filesystem and returns
// its extracted documentation.
func ParseFSFile(fsys fs.FS, path string) (*model.FileDoc, error) {
	if fsys == nil {
		return nil, errors.New("filesystem is nil")
	}
	if path == "" {
		return nil, ErrEmptyPath
	}

	src, err := fs.ReadFile(fsys, path)
	if err != nil {
		return nil, errors.Wrapf(err, "reading %s", path)
	}
	return ParseSource(path, src)
}

// ParseSource parses a single JS source buffer and returns its extracted documentation.
func ParseSource(path string, src []byte) (*model.FileDoc, error) {
	parser := tree_sitter.NewParser()
	lang := tree_sitter.NewLanguage(tree_sitter_javascript.Language())
	if err := parser.SetLanguage(lang); err != nil {
		parser.Close()
		return nil, errors.Wrapf(err, "setting javascript language")
	}
	defer parser.Close()

	tree := parser.Parse(src, nil)
	if tree == nil {
		return nil, errors.Errorf("parsing %s: got nil tree", path)
	}
	defer tree.Close()

	e := &extractor{src: src, path: path}
	return e.extract(tree.RootNode()), nil
}

// ParseDir parses all .js files in a directory (non-recursive).
//
// Parity note: this matches jsdocex's current behavior; the watcher is recursive,
// but initial parsing is not.
func ParseDir(dir string) ([]*model.FileDoc, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, errors.Wrapf(err, "reading dir %s", dir)
	}

	var docs []*model.FileDoc
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".js") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		fd, err := ParseFile(path)
		if err != nil {
			// Parity note: log but continue.
			fmt.Fprintf(os.Stderr, "warning: %v\n", err)
			continue
		}
		docs = append(docs, fd)
	}
	return docs, nil
}

// extractor holds parsing state for a single file.
type extractor struct {
	src  []byte
	path string
}

func (e *extractor) extract(root *tree_sitter.Node) *model.FileDoc {
	fd := &model.FileDoc{FilePath: e.path}

	// Walk top-level statements.
	count := int(root.ChildCount())
	for i := 0; i < count; i++ {
		child := root.Child(uint(i))
		e.processNode(child, fd)
	}

	return fd
}

// processNode inspects a single AST node and dispatches to the appropriate handler.
// It recurses into class bodies, export statements, and other containers.
func (e *extractor) processNode(node *tree_sitter.Node, fd *model.FileDoc) {
	if node == nil {
		return
	}

	nodeKind := node.Kind()

	// Expression statements wrap call_expression.
	if nodeKind == "expression_statement" {
		inner := node.Child(0)
		if inner == nil {
			return
		}
		e.processNode(inner, fd)
		return
	}

	switch nodeKind {
	case "call_expression":
		e.handleCallExpression(node, fd)

	// Recurse into class bodies to find __doc__ calls on methods.
	case "class_declaration", "class":
		e.recurseChildren(node, fd)
	case "class_body":
		e.recurseChildren(node, fd)
	case "method_definition":
		e.recurseChildren(node, fd)

	// Recurse into export statements.
	case "export_statement":
		e.recurseChildren(node, fd)

	// Recurse into lexical/variable declarations (const fn = ...).
	case "lexical_declaration", "variable_declaration":
		e.recurseChildren(node, fd)
	}
}

// recurseChildren walks all children of a node through processNode.
func (e *extractor) recurseChildren(node *tree_sitter.Node, fd *model.FileDoc) {
	for i := 0; i < int(node.ChildCount()); i++ {
		e.processNode(node.Child(uint(i)), fd)
	}
}

// handleCallExpression processes __package__, __doc__, and __example__ calls.
func (e *extractor) handleCallExpression(node *tree_sitter.Node, fd *model.FileDoc) {
	fnNode := node.ChildByFieldName("function")
	if fnNode == nil {
		return
	}
	fnName := e.nodeText(fnNode)

	argsNode := node.ChildByFieldName("arguments")

	// Handle doc`...` — tree-sitter parses it as call_expression with template_string child.
	if fnName == "doc" {
		e.handleDocTemplate(node, fd)
		return
	}

	// If we can't see an arguments node, we can't parse sentinel call args.
	if argsNode == nil {
		return
	}

	switch fnName {
	case "__package__":
		pkg := e.parsePackage(argsNode)
		if pkg != nil {
			pkg.SourceFile = e.path
			fd.Package = pkg
		}

	case "__doc__":
		sym := e.parseSymbolDoc(argsNode)
		if sym != nil {
			sym.SourceFile = e.path
			sym.Line = int(node.StartPosition().Row) + 1
			fd.Symbols = append(fd.Symbols, sym)
		}

	case "__example__":
		ex := e.parseExample(argsNode)
		if ex != nil {
			ex.SourceFile = e.path
			ex.Line = int(node.StartPosition().Row) + 1
			fd.Examples = append(fd.Examples, ex)
		}
	}
}

// handleDocTemplate processes doc`...` calls.
//
// tree-sitter represents doc`...` as a call_expression where the template_string
// is a direct child of the call_expression (not inside an arguments node).
func (e *extractor) handleDocTemplate(node *tree_sitter.Node, fd *model.FileDoc) {
	// Find the template_string as a direct child of the call_expression.
	var tmplNode *tree_sitter.Node
	for i := 0; i < int(node.ChildCount()); i++ {
		c := node.Child(uint(i))
		if c.Kind() == "template_string" {
			tmplNode = c
			break
		}
	}
	if tmplNode == nil {
		return
	}

	raw := e.templateRawText(tmplNode)
	frontmatter, prose := splitFrontmatter(raw)

	// Parse the frontmatter to find which symbol or package this belongs to.
	fm := parseFrontmatter(frontmatter)

	symbolName := fm["symbol"]
	packageName := fm["package"]

	if symbolName != "" {
		// Attach prose to the most recently added symbol with this name.
		for i := len(fd.Symbols) - 1; i >= 0; i-- {
			if fd.Symbols[i].Name == symbolName {
				fd.Symbols[i].Prose = strings.TrimSpace(prose)
				return
			}
		}
		// If not found yet, create a stub — the __doc__ may come after.
		fd.Symbols = append(fd.Symbols, &model.SymbolDoc{
			Name:       symbolName,
			Prose:      strings.TrimSpace(prose),
			SourceFile: e.path,
		})
		return
	}

	if packageName != "" {
		if fd.Package != nil {
			fd.Package.Prose = strings.TrimSpace(prose)
		}
		return
	}

	// Unattributed prose — attach to the last symbol if any.
	if len(fd.Symbols) > 0 {
		last := fd.Symbols[len(fd.Symbols)-1]
		if last.Prose == "" {
			last.Prose = strings.TrimSpace(prose)
		}
	}
}

// ---- JSON object parsing helpers ----

// parsePackage extracts __package__({...}) metadata.
func (e *extractor) parsePackage(argsNode *tree_sitter.Node) *model.Package {
	obj := e.firstObjectArg(argsNode)
	if obj == nil {
		return nil
	}
	raw := e.nodeText(obj)
	data := jsObjectToJSON(raw)

	var pkg model.Package
	if err := json.Unmarshal([]byte(data), &pkg); err != nil {
		return &model.Package{Name: extractStringField(data, "name")}
	}
	return &pkg
}

// parseSymbolDoc extracts __doc__("name", {...}) metadata.
func (e *extractor) parseSymbolDoc(argsNode *tree_sitter.Node) *model.SymbolDoc {
	// First arg: string name; second arg: object.
	args := e.collectArgs(argsNode)
	if len(args) == 0 {
		return nil
	}

	var name string
	var objText string

	if len(args) == 1 {
		// __doc__({name: "...", ...}) — name inside object.
		objText = e.nodeText(args[0])
		name = extractStringField(jsObjectToJSON(objText), "name")
	} else {
		// __doc__("name", {...})
		name = strings.Trim(e.nodeText(args[0]), `"'`+"`")
		objText = e.nodeText(args[1])
	}

	data := jsObjectToJSON(objText)
	var sym model.SymbolDoc
	if err := json.Unmarshal([]byte(data), &sym); err != nil {
		sym = model.SymbolDoc{}
	}
	if name != "" {
		sym.Name = name
	}
	return &sym
}

// parseExample extracts __example__({...}) metadata.
func (e *extractor) parseExample(argsNode *tree_sitter.Node) *model.Example {
	obj := e.firstObjectArg(argsNode)
	if obj == nil {
		return nil
	}
	raw := e.nodeText(obj)
	data := jsObjectToJSON(raw)

	var ex model.Example
	if err := json.Unmarshal([]byte(data), &ex); err != nil {
		ex = model.Example{ID: extractStringField(data, "id")}
	}
	return &ex
}

// ---- AST traversal helpers ----

func (e *extractor) firstObjectArg(argsNode *tree_sitter.Node) *tree_sitter.Node {
	for i := 0; i < int(argsNode.ChildCount()); i++ {
		c := argsNode.Child(uint(i))
		if c.Kind() == "object" {
			return c
		}
	}
	return nil
}

func (e *extractor) collectArgs(argsNode *tree_sitter.Node) []*tree_sitter.Node {
	var args []*tree_sitter.Node
	for i := 0; i < int(argsNode.ChildCount()); i++ {
		c := argsNode.Child(uint(i))
		k := c.Kind()
		if k != "," && k != "(" && k != ")" && k != "comment" {
			args = append(args, c)
		}
	}
	return args
}

func (e *extractor) nodeText(node *tree_sitter.Node) string {
	if node == nil {
		return ""
	}
	start := int(node.StartByte())
	end := int(node.EndByte())
	if start < 0 || end < start || end > len(e.src) {
		return ""
	}
	return string(e.src[start:end])
}

func (e *extractor) templateRawText(tmplNode *tree_sitter.Node) string {
	// template_string node — strip backticks.
	text := e.nodeText(tmplNode)
	text = strings.TrimPrefix(text, "`")
	text = strings.TrimSuffix(text, "`")
	return text
}

// ---- JS-to-JSON conversion ----

// jsObjectToJSON performs a best-effort conversion of a JS object literal
// to valid JSON so we can unmarshal it with encoding/json.
func jsObjectToJSON(js string) string {
	return convertJSToJSON(js)
}

func convertJSToJSON(input string) string {
	var sb strings.Builder
	i := 0
	n := len(input)

	for i < n {
		ch := input[i]

		switch {
		case ch == '/' && i+1 < n && input[i+1] == '/':
			// Line comment — skip to end of line.
			for i < n && input[i] != '\n' {
				i++
			}

		case ch == '/' && i+1 < n && input[i+1] == '*':
			// Block comment — skip.
			i += 2
			for i+1 < n && (input[i] != '*' || input[i+1] != '/') {
				i++
			}
			i += 2

		case ch == '\'':
			// Single-quoted string → double-quoted.
			sb.WriteByte('"')
			i++
			for i < n && input[i] != '\'' {
				if input[i] == '"' {
					sb.WriteByte('\\')
				}
				if input[i] == '\\' && i+1 < n {
					if input[i+1] == '\'' {
						sb.WriteByte('\'')
						i += 2
						continue
					}
					sb.WriteByte(input[i])
					i++
				}
				sb.WriteByte(input[i])
				i++
			}
			sb.WriteByte('"')
			i++ // closing '

		case ch == '"':
			// Double-quoted string — copy as-is.
			sb.WriteByte(ch)
			i++
			for i < n && input[i] != '"' {
				if input[i] == '\\' && i+1 < n {
					sb.WriteByte(input[i])
					i++
				}
				sb.WriteByte(input[i])
				i++
			}
			sb.WriteByte('"')
			i++ // closing "

		case ch == '`':
			// Template literal — treat as string, no interpolation support.
			sb.WriteByte('"')
			i++
			for i < n && input[i] != '`' {
				if input[i] == '"' {
					sb.WriteByte('\\')
				}
				if input[i] == '\\' && i+1 < n {
					sb.WriteByte(input[i])
					i++
				}
				sb.WriteByte(input[i])
				i++
			}
			sb.WriteByte('"')
			i++ // closing `

		case isIdentStart(ch):
			// Possibly an unquoted key — collect identifier.
			start := i
			for i < n && isIdentPart(input[i]) {
				i++
			}
			word := input[start:i]
			// Check if followed by ':' (object key).
			j := i
			for j < n && (input[j] == ' ' || input[j] == '\t') {
				j++
			}
			if j < n && input[j] == ':' && word != "true" && word != "false" && word != "null" {
				sb.WriteByte('"')
				sb.WriteString(word)
				sb.WriteByte('"')
			} else {
				sb.WriteString(word)
			}

		case ch == ',':
			// Check for trailing comma before } or ].
			j := i + 1
			for j < n && (input[j] == ' ' || input[j] == '\t' || input[j] == '\n' || input[j] == '\r') {
				j++
			}
			if j < n && (input[j] == '}' || input[j] == ']') {
				// Skip trailing comma.
				i++
			} else {
				sb.WriteByte(ch)
				i++
			}

		default:
			sb.WriteByte(ch)
			i++
		}
	}

	return sb.String()
}

func isIdentStart(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_' || c == '$'
}

func isIdentPart(c byte) bool {
	return isIdentStart(c) || (c >= '0' && c <= '9')
}

// extractStringField extracts a simple string value from a JSON-like string.
func extractStringField(data, field string) string {
	key := `"` + field + `"`
	idx := strings.Index(data, key)
	if idx == -1 {
		return ""
	}
	rest := data[idx+len(key):]
	colon := strings.Index(rest, ":")
	if colon == -1 {
		return ""
	}
	rest = strings.TrimSpace(rest[colon+1:])
	if len(rest) == 0 {
		return ""
	}
	if rest[0] == '"' {
		end := strings.Index(rest[1:], `"`)
		if end == -1 {
			return ""
		}
		return rest[1 : end+1]
	}
	return ""
}

// ---- Front-matter parsing ----

// splitFrontmatter splits a template string into YAML frontmatter and prose body.
// Frontmatter is delimited by --- lines.
func splitFrontmatter(text string) (string, string) {
	text = strings.TrimSpace(text)
	if !strings.HasPrefix(text, "---") {
		return "", text
	}
	// Find closing ---.
	rest := text[3:]
	end := strings.Index(rest, "\n---")
	if end == -1 {
		return "", text
	}
	fm := strings.TrimSpace(rest[:end])
	body := strings.TrimSpace(rest[end+4:])
	return fm, body
}

// parseFrontmatter parses a simple key: value YAML-like frontmatter.
func parseFrontmatter(fm string) map[string]string {
	result := make(map[string]string)
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
		val := strings.TrimSpace(line[idx+1:])
		result[key] = val
	}
	return result
}
