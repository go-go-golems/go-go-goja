#!/usr/bin/env bash
set -euo pipefail

# Inventory the Goja/xgoja-relevant surfaces for the sibling packages targeted by
# XGOJA-007. Run from the add-js-providers workspace root.
roots=(geppetto workspace-manager goja-git go-minitrace loupedeck)
patterns='goja|require\.Registry|RegisterNativeModule|RegisterRuntimeModules|ModuleName|func Register|NativeModule|ModuleLoader|RuntimeModule|runtimebridge|runtimeowner|jsverbs'

for root in "${roots[@]}"; do
  echo "### ${root}"
  if [[ ! -d "${root}" ]]; then
    echo "missing directory: ${root}" >&2
    continue
  fi
  (
    cd "${root}"
    printf 'module: '
    grep '^module ' go.mod || true
    rg -n "${patterns}" -S \
      --glob '!vendor' \
      --glob '!node_modules' \
      --glob '!ttmp' \
      --glob '!dist' \
      --glob '!tmp' \
      . || true
  )
  echo
 done
