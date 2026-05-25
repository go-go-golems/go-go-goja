#!/usr/bin/env bash
set -euo pipefail
roots=(go-go-goja loupedeck discord-bot css-visual-diff go-minitrace)
patterns='xgoja|jsverbs|RunJS|RunFile|RunString|goja|require\.Registry|cobra\.Command|GlazeCommand|script|sandbox|Runtime|RegisterRuntimeModule|NewRootCommand|RunIntoGlazeProcessor'
for root in "${roots[@]}"; do
  echo "### ${root}"
  if [[ ! -d "$root" ]]; then
    echo "missing directory: $root" >&2
    continue
  fi
  (
    cd "$root"
    printf 'module: '
    grep '^module ' go.mod || true
    rg -n "$patterns" -S \
      --glob '!vendor' --glob '!node_modules' --glob '!dist' --glob '!tmp' --glob '!ttmp/**/sources/**' \
      . || true
  )
  echo
 done
