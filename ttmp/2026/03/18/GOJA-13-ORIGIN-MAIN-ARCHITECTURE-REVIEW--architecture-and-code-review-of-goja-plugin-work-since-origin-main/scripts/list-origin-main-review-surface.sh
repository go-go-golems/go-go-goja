#!/usr/bin/env bash
set -euo pipefail

git log --oneline origin/main..HEAD
echo
git diff --stat origin/main..HEAD
echo
git diff --dirstat=files,0 origin/main..HEAD
echo
git diff --name-only origin/main..HEAD | grep -v '^ttmp/' || true
