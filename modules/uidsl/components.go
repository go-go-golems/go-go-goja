package uidsl

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"github.com/dop251/goja"
)

var allowedBadgeTones = map[string]bool{
	"default": true,
	"info":    true,
	"success": true,
	"warning": true,
	"danger":  true,
	"muted":   true,
}

type codeBlockOptions struct {
	Title       string
	LineNumbers bool
	Wrap        bool
	Copy        bool
	MaxHeight   string
	Class       string
}

func codeBlockNode(language string, source any, opts map[string]any) Node {
	lang := cssToken(language)
	if lang == "empty" {
		lang = "text"
	}
	options := parseCodeBlockOptions(opts)
	classes := []any{"ui-codeblock", "ui-codeblock--" + lang}
	if options.Wrap {
		classes = append(classes, "ui-codeblock--wrap")
	} else {
		classes = append(classes, "ui-codeblock--nowrap")
	}
	if options.LineNumbers {
		classes = append(classes, "ui-codeblock--line-numbers")
	}
	if options.Class != "" {
		classes = append(classes, options.Class)
	}
	preAttrs := map[string]any{}
	if options.MaxHeight != "" {
		preAttrs["style"] = map[string]any{"max-height": options.MaxHeight, "overflow": "auto"}
	}
	sourceText := stringifySource(source)
	code := &Element{Tag: "code", Attrs: attrsFromMap(map[string]any{"class": "language-" + lang}), Children: highlightCode(lang, sourceText)}
	if options.Title == "" && !options.Copy {
		preAttrs["class"] = classes
		return &Element{Tag: "pre", Attrs: attrsFromMap(preAttrs), Children: []Node{code}}
	}
	captionChildren := []Node{}
	if options.Title != "" {
		captionChildren = append(captionChildren, &Element{Tag: "span", Attrs: attrsFromMap(map[string]any{"class": "ui-codeblock__title"}), Children: []Node{&Text{Value: options.Title}}})
	}
	if options.Copy {
		captionChildren = append(captionChildren, &Element{Tag: "button", Attrs: attrsFromMap(map[string]any{"class": "ui-codeblock__copy", "type": "button"}), Children: []Node{&Text{Value: "Copy"}}})
	}
	preAttrs["class"] = "ui-codeblock__pre"
	return &Element{Tag: "figure", Attrs: attrsFromMap(map[string]any{"class": classes}), Children: []Node{
		&Element{Tag: "figcaption", Attrs: attrsFromMap(map[string]any{"class": "ui-codeblock__caption"}), Children: captionChildren},
		&Element{Tag: "pre", Attrs: attrsFromMap(preAttrs), Children: []Node{code}},
	}}
}

func parseCodeBlockOptions(opts map[string]any) codeBlockOptions {
	return codeBlockOptions{
		Title:       stringFromAny(opts["title"]),
		LineNumbers: boolFromAny(opts["lineNumbers"]),
		Wrap:        optionBool(opts, "wrap", true),
		Copy:        boolFromAny(opts["copy"]),
		MaxHeight:   stringFromAny(opts["maxHeight"]),
		Class:       stringFromAny(opts["class"]),
	}
}

func badgeNode(value any, opts map[string]any) Node {
	text := stringifySource(value)
	tone := cssToken(stringFromAny(opts["tone"]))
	if !allowedBadgeTones[tone] {
		tone = "default"
	}
	classes := []any{"ui-badge", "ui-badge--" + tone, "ui-badge--value-" + cssToken(text)}
	if className := stringFromAny(opts["class"]); className != "" {
		classes = append(classes, className)
	}
	attrs := map[string]any{"class": classes}
	if title := stringFromAny(opts["title"]); title != "" {
		attrs["title"] = title
	}
	return &Element{Tag: "span", Attrs: attrsFromMap(attrs), Children: []Node{&Text{Value: text}}}
}

type tabSpec struct {
	ID       string
	Label    string
	Content  Node
	Disabled bool
}

func tabsNode(id string, value goja.Value, opts map[string]any) (Node, error) {
	baseID := domToken(id, "tabs")
	tabs, err := parseTabs(value)
	if err != nil {
		return nil, err
	}
	if len(tabs) == 0 {
		return &Element{Tag: "div", Attrs: attrsFromMap(map[string]any{"class": []any{"ui-tabs", stringFromAny(opts["class"])}, "id": baseID})}, nil
	}
	assignTabIDs(tabs)
	selected := selectedTabIndex(tabs, opts["selected"])
	classes := []any{"ui-tabs"}
	if className := stringFromAny(opts["class"]); className != "" {
		classes = append(classes, className)
	}
	children := []Node{tabsStyleNode(baseID, tabs)}
	children = append(children, tabInputNodes(baseID, tabs, selected)...)
	children = append(children, &Element{Tag: "div", Attrs: attrsFromMap(map[string]any{"class": "ui-tabs__tablist", "role": "tablist"}), Children: tabLabelNodes(baseID, tabs)})
	panels := make([]Node, 0, len(tabs))
	for i, tab := range tabs {
		panelClasses := []any{"ui-tabs__panel"}
		if i == selected {
			panelClasses = append(panelClasses, "ui-tabs__panel--active")
		}
		panels = append(panels, &Element{Tag: "section", Attrs: attrsFromMap(map[string]any{"class": panelClasses, "data-tab": tab.ID}), Children: []Node{tab.Content}})
	}
	children = append(children, &Element{Tag: "div", Attrs: attrsFromMap(map[string]any{"class": "ui-tabs__panels"}), Children: panels})
	return &Element{Tag: "div", Attrs: attrsFromMap(map[string]any{"class": classes, "id": baseID}), Children: children}, nil
}

func parseTabs(value goja.Value) ([]tabSpec, error) {
	if value == nil || goja.IsUndefined(value) || goja.IsNull(value) {
		return nil, nil
	}
	exported := value.Export()
	items, ok := exported.([]any)
	if !ok {
		return nil, fmt.Errorf("ui.tabs expects an array of tab specs, got %T", exported)
	}
	tabs := make([]tabSpec, 0, len(items))
	for i, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("ui.tabs tab %d must be an object, got %T", i, item)
		}
		content, err := NormalizeExport(m["content"])
		if err != nil {
			return nil, err
		}
		label := stringFromAny(m["label"])
		if label == "" {
			label = fmt.Sprintf("Tab %d", i+1)
		}
		tabs = append(tabs, tabSpec{ID: stringFromAny(m["id"]), Label: label, Content: content, Disabled: boolFromAny(m["disabled"])})
	}
	return tabs, nil
}

func assignTabIDs(tabs []tabSpec) {
	seen := map[string]int{}
	for i := range tabs {
		base := domToken(tabs[i].ID, "")
		if base == "" {
			base = domToken(tabs[i].Label, fmt.Sprintf("tab-%d", i+1))
		}
		seen[base]++
		if seen[base] > 1 {
			tabs[i].ID = fmt.Sprintf("%s-%d", base, seen[base])
		} else {
			tabs[i].ID = base
		}
	}
}

func selectedTabIndex(tabs []tabSpec, selected any) int {
	fallback := -1
	for i, tab := range tabs {
		if !tab.Disabled && fallback == -1 {
			fallback = i
		}
	}
	if fallback == -1 {
		return 0
	}
	if idx, ok := intLike(selected); ok && idx >= 0 && idx < len(tabs) && !tabs[idx].Disabled {
		return idx
	}
	selectedID := domToken(stringFromAny(selected), "")
	if selectedID != "" {
		for i, tab := range tabs {
			if tab.ID == selectedID && !tab.Disabled {
				return i
			}
		}
	}
	return fallback
}

func tabInputNodes(baseID string, tabs []tabSpec, selected int) []Node {
	children := make([]Node, 0, len(tabs))
	for i, tab := range tabs {
		inputID := baseID + "-" + tab.ID
		inputAttrs := map[string]any{"class": "ui-tabs__radio", "type": "radio", "name": baseID, "id": inputID}
		if i == selected && !tab.Disabled {
			inputAttrs["checked"] = true
		}
		if tab.Disabled {
			inputAttrs["disabled"] = true
		}
		children = append(children, &Element{Tag: "input", Attrs: attrsFromMap(inputAttrs)})
	}
	return children
}

func tabLabelNodes(baseID string, tabs []tabSpec) []Node {
	children := make([]Node, 0, len(tabs))
	for _, tab := range tabs {
		inputID := baseID + "-" + tab.ID
		labelClasses := []any{"ui-tabs__tab"}
		if tab.Disabled {
			labelClasses = append(labelClasses, "ui-tabs__tab--disabled")
		}
		children = append(children, &Element{Tag: "label", Attrs: attrsFromMap(map[string]any{"class": labelClasses, "for": inputID}), Children: []Node{&Text{Value: tab.Label}}})
	}
	return children
}

func tabsStyleNode(baseID string, tabs []tabSpec) Node {
	var b strings.Builder
	fmt.Fprintf(&b, "#%s>.ui-tabs__radio{position:absolute;left:-9999px;}\n", baseID)
	fmt.Fprintf(&b, "#%s>.ui-tabs__panels>.ui-tabs__panel{display:none;}\n", baseID)
	for _, tab := range tabs {
		inputID := baseID + "-" + tab.ID
		fmt.Fprintf(&b, "#%s:checked~.ui-tabs__panels>[data-tab=\"%s\"]{display:block;}\n", inputID, tab.ID)
		fmt.Fprintf(&b, "#%s:checked~.ui-tabs__tablist>label[for=\"%s\"]{background:#fff;font-weight:900;}\n", inputID, inputID)
	}
	return &Element{Tag: "style", Attrs: attrsFromMap(map[string]any{"class": "ui-tabs__style"}), Children: []Node{&RawHTML{Value: b.String()}}}
}

func jsonBlockSource(value goja.Value) string {
	if value == nil || goja.IsUndefined(value) || goja.IsNull(value) {
		return "null"
	}
	exported := value.Export()
	if s, ok := exported.(string); ok {
		var parsed any
		if err := json.Unmarshal([]byte(s), &parsed); err == nil {
			if b, err := json.MarshalIndent(parsed, "", "  "); err == nil {
				return string(b)
			}
		}
		return s
	}
	if b, err := json.MarshalIndent(exported, "", "  "); err == nil {
		return string(b)
	}
	return stringifySource(exported)
}

func highlightCode(language string, source string) []Node {
	switch language {
	case "sql":
		return highlightSQLLike(source, sqlKeywords(), true)
	case "javascript", "js":
		return highlightSQLLike(source, jsKeywords(), false)
	case "json":
		return highlightJSON(source)
	default:
		return []Node{&Text{Value: source}}
	}
}

func highlightSQLLike(source string, keywords map[string]bool, sql bool) []Node {
	nodes := []Node{}
	for i := 0; i < len(source); {
		if sql && i+1 < len(source) && source[i] == '-' && source[i+1] == '-' {
			j := i + 2
			for j < len(source) && source[j] != '\n' {
				j++
			}
			nodes = append(nodes, tokenNode("comment", source[i:j]))
			i = j
			continue
		}
		if !sql && i+1 < len(source) && source[i] == '/' && source[i+1] == '/' {
			j := i + 2
			for j < len(source) && source[j] != '\n' {
				j++
			}
			nodes = append(nodes, tokenNode("comment", source[i:j]))
			i = j
			continue
		}
		if i+1 < len(source) && source[i] == '/' && source[i+1] == '*' {
			j := i + 2
			for j+1 < len(source) && (source[j] != '*' || source[j+1] != '/') {
				j++
			}
			if j+1 < len(source) {
				j += 2
			}
			nodes = append(nodes, tokenNode("comment", source[i:j]))
			i = j
			continue
		}
		if source[i] == '\'' || source[i] == '"' || (!sql && source[i] == '`') {
			j := scanString(source, i)
			nodes = append(nodes, tokenNode("string", source[i:j]))
			i = j
			continue
		}
		if isDigit(source[i]) {
			j := i + 1
			for j < len(source) && (isDigit(source[j]) || source[j] == '.') {
				j++
			}
			nodes = append(nodes, tokenNode("number", source[i:j]))
			i = j
			continue
		}
		if isIdentStart(source[i]) {
			j := i + 1
			for j < len(source) && isIdentPart(source[j]) {
				j++
			}
			word := source[i:j]
			lower := strings.ToLower(word)
			if keywords[lower] {
				nodes = append(nodes, tokenNode("keyword", word))
			} else if !sql && (lower == "true" || lower == "false" || lower == "null" || lower == "undefined") {
				nodes = append(nodes, tokenNode("literal", word))
			} else {
				nodes = append(nodes, &Text{Value: word})
			}
			i = j
			continue
		}
		nodes = append(nodes, &Text{Value: source[i : i+1]})
		i++
	}
	return coalesceText(nodes)
}

func highlightJSON(source string) []Node {
	nodes := []Node{}
	for i := 0; i < len(source); {
		if source[i] == '"' {
			j := scanString(source, i)
			kind := "string"
			k := j
			for k < len(source) && unicode.IsSpace(rune(source[k])) {
				k++
			}
			if k < len(source) && source[k] == ':' {
				kind = "key"
			}
			nodes = append(nodes, tokenNode(kind, source[i:j]))
			i = j
			continue
		}
		if isDigit(source[i]) || source[i] == '-' {
			j := i + 1
			for j < len(source) && (isDigit(source[j]) || strings.ContainsRune(".eE+-", rune(source[j]))) {
				j++
			}
			nodes = append(nodes, tokenNode("number", source[i:j]))
			i = j
			continue
		}
		matched := false
		for _, literal := range []string{"true", "false", "null"} {
			if strings.HasPrefix(source[i:], literal) {
				nodes = append(nodes, tokenNode("literal", literal))
				i += len(literal)
				matched = true
				break
			}
		}
		if matched {
			continue
		}
		nodes = append(nodes, &Text{Value: source[i : i+1]})
		i++
	}
	return coalesceText(nodes)
}

func tokenNode(kind string, text string) Node {
	return &Element{Tag: "span", Attrs: attrsFromMap(map[string]any{"class": []any{"ui-codeblock__token", "ui-codeblock__token--" + kind}}), Children: []Node{&Text{Value: text}}}
}

func scanString(source string, i int) int {
	quote := source[i]
	j := i + 1
	for j < len(source) {
		if source[j] == '\\' {
			j += 2
			continue
		}
		if source[j] == quote {
			return j + 1
		}
		j++
	}
	return j
}

func coalesceText(nodes []Node) []Node {
	out := []Node{}
	for _, n := range nodes {
		if text, ok := n.(*Text); ok {
			if len(out) > 0 {
				if prev, ok := out[len(out)-1].(*Text); ok {
					prev.Value += text.Value
					continue
				}
			}
		}
		out = append(out, n)
	}
	return out
}

func sqlKeywords() map[string]bool {
	return keywordSet("select from where join left right inner outer on as create table view index trigger insert update delete values into primary key references not null integer text real blob autoincrement case when then else end exists and or order by group having limit offset union all distinct count sum min max coalesce")
}

func jsKeywords() map[string]bool {
	return keywordSet("const let var function return if else for while do switch case break continue try catch finally throw new class extends import export require async await in of typeof instanceof")
}

func keywordSet(words string) map[string]bool {
	ret := map[string]bool{}
	for _, word := range strings.Fields(words) {
		ret[word] = true
	}
	return ret
}

func isDigit(b byte) bool { return b >= '0' && b <= '9' }
func isIdentStart(b byte) bool {
	return (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z') || b == '_'
}
func isIdentPart(b byte) bool { return isIdentStart(b) || isDigit(b) || b == '$' }

func stringifySource(value any) string {
	if value == nil {
		return ""
	}
	return fmt.Sprint(value)
}

func firstOptions(options []map[string]any) map[string]any {
	if len(options) == 0 || options[0] == nil {
		return map[string]any{}
	}
	return options[0]
}

func optionBool(opts map[string]any, key string, fallback bool) bool {
	if _, ok := opts[key]; !ok {
		return fallback
	}
	return boolFromAny(opts[key])
}

func domToken(value string, fallback string) string {
	token := cssToken(value)
	if token == "empty" {
		return fallback
	}
	if token[0] >= '0' && token[0] <= '9' {
		return "x-" + token
	}
	return token
}

func intLike(value any) (int, bool) {
	switch v := value.(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case float64:
		return int(v), true
	case string:
		n, err := strconv.Atoi(v)
		return n, err == nil
	default:
		return 0, false
	}
}
