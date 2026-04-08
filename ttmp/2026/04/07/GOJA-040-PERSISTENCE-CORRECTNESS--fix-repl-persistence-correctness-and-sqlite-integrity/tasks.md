# Tasks

## Completed

- [x] Create ticket `GOJA-040-PERSISTENCE-CORRECTNESS`
- [x] Write the persistence correctness analysis/design/implementation guide
- [x] Record an investigation diary
- [x] Validate the ticket with `docmgr doctor`
- [x] Upload the ticket bundle to reMarkable

## Planned implementation work

- [ ] Define the delete/read contract for durable sessions
- [x] Commit 1: hide soft-deleted sessions from list/load/restore/export/history paths
- [x] Replace process-local default session ID allocation with collision-resistant durable IDs
- [ ] Move SQLite integrity configuration to connection-open time
- [ ] Add regression tests for deleted-session behavior
- [ ] Add regression tests for multi-process-safe durable IDs
- [ ] Add regression tests or probes for SQLite foreign key enforcement
- [ ] Keep the implementation diary updated after each slice
- [ ] Commit each stable slice with focused commit messages
