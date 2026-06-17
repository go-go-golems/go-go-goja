# audit

`audit` provides reusable sinks and normalization for `gojahttp.AuditEvent` values emitted by planned routes.

The package is intentionally storage-agnostic:

- `MemorySink` stores records in memory for tests and demos.
- `LogSink` writes minimal JSON records to a logger for development, omitting request-header-derived metadata, IP information, arbitrary attributes, and free-form error reasons.
- `Sink` writes normalized records to any `Store` implementation.

`Normalizer` maps the runtime event into a storage-friendly `Record` with actor, resource, route, request, and outcome fields.

Secret-looking attributes are redacted recursively. Keys containing values such as `token`, `secret`, `password`, `cookie`, `session`, `authorization`, `credential`, `code`, or `capability` are stored as `[REDACTED]`.

A production application can implement:

```go
type SQLAuditStore struct { /* db handle */ }

func (s SQLAuditStore) InsertAuditRecord(ctx context.Context, record audit.Record) error {
    // INSERT into audit_event (...)
}
```

Then wire it into `gojahttp`:

```go
host := gojahttp.NewHost(gojahttp.HostOptions{
    Auth: gojahttp.AuthOptions{
        Audit: audit.Sink{Store: SQLAuditStore{db: db}},
    },
})
```
