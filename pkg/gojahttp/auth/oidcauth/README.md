# oidcauth

`oidcauth` is a provider-neutral OIDC browser-login adapter for `gojahttp` hosts. It uses the standard OIDC discovery/token/JWKS flow through `github.com/coreos/go-oidc/v3/oidc` and `golang.org/x/oauth2`.

## Intended production shape

```text
Browser -> Go /auth/login
  -> OIDC provider Authorization Code + PKCE
  -> Go /auth/callback
  -> verify state, code, ID token, nonce
  -> normalize issuer-scoped OIDC subject into an app user
  -> create server-side app session through sessionauth
  -> browser receives only the app session cookie
```

Do not expose identity-provider access or refresh tokens to browser JavaScript. Planned routes authenticate using the opaque app session cookie through `sessionauth.Manager`.

## Keycloak client settings

Recommended client settings for browser users:

- client type: OIDC
- flow: Authorization Code Flow
- PKCE: enabled / S256
- valid redirect URI: `https://your-app.example/auth/callback`
- web origins: your app origin
- direct access grants / password grant: disabled for browser clients
- implicit flow: disabled
- standard flow: enabled

Use Keycloak for identity, login, MFA, account lifecycle, federation, groups, and coarse roles. Keep application users, tenant memberships, resource ownership, and fine-grained authorization in the Go application.

## Go wiring sketch

```go
sessions, _ := sessionauth.New(sessionauth.Config{
    Store:       productionSessionStore,
    ActorLoader: appActorLoader,
})

handlers, _ := oidcauth.New(ctx, oidcauth.Config{
    IssuerURL:      "https://keycloak.example/realms/app",
    ClientID:       "goja-app",
    ClientSecret:   os.Getenv("KEYCLOAK_CLIENT_SECRET"),
    RedirectURL:    "https://your-app.example/auth/callback",
    AfterLoginURL:  "/",
    AfterLogoutURL: "/",
    SessionManager: sessions,
    UserNormalizer: oidcauth.UserNormalizerFunc(func(ctx context.Context, claims oidcauth.OIDCClaims) (oidcauth.UserSession, error) {
        user, err := users.UpsertFromOIDC(ctx, claims.Subject, claims.Email, claims.EmailVerified)
        if err != nil {
            return oidcauth.UserSession{}, err
        }
        memberships, err := membershipStore.ForUser(ctx, user.ID)
        if err != nil {
            return oidcauth.UserSession{}, err
        }
        return oidcauth.UserSession{
            UserID:        user.ID,
            Email:         user.Email,
            EmailVerified: user.EmailVerified,
            TenantIDs:     memberships.TenantIDs(),
            Claims:        map[string]any{"oidcSubject": claims.Subject},
        }, nil
    }),
})

mux.Handle("GET /auth/login", handlers.LoginHandler())
mux.Handle("GET /auth/callback", handlers.CallbackHandler())
mux.Handle("POST /auth/logout", handlers.LogoutHandler())
```
