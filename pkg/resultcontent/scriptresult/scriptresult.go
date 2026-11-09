// Package scriptresult reports the final value of a one-shot script run
// (eval, run) together with its result content: providers in the value are
// printed as "[content N: ...]" placeholders and, when a content directory is
// given, rendered and written to files whose paths are printed.
package scriptresult

import (
	"context"
	"fmt"
	"io"
	"reflect"
	"strings"
	"time"

	"github.com/dop251/goja"
	"github.com/go-go-golems/go-go-goja/pkg/resultcontent"
	"github.com/go-go-golems/go-go-goja/pkg/runtimeowner"
	"github.com/pkg/errors"
)

// ContentDirFlag is the conventional flag name for Options.ContentDir.
const ContentDirFlag = "content-dir"

// ContentDirHelp is the conventional help text for ContentDirFlag.
const ContentDirHelp = "Render result content (images returned by the script) and write it to this directory as content-N.<ext>"

// Options controls Build.
type Options struct {
	// ContentDir, when set, renders the providers and writes their content
	// there. When empty, providers are only named, not rendered.
	ContentDir string
	// FilePrefix is the file name prefix (default "content").
	FilePrefix string
	// Limits bounds extraction and rendering.
	Limits resultcontent.Limits
}

// Report is the content part of a script's final value.
type Report struct {
	// Text is the placeholder rendering of the value; empty when the value
	// holds no providers.
	Text       string
	Providers  int
	Rendering  resultcontent.Rendering
	Files      []resultcontent.WrittenFile
	ContentDir string
}

// HasContent reports whether the value held at least one provider.
func (r Report) HasContent() bool { return r.Providers > 0 }

// Fprint writes the placeholder text followed by one line per written file
// (or dropped item), or a note that content was not written.
func (r Report) Fprint(w io.Writer) error {
	if !r.HasContent() {
		return nil
	}
	lines := []string{r.Text}
	if r.ContentDir == "" {
		lines = append(lines, fmt.Sprintf("# %d content item(s) not written; pass --%s DIR to write them", r.Providers, ContentDirFlag))
	} else {
		for _, f := range r.Files {
			label := fmt.Sprintf("content %d", f.Provider)
			if f.Caption != "" {
				label += " " + f.Caption
			}
			lines = append(lines, fmt.Sprintf("# %s: %s", label, f.Path))
		}
		for _, note := range r.Rendering.Notes() {
			lines = append(lines, "# "+note)
		}
	}
	_, err := io.WriteString(w, strings.Join(lines, "\n")+"\n")
	return err
}

// Settle returns the fulfilled value when value is a promise, polling the
// owner until it settles; a rejection is returned as an error. Other values
// are returned unchanged.
func Settle(ctx context.Context, owner runtimeowner.RuntimeOwner, value goja.Value) (goja.Value, error) {
	if owner == nil {
		return nil, errors.New("scriptresult: runtime owner is nil")
	}
	promise, err := asPromise(ctx, owner, value)
	if err != nil || promise == nil {
		return value, err
	}
	for {
		ret, err := owner.Call(ctx, "scriptresult.promise-state", func(_ context.Context, vm *goja.Runtime) (any, error) {
			switch promise.State() {
			case goja.PromiseStatePending:
				return nil, nil
			case goja.PromiseStateRejected:
				return nil, errors.Errorf("promise rejected: %s", rejection(promise.Result()))
			case goja.PromiseStateFulfilled:
				return settled{value: promise.Result()}, nil
			default:
				return nil, nil
			}
		})
		if err != nil {
			return nil, err
		}
		if s, ok := ret.(settled); ok {
			return s.value, nil
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(5 * time.Millisecond):
		}
	}
}

type settled struct{ value goja.Value }

var promiseType = reflect.TypeOf((*goja.Promise)(nil))

func asPromise(ctx context.Context, owner runtimeowner.RuntimeOwner, value goja.Value) (*goja.Promise, error) {
	obj, ok := value.(*goja.Object)
	if !ok || obj == nil {
		return nil, nil
	}
	ret, err := owner.Call(ctx, "scriptresult.as-promise", func(_ context.Context, vm *goja.Runtime) (any, error) {
		// ExportType first: Export on a plain object would read every property.
		if obj.ExportType() != promiseType {
			return nil, nil
		}
		p, _ := obj.Export().(*goja.Promise)
		return p, nil
	})
	if err != nil {
		return nil, err
	}
	p, _ := ret.(*goja.Promise)
	return p, nil
}

func rejection(value goja.Value) string {
	if value == nil || goja.IsUndefined(value) {
		return "undefined"
	}
	if obj, ok := value.(*goja.Object); ok {
		if msg := obj.Get("message"); msg != nil && !goja.IsUndefined(msg) {
			return msg.String()
		}
	}
	return value.String()
}

// Build extracts providers from value on the owner goroutine and, when
// opts.ContentDir is set, renders them off the owner goroutine and writes the
// files. A value without providers yields an empty Report.
func Build(ctx context.Context, owner runtimeowner.RuntimeOwner, value goja.Value, opts Options) (Report, error) {
	if owner == nil {
		return Report{}, errors.New("scriptresult: runtime owner is nil")
	}
	ret, err := owner.Call(ctx, "scriptresult.extract", func(_ context.Context, vm *goja.Runtime) (any, error) {
		return resultcontent.Extract(vm, value, opts.Limits)
	})
	if err != nil {
		return Report{}, err
	}
	extraction, ok := ret.(resultcontent.Extraction)
	if !ok {
		return Report{}, errors.Errorf("scriptresult: unexpected extraction type %T", ret)
	}
	report := Report{Text: extraction.Text, Providers: len(extraction.Providers), ContentDir: strings.TrimSpace(opts.ContentDir)}
	if !report.HasContent() || report.ContentDir == "" {
		return report, nil
	}
	report.Rendering, report.Files, err = resultcontent.RenderToDir(ctx, extraction.Providers, opts.Limits, report.ContentDir, opts.FilePrefix)
	if err != nil {
		return report, err
	}
	return report, nil
}
