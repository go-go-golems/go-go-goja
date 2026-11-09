package jsverbs

import (
	"context"
	"strings"

	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/go-go-goja/pkg/resultcontent"
)

const (
	// ContentSectionSlug is the Glazed section every verb command gets for
	// result content options. Its values are not passed to the verb.
	ContentSectionSlug = "jsverbs-content"
	// ContentDirField is the --content-dir flag in ContentSectionSlug.
	ContentDirField = "content-dir"
)

// addContentSection adds the result-content section with --content-dir to a
// verb command unless the verb already defines a field with that name (the
// verb then owns the flag and content is reported as placeholders only).
func addContentSection(desc *cmds.CommandDescription) error {
	if desc == nil || desc.Schema == nil {
		return nil
	}
	conflict := false
	desc.Schema.ForEach(func(_ string, section schema.Section) {
		if _, ok := section.GetDefinitions().Get(ContentDirField); ok {
			conflict = true
		}
	})
	if conflict {
		return nil
	}
	section, err := schema.NewSection(ContentSectionSlug, "Result content",
		schema.WithDescription("Options for result content (images and other rich values) returned by the verb"),
		schema.WithFields(fields.New(ContentDirField, fields.TypeString,
			fields.WithDefault(""),
			fields.WithHelp("Render result content returned by the verb and write it to this directory as content-N.<ext>; the row field then holds the file path"))),
	)
	if err != nil {
		return err
	}
	desc.Schema.Set(ContentSectionSlug, section)
	return nil
}

func contentDirFromValues(parsedValues *values.Values) string {
	if parsedValues == nil {
		return ""
	}
	field, ok := parsedValues.GetField(ContentSectionSlug, ContentDirField)
	if !ok || field == nil {
		return ""
	}
	dir, _ := field.Value.(string)
	return strings.TrimSpace(dir)
}

// processResultContent replaces result content providers in an exported verb
// result. Without a content directory each provider becomes its placeholder
// string ("[content N: ...]"); with one, providers are rendered, their items
// written as files, and each provider becomes its file path (or a list of
// paths when it produced several items, or its placeholder when nothing was
// written). Glazed never sees a raw provider.
func (r *Registry) processResultContent(ctx context.Context, result interface{}, parsedValues *values.Values) (interface{}, error) {
	if result == nil {
		return nil, nil
	}
	limits := resultcontent.Limits{}
	if r != nil {
		limits = r.ResultContentLimits
	}
	replaced, providers := resultcontent.ExtractExported(result, limits)
	if len(providers) == 0 {
		return result, nil
	}
	dir := contentDirFromValues(parsedValues)
	if dir == "" {
		return replaced, nil
	}
	_, files, err := resultcontent.RenderToDir(ctx, providers, limits, dir, resultcontent.DefaultFilePrefix)
	if err != nil {
		return nil, err
	}
	paths := resultcontent.PathsByProvider(files)
	withPaths, _ := resultcontent.MapExported(result, limits, func(n int, p resultcontent.Provider) any {
		switch len(paths[n]) {
		case 0:
			return resultcontent.Placeholder(n, p)
		case 1:
			return paths[n][0]
		default:
			out := make([]any, 0, len(paths[n]))
			for _, path := range paths[n] {
				out = append(out, path)
			}
			return out
		}
	})
	return withPaths, nil
}
