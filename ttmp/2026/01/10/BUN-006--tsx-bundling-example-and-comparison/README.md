# TSX bundling example and comparison

This is the document workspace for ticket BUN-006.

## Structure

- **design/**: Design documents and architecture notes
- **reference/**: Reference documentation and API contracts
- **playbooks/**: Operational playbooks and procedures
- **scripts/**: Utility scripts and automation
- **sources/**: External sources and imported documents
- **various/**: Scratch or meeting notes, working notes
- **archive/**: Optional space for deprecated or reference-only artifacts

## Getting Started

Use docmgr commands to manage this workspace:

- Add documents: `docmgr doc add --ticket BUN-006 --doc-type design-doc --title "My Design"`
- Import sources: `docmgr import file --ticket BUN-006 --file /path/to/doc.md`
- Update metadata: `docmgr meta update --ticket BUN-006 --field Status --value review`
