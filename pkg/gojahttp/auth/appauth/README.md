# appauth

`appauth` is a small app-owned authorization helper package for `gojahttp` planned routes. It intentionally starts with explicit Go checks instead of a policy engine.

Use it for early monoliths, demos, and production systems whose authorization rules are still readable as normal Go code. Graduate to Casbin, OpenFGA, OPA, Cedar, or Keycloak Authorization Services only when the rules become too large or relationship-heavy for explicit tests and code review.

## Model

The package defines minimal contracts for:

- users,
- tenants,
- memberships,
- resources,
- resource resolution,
- action authorization.

It implements these `gojahttp` interfaces:

```go
appauth.Resolver    -> gojahttp.ResourceResolver
appauth.Authorizer -> gojahttp.Authorizer
```

## Deny-by-default actions

Built-in action constants are intentionally small:

```go
appauth.ActionUserSelfRead
appauth.ActionUserSelfUpdate
appauth.ActionProjectRead
appauth.ActionProjectUpdate
appauth.ActionOrgInvite
```

Unknown actions deny. Missing actors deny. Missing resources deny. Missing tenant roles deny.

## Wiring sketch

```go
store := appauth.NewMemoryStore()
store.AddUser(appauth.User{ID: "u1", KeycloakSub: "..."})
store.AddMembership(appauth.Membership{UserID: "u1", TenantID: "o1", Role: "admin"})
store.AddResource(appauth.Resource{Type: "project", ID: "p1", TenantID: "o1"})

host := gojahttp.NewHost(gojahttp.HostOptions{
    Auth: gojahttp.AuthOptions{
        Authenticator: sessions,
        CSRF:          sessions,
        Resources:     appauth.Resolver{Store: store},
        Authorizer:    appauth.Authorizer{Memberships: store},
    },
})
```

This is a starting point, not a product policy. Real applications should usually wrap or replace `Authorizer` with their own explicit action switch and negative authorization tests.
