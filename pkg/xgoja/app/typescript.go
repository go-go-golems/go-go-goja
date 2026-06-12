package app

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/evanw/esbuild/pkg/api"
	"github.com/go-go-golems/go-go-goja/pkg/jsverbs"
	"github.com/go-go-golems/go-go-goja/pkg/tsscript"
)

const sourceLanguageTypeScript = "typescript"

func applyTypeScriptScanOptions(source JSVerbSourceSpec, options *jsverbs.ScanOptions, runtimeAliases []string) {
	if options == nil || source.TypeScript == nil || !source.TypeScript.Enabled {
		return
	}
	tsOptions := tsscriptOptionsFromRuntimeSpec(source.TypeScript)
	tsOptions.External = appendUniqueStrings(tsOptions.External, runtimeAliases...)
	options.SourceTransform = func(file jsverbs.SourceFile) (jsverbs.SourceFile, error) {
		if !tsscript.IsTypeScriptPath(file.Path) {
			return file, nil
		}
		original := append([]byte(nil), file.Source...)
		artifact, err := tsscript.TransformSource(tsscript.Source{
			Path:       file.Path,
			AbsPath:    file.AbsPath,
			ResolveDir: file.ResolveDir,
			Contents:   file.Source,
		}, tsOptions)
		if err != nil {
			return file, err
		}
		file.Source = artifact.Code
		file.OriginalSource = original
		file.Language = sourceLanguageTypeScript
		return file, nil
	}
	options.RuntimeTransform = func(input jsverbs.RuntimeTransformInput) ([]byte, error) {
		if input.Language != sourceLanguageTypeScript {
			out := append([]byte(input.Prelude), input.Source...)
			out = append(out, []byte(input.Overlay)...)
			return out, nil
		}
		sourceWithOverlay := append([]byte(input.Prelude), input.OriginalSource...)
		if len(sourceWithOverlay) == 0 {
			sourceWithOverlay = append([]byte(nil), input.Source...)
		}
		sourceWithOverlay = append(sourceWithOverlay, '\n')
		sourceWithOverlay = append(sourceWithOverlay, []byte(input.Overlay)...)
		compileSource := tsscript.Source{
			Path:       input.RelPath,
			AbsPath:    input.AbsPath,
			ResolveDir: input.ResolveDir,
			Contents:   sourceWithOverlay,
		}
		if source.TypeScript.Bundle {
			var artifact *tsscript.Artifact
			var err error
			if input.RootFS != nil {
				artifact, err = tsscript.BundleVirtualEntryFS(input.RootFS, compileSource, tsOptions)
			} else {
				artifact, err = tsscript.BundleVirtualEntry(compileSource, tsOptions)
			}
			if err != nil {
				return nil, fmt.Errorf("bundle TypeScript jsverb %s: %w", input.RelPath, err)
			}
			return artifact.Code, nil
		}
		artifact, err := tsscript.TransformSource(compileSource, tsOptions)
		if err != nil {
			return nil, fmt.Errorf("transform TypeScript jsverb %s: %w", input.RelPath, err)
		}
		return artifact.Code, nil
	}
}

func appendUniqueStrings(values []string, extra ...string) []string {
	out := append([]string(nil), values...)
	seen := map[string]struct{}{}
	for _, value := range out {
		seen[value] = struct{}{}
	}
	for _, value := range extra {
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

func tsscriptOptionsFromRuntimeSpec(spec *TypeScriptSpec) tsscript.Options {
	if spec == nil {
		return tsscript.Options{}
	}
	return tsscript.Options{
		Target:    targetFromString(spec.Target),
		Format:    formatFromString(spec.Format),
		Platform:  platformFromString(spec.Platform),
		External:  append([]string(nil), spec.External...),
		Define:    cloneStringMap(spec.Define),
		Tsconfig:  tsconfigPath(spec.Tsconfig),
		Sourcemap: sourcemapFromString(spec.Sourcemap),
	}
}

func targetFromString(value string) api.Target {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "es5":
		return api.ES5
	case "es2016":
		return api.ES2016
	case "es2017":
		return api.ES2017
	case "es2018":
		return api.ES2018
	case "es2019":
		return api.ES2019
	case "es2020":
		return api.ES2020
	case "es2021":
		return api.ES2021
	case "es2022":
		return api.ES2022
	case "es2023":
		return api.ES2023
	case "es2024":
		return api.ES2024
	case "esnext":
		return api.ESNext
	default:
		return api.ES2015
	}
}

func formatFromString(value string) api.Format {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "iife":
		return api.FormatIIFE
	case "esm":
		return api.FormatESModule
	default:
		return api.FormatCommonJS
	}
}

func platformFromString(value string) api.Platform {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "browser":
		return api.PlatformBrowser
	case "node":
		return api.PlatformNode
	default:
		return api.PlatformNeutral
	}
}

func sourcemapFromString(value string) api.SourceMap {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "inline":
		return api.SourceMapInline
	case "external", "linked":
		return api.SourceMapLinked
	case "both":
		return api.SourceMapInlineAndExternal
	default:
		return api.SourceMapNone
	}
}

func tsconfigPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	return filepath.Clean(path)
}
