---
Title: Workspace rollout inventory
Ticket: BUMP-GOJA-ROLLOUT
Status: active
Topics:
    - go
    - tooling
    - maintenance
DocType: reference
Intent: short-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Generated inventory of target repositories and tooling gaps."
LastUpdated: 2026-06-06T23:30:00-04:00
WhatFor: "Use as evidence for repository ordering."
WhenToUse: "Refresh before continuing the rollout."
---

# Workspace rollout inventory

Workspace: `/home/manuel/workspaces/2026-06-06/bump-goja`
Excluded: `glazed, go-go-goja`
Repositories inventoried: 20

| repo | goja | glazed | logcopter dep | logcopter gen | bump | glazed-lint | logcopter-check | deps |
|---|---|---|---|---|---|---|---|---|
| cozodb-goja | no | no | no | no | no | no | no | XXX |
| css-visual-diff | yes | yes | yes | yes | yes | yes | yes | geppetto, glazed, go-go-goja, logcopter, pinocchio |
| discord-bot | yes | yes | yes | yes | yes | yes | yes | glazed, go-go-goja, logcopter |
| go-go-app-inventory | yes | yes | yes | yes | no | yes | yes | geppetto, glazed, go-go-goja, go-go-os-backend, go-go-os-chat, logcopter, pinocchio, plz-confirm |
| go-go-gepa | yes | yes | yes | yes | no | no | yes | clay, geppetto, glazed, go-go-goja, go-go-os-backend, logcopter |
| go-go-host | yes | yes | yes | yes | yes | yes | yes | glazed, go-go-goja, logcopter |
| go-go-os-backend | yes | no | yes | yes | yes | yes | yes | go-go-goja, logcopter |
| go-minitrace | yes | yes | yes | yes | yes | yes | yes | clay, glazed, go-go-goja, logcopter |
| goja-git | yes | yes | yes | yes | yes | yes | yes | glazed, go-go-goja, logcopter |
| goja-github-actions | yes | yes | yes | yes | no | no | yes | glazed, go-go-goja, logcopter |
| goja-text | yes | yes | yes | yes | yes | no | yes | glazed, go-go-goja, logcopter, sanitize |
| jesus | yes | yes | yes | yes | no | yes | yes | clay, geppetto, glazed, go-go-goja, go-go-mcp, logcopter, pinocchio |
| js-analyzer | yes | yes | yes | no | no | yes | yes | glazed, go-go-goja, logcopter |
| loupedeck | yes | yes | yes | yes | yes | yes | yes | glazed, go-go-goja, logcopter |
| pinocchio | yes | yes | yes | yes | yes | yes | yes | bobatea, clay, geppetto, glazed, go-go-goja, logcopter, sanitize, sessionstream, uhoh |
| plz-confirm | yes | yes | yes | yes | yes | yes | yes | glazed, go-go-goja, logcopter |
| scraper | yes | yes | yes | yes | yes | no | yes | glazed, go-go-goja, logcopter, sessionstream |
| smailnail | yes | yes | yes | yes | no | yes | yes | clay, geppetto, glazed, go-go-goja, go-go-mcp, logcopter |
| vm-system | yes | yes | yes | yes | yes | yes | yes | glazed, go-go-goja, logcopter |
| workspace-manager | yes | yes | yes | yes | yes | yes | yes | clay, glazed, go-go-goja, logcopter |

## JSON

```json
[
  {
    "ci_mentions_glazed_lint": false,
    "ci_mentions_logcopter": false,
    "depends_on_glazed": false,
    "depends_on_go_go_goja": false,
    "go_go_golems_deps": [
      "XXX"
    ],
    "has_bump_target": false,
    "has_generated_logcopter_files": false,
    "has_glazed_lint_target": false,
    "has_logcopter_check_target": false,
    "has_logcopter_dependency": false,
    "has_logcopter_generate_go": false,
    "has_makefile": true,
    "lefthook_mentions_lint": true,
    "module": "github.com/go-go-golems/XXX",
    "path": "/home/manuel/workspaces/2026-06-06/bump-goja/cozodb-goja",
    "repo": "cozodb-goja"
  },
  {
    "ci_mentions_glazed_lint": true,
    "ci_mentions_logcopter": false,
    "depends_on_glazed": true,
    "depends_on_go_go_goja": true,
    "go_go_golems_deps": [
      "geppetto",
      "glazed",
      "go-go-goja",
      "logcopter",
      "pinocchio"
    ],
    "has_bump_target": true,
    "has_generated_logcopter_files": true,
    "has_glazed_lint_target": true,
    "has_logcopter_check_target": true,
    "has_logcopter_dependency": true,
    "has_logcopter_generate_go": true,
    "has_makefile": true,
    "lefthook_mentions_lint": true,
    "module": "github.com/go-go-golems/css-visual-diff",
    "path": "/home/manuel/workspaces/2026-06-06/bump-goja/css-visual-diff",
    "repo": "css-visual-diff"
  },
  {
    "ci_mentions_glazed_lint": true,
    "ci_mentions_logcopter": false,
    "depends_on_glazed": true,
    "depends_on_go_go_goja": true,
    "go_go_golems_deps": [
      "glazed",
      "go-go-goja",
      "logcopter"
    ],
    "has_bump_target": true,
    "has_generated_logcopter_files": true,
    "has_glazed_lint_target": true,
    "has_logcopter_check_target": true,
    "has_logcopter_dependency": true,
    "has_logcopter_generate_go": true,
    "has_makefile": true,
    "lefthook_mentions_lint": true,
    "module": "github.com/go-go-golems/discord-bot",
    "path": "/home/manuel/workspaces/2026-06-06/bump-goja/discord-bot",
    "repo": "discord-bot"
  },
  {
    "ci_mentions_glazed_lint": true,
    "ci_mentions_logcopter": false,
    "depends_on_glazed": true,
    "depends_on_go_go_goja": true,
    "go_go_golems_deps": [
      "geppetto",
      "glazed",
      "go-go-goja",
      "go-go-os-backend",
      "go-go-os-chat",
      "logcopter",
      "pinocchio",
      "plz-confirm"
    ],
    "has_bump_target": false,
    "has_generated_logcopter_files": true,
    "has_glazed_lint_target": true,
    "has_logcopter_check_target": true,
    "has_logcopter_dependency": true,
    "has_logcopter_generate_go": true,
    "has_makefile": true,
    "lefthook_mentions_lint": true,
    "module": "github.com/go-go-golems/go-go-app-inventory",
    "path": "/home/manuel/workspaces/2026-06-06/bump-goja/go-go-app-inventory",
    "repo": "go-go-app-inventory"
  },
  {
    "ci_mentions_glazed_lint": false,
    "ci_mentions_logcopter": false,
    "depends_on_glazed": true,
    "depends_on_go_go_goja": true,
    "go_go_golems_deps": [
      "clay",
      "geppetto",
      "glazed",
      "go-go-goja",
      "go-go-os-backend",
      "logcopter"
    ],
    "has_bump_target": false,
    "has_generated_logcopter_files": true,
    "has_glazed_lint_target": false,
    "has_logcopter_check_target": true,
    "has_logcopter_dependency": true,
    "has_logcopter_generate_go": true,
    "has_makefile": true,
    "lefthook_mentions_lint": true,
    "module": "github.com/go-go-golems/go-go-gepa",
    "path": "/home/manuel/workspaces/2026-06-06/bump-goja/go-go-gepa",
    "repo": "go-go-gepa"
  },
  {
    "ci_mentions_glazed_lint": false,
    "ci_mentions_logcopter": false,
    "depends_on_glazed": true,
    "depends_on_go_go_goja": true,
    "go_go_golems_deps": [
      "glazed",
      "go-go-goja",
      "logcopter"
    ],
    "has_bump_target": true,
    "has_generated_logcopter_files": true,
    "has_glazed_lint_target": true,
    "has_logcopter_check_target": true,
    "has_logcopter_dependency": true,
    "has_logcopter_generate_go": true,
    "has_makefile": true,
    "lefthook_mentions_lint": true,
    "module": "github.com/go-go-golems/go-go-host",
    "path": "/home/manuel/workspaces/2026-06-06/bump-goja/go-go-host",
    "repo": "go-go-host"
  },
  {
    "ci_mentions_glazed_lint": false,
    "ci_mentions_logcopter": false,
    "depends_on_glazed": false,
    "depends_on_go_go_goja": true,
    "go_go_golems_deps": [
      "go-go-goja",
      "logcopter"
    ],
    "has_bump_target": true,
    "has_generated_logcopter_files": true,
    "has_glazed_lint_target": true,
    "has_logcopter_check_target": true,
    "has_logcopter_dependency": true,
    "has_logcopter_generate_go": true,
    "has_makefile": true,
    "lefthook_mentions_lint": true,
    "module": "github.com/go-go-golems/go-go-os-backend",
    "path": "/home/manuel/workspaces/2026-06-06/bump-goja/go-go-os-backend",
    "repo": "go-go-os-backend"
  },
  {
    "ci_mentions_glazed_lint": true,
    "ci_mentions_logcopter": false,
    "depends_on_glazed": true,
    "depends_on_go_go_goja": true,
    "go_go_golems_deps": [
      "clay",
      "glazed",
      "go-go-goja",
      "logcopter"
    ],
    "has_bump_target": true,
    "has_generated_logcopter_files": true,
    "has_glazed_lint_target": true,
    "has_logcopter_check_target": true,
    "has_logcopter_dependency": true,
    "has_logcopter_generate_go": true,
    "has_makefile": true,
    "lefthook_mentions_lint": true,
    "module": "github.com/go-go-golems/go-minitrace",
    "path": "/home/manuel/workspaces/2026-06-06/bump-goja/go-minitrace",
    "repo": "go-minitrace"
  },
  {
    "ci_mentions_glazed_lint": true,
    "ci_mentions_logcopter": false,
    "depends_on_glazed": true,
    "depends_on_go_go_goja": true,
    "go_go_golems_deps": [
      "glazed",
      "go-go-goja",
      "logcopter"
    ],
    "has_bump_target": true,
    "has_generated_logcopter_files": true,
    "has_glazed_lint_target": true,
    "has_logcopter_check_target": true,
    "has_logcopter_dependency": true,
    "has_logcopter_generate_go": true,
    "has_makefile": true,
    "lefthook_mentions_lint": true,
    "module": "github.com/go-go-golems/goja-git",
    "path": "/home/manuel/workspaces/2026-06-06/bump-goja/goja-git",
    "repo": "goja-git"
  },
  {
    "ci_mentions_glazed_lint": false,
    "ci_mentions_logcopter": false,
    "depends_on_glazed": true,
    "depends_on_go_go_goja": true,
    "go_go_golems_deps": [
      "glazed",
      "go-go-goja",
      "logcopter"
    ],
    "has_bump_target": false,
    "has_generated_logcopter_files": true,
    "has_glazed_lint_target": false,
    "has_logcopter_check_target": true,
    "has_logcopter_dependency": true,
    "has_logcopter_generate_go": true,
    "has_makefile": true,
    "lefthook_mentions_lint": true,
    "module": "github.com/go-go-golems/goja-github-actions",
    "path": "/home/manuel/workspaces/2026-06-06/bump-goja/goja-github-actions",
    "repo": "goja-github-actions"
  },
  {
    "ci_mentions_glazed_lint": false,
    "ci_mentions_logcopter": true,
    "depends_on_glazed": true,
    "depends_on_go_go_goja": true,
    "go_go_golems_deps": [
      "glazed",
      "go-go-goja",
      "logcopter",
      "sanitize"
    ],
    "has_bump_target": true,
    "has_generated_logcopter_files": true,
    "has_glazed_lint_target": false,
    "has_logcopter_check_target": true,
    "has_logcopter_dependency": true,
    "has_logcopter_generate_go": true,
    "has_makefile": true,
    "lefthook_mentions_lint": true,
    "module": "github.com/go-go-golems/goja-text",
    "path": "/home/manuel/workspaces/2026-06-06/bump-goja/goja-text",
    "repo": "goja-text"
  },
  {
    "ci_mentions_glazed_lint": true,
    "ci_mentions_logcopter": false,
    "depends_on_glazed": true,
    "depends_on_go_go_goja": true,
    "go_go_golems_deps": [
      "clay",
      "geppetto",
      "glazed",
      "go-go-goja",
      "go-go-mcp",
      "logcopter",
      "pinocchio"
    ],
    "has_bump_target": false,
    "has_generated_logcopter_files": true,
    "has_glazed_lint_target": true,
    "has_logcopter_check_target": true,
    "has_logcopter_dependency": true,
    "has_logcopter_generate_go": true,
    "has_makefile": true,
    "lefthook_mentions_lint": true,
    "module": "github.com/go-go-golems/jesus",
    "path": "/home/manuel/workspaces/2026-06-06/bump-goja/jesus",
    "repo": "jesus"
  },
  {
    "ci_mentions_glazed_lint": false,
    "ci_mentions_logcopter": false,
    "depends_on_glazed": true,
    "depends_on_go_go_goja": true,
    "go_go_golems_deps": [
      "glazed",
      "go-go-goja",
      "logcopter"
    ],
    "has_bump_target": false,
    "has_generated_logcopter_files": true,
    "has_glazed_lint_target": true,
    "has_logcopter_check_target": true,
    "has_logcopter_dependency": true,
    "has_logcopter_generate_go": false,
    "has_makefile": true,
    "lefthook_mentions_lint": false,
    "module": "github.com/go-go-golems/js-analyzer",
    "path": "/home/manuel/workspaces/2026-06-06/bump-goja/js-analyzer",
    "repo": "js-analyzer"
  },
  {
    "ci_mentions_glazed_lint": true,
    "ci_mentions_logcopter": false,
    "depends_on_glazed": true,
    "depends_on_go_go_goja": true,
    "go_go_golems_deps": [
      "glazed",
      "go-go-goja",
      "logcopter"
    ],
    "has_bump_target": true,
    "has_generated_logcopter_files": true,
    "has_glazed_lint_target": true,
    "has_logcopter_check_target": true,
    "has_logcopter_dependency": true,
    "has_logcopter_generate_go": true,
    "has_makefile": true,
    "lefthook_mentions_lint": true,
    "module": "github.com/go-go-golems/loupedeck",
    "path": "/home/manuel/workspaces/2026-06-06/bump-goja/loupedeck",
    "repo": "loupedeck"
  },
  {
    "ci_mentions_glazed_lint": true,
    "ci_mentions_logcopter": true,
    "depends_on_glazed": true,
    "depends_on_go_go_goja": true,
    "go_go_golems_deps": [
      "bobatea",
      "clay",
      "geppetto",
      "glazed",
      "go-go-goja",
      "logcopter",
      "sanitize",
      "sessionstream",
      "uhoh"
    ],
    "has_bump_target": true,
    "has_generated_logcopter_files": true,
    "has_glazed_lint_target": true,
    "has_logcopter_check_target": true,
    "has_logcopter_dependency": true,
    "has_logcopter_generate_go": true,
    "has_makefile": true,
    "lefthook_mentions_lint": true,
    "module": "github.com/go-go-golems/pinocchio",
    "path": "/home/manuel/workspaces/2026-06-06/bump-goja/pinocchio",
    "repo": "pinocchio"
  },
  {
    "ci_mentions_glazed_lint": false,
    "ci_mentions_logcopter": false,
    "depends_on_glazed": true,
    "depends_on_go_go_goja": true,
    "go_go_golems_deps": [
      "glazed",
      "go-go-goja",
      "logcopter"
    ],
    "has_bump_target": true,
    "has_generated_logcopter_files": true,
    "has_glazed_lint_target": true,
    "has_logcopter_check_target": true,
    "has_logcopter_dependency": true,
    "has_logcopter_generate_go": true,
    "has_makefile": true,
    "lefthook_mentions_lint": true,
    "module": "github.com/go-go-golems/plz-confirm",
    "path": "/home/manuel/workspaces/2026-06-06/bump-goja/plz-confirm",
    "repo": "plz-confirm"
  },
  {
    "ci_mentions_glazed_lint": false,
    "ci_mentions_logcopter": true,
    "depends_on_glazed": true,
    "depends_on_go_go_goja": true,
    "go_go_golems_deps": [
      "glazed",
      "go-go-goja",
      "logcopter",
      "sessionstream"
    ],
    "has_bump_target": true,
    "has_generated_logcopter_files": true,
    "has_glazed_lint_target": false,
    "has_logcopter_check_target": true,
    "has_logcopter_dependency": true,
    "has_logcopter_generate_go": true,
    "has_makefile": true,
    "lefthook_mentions_lint": true,
    "module": "github.com/go-go-golems/scraper",
    "path": "/home/manuel/workspaces/2026-06-06/bump-goja/scraper",
    "repo": "scraper"
  },
  {
    "ci_mentions_glazed_lint": true,
    "ci_mentions_logcopter": false,
    "depends_on_glazed": true,
    "depends_on_go_go_goja": true,
    "go_go_golems_deps": [
      "clay",
      "geppetto",
      "glazed",
      "go-go-goja",
      "go-go-mcp",
      "logcopter"
    ],
    "has_bump_target": false,
    "has_generated_logcopter_files": true,
    "has_glazed_lint_target": true,
    "has_logcopter_check_target": true,
    "has_logcopter_dependency": true,
    "has_logcopter_generate_go": true,
    "has_makefile": true,
    "lefthook_mentions_lint": true,
    "module": "github.com/go-go-golems/smailnail",
    "path": "/home/manuel/workspaces/2026-06-06/bump-goja/smailnail",
    "repo": "smailnail"
  },
  {
    "ci_mentions_glazed_lint": false,
    "ci_mentions_logcopter": false,
    "depends_on_glazed": true,
    "depends_on_go_go_goja": true,
    "go_go_golems_deps": [
      "glazed",
      "go-go-goja",
      "logcopter"
    ],
    "has_bump_target": true,
    "has_generated_logcopter_files": true,
    "has_glazed_lint_target": true,
    "has_logcopter_check_target": true,
    "has_logcopter_dependency": true,
    "has_logcopter_generate_go": true,
    "has_makefile": true,
    "lefthook_mentions_lint": false,
    "module": "github.com/go-go-golems/vm-system",
    "path": "/home/manuel/workspaces/2026-06-06/bump-goja/vm-system",
    "repo": "vm-system"
  },
  {
    "ci_mentions_glazed_lint": true,
    "ci_mentions_logcopter": false,
    "depends_on_glazed": true,
    "depends_on_go_go_goja": true,
    "go_go_golems_deps": [
      "clay",
      "glazed",
      "go-go-goja",
      "logcopter"
    ],
    "has_bump_target": true,
    "has_generated_logcopter_files": true,
    "has_glazed_lint_target": true,
    "has_logcopter_check_target": true,
    "has_logcopter_dependency": true,
    "has_logcopter_generate_go": true,
    "has_makefile": true,
    "lefthook_mentions_lint": true,
    "module": "github.com/go-go-golems/workspace-manager",
    "path": "/home/manuel/workspaces/2026-06-06/bump-goja/workspace-manager",
    "repo": "workspace-manager"
  }
]
```
