---
Title: Provider API audit for v2 provider graph
Ticket: XGOJA-ARCH-001
Status: active
Topics:
    - xgoja
    - provider-api
DocType: source
Summary: Initial audit conclusion that the v2 provider graph can wrap existing provider runtime APIs for the hard cutover MVP.
LastUpdated: 2026-06-12T15:45:00-04:00
WhatFor: Use when reviewing whether v2 requires provider API changes before provider manifests are introduced.
WhenToUse: Before changing providerapi.Module, CommandSetProvider, VerbSource, HelpSource, or registry APIs.
---

# Provider API audit for v2 provider graph

Initial conclusion: do not change provider runtime APIs for the hard cutover MVP.

Existing APIs already expose the runtime capabilities needed by the first v2 provider graph:

- `providerapi.ProviderRegistry.Packages()` lists registered provider packages.
- `ResolveModule(packageID, moduleName)` resolves selected runtime modules.
- `ResolveCommandSetProvider(packageID, providerName)` resolves provider command sets.
- `ResolveVerbSource(packageID, sourceName)` resolves provider-shipped jsverb sources.
- `ResolveHelpSource(packageID, sourceName)` resolves provider-shipped help docs.
- `providerapi.Module.TypeScript` carries declaration metadata for selected runtime modules.

The v2 provider graph should wrap these APIs and provide central validation:

- selected provider exists;
- selected module exists;
- runtime module aliases are unique;
- command set exists;
- strict declaration mode fails when a selected module lacks a TypeScript descriptor;
- runtime module aliases can be passed to source compilation as automatic externals.

Potential future API additions are deferred:

- static provider manifests for scanning/help search without building;
- richer command-set source dependency descriptors;
- provider-owned config schemas in a static catalog;
- provider package self-description metadata.

These future additions should complement provider Go registration rather than replacing it.
