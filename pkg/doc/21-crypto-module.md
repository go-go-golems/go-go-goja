---
Title: Crypto Module
Slug: crypto-module
Short: Generate UUIDs, random bytes, and SHA/MD5 hashes from JavaScript
Topics:
- crypto
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

The `crypto` module provides basic cryptographic helpers for Goja runtimes. It exposes UUID generation, random byte generation, and Node-style `createHash` digest objects.

The module is aliased as both `crypto` and `node:crypto`. It is intentionally a lightweight subset, not a full Node.js crypto polyfill.

## JavaScript usage

```javascript
const crypto = require("crypto");

const id = crypto.randomUUID();
// e.g. "550e8400-e29b-41d4-a716-446655440000"

const buf = crypto.randomBytes(16);
// Buffer with 16 random bytes

const hash = crypto.createHash("sha256")
  .update("hello")
  .digest("hex");
// "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
```

## Module API

### `randomUUID()`

Generates a version-4 UUID string.

### `randomBytes(size)`

Generates a `Buffer` containing `size` cryptographically secure random bytes. Throws a type error when `size` is negative.

### `createHash(algorithm)`

Creates a hash object for the named algorithm. Supported algorithms:

- `sha256`
- `sha512`
- `sha1`
- `md5`

Returns a Go-backed object with two methods:

- `update(data)` — appends data to the hash state and returns the same object for chaining.
- `digest(encoding?)` — finalizes the hash and returns the digest.
  - Without an encoding, returns a `Buffer`.
  - With `"hex"`, returns a hex string.
  - With `"base64"`, returns a Base64 string.
  - Any other encoding throws a type error.

## Security notes

This module exists primarily for Node.js compatibility and convenience helpers such as request signatures and content hashing. SHA-1 and MD5 are provided because callers may need to match existing third-party interfaces. For new designs requiring collision resistance, prefer `sha256`.

## Troubleshooting

| Problem | Cause | Solution |
|---|---|---|
| "unsupported hash algorithm" error | Algorithm name is not in the supported list | Use `sha256`, `sha512`, `sha1`, or `md5` |
| "randomBytes size must be >= 0" error | Negative size passed | Pass a non-negative integer size |
| Data is not a string or Buffer inside `update()` | The runtime `buffer` module decodes the value automatically | Ensure the `buffer` module is available in the runtime |
