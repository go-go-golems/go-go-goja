// Package hostauth contains generated-host authentication configuration and
// host-service helpers for xgoja.
//
// The package deliberately sits outside JavaScript-facing modules such as
// express. It parses host-owned auth infrastructure configuration, resolves
// secure defaults, and defines typed host-service payloads that command
// providers and module providers can discover through providerapi.HostServices.
package hostauth
