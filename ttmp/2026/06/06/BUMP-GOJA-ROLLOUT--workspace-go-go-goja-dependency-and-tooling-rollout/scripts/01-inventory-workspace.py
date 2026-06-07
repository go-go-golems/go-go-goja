#!/usr/bin/env python3
"""Inventory Go repositories in the bump-goja workspace for tooling rollout planning.

The script intentionally excludes glazed and go-go-goja because the rollout target is
all other repositories in the workspace. It prints Markdown so the output can be
pasted into the ticket design document.
"""

from __future__ import annotations

import json
import os
import re
import subprocess
from pathlib import Path

WORKSPACE = Path(os.environ.get("WORKSPACE", "/home/manuel/workspaces/2026-06-06/bump-goja"))
EXCLUDED = {"glazed", "go-go-goja"}
GO_GO_GOLEMS_RE = re.compile(r"github\.com/go-go-golems/([A-Za-z0-9_.-]+)")
GOJA_RE = re.compile(r"github\.com/go-go-golems/go-go-goja")
GLAZED_RE = re.compile(r"github\.com/go-go-golems/glazed")
LOGCOPTER_RE = re.compile(r"github\.com/go-go-golems/logcopter")


def run(cmd: list[str], cwd: Path) -> str:
    env = dict(os.environ)
    env["GOWORK"] = "off"
    try:
        return subprocess.check_output(cmd, cwd=cwd, env=env, text=True, stderr=subprocess.DEVNULL).strip()
    except subprocess.CalledProcessError:
        return ""


def read(path: Path) -> str:
    try:
        return path.read_text()
    except FileNotFoundError:
        return ""


def has_target(makefile: str, target: str) -> bool:
    return bool(re.search(rf"(?m)^{re.escape(target)}\s*:", makefile))


def main() -> None:
    repos: list[dict[str, object]] = []
    for go_mod in sorted(WORKSPACE.glob("*/go.mod")):
        repo = go_mod.parent
        name = repo.name
        if name in EXCLUDED:
            continue
        go_mod_text = read(go_mod)
        makefile = read(repo / "Makefile")
        workflows = "\n".join(read(p) for p in sorted((repo / ".github" / "workflows").glob("*.y*ml")))
        lefthook = read(repo / "lefthook.yml") + read(repo / ".lefthook.yml")
        ggg_deps = sorted(set(GO_GO_GOLEMS_RE.findall(go_mod_text)))
        repos.append({
            "repo": name,
            "path": str(repo),
            "module": run(["go", "list", "-m"], repo),
            "go_go_golems_deps": [d for d in ggg_deps if d != name],
            "depends_on_go_go_goja": bool(GOJA_RE.search(go_mod_text)),
            "depends_on_glazed": bool(GLAZED_RE.search(go_mod_text)),
            "has_logcopter_dependency": bool(LOGCOPTER_RE.search(go_mod_text)),
            "has_logcopter_generate_go": (repo / "logcopter_generate.go").exists(),
            "has_generated_logcopter_files": bool(list(repo.glob("**/logcopter.go"))),
            "has_makefile": (repo / "Makefile").exists(),
            "has_bump_target": has_target(makefile, "bump-go-go-golems"),
            "has_glazed_lint_target": has_target(makefile, "glazed-lint"),
            "has_logcopter_check_target": has_target(makefile, "logcopter-check"),
            "ci_mentions_glazed_lint": "glazed-lint" in workflows,
            "ci_mentions_logcopter": "logcopter" in workflows,
            "lefthook_mentions_lint": "lint" in lefthook,
        })

    print("# Workspace rollout inventory\n")
    print(f"Workspace: `{WORKSPACE}`")
    print(f"Excluded: `{', '.join(sorted(EXCLUDED))}`")
    print(f"Repositories inventoried: {len(repos)}\n")
    headers = [
        "repo", "goja", "glazed", "logcopter dep", "logcopter gen", "bump", "glazed-lint", "logcopter-check", "deps",
    ]
    print("| " + " | ".join(headers) + " |")
    print("|" + "|".join(["---"] * len(headers)) + "|")
    for r in repos:
        print("| {repo} | {goja} | {glazed} | {logdep} | {loggen} | {bump} | {glint} | {lcheck} | {deps} |".format(
            repo=r["repo"],
            goja="yes" if r["depends_on_go_go_goja"] else "no",
            glazed="yes" if r["depends_on_glazed"] else "no",
            logdep="yes" if r["has_logcopter_dependency"] else "no",
            loggen="yes" if r["has_logcopter_generate_go"] else "no",
            bump="yes" if r["has_bump_target"] else "no",
            glint="yes" if r["has_glazed_lint_target"] else "no",
            lcheck="yes" if r["has_logcopter_check_target"] else "no",
            deps=", ".join(r["go_go_golems_deps"]) or "—",
        ))
    print("\n## JSON\n")
    print("```json")
    print(json.dumps(repos, indent=2, sort_keys=True))
    print("```")


if __name__ == "__main__":
    main()
