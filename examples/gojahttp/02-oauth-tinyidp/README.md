# tiny-idp OAuth planned-route smoke

This focused example exercises an Express-compatible planned OAuth route with a
fake verifier that has the same `programauth.OAuthBearerAuthenticator` contract
as the RFC 7662 tiny-idp verifier.

Run:

```bash
go run .
```

The smoke proves missing, invalid, and valid Bearer credentials are rejected or
accepted before the handler executes, and that the handler receives only the
redacted OAuth subject.
