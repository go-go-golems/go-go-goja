# repldb migration fixtures

`repl-v1.sqlite` is a real SQLite schema-v1 database used by migration tests. It contains:

- session `fixture-session` with persistent-profile metadata;
- evaluation cell 1 and raw replay source;
- one console event;
- one binding and binding version;
- one binding documentation row;
- `repldb_meta.schema_version = 1`.

Treat the fixture as immutable historical input. Opening it with the current binary applies schema v2 (`session_leases`) while preserving these v1 rows. Keep this file and add new fixtures rather than regenerating old versions from the latest schema. Before committing a fixture, run:

```bash
sqlite3 repl-v1.sqlite 'PRAGMA integrity_check;'
```
