package uidsl

import (
	"github.com/go-go-golems/go-go-goja/modules"
	"github.com/go-go-golems/go-go-goja/pkg/tsgen/spec"
)

var _ modules.TypeScriptDeclarer = (*Registrar)(nil)

func (r *Registrar) TypeScriptModule() *spec.Module {
	return &spec.Module{
		Name:        "ui.dsl",
		Description: "Server-rendered HTML node DSL for go-go-goja runtimes.",
		Functions: []spec.Function{
			{Name: "page", Params: []spec.Param{{Name: "children", Type: spec.Unknown(), Variadic: true}}, Returns: spec.Named("Node")},
			{Name: "fragment", Params: []spec.Param{{Name: "children", Type: spec.Unknown(), Variadic: true}}, Returns: spec.Named("Node")},
			{Name: "text", Params: []spec.Param{{Name: "value", Type: spec.Unknown()}}, Returns: spec.Named("Node")},
			{Name: "raw", Params: []spec.Param{{Name: "html", Type: spec.String()}}, Returns: spec.Named("Node")},
			{Name: "render", Params: []spec.Param{{Name: "value", Type: spec.Unknown()}}, Returns: spec.String()},
			{Name: "codeBlock", Params: []spec.Param{{Name: "language", Type: spec.String()}, {Name: "source", Type: spec.Unknown()}, {Name: "options", Type: spec.Unknown(), Optional: true}}, Returns: spec.Named("Node")},
			{Name: "sql", Params: []spec.Param{{Name: "source", Type: spec.Unknown()}, {Name: "options", Type: spec.Unknown(), Optional: true}}, Returns: spec.Named("Node")},
			{Name: "js", Params: []spec.Param{{Name: "source", Type: spec.Unknown()}, {Name: "options", Type: spec.Unknown(), Optional: true}}, Returns: spec.Named("Node")},
			{Name: "jsonBlock", Params: []spec.Param{{Name: "value", Type: spec.Unknown()}, {Name: "options", Type: spec.Unknown(), Optional: true}}, Returns: spec.Named("Node")},
			{Name: "badge", Params: []spec.Param{{Name: "value", Type: spec.Unknown()}, {Name: "options", Type: spec.Unknown(), Optional: true}}, Returns: spec.Named("Node")},
			{Name: "tabs", Params: []spec.Param{{Name: "id", Type: spec.String()}, {Name: "tabs", Type: spec.Unknown()}, {Name: "options", Type: spec.Unknown(), Optional: true}}, Returns: spec.Named("Node")},
		},
		RawDTS: []string{
			"export type Node = unknown;",
			"export type Attrs = Record<string, unknown>;",
			"export type Child = Node | string | number | boolean | null | undefined;",
			"export type Tag = (attrsOrChild?: Attrs | Child, ...children: Child[]) => Node;",
			"export const table: ((id: string) => unknown) & { fromRows(id: string, rows: unknown[]): unknown };",
			"export const html: Tag; export const head: Tag; export const body: Tag; export const title: Tag;",
			"export const meta: Tag; export const link: Tag; export const script: Tag; export const style: Tag;",
			"export const main: Tag; export const div: Tag; export const span: Tag; export const h1: Tag; export const h2: Tag; export const h3: Tag; export const h4: Tag;",
			"export const p: Tag; export const a: Tag; export const form: Tag; export const input: Tag; export const button: Tag; export const select: Tag; export const option: Tag;",
			"export const ul: Tag; export const ol: Tag; export const li: Tag; export const thead: Tag; export const tbody: Tag; export const tr: Tag; export const th: Tag; export const td: Tag;",
			"export const section: Tag; export const article: Tag; export const header: Tag; export const footer: Tag; export const nav: Tag; export const label: Tag; export const textarea: Tag;",
			"export const strong: Tag; export const em: Tag; export const small: Tag; export const pre: Tag; export const code: Tag; export const img: Tag; export const br: Tag; export const hr: Tag;",
		},
	}
}
