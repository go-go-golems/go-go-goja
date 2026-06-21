# Tasks

## TODO

- [x] Add tasks here

- [x] Create programmatic auth design ticket
- [x] Collect current auth surface evidence
- [x] Write token and device login implementation guide
- [x] Upload implementation guide bundle to reMarkable
- [x] Download best-practice sources for programmatic auth review
- [x] Write best-practice code review and opinionated JavaScript API design
- [x] Upload revised programmatic auth review bundle to reMarkable
- [x] Promote rate limiting to a planned-route primitive in the programmatic auth review
- [x] Phase 1A: Add RateLimitSpec, RateLimiter, memory limiter, and Enforcer pre/post route enforcement
- [x] Phase 1B: Add Go planned-route rate limit builders and core tests
- [x] Phase 1C: Add Express express.rateLimit(...) builder, .rateLimit(...) route methods, DTS, and integration tests
- [x] Phase 1D: Wire generated hostauth services with a default in-memory RateLimiter and tests
- [x] Phase 1E: Validate focused packages and commit rate limiting implementation
- [x] Phase 2: Add AuthResult and safe ctx.auth projection
- [x] Phase 3: Add typed programmatic grants and first-class agent model
- [x] Phase 4: Add API token issue/list/revoke/authenticate and bearer planned-route auth
- [x] Phase 5: Add auth.agents and auth.tokens fluent JavaScript APIs
- [x] Phase 6: Add route auth restriction builders for agent/session/anyOf
- [x] Phase 7: Add access and rotating refresh token families
- [x] Phase 8: Add device authorization flow and native polling/token handlers
- [x] Phase 9: Add generated examples, smoke tests, help docs, and final reMarkable bundle
- [x] Phase 10: Enforce device approval grant intersection and add regression tests
- [x] Phase 11A: Design SQL-backed programauth store schema and transaction contracts
- [ ] Phase 11B: Implement SQL-backed agent and API-token stores
- [ ] Phase 11C: Implement SQL-backed access/refresh token stores with transactional refresh rotation
- [ ] Phase 11D: Implement SQL-backed device authorization store with atomic approval/consume transitions
- [ ] Phase 11E: Wire generated hostauth configuration to select durable programauth stores
- [ ] Phase 11F: Validate SQL programauth stores and document production migration notes
