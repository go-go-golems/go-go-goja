# capability

`capability` provides narrow bearer-token helpers for intentional delegation flows such as:

- organization invite acceptance,
- email verification,
- password reset,
- one-time downloads,
- scoped API tokens.

A capability token is not a general permission system. It is an unguessable token that grants one specific authority for one specific purpose/resource/subject until expiry or revocation.

Rules enforced by `Service.Issue`:

- purpose is required,
- subject or resource is required,
- expiry or TTL is required,
- raw token is returned once,
- only the hash is stored,
- raw token is not included in audit attributes.

The package includes an in-memory store for tests/demos, `capability/sqlstore` for durable SQLite/Postgres-backed token hashes, and a concrete org-invite helper:

```go
issued, err := svc.IssueOrgInvite(ctx, capability.OrgInviteSpec{
    OrgID:     "o1",
    Email:     "new@example.test",
    Role:      "viewer",
    TTL:       7 * 24 * time.Hour,
    CreatedBy: "u1",
})

// Send issued.Token to the invited email address. Do not log it.

accepted, err := svc.AcceptOrgInvite(ctx, issued.Token)
// accepted.OrgID, accepted.Email, accepted.Role can now create the membership.
```

Production hosts can wire the SQL store like this:

```go
store, err := sqlstore.New(sqlstore.Config{DB: db, Dialect: sqlstore.DialectPostgres})
err = store.ApplySchema(ctx) // examples/tests; production can run the DDL through migrations
svc := capability.Service{Store: store, Audit: auditSink}
```

Production stores should make `Redeem` atomic so single-use capabilities cannot be redeemed twice under concurrent requests. `capability/sqlstore` enforces this with transaction-scoped validation and a conditional `used_at IS NULL` update.
