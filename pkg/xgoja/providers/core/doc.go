// Package core exposes safe, data-oriented first-party go-go-goja modules as
// an xgoja provider package.
//
// The provider package ID is "go-go-goja-core". It registers path, node:path,
// yaml, crypto, node:crypto, time, timer, events, and node:events. These
// modules are intended as the default first-party provider set for generated
// binaries that need common JavaScript helpers without filesystem, process, or
// database host capabilities.
package core
