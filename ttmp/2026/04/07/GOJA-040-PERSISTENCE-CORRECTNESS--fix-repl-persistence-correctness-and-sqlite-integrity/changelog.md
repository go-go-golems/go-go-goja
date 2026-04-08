# Changelog

## 2026-04-07

- Created ticket `GOJA-040-PERSISTENCE-CORRECTNESS`.
- Added a detailed design and implementation guide for the persistence correctness PR.
- Captured the three main problem areas: deleted-session visibility, durable session ID collisions, and connection-local SQLite integrity settings.
- Validated the ticket with `docmgr doctor` and uploaded the bundle to reMarkable.
- Implemented the first code slice: hidden-session semantics for deleted durable sessions, plus regression tests in `repldb` and `replapi`.
