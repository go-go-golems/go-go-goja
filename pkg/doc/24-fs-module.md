---
Title: File System Module
Slug: fs-module
Short: Promise-based and synchronous file system helpers for Goja runtimes
Topics:
- fs
- modules
- goja
- javascript
Commands:
- goja-repl
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: GeneralTopic
---

The `fs` module provides Node-style file system access for Goja runtimes. It is aliased as both `fs` and `node:fs`. The module exports both asynchronous promise-based functions and synchronous blocking variants.

Async operations run on background goroutines and resolve through the runtime owner, so they do not freeze the JavaScript event loop. Sync operations block the runtime goroutine and should be used sparingly.

## JavaScript usage

```javascript
const fs = require("fs");

await fs.mkdir("/tmp/demo", { recursive: true });
await fs.writeFile("/tmp/demo/hello.txt", "hello world");
const text = await fs.readFile("/tmp/demo/hello.txt", "utf8");
console.log(text);

const exists = await fs.exists("/tmp/demo/hello.txt");

const entries = await fs.readdir("/tmp/demo");

const stats = await fs.stat("/tmp/demo/hello.txt");
console.log(stats.size, stats.isFile);
```

## Async API

All async functions return Promises.

### `readFile(path, encoding?)`

Reads the entire file. Without an encoding, resolves to a `Buffer`. With `"utf8"`, resolves to a string.

### `writeFile(path, data, options?)`

Writes `data` to `path`, overwriting if necessary. `data` can be a string, `Buffer`, `Uint8Array`, or `DataView`. `options` may include:
- `encoding` — string encoding when `data` is a string.
- `mode` — numeric file permission mode (default `0o644`).

### `appendFile(path, data, options?)`

Appends `data` to `path`. Accepts the same data types and options as `writeFile`.

### `exists(path)`

Resolves to `true` if the path exists, `false` otherwise.

### `mkdir(path, options?)`

Creates a directory. `options` may include:
- `recursive` — create parent directories as needed (default `false`).
- `mode` — permission mode (default `0o755`).

### `readdir(path)`

Resolves to an array of entry names in the directory.

### `stat(path)`

Resolves to a `FileStats` object:
- `name` — base name of the file.
- `size` — size in bytes.
- `mode` — file mode bits.
- `modTime` — modification time as an ISO8601 string.
- `isDir` — `true` if the entry is a directory.
- `isFile` — `true` if the entry is a regular file.

### `unlink(path)`

Deletes a file.

### `rename(oldPath, newPath)`

Renames or moves a file or directory.

### `copyFile(src, dst)`

Copies a file from `src` to `dst`.

### `rm(path, options?)`

Removes a file or directory. `options` may include:
- `recursive` — remove directories and their contents (default `false`).
- `force` — do not throw when the path does not exist (default `false`).

## Sync API

Each async function has a synchronous counterpart:

- `readFileSync(path, encoding?)` — returns string or Buffer.
- `writeFileSync(path, data, options?)`
- `appendFileSync(path, data, options?)`
- `existsSync(path)` — returns boolean.
- `mkdirSync(path, options?)`
- `readdirSync(path)` — returns `string[]`.
- `statSync(path)` — returns `FileStats`.
- `unlinkSync(path)`
- `renameSync(oldPath, newPath)`
- `copyFileSync(src, dst)`
- `rmSync(path, options?)`

## Read-only backends

For embedded assets, you can create a read-only `fs` module from a Go `embed.FS` or any `fs.FS`. This is useful for serving static files or asset bundles:

```go
assetsMod := fs.New(
    fs.WithName("fs:assets"),
    fs.WithBackend(fs.NewReadOnlyFSBackend(
        fs.FSMount{FS: embedFS, Root: "app/public", Mount: "/"},
    )),
)
```

Read-only modules reject writes and return `EROFS` errors.

## Troubleshooting

| Problem | Cause | Solution |
|---|---|---|
| "fs module requires runtime services" panic | The module is used in a runtime without owner services | Build the engine with an owner-based runtime factory |
| File not found inside embedded FS | Wrong mount root or virtual path | Use `cleanVirtualPath` logic and mount with matching prefixes |
| Promises never resolve | Background goroutine blocked or runtime closed | Ensure the runtime context is alive and the owner event loop is running |
