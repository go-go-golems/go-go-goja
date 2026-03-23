# Changelog

## 2026-03-20

- Initial workspace created
- Added a detailed design document analyzing whether `cmd/gen-dts` can generate declarations for runtime plugins, with current-state architecture, gap analysis, proposed contract changes, phased implementation, pseudocode, risks, and file references.
- Added an investigation diary capturing repository reads, exact commands, the current `gen-dts` plugin miss failure, and the ticket-local manifest probe experiment.
- Added `scripts/plugin_manifest_probe.go` to load example plugins and demonstrate that current manifests support structural `.d.ts` generation but not accurate typed signatures.

## 2026-03-20

Completed evidence-backed analysis for plugin-aware gen-dts support, documented the current manifest limitations, and added a ticket-local probe script that demonstrates best-effort plugin declaration generation.

### Related Files

- /home/manuel/workspaces/2026-03-01/generate-js-types/go-go-goja/cmd/gen-dts/main.go — Current built-in-only generator behavior
- /home/manuel/workspaces/2026-03-01/generate-js-types/go-go-goja/pkg/hashiplugin/contract/jsmodule.proto — Manifest lacks signature metadata today
- /home/manuel/workspaces/2026-03-01/generate-js-types/go-go-goja/ttmp/2026/03/20/GOJA-15-GEN-DTS-PLUGINS--extend-gen-dts-to-generate-typescript-declarations-for-plugins/scripts/plugin_manifest_probe.go — Experiment artifact for manifest-driven declaration sketch


## 2026-03-20

Validated the cited Go packages with targeted tests, confirmed ticket document frontmatter, uploaded the bundle to reMarkable, and recorded that docmgr doctor is currently blocked by a local nil-pointer panic.

### Related Files

- /home/manuel/workspaces/2026-03-01/generate-js-types/go-go-goja/ttmp/2026/03/20/GOJA-15-GEN-DTS-PLUGINS--extend-gen-dts-to-generate-typescript-declarations-for-plugins/reference/01-investigation-diary.md — Recorded validation and upload evidence
- /home/manuel/workspaces/2026-03-01/generate-js-types/go-go-goja/ttmp/2026/03/20/GOJA-15-GEN-DTS-PLUGINS--extend-gen-dts-to-generate-typescript-declarations-for-plugins/tasks.md — Task status and doctor fallback note


## 2026-03-20

Revised the ticket after user clarification: the primary design target is now plugin-author-facing `.d.ts` generation from source-owned SDK metadata, not host-side generation from discovered plugin binaries. Updated the design doc, ticket summary, diary, and tasks accordingly, then prepared the revised bundle for re-upload.

## 2026-03-20

Uploaded the revised plugin-author-facing bundle to reMarkable as GOJA-15 Plugin Author DTS Analysis and verified the remote directory now contains both the original and corrected bundles.

### Related Files

- /home/manuel/workspaces/2026-03-01/generate-js-types/go-go-goja/ttmp/2026/03/20/GOJA-15-GEN-DTS-PLUGINS--extend-gen-dts-to-generate-typescript-declarations-for-plugins/design-doc/01-extending-gen-dts-to-generate-declarations-for-runtime-plugins.md — Corrected design doc uploaded to reMarkable
- /home/manuel/workspaces/2026-03-01/generate-js-types/go-go-goja/ttmp/2026/03/20/GOJA-15-GEN-DTS-PLUGINS--extend-gen-dts-to-generate-typescript-declarations-for-plugins/index.md — Ticket summary now reflects the corrected scope

