# Changelog

## 2026-06-09

- Initial workspace created


## 2026-06-09

Created implementation guide, task list, and initial diary for exposing explicit-only jsverb discovery in xgoja.

### Related Files

- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/ttmp/2026/06/09/xgoja-jsverbs-loader-only-sources--separate-jsverb-command-discovery-from-require-loader-sources/design-doc/01-implementation-guide.md — Ticket implementation guide
- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/ttmp/2026/06/09/xgoja-jsverbs-loader-only-sources--separate-jsverb-command-discovery-from-require-loader-sources/reference/01-investigation-diary.md — Initial diary entry


## 2026-06-09

Implemented clean-break __verb__-only jsverb command discovery and removed public-function compatibility option; targeted xgoja/jsverbs/http tests pass.

### Related Files

- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/pkg/jsverbs/model.go — Removed IncludePublicFunctions scan option
- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/pkg/jsverbs/scan.go — Removed implicit public-function command discovery
- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/pkg/xgoja/providers/http/serve_test.go — Regression coverage for helper modules without helper commands
- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/ttmp/2026/06/09/xgoja-jsverbs-loader-only-sources--separate-jsverb-command-discovery-from-require-loader-sources/reference/01-investigation-diary.md — Implementation diary step 2


## 2026-06-09

Validated ClubMed minitrace-viz with local clean-break xgoja: direct smoke passed and devctl up reached healthy service.

### Related Files

- /home/manuel/workspaces/2026-06-07/club-meetup-site/ClubMedMeetup/minitrace-viz/xgoja.yaml — Dependent app source includes site/server/lib helper files
- /home/manuel/workspaces/2026-06-07/club-meetup-site/ClubMedMeetup/scripts/devctl/clubmed.py — devctl launch/build path validated


## 2026-06-09

Committed clean-break jsverb metadata implementation as 833edd587db205318cc408464053566d74c11b5f and updated diary with validation.

### Related Files

- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/pkg/jsverbs/scan.go — Commit 833edd5 removes implicit public-function discovery
- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/ttmp/2026/06/09/xgoja-jsverbs-loader-only-sources--separate-jsverb-command-discovery-from-require-loader-sources/reference/01-investigation-diary.md — Records commit and validation results

