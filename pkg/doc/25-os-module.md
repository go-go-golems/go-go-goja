---
Title: OS Module
Slug: os-module
Short: Inspect host operating system properties from JavaScript
Topics:
- os
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

The `os` module exposes basic host operating system introspection helpers. It is aliased as both `os` and `node:os`. The module is read-only and safe to use in any runtime context.

## JavaScript usage

```javascript
const os = require("os");

console.log(os.platform());   // "linux", "darwin", "windows"...
console.log(os.arch());       // "amd64", "arm64"...
console.log(os.homedir());    // "/home/user"
console.log(os.tmpdir());     // "/tmp"
console.log(os.hostname());   // "my-machine"
console.log(os.EOL);          // "\n" or "\r\n"

const cpus = os.cpus();
console.log(cpus.length);     // number of logical CPUs
```

## Module API

### `homedir()`

Returns the current user's home directory path.

### `tmpdir()`

Returns the default temporary directory path.

### `platform()`

Returns the Go operating system name (`runtime.GOOS`), such as `linux`, `darwin`, or `windows`.

### `arch()`

Returns the Go architecture name (`runtime.GOARCH`), such as `amd64` or `arm64`.

### `hostname()`

Returns the host name reported by the operating system.

### `release()`

Returns the same value as `platform()`. Provided for Node.js compatibility.

### `type()`

Returns the same value as `platform()`. Provided for Node.js compatibility.

### `cpus()`

Returns an array of CPU objects. The current implementation returns lightweight stubs with `model`, `speed`, and `times` fields so that generic Node.js code does not fail.

### `os.EOL`

A string constant: `\n` on Unix-like systems and `\r\n` on Windows.

## Troubleshooting

| Problem | Cause | Solution |
|---|---|---|
| `cpus()` array is empty | Zero logical processors reported by Go runtime | This is extremely unlikely; check the Go runtime environment |
| `hostname()` returns an error | OS call fails | Verify process permissions for reading the host name |
