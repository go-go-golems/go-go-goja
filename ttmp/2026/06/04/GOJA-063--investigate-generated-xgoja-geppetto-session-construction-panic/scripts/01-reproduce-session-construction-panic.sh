#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT=${REPO_ROOT:-/home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja}
GEPPETTO_ROOT=${GEPPETTO_ROOT:-/home/manuel/workspaces/2026-06-03/goja-runtime-flags/geppetto}
PINOCCHIO_ROOT=${PINOCCHIO_ROOT:-/home/manuel/workspaces/2026-06-03/goja-runtime-flags/pinocchio}
WORK_DIR=${WORK_DIR:-/tmp/xgoja-geppetto-session-panic-repro}
PROFILE_REGISTRIES=${PROFILE_REGISTRIES:-$PINOCCHIO_ROOT/examples/js/profiles/basic.yaml}
PROFILE=${PROFILE:-assistant}
SESSION=${SESSION:-xgoja-pinocchio-profile-smoke}

rm -rf "$WORK_DIR"
mkdir -p "$WORK_DIR/verbs"

cat > "$WORK_DIR/xgoja.yaml" <<EOF
name: geppetto-session-panic-repro
appName: geppetto-session-panic-repro
target:
  kind: xgoja
  output: $WORK_DIR/geppetto-session-panic-repro
packages:
  - id: geppetto
    import: github.com/go-go-golems/geppetto/pkg/js/modules/geppetto/provider
    replace: $GEPPETTO_ROOT
runtimes:
  main:
    modules:
      - package: geppetto
        name: geppetto
        as: geppetto
commands:
  jsverbs:
    enabled: true
    runtime: main
    name: verbs
jsverbs:
  - id: local
    path: ./verbs
    embed: true
EOF

cat > "$WORK_DIR/verbs/profile_smoke.js" <<'EOF'
__package__({ name: "pinocchio" });

function profileSmoke(sessionId) {
  const gp = require("geppetto");
  const settings = gp.inferenceProfiles.resolve();
  const snapshot = settings.toJSON();
  const agent = gp.agent()
    .name("xgoja-pinocchio-profile-smoke")
    .inference(settings)
    .build();
  const session = agent.session().id(sessionId).build();

  return {
    profile: snapshot.provenance?.profileSlug || "",
    registry: snapshot.provenance?.registrySlug || "",
    model: snapshot.chat?.engine || "",
    apiType: snapshot.chat?.api_type || "",
    session: session.id(),
    hasSessionNext: typeof session.next === "function",
  };
}

__verb__("profileSmoke", {
  name: "profile-smoke",
  short: "Reproduce no-inference Geppetto session construction panic",
  fields: {
    sessionId: {
      argument: true,
      default: "xgoja-pinocchio-profile-smoke",
      help: "Session ID"
    }
  }
});
EOF

cd "$REPO_ROOT"
go run ./cmd/xgoja build \
  -f "$WORK_DIR/xgoja.yaml" \
  --xgoja-replace "$REPO_ROOT" \
  --keep-work \
  --work-dir "$WORK_DIR/work"

set +e
"$WORK_DIR/geppetto-session-panic-repro" verbs pinocchio profile-smoke "$SESSION" \
  --profile-registries "$PROFILE_REGISTRIES" \
  --profile "$PROFILE" \
  --output json
status=$?
set -e

echo "repro_exit_status=$status"
exit "$status"
