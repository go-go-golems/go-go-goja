#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../../../.." && pwd)"
cd "$repo_root"

echo "# Commands with runtime settings and selected module descriptor usage"
rg -n 'Runtime string|settings\.Runtime|selectedModules|selectedModuleDescriptors|sectionsForRuntimeProfile' pkg/xgoja/app/*.go

echo
echo "# Provider capability iteration sites"
rg -n 'PackageCapabilities|CollectConfigSections|InitRuntimeFromSections|ResolvePackageCapabilities' pkg/xgoja 2>/dev/null || true

echo
echo "# Tests touching runtime overrides / module sections"
rg -n 'RuntimeOverride|runtime override|sectionsForRuntimeProfile|PackageCapabilities' pkg/xgoja/app/*_test.go pkg/xgoja/providerutil/*_test.go
