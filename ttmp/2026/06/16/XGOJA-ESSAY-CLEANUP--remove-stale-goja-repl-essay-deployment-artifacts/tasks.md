# Tasks

## TODO

- [x] Verify live GitOps and cluster state no longer contain goja-essay before deleting remaining source-repo deployment references
- [x] Remove the stale goja-essay GitOps target from deploy/gitops-targets.json
- [x] Remove the goja-repl essay command registration and delete cmd/goja-repl/essay.go
- [x] Delete pkg/replessay/ and essay-specific tests
- [x] Delete the essay React app under web/ if no non-essay surface uses it
- [x] Remove, replace, or defer the root Dockerfile so it no longer runs essay by default
- [x] Delete the essay-specific publish-image workflow; auth-host publishing will be added separately
- [x] Preserve pkg/replapi, pkg/repldb, pkg/replhttp, pkg/replsession, and non-essay goja-repl commands
- [x] Run build/test validation after cleanup
