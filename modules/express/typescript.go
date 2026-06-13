package express

import (
	"github.com/go-go-golems/go-go-goja/modules"
	"github.com/go-go-golems/go-go-goja/pkg/tsgen/spec"
)

var _ modules.TypeScriptDeclarer = (*Registrar)(nil)

func (r *Registrar) TypeScriptModule() *spec.Module {
	name := "express"
	if r != nil && r.name != "" {
		name = r.name
	}
	return &spec.Module{
		Name:        name,
		Description: "Express-style HTTP route registration for go-go-goja hosts.",
		Functions: []spec.Function{
			{Name: "app", Returns: spec.Named("App")},
		},
		RawDTS: []string{
			"export interface App {",
			"  get(pattern: string, handler: Handler): void;",
			"  post(pattern: string, handler: Handler): void;",
			"  put(pattern: string, handler: Handler): void;",
			"  patch(pattern: string, handler: Handler): void;",
			"  delete(pattern: string, handler: Handler): void;",
			"  all(pattern: string, handler: Handler): void;",
			"  mount(prefix: string, handler: MountableHandler, options?: MountOptions): void;",
			"  mountHandler(prefix: string, handler: MountableHandler, options?: MountOptions): void;",
			"  static(prefix: string, directory: string): void;",
			"  staticFromAssetsModule(prefix: string, assetsModule: unknown, root: string): void;",
			"}",
			"export interface MountableHandler {}",
			"export interface MountOptions { stripPrefix?: boolean; excludePrefixes?: string[]; }",
			"export type Handler = (req: Request, res: Response) => unknown;",
			"export interface Request {",
			"  method: string;",
			"  url: string;",
			"  path: string;",
			"  query: Record<string, string | string[]>;",
			"  params: Record<string, string>;",
			"  headers: Record<string, string>;",
			"  cookies: Record<string, string>;",
			"  session: Session | null;",
			"  ip: string;",
			"  body: unknown;",
			"  rawBody: string;",
			"}",
			"export interface Session { id: string; isNew: boolean; cookieName: string; }",
			"export interface Response {",
			"  status(code: number): Response;",
			"  set(name: string, value: string): Response;",
			"  type(value: string): Response;",
			"  json(value: unknown): void;",
			"  send(value?: unknown): void;",
			"  html(value: unknown): void;",
			"  redirect(url: string): void;",
			"  redirect(status: number, url: string): void;",
			"  end(): void;",
			"}",
		},
	}
}
