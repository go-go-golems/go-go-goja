# Tasks

## TODO

- [x] Create the GOJA-15 ticket workspace, design doc, and diary
- [x] Gather evidence from the built-in generator, plugin SDK, and the runtime plugin path as background context
- [x] Run a ticket-local experiment to understand the initial host-side framing
- [x] Revise the ticket to the corrected plugin-author-facing scope
- [x] Write an intern-oriented analysis and implementation guide for plugin-writer `.d.ts` generation
- [x] Update ticket relations, changelog, and index metadata
- [ ] Run `docmgr doctor --ticket GOJA-15-GEN-DTS-PLUGINS --stale-after 30` successfully
- [x] Upload the ticket bundle to reMarkable and verify the remote listing
- [x] Re-upload the revised bundle to reMarkable after the scope correction

## Notes

- The initial analysis over-focused on host-side discovered plugins. The revised design document now treats that as secondary background rather than the main solution.
- `docmgr doctor --ticket GOJA-15-GEN-DTS-PLUGINS --stale-after 30` currently crashes in the local `docmgr` installation with a nil-pointer panic. As a fallback, `docmgr validate frontmatter --doc ... --suggest-fixes` passed for the ticket index, design doc, and diary.
