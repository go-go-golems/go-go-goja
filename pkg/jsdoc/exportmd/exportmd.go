// Package exportmd produces a deterministic single-file Markdown representation of a DocStore.
package exportmd

import (
	"context"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"

	"github.com/pkg/errors"

	"github.com/go-go-golems/go-go-goja/pkg/jsdoc/model"
)

type Options struct {
	// TOCDepth controls ToC depth (levels). Values <= 0 default to 3.
	TOCDepth int
}

type tocItem struct {
	Level  int
	Text   string
	Anchor string
}

func Write(ctx context.Context, store *model.DocStore, w io.Writer, opts Options) error {
	if store == nil {
		return errors.New("store is nil")
	}
	if w == nil {
		return errors.New("writer is nil")
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	tocDepth := opts.TOCDepth
	if tocDepth <= 0 {
		tocDepth = 3
	}

	var toc []tocItem
	var body strings.Builder
	addHeading := func(level int, text string) {
		anchor := anchorize(text)
		toc = append(toc, tocItem{Level: level, Text: text, Anchor: anchor})
		body.WriteString(strings.Repeat("#", level))
		body.WriteString(" ")
		body.WriteString(text)
		body.WriteString("\n\n")
	}

	// Packages
	addHeading(2, "Packages")
	pkgNames := make([]string, 0, len(store.ByPackage))
	for name := range store.ByPackage {
		pkgNames = append(pkgNames, name)
	}
	sort.Strings(pkgNames)
	if len(pkgNames) == 0 {
		body.WriteString("_No packages._\n\n")
	} else {
		for _, name := range pkgNames {
			pkg := store.ByPackage[name]
			addHeading(3, fmt.Sprintf("Package: %s", pkg.Name))
			if pkg.Title != "" {
				body.WriteString(fmt.Sprintf("**Title:** %s\n\n", pkg.Title))
			}
			if pkg.Description != "" {
				body.WriteString(pkg.Description)
				body.WriteString("\n\n")
			}
			if pkg.Prose != "" {
				body.WriteString(pkg.Prose)
				body.WriteString("\n\n")
			}
			if pkg.SourceFile != "" {
				body.WriteString(fmt.Sprintf("**Source:** `%s`\n\n", pkg.SourceFile))
			}
		}
	}

	// Symbols
	addHeading(2, "Symbols")
	symNames := make([]string, 0, len(store.BySymbol))
	for name := range store.BySymbol {
		symNames = append(symNames, name)
	}
	sort.Strings(symNames)
	if len(symNames) == 0 {
		body.WriteString("_No symbols._\n\n")
	} else {
		for _, name := range symNames {
			sym := store.BySymbol[name]
			addHeading(3, fmt.Sprintf("Symbol: %s", sym.Name))
			if sym.Summary != "" {
				body.WriteString(sym.Summary)
				body.WriteString("\n\n")
			}
			if len(sym.Params) > 0 {
				body.WriteString("**Parameters**\n\n")
				for _, p := range sym.Params {
					if p.Type != "" {
						body.WriteString(fmt.Sprintf("- `%s` (%s): %s\n", p.Name, p.Type, p.Description))
					} else {
						body.WriteString(fmt.Sprintf("- `%s`: %s\n", p.Name, p.Description))
					}
				}
				body.WriteString("\n")
			}
			if sym.Returns.Type != "" || sym.Returns.Description != "" {
				body.WriteString("**Returns**\n\n")
				if sym.Returns.Type != "" {
					body.WriteString(fmt.Sprintf("- (%s) %s\n\n", sym.Returns.Type, sym.Returns.Description))
				} else {
					body.WriteString(fmt.Sprintf("- %s\n\n", sym.Returns.Description))
				}
			}
			if sym.Prose != "" {
				body.WriteString(sym.Prose)
				body.WriteString("\n\n")
			}
			if sym.SourceFile != "" {
				loc := sym.SourceFile
				if sym.Line > 0 {
					loc = fmt.Sprintf("%s:%d", sym.SourceFile, sym.Line)
				}
				body.WriteString(fmt.Sprintf("**Source:** `%s`\n\n", loc))
			}
		}
	}

	// Examples
	addHeading(2, "Examples")
	exIDs := make([]string, 0, len(store.ByExample))
	for id := range store.ByExample {
		exIDs = append(exIDs, id)
	}
	sort.Strings(exIDs)
	if len(exIDs) == 0 {
		body.WriteString("_No examples._\n\n")
	} else {
		for _, id := range exIDs {
			ex := store.ByExample[id]
			title := ex.ID
			if ex.Title != "" {
				title = fmt.Sprintf("%s — %s", ex.ID, ex.Title)
			}
			addHeading(3, fmt.Sprintf("Example: %s", title))
			if len(ex.Symbols) > 0 {
				sort.Strings(ex.Symbols)
				body.WriteString("**Symbols:** ")
				body.WriteString(strings.Join(ex.Symbols, ", "))
				body.WriteString("\n\n")
			}
			if ex.Body != "" {
				body.WriteString("```js\n")
				body.WriteString(ex.Body)
				if !strings.HasSuffix(ex.Body, "\n") {
					body.WriteString("\n")
				}
				body.WriteString("```\n\n")
			}
			if ex.SourceFile != "" {
				loc := ex.SourceFile
				if ex.Line > 0 {
					loc = fmt.Sprintf("%s:%d", ex.SourceFile, ex.Line)
				}
				body.WriteString(fmt.Sprintf("**Source:** `%s`\n\n", loc))
			}
		}
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	var out strings.Builder
	out.WriteString("# JSDoc Export\n\n")
	out.WriteString("## Table of Contents\n\n")
	out.WriteString(renderTOC(toc, tocDepth))
	out.WriteString("\n")
	out.WriteString(body.String())

	_, err := io.WriteString(w, out.String())
	return errors.Wrap(err, "writing markdown")
}

func renderTOC(items []tocItem, depth int) string {
	var b strings.Builder
	for _, it := range items {
		if it.Level > depth {
			continue
		}
		indent := strings.Repeat("  ", maxInt(0, it.Level-2))
		b.WriteString(indent)
		b.WriteString("- [")
		b.WriteString(it.Text)
		b.WriteString("](#")
		b.WriteString(it.Anchor)
		b.WriteString(")\n")
	}
	if b.Len() == 0 {
		return "_No sections._\n"
	}
	return b.String()
}

var nonWord = regexp.MustCompile(`[^a-z0-9- ]+`)

func anchorize(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = nonWord.ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "--", "-")
	return s
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
