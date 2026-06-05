---
Title: Research Logbook
Ticket: 20260603-go-codegen-patterns
Status: active
Topics:
    - go
    - code-generation
    - patterns
    - metaprogramming
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ttmp/2026/06/03/20260603-go-codegen-patterns--go-code-generation-patterns-deep-research/design-doc/01-go-code-generation-patterns-comprehensive-research-report.md
      Note: Linked research report
ExternalSources: []
Summary: ""
LastUpdated: 2026-06-03T17:20:00Z
WhatFor: ""
WhenToUse: ""
---


# Research Logbook

## Goal

Keep a running evaluation of every external resource consulted during the Go code generation patterns deep-dive. For each resource, record what we were researching, why we chose it, how we found it, what was useful, what was useless, what is out of date, and what would need updating.

---

## Resource 1: Go Wiki — GoGenerateTools

- **Path / URL**: https://go.dev/wiki/GoGenerateTools
- **Saved as**: `sources/articles/01-go-wiki-generate-tools.md`
- **Date accessed**: 2026-06-03

### What I was researching
A canonical list of `go generate`-compatible tools in the Go ecosystem.

### What I was looking for in this document in particular
An authoritative inventory of common tools: stringer, enumer, goyacc, wire, msgp, protobuf, thriftrw, avro, deep-copy, interface-extractor, etc.

### Why I chose it
The Go Wiki is maintained by the Go team and is the closest thing to an official registry of `go generate` tools.

### How I found the resource itself
Kagi web search: `Go code generation patterns go generate tools list`.

### What I found useful in the document
- Confirms `stringer` is the canonical enum-to-string generator.
- Lists `Wire` (compile-time dependency injection) as a code generator.
- Lists `msgp`, `protobuf`, `thriftrw`, `gogen-avro` for serialization formats.
- Mentions `bundle` (from x/tools) for single-file package inlining.

### What I didn't find useful
- It is explicitly "incomplete"; many newer tools (e.g., `sqlc`, `ent`) are missing.
- No usage examples or recipes.

### What is out of date / what was wrong
- The wiki has not been aggressively updated since generics landed. Some pre-generics tools (e.g., `go-syncmap`) are now less necessary because generics cover many cases.
- `jsonenums` is listed but `enumer` (with json support) has superseded it for many.

### What would need updating
- Add `sqlc`, `ent`, `oapi-codegen`, `templ`, `gqlgen`.
- Flag pre-generics tools that are obsoleted by Go 1.18+ generics.

---

## Resource 2: DoltHub Blog — Generating Golang Source Files at Build-Time with go:generate

- **Path / URL**: https://www.dolthub.com/blog/2025-05-09-go-generate/
- **Saved as**: `sources/articles/02-dolthub-go-generate.md`
- **Date accessed**: 2026-06-03

### What I was researching
A modern (2025), practical walkthrough of `go generate` including real-world examples from a large Go project (Dolt).

### What I was looking for in this document in particular
How a production Go project uses `go generate` day-to-day, what pitfalls they hit, and what their CI integration looks like.

### Why I chose it
Dolt is a serious Go project with many submodules. The blog post is recent (May 2025) and touches on protobuf, gRPC, and stringer.

### How I found the resource itself
Kagi web search, first page result.

### What I found useful in the document
- Clear distinction between `go generate` (one-off) and generating at build time.
- Mentions `go build` will not trigger `go generate` automatically — important for CI.
- Shows that Google projects (gRPC-go, protobuf-go) use `go generate` as a model.
- Good discussion of `stringer` and the generated output.

### What I didn't find useful
- No coverage of AST-based generation or template-based binary builders.
- Very Dolt-specific protobuf setup may not generalize to smaller projects.

### What is out of date / what was wrong
- Nothing obviously wrong; it is current as of 2025.

### What would need updating
- Could add a section on `go tool` (proposed/landed in Go 1.24 tool dependencies).

---

## Resource 3: dev.to — Metaprogramming with Go (hlubek)

- **Path / URL**: https://dev.to/hlubek/metaprogramming-with-go-or-how-to-build-code-generators-that-parse-go-code-2k3j
- **Saved as**: `sources/articles/03-metaprogramming-go-devto.md`
- **Date accessed**: 2026-06-03

### What I was researching
How to write custom code generators that parse Go source, with AST traversal examples.

### What I was looking for in this document in particular
A hands-on tutorial showing `go/ast`, `go/parser`, `go/token`, and `go/format` in a realistic generator context.

### Why I chose it
The title explicitly targets building code generators that parse Go code. It is a well-known dev.to post frequently cited in Go codegen discussions.

### How I found the resource itself
Kagi web search: `metaprogramming with go how to build code generators`.

### What I found useful in the document
- Good explanation of the `go generate` directive mechanism.
- Shows how to use `go/parser` to parse a source file into an `ast.File`.
- Mentions `go/format` to pretty-print generated AST back to source.
- Provides motivation for why metaprogramming matters in Go (boilerplate reduction).

### What I didn't find useful
- Less depth than Eli Bendersky's post on AST rewriting.
- No complete working generator example; more conceptual.

### What is out of date / what was wrong
- Published 2020. References pre-generics patterns heavily; some of those (e.g., generating typed wrappers for each type) are now simplified by generics.
- Still conceptually valid.

### What would need updating
- Add a "generics era" addendum: which patterns are now better served by type parameters.
- Include a small working repo with `go:generate` + AST example.

---

## Resource 4: Eli Bendersky — Rewriting Go source code with AST tooling

- **Path / URL**: https://eli.thegreenplace.net/2021/rewriting-go-source-code-with-ast-tooling/
- **Saved as**: `sources/articles/04-eli-bendersky-ast.md`
- **Date accessed**: 2026-06-03

### What I was researching
Deep-dive into AST-based source rewriting in Go, including `inspector` and `astutil`.

### What I was looking for in this document in particular
Concrete examples of replacing function bodies, modifying imports, and using `golang.org/x/tools/go/ast/astutil`.

### Why I chose it
Eli Bendersky's posts are consistently high-quality, deep, and accurate. This article is from 2021 and uses the modern `x/tools` packages.

### How I found the resource itself
Kagi web search: `go/ast rewriting source code ast tooling`.

### What I found useful in the document
- Excellent explanation of `ast.Inspect` vs `ast.Walk`.
- Shows `inspector` from `x/tools/go/ast/inspector` for faster traversal.
- Covers adding/removing imports with `astutil.AddImport` and `astutil.DeleteImport`.
- Discusses the subtleties of preserving comments and formatting.

### What I didn't find useful
- The examples are small; a large-scale refactoring example would help.

### What is out of date / what was wrong
- Nothing is wrong; 2021 is recent enough that the APIs are stable.

### What would need updating
- Could cover `x/tools/go/analysis` framework for static-analysis-driven rewriting.

---

## Resource 5: josestg.com — How to Build a Pluggable Library in Go

- **Path / URL**: https://www.josestg.com/posts/golang/how-to-build-a-pluggable-library-in-go/
- **Saved as**: `sources/articles/05-pluggable-library-go.md`
- **Date accessed**: 2026-06-03

### What I was researching
Plugin architectures in Go, including `plugin` package and compile-time composition.

### What I was looking for in this document in particular
Comparison between runtime `plugin` package and compile-time import-based plugins (like xcaddy).

### Why I chose it
Kagi surfaced it under "pluggable library" queries. The article compares the two dominant approaches.

### How I found the resource itself
Kagi web search: `Go dynamically build custom binary multiple packages plugin architecture`.

### What I found useful in the document
- Explains that Go `plugin` package requires shared-object `.so`/`.dll` builds and has platform limitations.
- Makes the case for compile-time composition via generated `main.go` imports.
- Mentions that `go build -buildmode=plugin` is brittle across Go versions.

### What I didn't find useful
- Very short; does not show the actual template/codegen part.
- The example is toy-level.

### What is out of date / what was wrong
- The Go `plugin` package remains poorly supported on Windows and macOS; the article understates this.

### What would need updating
- Add a worked example of generating an import block and compiling a binary, like xcaddy does.

---

## Resource 6: Caddy Docs — Extending Caddy

- **Path / URL**: https://caddyserver.com/docs/extending-caddy
- **Saved as**: `sources/articles/06-caddy-extending.md`
- **Date accessed**: 2026-06-03

### What I was researching
How Caddy implements its plugin system and how xcaddy leverages it.

### What I was looking for in this document in particular
The registration mechanism (`caddy.RegisterModule`) and how blank imports trigger registration.

### Why I chose it
Because the user explicitly called out xcaddy as an example. Understanding the host (Caddy) plugin model is essential to understand the builder (xcaddy).

### How I found the resource itself
Kagi web search: `xcaddy extending caddy`.

### What I found useful in the document
- Clear explanation: Caddy plugins register themselves via `init()` functions that call `caddy.RegisterModule()`.
- The blank import (`_ "plugin"`) is sufficient because the plugin's `init()` runs at startup.
- Shows the module lifecycle: provision, validate, start, stop.

### What I didn't find useful
- No mention of how the binary is actually built; that's in xcaddy, not Caddy docs.

### What is out of date / what was wrong
- Nothing; actively maintained docs.

### What would need updating
- Link directly to xcaddy's builder code and README.

---

## Resource 7: xcaddy GitHub + Source

- **Path / URL**: https://github.com/caddyserver/xcaddy
- **Saved as**: `sources/articles/07-xcaddy-github.md`, `07-xcaddy-readme.md`, `07-xcaddy-builder-go.md`, `10-xcaddy-environment.go.md`
- **Date accessed**: 2026-06-03

### What I was researching
The canonical example of "generate a custom Go binary by writing a template-based main.go and importing selected packages".

### What I was looking for in this document in particular
The exact mechanics: template definition, temp folder creation, `go mod init`, version pinning, build flags, and cleanup.

### Why I chose it
The user explicitly said "one of the examples I was thinking of in particular is xcaddy, btw".

### How I found the resource itself
Kagi web search: `xcaddy go source code github caddy plugin builder code generation import integration`.

### What I found useful in the document
- `environment.go` contains the core pattern:
  1. Parse a `text/template` (`mainModuleTemplate`) with plugin list.
  2. Write `main.go` to a temp folder.
  3. Run `go mod init caddy`.
  4. Run `go mod edit -replace` for local overrides.
  5. Run `go get` for each dependency with version pinning.
  6. Run `go build -o <output>`.
- Handles semantic import versioning (SIV) via `versionedModulePath()`.
- Supports `go:embed` for static assets via an additional template (`embeddedModuleTemplate`).
- Windows resource embedding, race detector, debug builds, PGO.

### What I didn't find useful
- No high-level architecture doc inside the repo; you have to read the code.

### What is out of date / what was wrong
- Nothing major; actively maintained.
- Some comments about macOS High Sierra quirks are historical but harmless.

### What would need updating
- A top-level `ARCHITECTURE.md` explaining the template, module, and build phases would be great.

---

## Resource 8: zakariaamine.com — AST package generate function comments

- **Path / URL**: https://www.zakariaamine.com/2022-09-22/ast-package-generate-function-comments/
- **Saved as**: `sources/articles/08-zakaria-ast.md`
- **Date accessed**: 2026-06-03

### What I was researching
Practical example of using `go/ast` to add doc comments to generated functions.

### What I was looking for in this document in particular
A concrete, copy-pasteable example of parsing a file and injecting comments.

### Why I chose it
It was a top Kagi result for `go/ast generate function comments` and looked hands-on.

### How I found the resource itself
Kagi web search.

### What I found useful in the document
- Good small example of `parser.ParseFile`.
- Shows `go/ast` comment node manipulation.

### What I didn't find useful
- Narrow scope (only comments). Less comprehensive than Eli Bendersky's post.

### What is out of date / what was wrong
- Nothing wrong.

### What would need updating
- Add `go/format` step to show how to write the modified AST back to a `.go` file.

---

## Resource 9: go-cookbook.com — AST Manipulation

- **Path / URL**: https://go-cookbook.com/snippets/other-topics/ast-manipulation
- **Saved as**: `sources/articles/09-go-cookbook-ast.md`
- **Date accessed**: 2026-06-03

### What I was researching
Concise AST traversal examples for beginners.

### What I was looking for in this document in particular
A minimal, runnable snippet for `ast.Inspect`.

### Why I chose it
"Go Cookbook" branding suggests copy/paste recipes.

### How I found the resource itself
Kagi web search.

### What I found useful in the document
- Very short, runnable `ast.Inspect` example.

### What I didn't find useful
- No modification or generation; purely read-only traversal.

### What is out of date / what was wrong
- Nothing wrong; just shallow.

### What would need updating
- Add a "modify the AST and write it back" recipe.

---

## Resource 10: Go FAQ — go-generate-code-generation

- **Path / URL**: https://www.gofaq.org/en/go-generate-code-generation/
- **Saved as**: Not saved separately; content was redundant with Wiki + DoltHub.
- **Date accessed**: 2026-06-03

### What I was researching
A concise FAQ on `go generate`.

### What I was looking for in this document in particular
Quick validation that `go generate` is not run by `go build`.

### Why I chose it
Kagi result #8, looked like a FAQ.

### How I found the resource itself
Kagi search.

### What I found useful in the document
- Confirms `go generate` is opt-in and not part of the normal build.
- Mentions common tools (mockery, stringer, protoc-gen-go).

### What I didn't find useful
- Very shallow; adds nothing beyond Wiki/DoltHub.

### What is out of date / what was wrong
- Nothing wrong.

### What would need updating
N/A. Not worth saving as a primary source.

---

## Resource 11: stringer source (golang/tools)

- **Path / URL**: https://raw.githubusercontent.com/golang/tools/master/cmd/stringer/stringer.go
- **Saved as**: `sources/articles/17-stringer-source.md`
- **Date accessed**: 2026-06-03

### What I was researching
The canonical standard-library-adjacent code generator to understand how Google writes generators.

### What I was looking for in this document in particular
How `stringer` parses Go files, finds constants/enums, and emits a `String()` method.

### Why I chose it
`stringer` is the most widely used `go generate` tool and ships under `golang.org/x/tools`.

### How I found the resource itself
GitHub raw file fetch, pointed to by the Go Wiki.

### What I found useful in the document
- Shows a hybrid approach: `go/ast` for parsing, `text/template` could be used but stringer hand-writes via `fmt.Fprintf` for speed.
- Uses `types.Eval` from `x/tools/go/packages` for type checking.
- Demonstrates how to handle build-tags and file filtering.
- Large (~900 lines), battle-tested, and well-commented.

### What I didn't find useful
- Very dense. Not a tutorial.

### What is out of date / what was wrong
- Nothing; actively maintained.

### What would need updating
- An annotated guide (blog post or doc) walking through `stringer.go` would be community-valuable.

---

## Resource 12: Protocol Buffers Go Generated Code Guide

- **Path / URL**: https://protobuf.dev/reference/go/go-generated/
- **Saved as**: Not saved separately (live reference).
- **Date accessed**: 2026-06-03

### What I was researching
The canonical schema-first code generation pipeline: `.proto` → `protoc-gen-go` → `.pb.go`.

### What I was looking for in this document in particular
Exact output contracts: struct tags, accessor methods, `MessageV2` interface, `protoreflect` integration.

### Why I chose it
Protocol Buffers is the most widely used schema-first generator in Go.

### How I found the resource itself
Kagi search: `protoc-gen-go generated code guide`.

### What I found useful in the document
- Precise specification of generated structs, methods, and tags.
- Explains the `opaque` API mode introduced in protobuf-go v2.
- Documents the `google.golang.org/protobuf` runtime contract.

### What I didn't find useful
- No discussion of custom protobuf plugins or how to write your own `protoc-gen-*`.

### What is out of date / what was wrong
- The opaque API is relatively new (2024); make sure you're reading the opaque guide, not the open guide, if using the latest runtime.

### What would need updating
- Add a "Writing a custom protoc plugin in Go" tutorial section.

---

## Resource 13: oapi-codegen / OpenAPI Go codegen ecosystem

- **Path / URL**: https://github.com/oapi-codegen/oapi-codegen
- **Saved as**: Not saved separately (referenced from search results).
- **Date accessed**: 2026-06-03

### What I was researching
Schema-first HTTP API code generation from OpenAPI specs.

### What I was looking for in this document in particular
Whether OpenAPI → Go is as mature as Protobuf → Go.

### Why I chose it
Kagi results pushed `oapi-codegen` as the dominant OpenAPI generator for Go.

### How I found the resource itself
Kagi search: `Go generate struct from protobuf JSON schema OpenAPI swagger codegen`.

### What I found useful in the document
- Generates server stubs, client SDKs, and type models from an OpenAPI 3 spec.
- Supports multiple output styles (strict server, chi, gin, echo).

### What I didn't find useful
- Did not read the full README in depth; surface-level only.

### What is out of date / what was wrong
- Nothing apparent.

### What would need updating
- Deep-dive reading if this project adopts OpenAPI-first design.

---

## Resource 14: Google Wire README

- **Path / URL**: https://github.com/google/wire
- **Saved as**: `sources/articles/14-google-wire-readme.md`
- **Date accessed**: 2026-06-03

### What I was researching
Compile-time dependency injection as a code generation pattern.

### What I was looking for in this document in particular
How Wire's provider/injector model works, and whether it's a good fit for CLI tooling.

### Why I chose it
Wire is a well-known Google project that uses code generation for DI; it's listed in the Go Wiki.

### How I found the resource itself
GitHub raw README fetch.

### What I found useful in the document
- Wire generates `wire_gen.go` from `wire.go` injector functions.
- Enforces compile-time contract: if dependencies change, generated code fails to compile, alerting the developer.
- Good analogy for "generated code should be checked in and reviewed".

### What I didn't find useful
- Wire is narrowly focused on DI; not a general-purpose codegen pattern.

### What is out of date / what was wrong
- Wire has not seen major updates recently; still valid but not actively evolving.

### What would need updating
- Evaluate whether generics + functional options reduce the need for Wire in new projects.

---

## Resource 15: Masterminds/sprig

- **Path / URL**: https://github.com/Masterminds/sprig
- **Saved as**: `sources/articles/13-masterminds-sprig.md`
- **Date accessed**: 2026-06-03

### What I was researching
Template function libraries that improve `text/template` for code generation.

### What I was looking for in this document in particular
Whether sprig is safe/idiomatic for generating Go source (e.g., string quoting, map iteration).

### Why I chose it
Sprig is the most popular template function library for Go; xcaddy uses standard `text/template`, but many generators benefit from richer template functions.

### How I found the resource itself
Prior knowledge; fetched to verify current feature set.

### What I found useful in the document
- Adds 100+ functions to templates: `snakecase`, `camelcase`, `quote`, `indent`, `dict`, `list`.
- Very useful for naming conventions when generating identifiers.

### What I didn't find useful
- Not specifically about code generation; general-purpose template utility.

### What is out of date / what was wrong
- Nothing.

### What would need updating
N/A. It's a utility, not a learning resource.

---

## Resource 16: campoy/justforfunc — AST episode

- **Path / URL**: https://raw.githubusercontent.com/campoy/justforfunc/master/16-ast/ast.go
- **Saved as**: `sources/articles/16-campoy-justforfunc-ast.md`
- **Date accessed**: 2026-06-03

### What I was researching
A canonical video+code example of AST manipulation from the "justforfunc" series.

### What I was looking for in this document in particular
A small, complete AST program that can be run as a script.

### Why I chose it
Francesc Campoy's justforfunc series is legendary for teaching Go internals.

### How I found the resource itself
GitHub raw file fetch (known from community references).

### What I found useful in the document
- The source file is very short (14 lines in the fetched version); likely the episode script, not the full example.

### What I didn't find useful
- The fetched file is truncated/incomplete. The real content is likely in the episode video, not the repo.

### What is out of date / what was wrong
- Fetched file is not the complete example.

### What would need updating
- Find the correct GitHub path for the full episode 16 code, or use the YouTube transcript if needed.

---

## Resource 17: Various Reddit / Reddit threads

- **Path / URL**: r/golang threads on schema management and codegen
- **Saved as**: Not saved separately.
- **Date accessed**: 2026-06-03

### What I was researching
Community sentiment on whether to go schema-first (proto/OpenAPI) or code-first.

### What I was looking for in this document in particular
Whether Go developers prefer protobuf, OpenAPI, or hand-written structs.

### Why I chose it
Kagi search surfaced several active Reddit threads from 2024–2025.

### How I found the resource itself
Kagi web search.

### What I found useful in the document
- 2024 thread confirms protobuf is still dominant but OpenAPI-first is gaining for REST APIs.
- Several developers note that `protoc-gen-openapi` bridges the gap.

### What I didn't find useful
- Reddit threads are noisy and opinion-heavy; light on technical specifics.

### What is out of date / what was wrong
- Opinions may not reflect best practices.

### What would need updating
N/A. Community sentiment is perishable.

---

## Resource 18: Leapcell — Type-safe database operations with sqlc and go generate

- **Path / URL**: https://leapcell.io/blog/type-safe-database-operations-in-go-with-go-generate-and-sqlc
- **Saved as**: Not saved separately.
- **Date accessed**: 2026-06-03

### What I was researching
SQL-first code generation in Go as a representative of the "schema/contract-first" pattern.

### What I was looking for in this document in particular
How `sqlc` uses `go generate` to turn `.sql` files into type-safe Go code.

### Why I chose it
Kagi surfaced it; `sqlc` is one of the most lauded Go codegen tools in recent years.

### How I found the resource itself
Kagi web search.

### What I found useful in the document
- Confirms that `sqlc` uses `//go:generate sqlc generate` directives.
- Emphasizes type-safe database queries without an ORM.

### What I didn't find useful
- Did not save full article; just noted the pattern.

### What is out of date / what was wrong
- Nothing; `sqlc` is actively maintained.

### What would need updating
- Re-fetch and defuddle if deep comparison with `ent` or `gorm` is needed.

---

## Summary Table

| # | Resource | Saved? | Quality | Staleness | Action |
|---|----------|--------|---------|-----------|--------|
| 1 | Go Wiki: GoGenerateTools | Yes | Medium | Some pre-generics entries | Add modern tools |
| 2 | DoltHub go:generate blog | Yes | High | Current | Good reference |
| 3 | dev.to metaprogramming | Yes | Medium | Pre-generics bias | Add generics note |
| 4 | Eli Bendersky AST | Yes | High | Current | Excellent reference |
| 5 | josestg pluggable library | Yes | Low | Toy example | Needs worked example |
| 6 | Caddy extending docs | Yes | High | Current | Link to xcaddy |
| 7 | xcaddy GitHub + source | Yes | Very High | Current | Primary pattern ref |
| 8 | zakariaamine AST | Yes | Medium | Current | Shallow |
| 9 | go-cookbook AST | Yes | Low | Current | Too shallow |
| 10 | Go FAQ generate | No | Low | Current | Skip |
| 11 | stringer source | Yes | Very High | Current | Canonical generator |
| 12 | protobuf.dev guide | No (live) | High | Some old/open vs opaque | Verify version |
| 13 | oapi-codegen | No | Medium | Current | Deep-dive if needed |
| 14 | Google Wire | Yes | High | Stable | DI-specific |
| 15 | Masterminds/sprig | Yes | Medium | Current | Utility library |
| 16 | campoy justforfunc AST | Yes | Low | Incomplete file | Find full source |
| 17 | Reddit threads | No | Low | Perishable | Community sentiment only |
| 18 | sqlc / leapcell | No | Medium | Current | Noted pattern |
