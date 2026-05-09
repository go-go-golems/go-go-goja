package uidsl

import (
	"fmt"
	"net/url"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/dop251/goja"
)

type TableBuilder struct {
	ID        string
	Rows      []map[string]any
	features  TableFeatures
	dataFn    goja.Callable
	columnsFn goja.Callable
}

type TableFeatures struct {
	Pagination   bool
	Sorting      bool
	ColumnPicker bool
	Filters      bool
	PageSize     int
}

type ColumnSpec struct {
	Name       string
	Label      string
	Kind       string
	Sortable   bool
	Filterable bool
	Align      string
	LinkHref   string
	LinkFn     goja.Callable
}

type RenderContext struct {
	Query  map[string]any    `json:"query"`
	Page   map[string]any    `json:"page"`
	Order  map[string]any    `json:"order"`
	State  map[string]any    `json:"state"`
	Filter map[string]any    `json:"filter"`
	Params map[string]string `json:"params"`
}

func newTableBuilder(id string) *TableBuilder {
	return &TableBuilder{ID: id, Rows: []map[string]any{}}
}

func tableFromRows(id string, value goja.Value) (*TableBuilder, error) {
	rows, err := rowsFromValue(value)
	if err != nil {
		return nil, err
	}
	return &TableBuilder{ID: id, Rows: rows}, nil
}

func tableBuilderObject(vm *goja.Runtime, builder *TableBuilder) goja.Value {
	obj := vm.NewObject()
	_ = obj.Set("features", func(callback goja.Value) goja.Value {
		if fn, ok := goja.AssertFunction(callback); ok {
			_, _ = fn(goja.Undefined(), featureBuilderObject(vm, &builder.features))
		}
		return obj
	})
	_ = obj.Set("data", func(callback goja.Value) goja.Value {
		if fn, ok := goja.AssertFunction(callback); ok {
			builder.dataFn = fn
		}
		return obj
	})
	_ = obj.Set("columns", func(callback goja.Value) goja.Value {
		if fn, ok := goja.AssertFunction(callback); ok {
			builder.columnsFn = fn
		}
		return obj
	})
	_ = obj.Set("render", func(input map[string]any) (Node, error) {
		return builder.render(vm, input)
	})
	return obj
}

func (t *TableBuilder) render(vm *goja.Runtime, input map[string]any) (Node, error) {
	ctx := t.context(input)
	rows := append([]map[string]any(nil), t.Rows...)
	total := len(rows)
	if t.dataFn != nil {
		value, err := t.dataFn(goja.Undefined(), vm.ToValue(ctx))
		if err != nil {
			return nil, err
		}
		dataRows, dataTotal, err := dataResultFromValue(value)
		if err != nil {
			return nil, err
		}
		rows = dataRows
		total = dataTotal
	}
	columns, err := t.resolveColumns(vm, ctx, rows)
	if err != nil {
		return nil, err
	}
	if t.dataFn == nil {
		rows = filterRows(rows, columns, ctx.Filter)
		rows = sortRows(rows, columns, ctx.Order)
		total = len(rows)
		if t.features.Pagination {
			rows = paginateRows(rows, ctx)
		}
	}
	return t.node(vm, ctx, rows, total, columns), nil
}

func (t *TableBuilder) context(input map[string]any) RenderContext {
	query := mapFromAny(input["query"])
	params := stringMapFromAny(input["params"])
	pageSize := intFromAny(query["limit"], t.features.PageSize)
	if pageSize <= 0 {
		pageSize = 25
	}
	if pageSize > 500 {
		pageSize = 500
	}
	pageIndex := intFromAny(query["page"], 1)
	if pageIndex < 1 {
		pageIndex = 1
	}
	sortKey := stringFromAny(query["sort"])
	dir := strings.ToLower(stringFromAny(query["dir"]))
	if dir != "desc" {
		dir = "asc"
	}
	return RenderContext{
		Query:  query,
		Params: params,
		State:  map[string]any{},
		Filter: filterMapFromQuery(query),
		Page: map[string]any{
			"index":  pageIndex,
			"size":   pageSize,
			"limit":  pageSize,
			"offset": (pageIndex - 1) * pageSize,
		},
		Order: map[string]any{"key": sortKey, "dir": dir},
	}
}

func (t *TableBuilder) resolveColumns(vm *goja.Runtime, ctx RenderContext, rows []map[string]any) ([]ColumnSpec, error) {
	if t.columnsFn == nil {
		cols := tableColumns(rows)
		ret := make([]ColumnSpec, 0, len(cols))
		for _, col := range cols {
			ret = append(ret, ColumnSpec{Name: col, Label: col, Kind: "text", Filterable: true})
		}
		return ret, nil
	}
	cols := []ColumnSpec{}
	cb := columnBuilderObject(vm, &cols)
	value, err := t.columnsFn(goja.Undefined(), cb, vm.ToValue(ctx))
	if err != nil {
		return nil, err
	}
	if exported := value.Export(); exported != nil {
		if parsed := columnsFromExport(exported); len(parsed) > 0 {
			return parsed, nil
		}
	}
	return cols, nil
}

func (t *TableBuilder) node(vm *goja.Runtime, ctx RenderContext, rows []map[string]any, total int, columns []ColumnSpec) Node {
	headCells := make([]Node, 0, len(columns))
	for _, column := range columns {
		label := column.Label
		if label == "" {
			label = column.Name
		}
		var child Node = &Text{Value: label}
		if t.features.Sorting && column.Sortable {
			dir := "asc"
			if stringFromAny(ctx.Order["key"]) == column.Name && stringFromAny(ctx.Order["dir"]) == "asc" {
				dir = "desc"
			}
			child = &Element{Tag: "a", Attrs: map[string]any{"href": queryHref(ctx.Query, map[string]any{"sort": column.Name, "dir": dir, "page": 1})}, Children: []Node{&Text{Value: label}}}
		}
		headCells = append(headCells, &Element{Tag: "th", Attrs: map[string]any{"data-column": column.Name}, Children: []Node{child}})
	}
	bodyRows := make([]Node, 0, len(rows))
	for _, row := range rows {
		cells := make([]Node, 0, len(columns))
		for _, column := range columns {
			attrs := map[string]any{"data-column": column.Name}
			if column.Align != "" {
				attrs["class"] = "align-" + column.Align
			}
			cells = append(cells, &Element{Tag: "td", Attrs: attrs, Children: []Node{cellNode(vm, row, row[column.Name], column)}})
		}
		bodyRows = append(bodyRows, &Element{Tag: "tr", Children: cells})
	}
	if len(bodyRows) == 0 {
		bodyRows = append(bodyRows, &Element{Tag: "tr", Attrs: map[string]any{"class": "ui-table-empty-row"}, Children: []Node{
			&Element{Tag: "td", Attrs: map[string]any{"class": "ui-table-empty", "colspan": len(columns)}, Children: []Node{&Text{Value: "No rows match the current filters."}}},
		}})
	}
	attrs := map[string]any{"class": tableClass(t.features)}
	if t.ID != "" {
		attrs["id"] = t.ID
	}
	table := &Element{Tag: "table", Attrs: attrs, Children: []Node{
		&Element{Tag: "thead", Children: []Node{&Element{Tag: "tr", Children: headCells}}},
		&Element{Tag: "tbody", Children: bodyRows},
	}}
	children := []Node{}
	if t.features.Filters {
		children = append(children, filtersNode(ctx, columns))
	}
	children = append(children, table)
	if t.features.Pagination {
		children = append(children, paginationNode(ctx, total))
	}
	if len(children) == 1 {
		return children[0]
	}
	return &Fragment{Children: children}
}

func featureBuilderObject(vm *goja.Runtime, features *TableFeatures) goja.Value {
	obj := vm.NewObject()
	_ = obj.Set("pagination", func(options ...map[string]any) goja.Value {
		features.Pagination = true
		if len(options) > 0 {
			features.PageSize = intFromAny(options[0]["size"], features.PageSize)
		}
		return obj
	})
	_ = obj.Set("sorting", func(_ ...any) goja.Value { features.Sorting = true; return obj })
	_ = obj.Set("filters", func(_ ...any) goja.Value { features.Filters = true; return obj })
	_ = obj.Set("columnPicker", func(_ ...any) goja.Value { features.ColumnPicker = true; return obj })
	return obj
}

func columnBuilderObject(vm *goja.Runtime, columns *[]ColumnSpec) goja.Value {
	obj := vm.NewObject()
	add := func(kind string, name string) goja.Value {
		col := ColumnSpec{Name: name, Label: name, Kind: kind}
		*columns = append(*columns, col)
		return columnObject(vm, &(*columns)[len(*columns)-1], columns)
	}
	for _, kind := range []string{"text", "badge", "money", "date", "tags"} {
		kind := kind
		_ = obj.Set(kind, func(name string, _ ...map[string]any) goja.Value { return add(kind, name) })
	}
	return obj
}

func columnObject(vm *goja.Runtime, col *ColumnSpec, columns *[]ColumnSpec) goja.Value {
	obj := vm.NewObject()
	_ = obj.Set("label", func(label string) goja.Value { col.Label = label; return obj })
	_ = obj.Set("sortable", func(_ ...any) goja.Value { col.Sortable = true; return obj })
	_ = obj.Set("filterable", func(_ ...any) goja.Value { col.Filterable = true; return obj })
	_ = obj.Set("align", func(align string) goja.Value { col.Align = align; return obj })
	_ = obj.Set("mono", func(_ ...any) goja.Value { return obj })
	_ = obj.Set("truncate", func(_ ...any) goja.Value { return obj })
	_ = obj.Set("link", func(target goja.Value) goja.Value {
		if fn, ok := goja.AssertFunction(target); ok {
			col.LinkFn = fn
		} else if !goja.IsUndefined(target) && !goja.IsNull(target) {
			col.LinkHref = fmt.Sprint(target.Export())
		}
		return obj
	})
	for _, kind := range []string{"text", "badge", "money", "date", "tags"} {
		kind := kind
		_ = obj.Set(kind, func(name string, _ ...map[string]any) goja.Value {
			next := ColumnSpec{Name: name, Label: name, Kind: kind}
			*columns = append(*columns, next)
			return columnObject(vm, &(*columns)[len(*columns)-1], columns)
		})
	}
	return obj
}

func rowsFromValue(value goja.Value) ([]map[string]any, error) {
	if value == nil || goja.IsUndefined(value) || goja.IsNull(value) {
		return []map[string]any{}, nil
	}
	exported := value.Export()
	rows, ok := rowsFromExport(exported)
	if !ok {
		return nil, fmt.Errorf("ui.table.fromRows expects an array, got %T", exported)
	}
	return rows, nil
}

func rowsFromExport(exported any) ([]map[string]any, bool) {
	if exported == nil {
		return []map[string]any{}, true
	}
	if slice, ok := exported.([]any); ok {
		return rowsFromSlice(slice), true
	}
	rv := reflect.ValueOf(exported)
	if !rv.IsValid() || (rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array) {
		return nil, false
	}
	rows := make([]map[string]any, 0, rv.Len())
	for i := 0; i < rv.Len(); i++ {
		rows = append(rows, rowFromAny(rv.Index(i).Interface()))
	}
	return rows, true
}

func rowsFromSlice(slice []any) []map[string]any {
	rows := make([]map[string]any, 0, len(slice))
	for _, item := range slice {
		rows = append(rows, rowFromAny(item))
	}
	return rows
}

func rowFromAny(item any) map[string]any {
	if row, ok := item.(map[string]any); ok {
		return row
	}
	rv := reflect.ValueOf(item)
	if rv.IsValid() && rv.Kind() == reflect.Map && rv.Type().Key().Kind() == reflect.String {
		row := map[string]any{}
		iter := rv.MapRange()
		for iter.Next() {
			row[iter.Key().String()] = iter.Value().Interface()
		}
		return row
	}
	return map[string]any{"value": item}
}

func dataResultFromValue(value goja.Value) ([]map[string]any, int, error) {
	exported := value.Export()
	if rows, ok := rowsFromExport(exported); ok {
		return rows, len(rows), nil
	}
	m, ok := exported.(map[string]any)
	if !ok {
		return nil, 0, fmt.Errorf("table data callback must return rows array or {rows,total}, got %T", exported)
	}
	rows, _ := rowsFromExport(m["rows"])
	total := intFromAny(m["total"], len(rows))
	return rows, total, nil
}

func tableColumns(rows []map[string]any) []string {
	seen := map[string]struct{}{}
	columns := []string{}
	for _, row := range rows {
		keys := make([]string, 0, len(row))
		for key := range row {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			columns = append(columns, key)
		}
	}
	return columns
}

func columnsFromExport(exported any) []ColumnSpec {
	items, ok := exported.([]any)
	if !ok {
		return nil
	}
	ret := []ColumnSpec{}
	for _, item := range items {
		if m, ok := item.(map[string]any); ok {
			name := stringFromAny(m["name"])
			if name == "" {
				continue
			}
			ret = append(ret, ColumnSpec{Name: name, Label: stringFromAny(m["label"]), Kind: stringFromAny(m["kind"]), Sortable: boolFromAny(m["sortable"]), Filterable: boolFromAny(m["filterable"]), Align: stringFromAny(m["align"])})
		}
	}
	return ret
}

func tableClass(features TableFeatures) []any {
	classes := []any{"ui-table"}
	if features.Pagination {
		classes = append(classes, "ui-table--pagination")
	}
	if features.Sorting {
		classes = append(classes, "ui-table--sorting")
	}
	if features.ColumnPicker {
		classes = append(classes, "ui-table--column-picker")
	}
	if features.Filters {
		classes = append(classes, "ui-table--filters")
	}
	return classes
}

func paginationNode(ctx RenderContext, total int) Node {
	page := intFromAny(ctx.Page["index"], 1)
	limit := intFromAny(ctx.Page["limit"], 25)
	pages := 1
	if limit > 0 && total > 0 {
		pages = (total + limit - 1) / limit
	}
	children := []Node{&Text{Value: fmt.Sprintf("Page %d of %d (%d rows)", page, pages, total)}}
	if page > 1 {
		children = append(children, &Text{Value: " "}, &Element{Tag: "a", Attrs: map[string]any{"href": queryHref(ctx.Query, map[string]any{"page": page - 1})}, Children: []Node{&Text{Value: "Previous"}}})
	}
	if page < pages {
		children = append(children, &Text{Value: " "}, &Element{Tag: "a", Attrs: map[string]any{"href": queryHref(ctx.Query, map[string]any{"page": page + 1})}, Children: []Node{&Text{Value: "Next"}}})
	}
	return &Element{Tag: "nav", Attrs: map[string]any{"class": "ui-table-pagination"}, Children: children}
}

func cellNode(vm *goja.Runtime, row map[string]any, value any, col ColumnSpec) Node {
	text := formatCell(value, col)
	var child Node
	switch col.Kind {
	case "badge":
		child = &Element{Tag: "span", Attrs: map[string]any{"class": []any{"ui-badge", "ui-badge--" + cssToken(text)}}, Children: []Node{&Text{Value: text}}}
	case "tags":
		parts := splitTags(value)
		children := make([]Node, 0, len(parts))
		for i, part := range parts {
			if i > 0 {
				children = append(children, &Text{Value: " "})
			}
			children = append(children, &Element{Tag: "span", Attrs: map[string]any{"class": []any{"ui-tag", "ui-tag--" + cssToken(part)}}, Children: []Node{&Text{Value: part}}})
		}
		child = &Fragment{Children: children}
	default:
		child = &Text{Value: text}
	}
	if href := linkHref(vm, row, value, col); href != "" {
		return &Element{Tag: "a", Attrs: map[string]any{"href": href}, Children: []Node{child}}
	}
	return child
}

func linkHref(vm *goja.Runtime, row map[string]any, value any, col ColumnSpec) string {
	if col.LinkFn != nil {
		ret, err := col.LinkFn(goja.Undefined(), vm.ToValue(row), vm.ToValue(value))
		if err != nil || goja.IsUndefined(ret) || goja.IsNull(ret) {
			return ""
		}
		return fmt.Sprint(ret.Export())
	}
	if col.LinkHref == "" {
		return ""
	}
	href := col.LinkHref
	for key, value := range row {
		href = strings.ReplaceAll(href, "{"+key+"}", url.QueryEscape(fmt.Sprint(value)))
	}
	return href
}

func formatCell(value any, col ColumnSpec) string {
	if value == nil {
		return ""
	}
	if col.Kind == "money" {
		cents := intFromAny(value, 0)
		return fmt.Sprintf("$%d.%02d", cents/100, absInt(cents%100))
	}
	return fmt.Sprint(value)
}

func filtersNode(ctx RenderContext, columns []ColumnSpec) Node {
	children := []Node{}
	for _, key := range []string{"sort", "dir"} {
		if value := stringFromAny(ctx.Query[key]); value != "" {
			children = append(children, &Element{Tag: "input", Attrs: map[string]any{"type": "hidden", "name": key, "value": value}})
		}
	}
	children = append(children,
		&Element{Tag: "label", Children: []Node{&Text{Value: "Search "}, &Element{Tag: "input", Attrs: map[string]any{"type": "search", "name": "q", "value": stringFromAny(ctx.Filter["q"]), "placeholder": "all columns"}}}},
	)
	for _, column := range columns {
		if !column.Filterable {
			continue
		}
		label := column.Label
		if label == "" {
			label = column.Name
		}
		name := "filter." + column.Name
		children = append(children, &Element{Tag: "label", Children: []Node{&Text{Value: label + " "}, &Element{Tag: "input", Attrs: map[string]any{"type": "search", "name": name, "value": stringFromAny(ctx.Filter[column.Name])}}}})
	}
	children = append(children,
		&Element{Tag: "button", Attrs: map[string]any{"type": "submit"}, Children: []Node{&Text{Value: "Filter"}}},
		&Text{Value: " "},
		&Element{Tag: "a", Attrs: map[string]any{"href": "?"}, Children: []Node{&Text{Value: "Clear"}}},
	)
	return &Element{Tag: "form", Attrs: map[string]any{"class": "ui-table-filters", "method": "get"}, Children: children}
}

func filterRows(rows []map[string]any, columns []ColumnSpec, filter map[string]any) []map[string]any {
	if len(filter) == 0 {
		return rows
	}
	q := strings.ToLower(strings.TrimSpace(stringFromAny(filter["q"])))
	out := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		if q != "" {
			matched := false
			for _, column := range columns {
				if strings.Contains(strings.ToLower(fmt.Sprint(row[column.Name])), q) {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}
		ok := true
		for _, column := range columns {
			want := strings.ToLower(strings.TrimSpace(stringFromAny(filter[column.Name])))
			if want == "" {
				continue
			}
			if !strings.Contains(strings.ToLower(fmt.Sprint(row[column.Name])), want) {
				ok = false
				break
			}
		}
		if ok {
			out = append(out, row)
		}
	}
	return out
}

func sortRows(rows []map[string]any, columns []ColumnSpec, order map[string]any) []map[string]any {
	key := stringFromAny(order["key"])
	if key == "" {
		return rows
	}
	allowed := false
	for _, column := range columns {
		if column.Name == key && column.Sortable {
			allowed = true
			break
		}
	}
	if !allowed {
		return rows
	}
	desc := stringFromAny(order["dir"]) == "desc"
	out := append([]map[string]any(nil), rows...)
	sort.SliceStable(out, func(i, j int) bool {
		cmp := compareAny(out[i][key], out[j][key])
		if desc {
			return cmp > 0
		}
		return cmp < 0
	})
	return out
}

func paginateRows(rows []map[string]any, ctx RenderContext) []map[string]any {
	limit := intFromAny(ctx.Page["limit"], 25)
	offset := intFromAny(ctx.Page["offset"], 0)
	if limit <= 0 || offset >= len(rows) {
		if offset >= len(rows) {
			return []map[string]any{}
		}
		return rows
	}
	end := offset + limit
	if end > len(rows) {
		end = len(rows)
	}
	return rows[offset:end]
}

func compareAny(a, b any) int {
	af, aok := numeric(a)
	bf, bok := numeric(b)
	if aok && bok {
		switch {
		case af < bf:
			return -1
		case af > bf:
			return 1
		default:
			return 0
		}
	}
	as := strings.ToLower(fmt.Sprint(a))
	bs := strings.ToLower(fmt.Sprint(b))
	return strings.Compare(as, bs)
}

func numeric(v any) (float64, bool) {
	switch x := v.(type) {
	case int:
		return float64(x), true
	case int64:
		return float64(x), true
	case float64:
		return x, true
	case string:
		f, err := strconv.ParseFloat(x, 64)
		return f, err == nil
	default:
		return 0, false
	}
}

func queryHref(query map[string]any, updates map[string]any) string {
	values := url.Values{}
	for key, value := range query {
		if value == nil {
			continue
		}
		values.Set(key, fmt.Sprint(value))
	}
	for key, value := range updates {
		values.Set(key, fmt.Sprint(value))
	}
	encoded := values.Encode()
	if encoded == "" {
		return "?"
	}
	return "?" + encoded
}

func filterMapFromQuery(query map[string]any) map[string]any {
	ret := map[string]any{}
	if q := strings.TrimSpace(stringFromAny(query["q"])); q != "" {
		ret["q"] = q
	}
	for key, value := range query {
		if value == nil || strings.TrimSpace(stringFromAny(value)) == "" {
			continue
		}
		if strings.HasPrefix(key, "filter.") {
			ret[strings.TrimPrefix(key, "filter.")] = value
		}
		if strings.HasPrefix(key, "filter_") {
			ret[strings.TrimPrefix(key, "filter_")] = value
		}
	}
	return ret
}

func splitTags(value any) []string {
	if value == nil {
		return []string{}
	}
	switch v := value.(type) {
	case []any:
		parts := make([]string, 0, len(v))
		for _, item := range v {
			if s := strings.TrimSpace(fmt.Sprint(item)); s != "" {
				parts = append(parts, s)
			}
		}
		return parts
	case []string:
		return v
	default:
		text := fmt.Sprint(value)
		fields := strings.FieldsFunc(text, func(r rune) bool { return r == ',' || r == ';' })
		parts := []string{}
		for _, field := range fields {
			if s := strings.TrimSpace(field); s != "" {
				parts = append(parts, s)
			}
		}
		return parts
	}
}

func cssToken(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	ret := strings.Trim(b.String(), "-")
	if ret == "" {
		return "empty"
	}
	return ret
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func mapFromAny(value any) map[string]any {
	ret := map[string]any{}
	if m, ok := value.(map[string]any); ok {
		for k, v := range m {
			ret[k] = v
		}
	}
	return ret
}

func stringMapFromAny(value any) map[string]string {
	ret := map[string]string{}
	if m, ok := value.(map[string]any); ok {
		for k, v := range m {
			ret[k] = fmt.Sprint(v)
		}
	}
	return ret
}

func intFromAny(value any, fallback int) int {
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case string:
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func stringFromAny(value any) string {
	if value == nil {
		return ""
	}
	return fmt.Sprint(value)
}

func boolFromAny(value any) bool {
	switch v := value.(type) {
	case bool:
		return v
	case string:
		return v == "true"
	default:
		return false
	}
}
