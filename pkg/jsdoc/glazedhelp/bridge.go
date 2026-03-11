package glazedhelp

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	glazedhelp "github.com/go-go-golems/glazed/pkg/help"
	helpmodel "github.com/go-go-golems/glazed/pkg/help/model"
	jsdocmodel "github.com/go-go-golems/go-go-goja/pkg/jsdoc/model"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

type Options struct {
	CorpusSlug        string
	CorpusTitle       string
	SlugPrefix        string
	DefaultCommands   []string
	RootIsTopLevel    bool
	PackagesTopLevel  bool
	IncludeExamples   bool
	IncludeSourceInfo bool
}

type Result struct {
	Sections     []*helpmodel.Section
	RootSlug     string
	PackageSlugs map[string]string
	SymbolSlugs  map[string]string
	ExampleSlugs map[string]string
}

type Composer struct {
	store         *jsdocmodel.DocStore
	result        *Result
	slugToPackage map[string]*jsdocmodel.Package
	slugToSymbol  map[string]*jsdocmodel.SymbolDoc
	slugToExample map[string]*jsdocmodel.Example
}

func BuildSections(store *jsdocmodel.DocStore, opts Options) (*Result, error) {
	if store == nil {
		return nil, errors.New("store is nil")
	}

	opts = withDefaults(opts)

	result := &Result{
		RootSlug:     rootSlug(opts),
		PackageSlugs: map[string]string{},
		SymbolSlugs:  map[string]string{},
		ExampleSlugs: map[string]string{},
	}

	fileToPackage := buildFileToPackage(store)

	result.Sections = append(result.Sections, buildRootSection(store, result.RootSlug, opts))

	for _, name := range sortedKeys(store.ByPackage) {
		pkg := store.ByPackage[name]
		slug := packageSlug(opts, pkg.Name)
		result.PackageSlugs[pkg.Name] = slug
		result.Sections = append(result.Sections, buildPackageSection(pkg, slug, result.RootSlug, opts))
	}

	for _, name := range sortedKeys(store.BySymbol) {
		sym := store.BySymbol[name]
		slug := symbolSlug(opts, sym.Name)
		result.SymbolSlugs[sym.Name] = slug
		result.Sections = append(result.Sections, buildSymbolSection(sym, slug, inferPackageSlug(sym.SourceFile, fileToPackage, result.PackageSlugs), result.RootSlug, opts))
	}

	if opts.IncludeExamples {
		for _, id := range sortedKeys(store.ByExample) {
			ex := store.ByExample[id]
			slug := exampleSlug(opts, ex.ID)
			result.ExampleSlugs[ex.ID] = slug
			result.Sections = append(result.Sections, buildExampleSection(ex, slug, inferExamplePackageSlug(ex, fileToPackage, result), result.RootSlug, opts, result.SymbolSlugs))
		}
	}

	sort.SliceStable(result.Sections, func(i, j int) bool {
		if result.Sections[i].Order != result.Sections[j].Order {
			return result.Sections[i].Order < result.Sections[j].Order
		}
		return result.Sections[i].Slug < result.Sections[j].Slug
	})

	return result, nil
}

func LoadIntoHelpSystem(ctx context.Context, hs *glazedhelp.HelpSystem, result *Result) error {
	if hs == nil {
		return errors.New("help system is nil")
	}
	if result == nil {
		return errors.New("result is nil")
	}

	for _, section := range result.Sections {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := hs.Store.Upsert(ctx, section); err != nil {
			return errors.Wrapf(err, "upsert section %s", section.Slug)
		}
	}
	return nil
}

func BuildMarkdownFiles(result *Result) (map[string][]byte, error) {
	if result == nil {
		return nil, errors.New("result is nil")
	}

	out := map[string][]byte{}
	for _, section := range result.Sections {
		if section == nil {
			continue
		}

		meta := map[string]any{
			"Title":          section.Title,
			"Slug":           section.Slug,
			"Short":          section.Short,
			"SectionType":    section.SectionType.String(),
			"Topics":         section.Topics,
			"Commands":       section.Commands,
			"Flags":          section.Flags,
			"IsTopLevel":     section.IsTopLevel,
			"IsTemplate":     section.IsTemplate,
			"ShowPerDefault": section.ShowPerDefault,
			"Order":          section.Order,
		}
		if section.SubTitle != "" {
			meta["SubTitle"] = section.SubTitle
		}

		frontmatter, err := yaml.Marshal(meta)
		if err != nil {
			return nil, errors.Wrapf(err, "marshal frontmatter for %s", section.Slug)
		}

		var buf bytes.Buffer
		buf.WriteString("---\n")
		buf.Write(frontmatter)
		buf.WriteString("---\n\n")
		buf.WriteString(strings.TrimSpace(section.Content))
		buf.WriteString("\n")

		out[sectionMarkdownPath(section)] = buf.Bytes()
	}

	return out, nil
}

func WriteMarkdownFiles(dir string, files map[string][]byte) error {
	if dir == "" {
		return errors.New("dir is empty")
	}

	for path, content := range files {
		fullPath := filepath.Join(dir, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			return errors.Wrapf(err, "mkdir %s", filepath.Dir(fullPath))
		}
		if err := os.WriteFile(fullPath, content, 0o644); err != nil {
			return errors.Wrapf(err, "write %s", fullPath)
		}
	}
	return nil
}

func NewComposer(store *jsdocmodel.DocStore, result *Result) *Composer {
	c := &Composer{
		store:         store,
		result:        result,
		slugToPackage: map[string]*jsdocmodel.Package{},
		slugToSymbol:  map[string]*jsdocmodel.SymbolDoc{},
		slugToExample: map[string]*jsdocmodel.Example{},
	}

	if result != nil {
		for name, slug := range result.PackageSlugs {
			if store != nil {
				c.slugToPackage[slug] = store.ByPackage[name]
			}
		}
		for name, slug := range result.SymbolSlugs {
			if store != nil {
				c.slugToSymbol[slug] = store.BySymbol[name]
			}
		}
		for id, slug := range result.ExampleSlugs {
			if store != nil {
				c.slugToExample[slug] = store.ByExample[id]
			}
		}
	}

	return c
}

func (c *Composer) ComposePage(ctx context.Context, slug string, hs *glazedhelp.HelpSystem) (*glazedhelp.ComposedPage, error) {
	if c == nil || c.result == nil || hs == nil {
		return nil, glazedhelp.ErrPageNotComposed
	}

	if !c.knowsSlug(slug) {
		return nil, glazedhelp.ErrPageNotComposed
	}

	section, err := hs.GetSectionWithSlug(slug)
	if err != nil {
		return nil, err
	}

	parts := []glazedhelp.PagePart{
		markdownPart{
			kind:  "section-body",
			order: 10,
			body:  renderSectionBody(section),
		},
	}

	if metadata := c.renderMetadata(slug); metadata != "" {
		parts = append(parts, markdownPart{
			kind:  "jsdoc-metadata",
			order: 20,
			body:  metadata,
		})
	}

	if related := c.renderRelated(ctx, slug, hs); related != "" {
		parts = append(parts, markdownPart{
			kind:  "related-sections",
			order: 30,
			body:  related,
		})
	}

	return &glazedhelp.ComposedPage{
		Slug:  slug,
		Title: section.Title,
		Parts: parts,
	}, nil
}

type markdownPart struct {
	kind  string
	order int
	body  string
}

func (p markdownPart) Kind() string { return p.kind }
func (p markdownPart) Order() int   { return p.order }
func (p markdownPart) RenderMarkdown(_ context.Context) (string, error) {
	return p.body, nil
}

func withDefaults(opts Options) Options {
	if opts.CorpusSlug == "" {
		opts.CorpusSlug = "jsdoc"
	}
	if opts.CorpusTitle == "" {
		opts.CorpusTitle = "JavaScript API"
	}
	if !opts.RootIsTopLevel {
		opts.RootIsTopLevel = true
	}
	if !opts.IncludeExamples {
		opts.IncludeExamples = true
	}
	if !opts.IncludeSourceInfo {
		opts.IncludeSourceInfo = true
	}
	return opts
}

func rootSlug(opts Options) string {
	if opts.SlugPrefix == "" {
		return normalizeSlugPart(opts.CorpusSlug)
	}
	return strings.Trim(strings.TrimSuffix(opts.SlugPrefix, "/"), "/")
}

func packageSlug(opts Options, pkgName string) string {
	return joinSlug(rootSlug(opts), "package", pkgName)
}

func symbolSlug(opts Options, symbolName string) string {
	return joinSlug(rootSlug(opts), "symbol", symbolName)
}

func exampleSlug(opts Options, exampleID string) string {
	return joinSlug(rootSlug(opts), "example", exampleID)
}

func joinSlug(parts ...string) string {
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = normalizeSlugPart(part)
		if part == "" {
			continue
		}
		out = append(out, part)
	}
	return strings.Join(out, "/")
}

func normalizeSlugPart(part string) string {
	part = strings.TrimSpace(strings.ToLower(part))
	part = strings.ReplaceAll(part, " ", "-")
	part = strings.ReplaceAll(part, "_", "-")
	part = strings.ReplaceAll(part, ".", "-")
	var out strings.Builder
	lastDash := false
	for _, r := range part {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			out.WriteRune(r)
			lastDash = false
		case r == '/' || r == '-':
			if out.Len() == 0 || lastDash {
				continue
			}
			out.WriteRune(r)
			lastDash = true
		}
	}
	return strings.Trim(out.String(), "/-")
}

func buildRootSection(store *jsdocmodel.DocStore, slug string, opts Options) *helpmodel.Section {
	var packages []string
	for _, name := range sortedKeys(store.ByPackage) {
		packages = append(packages, fmt.Sprintf("- `%s`", name))
	}

	content := []string{
		fmt.Sprintf("Documentation extracted from JavaScript sources for **%s**.", opts.CorpusTitle),
		"## Summary",
		fmt.Sprintf("- Packages: %d", len(store.ByPackage)),
		fmt.Sprintf("- Symbols: %d", len(store.BySymbol)),
		fmt.Sprintf("- Examples: %d", len(store.ByExample)),
	}
	if len(packages) > 0 {
		content = append(content, "## Packages", strings.Join(packages, "\n"))
	}

	return &helpmodel.Section{
		Slug:           slug,
		SectionType:    helpmodel.SectionGeneralTopic,
		Title:          opts.CorpusTitle,
		Short:          "Browse documentation extracted from JavaScript sources.",
		Content:        strings.Join(content, "\n\n"),
		IsTopLevel:     opts.RootIsTopLevel,
		ShowPerDefault: true,
		Order:          10,
		Commands:       append([]string(nil), opts.DefaultCommands...),
	}
}

func buildPackageSection(pkg *jsdocmodel.Package, slug, rootSlug string, opts Options) *helpmodel.Section {
	title := pkg.Title
	if title == "" {
		title = pkg.Name
	}
	short := pkg.Description
	if short == "" {
		short = fmt.Sprintf("Package documentation for %s.", pkg.Name)
	}

	var sections []string
	if pkg.Description != "" {
		sections = append(sections, pkg.Description)
	}
	if pkg.Prose != "" {
		sections = append(sections, pkg.Prose)
	}

	var meta []string
	if pkg.Category != "" {
		meta = append(meta, fmt.Sprintf("- **Category:** %s", pkg.Category))
	}
	if pkg.Version != "" {
		meta = append(meta, fmt.Sprintf("- **Version:** %s", pkg.Version))
	}
	if pkg.Guide != "" {
		meta = append(meta, fmt.Sprintf("- **Guide:** `%s`", pkg.Guide))
	}
	if len(pkg.SeeAlso) > 0 {
		meta = append(meta, fmt.Sprintf("- **See also:** %s", strings.Join(wrapCode(pkg.SeeAlso), ", ")))
	}
	if opts.IncludeSourceInfo && pkg.SourceFile != "" {
		meta = append(meta, fmt.Sprintf("- **Source:** `%s`", pkg.SourceFile))
	}
	if len(meta) > 0 {
		sections = append(sections, "## Package Metadata\n\n"+strings.Join(meta, "\n"))
	}

	return &helpmodel.Section{
		Slug:           slug,
		SectionType:    helpmodel.SectionGeneralTopic,
		Title:          title,
		Short:          short,
		Content:        strings.Join(sections, "\n\n"),
		Topics:         []string{rootSlug},
		IsTopLevel:     opts.PackagesTopLevel,
		ShowPerDefault: true,
		Order:          100,
		Commands:       append([]string(nil), opts.DefaultCommands...),
	}
}

func buildSymbolSection(sym *jsdocmodel.SymbolDoc, slug, packageSlug, rootSlug string, opts Options) *helpmodel.Section {
	short := sym.Summary
	if short == "" {
		short = fmt.Sprintf("API documentation for %s.", sym.Name)
	}

	var sections []string
	if sym.Summary != "" {
		sections = append(sections, sym.Summary)
	}
	if sym.Prose != "" {
		sections = append(sections, sym.Prose)
	}
	if params := renderParams(sym.Params); params != "" {
		sections = append(sections, "## Parameters\n\n"+params)
	}
	if returns := renderReturns(sym.Returns); returns != "" {
		sections = append(sections, "## Returns\n\n"+returns)
	}
	if len(sym.Concepts) > 0 {
		sections = append(sections, "## Concepts\n\n"+strings.Join(bulletList(sym.Concepts), "\n"))
	}
	if len(sym.Tags) > 0 {
		sections = append(sections, "## Tags\n\n"+strings.Join(bulletList(sym.Tags), "\n"))
	}
	if len(sym.Related) > 0 {
		sections = append(sections, "## Related Symbols\n\n"+strings.Join(bulletList(wrapCode(sym.Related)), "\n"))
	}

	var meta []string
	if sym.DocPage != "" {
		meta = append(meta, fmt.Sprintf("- **Doc page:** `%s`", sym.DocPage))
	}
	if opts.IncludeSourceInfo && sym.SourceFile != "" {
		loc := sym.SourceFile
		if sym.Line > 0 {
			loc = fmt.Sprintf("%s:%d", sym.SourceFile, sym.Line)
		}
		meta = append(meta, fmt.Sprintf("- **Source:** `%s`", loc))
	}
	if len(meta) > 0 {
		sections = append(sections, "## Metadata\n\n"+strings.Join(meta, "\n"))
	}

	topics := []string{rootSlug}
	if packageSlug != "" {
		topics = append(topics, packageSlug)
	}
	topics = append(topics, sym.Concepts...)
	topics = append(topics, sym.Tags...)

	return &helpmodel.Section{
		Slug:           slug,
		SectionType:    helpmodel.SectionGeneralTopic,
		Title:          sym.Name,
		Short:          short,
		Content:        strings.Join(sections, "\n\n"),
		Topics:         uniqueStrings(topics),
		ShowPerDefault: true,
		Order:          200,
		Commands:       append([]string(nil), opts.DefaultCommands...),
	}
}

func buildExampleSection(ex *jsdocmodel.Example, slug, packageSlug, rootSlug string, opts Options, symbolSlugs map[string]string) *helpmodel.Section {
	title := ex.Title
	if title == "" {
		title = ex.ID
	}
	short := ex.Title
	if short == "" {
		short = fmt.Sprintf("Example documentation for %s.", ex.ID)
	}

	var sections []string
	if ex.Title != "" {
		sections = append(sections, ex.Title)
	}
	if len(ex.Symbols) > 0 {
		sections = append(sections, "## Symbols\n\n"+strings.Join(bulletList(wrapCode(ex.Symbols)), "\n"))
	}
	if len(ex.Concepts) > 0 {
		sections = append(sections, "## Concepts\n\n"+strings.Join(bulletList(ex.Concepts), "\n"))
	}
	if len(ex.Tags) > 0 {
		sections = append(sections, "## Tags\n\n"+strings.Join(bulletList(ex.Tags), "\n"))
	}
	if ex.Body != "" {
		sections = append(sections, "## Source\n\n```js\n"+strings.TrimSpace(ex.Body)+"\n```")
	}

	var meta []string
	if ex.DocPage != "" {
		meta = append(meta, fmt.Sprintf("- **Doc page:** `%s`", ex.DocPage))
	}
	if opts.IncludeSourceInfo && ex.SourceFile != "" {
		loc := ex.SourceFile
		if ex.Line > 0 {
			loc = fmt.Sprintf("%s:%d", ex.SourceFile, ex.Line)
		}
		meta = append(meta, fmt.Sprintf("- **Source:** `%s`", loc))
	}
	if len(meta) > 0 {
		sections = append(sections, "## Metadata\n\n"+strings.Join(meta, "\n"))
	}

	topics := []string{rootSlug}
	if packageSlug != "" {
		topics = append(topics, packageSlug)
	}
	topics = append(topics, ex.Concepts...)
	topics = append(topics, ex.Tags...)
	for _, symbol := range ex.Symbols {
		if symbolSlug, ok := symbolSlugs[symbol]; ok {
			topics = append(topics, symbolSlug)
		}
	}

	return &helpmodel.Section{
		Slug:           slug,
		SectionType:    helpmodel.SectionExample,
		Title:          title,
		Short:          short,
		Content:        strings.Join(sections, "\n\n"),
		Topics:         uniqueStrings(topics),
		ShowPerDefault: true,
		Order:          300,
		Commands:       append([]string(nil), opts.DefaultCommands...),
	}
}

func buildFileToPackage(store *jsdocmodel.DocStore) map[string]string {
	fileToPackage := map[string]string{}
	for _, fd := range store.Files {
		if fd != nil && fd.Package != nil {
			fileToPackage[fd.FilePath] = fd.Package.Name
		}
	}
	return fileToPackage
}

func inferPackageSlug(sourceFile string, fileToPackage map[string]string, packageSlugs map[string]string) string {
	if packageName, ok := fileToPackage[sourceFile]; ok {
		return packageSlugs[packageName]
	}
	return ""
}

func inferExamplePackageSlug(ex *jsdocmodel.Example, fileToPackage map[string]string, result *Result) string {
	if ex == nil || result == nil {
		return ""
	}
	if packageSlug := inferPackageSlug(ex.SourceFile, fileToPackage, result.PackageSlugs); packageSlug != "" {
		return packageSlug
	}
	for _, symbol := range ex.Symbols {
		if symbolSlug, ok := result.SymbolSlugs[symbol]; ok {
			return packageSlugFromSymbolSlug(symbolSlug)
		}
	}
	return ""
}

func packageSlugFromSymbolSlug(symbolSlug string) string {
	return ""
}

func sortedKeys[T any](m map[string]T) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func renderParams(params []jsdocmodel.Param) string {
	if len(params) == 0 {
		return ""
	}
	lines := make([]string, 0, len(params))
	for _, param := range params {
		switch {
		case param.Type != "" && param.Description != "":
			lines = append(lines, fmt.Sprintf("- `%s` (%s): %s", param.Name, param.Type, param.Description))
		case param.Type != "":
			lines = append(lines, fmt.Sprintf("- `%s` (%s)", param.Name, param.Type))
		case param.Description != "":
			lines = append(lines, fmt.Sprintf("- `%s`: %s", param.Name, param.Description))
		default:
			lines = append(lines, fmt.Sprintf("- `%s`", param.Name))
		}
	}
	return strings.Join(lines, "\n")
}

func renderReturns(ret jsdocmodel.ReturnInfo) string {
	switch {
	case ret.Type != "" && ret.Description != "":
		return fmt.Sprintf("- (%s) %s", ret.Type, ret.Description)
	case ret.Type != "":
		return fmt.Sprintf("- %s", ret.Type)
	case ret.Description != "":
		return fmt.Sprintf("- %s", ret.Description)
	default:
		return ""
	}
}

func bulletList(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, "- "+value)
	}
	return out
}

func wrapCode(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, fmt.Sprintf("`%s`", value))
	}
	return out
}

func uniqueStrings(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func sectionMarkdownPath(section *helpmodel.Section) string {
	return filepath.ToSlash(filepath.Join(sectionTypeDir(section.SectionType), section.Slug+".md"))
}

func sectionTypeDir(sectionType helpmodel.SectionType) string {
	switch sectionType {
	case helpmodel.SectionGeneralTopic:
		return "topics"
	case helpmodel.SectionExample:
		return "examples"
	case helpmodel.SectionApplication:
		return "applications"
	case helpmodel.SectionTutorial:
		return "tutorials"
	}
	return "topics"
}

func renderSectionBody(section *glazedhelp.Section) string {
	var body strings.Builder
	body.WriteString("# ")
	body.WriteString(section.Title)
	body.WriteString("\n\n")
	if section.Short != "" {
		body.WriteString(section.Short)
		body.WriteString("\n\n")
	}
	body.WriteString(strings.TrimSpace(section.Content))
	return body.String()
}

func (c *Composer) renderMetadata(slug string) string {
	switch {
	case c.slugToPackage[slug] != nil:
		pkg := c.slugToPackage[slug]
		lines := []string{fmt.Sprintf("- **Package name:** `%s`", pkg.Name)}
		if pkg.SourceFile != "" {
			lines = append(lines, fmt.Sprintf("- **Extracted from:** `%s`", pkg.SourceFile))
		}
		return "## JSDoc Identity\n\n" + strings.Join(lines, "\n")
	case c.slugToSymbol[slug] != nil:
		sym := c.slugToSymbol[slug]
		lines := []string{fmt.Sprintf("- **Symbol name:** `%s`", sym.Name)}
		if sym.SourceFile != "" {
			loc := sym.SourceFile
			if sym.Line > 0 {
				loc = fmt.Sprintf("%s:%d", sym.SourceFile, sym.Line)
			}
			lines = append(lines, fmt.Sprintf("- **Extracted from:** `%s`", loc))
		}
		return "## JSDoc Identity\n\n" + strings.Join(lines, "\n")
	case c.slugToExample[slug] != nil:
		ex := c.slugToExample[slug]
		lines := []string{fmt.Sprintf("- **Example id:** `%s`", ex.ID)}
		if ex.SourceFile != "" {
			loc := ex.SourceFile
			if ex.Line > 0 {
				loc = fmt.Sprintf("%s:%d", ex.SourceFile, ex.Line)
			}
			lines = append(lines, fmt.Sprintf("- **Extracted from:** `%s`", loc))
		}
		return "## JSDoc Identity\n\n" + strings.Join(lines, "\n")
	default:
		return ""
	}
}

func (c *Composer) renderRelated(ctx context.Context, slug string, hs *glazedhelp.HelpSystem) string {
	sections, err := hs.QuerySections("topic:" + slug)
	if err != nil {
		return ""
	}
	var related []*glazedhelp.Section
	for _, section := range sections {
		if section.Slug == slug {
			continue
		}
		related = append(related, section)
	}
	if len(related) == 0 {
		return ""
	}

	groups := map[helpmodel.SectionType][]string{}
	for _, section := range related {
		if err := ctx.Err(); err != nil {
			return ""
		}
		groups[section.SectionType] = append(groups[section.SectionType], fmt.Sprintf("- `%s` — %s", section.Slug, section.Title))
	}

	var blocks []string
	if topics := groups[helpmodel.SectionGeneralTopic]; len(topics) > 0 {
		sort.Strings(topics)
		blocks = append(blocks, "### Topics\n\n"+strings.Join(topics, "\n"))
	}
	if examples := groups[helpmodel.SectionExample]; len(examples) > 0 {
		sort.Strings(examples)
		blocks = append(blocks, "### Examples\n\n"+strings.Join(examples, "\n"))
	}
	if applications := groups[helpmodel.SectionApplication]; len(applications) > 0 {
		sort.Strings(applications)
		blocks = append(blocks, "### Applications\n\n"+strings.Join(applications, "\n"))
	}
	if tutorials := groups[helpmodel.SectionTutorial]; len(tutorials) > 0 {
		sort.Strings(tutorials)
		blocks = append(blocks, "### Tutorials\n\n"+strings.Join(tutorials, "\n"))
	}
	if len(blocks) == 0 {
		return ""
	}
	return "## Related Help Pages\n\n" + strings.Join(blocks, "\n\n")
}

func (c *Composer) knowsSlug(slug string) bool {
	if c.result == nil {
		return false
	}
	if slug == c.result.RootSlug {
		return true
	}
	for _, known := range c.result.PackageSlugs {
		if known == slug {
			return true
		}
	}
	for _, known := range c.result.SymbolSlugs {
		if known == slug {
			return true
		}
	}
	for _, known := range c.result.ExampleSlugs {
		if known == slug {
			return true
		}
	}
	return false
}

// BuildSectionFS materializes bridged sections into an fs.FS rooted at dir.
func BuildSectionFS(dir string, result *Result) (fs.FS, error) {
	files, err := BuildMarkdownFiles(result)
	if err != nil {
		return nil, err
	}
	if err := WriteMarkdownFiles(dir, files); err != nil {
		return nil, err
	}
	return os.DirFS(dir), nil
}
