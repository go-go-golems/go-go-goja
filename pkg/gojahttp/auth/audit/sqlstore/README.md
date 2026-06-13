# audit/sqlstore

`audit/sqlstore` is a `database/sql` backed implementation of `audit.Store`.
It persists already-normalized `audit.Record` values; callers should use
`audit.Sink{Store: store}` so redaction and request metadata normalization happen
before inserts.

## Common operational queries

Denied or failed route outcomes:

```sql
SELECT created_at, event, outcome, reason, actor_id, tenant_id, resource_type, resource_id, action, request_id
FROM auth_audit_records
WHERE outcome IN ('denied', 'failed')
ORDER BY created_at DESC
LIMIT 100;
```

Authorization denials for a tenant:

```sql
SELECT created_at, event, reason, actor_id, resource_type, resource_id, action, request_id
FROM auth_audit_records
WHERE outcome = 'denied'
  AND tenant_id = $1
ORDER BY created_at DESC
LIMIT 100;
```

Recent writes to a resource:

```sql
SELECT created_at, event, outcome, actor_id, action, request_id
FROM auth_audit_records
WHERE resource_type = $1
  AND resource_id = $2
ORDER BY created_at DESC
LIMIT 100;
```

The helper `QueryByOutcome(ctx, "denied", 100)` exists for examples and smoke
tests. Production applications can query the table directly or build their own
app-specific report layer.
