#!/usr/bin/env python3
import argparse
import difflib
import json
import os
import re
import shutil
import subprocess
import sys
import tempfile
from pathlib import Path


def run(cmd, cwd=None, env=None, capture_output=False):
    kwargs = {
        "cwd": cwd,
        "env": env,
        "check": True,
        "text": True,
    }
    if capture_output:
        kwargs["stdout"] = subprocess.PIPE
        kwargs["stderr"] = subprocess.PIPE
    return subprocess.run(cmd, **kwargs)


def load_targets(config_path: Path):
    payload = json.loads(config_path.read_text())
    targets = payload.get("targets", [])
    if not isinstance(targets, list) or not targets:
        raise ValueError(f"no targets defined in {config_path}")
    required = {"name", "gitops_repo", "gitops_branch", "manifest_path", "container_name"}
    for target in targets:
        missing = sorted(required - set(target))
        if missing:
            raise ValueError(f"target {target!r} is missing keys: {', '.join(missing)}")
    return targets


def select_targets(targets, target_name: str | None, all_targets: bool):
    if target_name and all_targets:
        raise ValueError("use either --target or --all-targets, not both")
    if all_targets:
        return targets
    if target_name:
        for target in targets:
            if target["name"] == target_name:
                return [target]
        raise ValueError(f"target {target_name!r} not found")
    raise ValueError("select a target with --target or pass --all-targets")


def sanitize_ref_fragment(value: str) -> str:
    return re.sub(r"[^a-zA-Z0-9._/-]+", "-", value).strip("-")


def image_tag_fragment(image: str) -> str:
    if ":" in image:
        return sanitize_ref_fragment(image.split(":", 1)[1])
    return sanitize_ref_fragment(image.rsplit("/", 1)[-1])


def image_repo_fragment(image: str) -> str:
    repo = image.rsplit("/", 1)[-1]
    return sanitize_ref_fragment(repo.split(":", 1)[0])


def patch_manifest_image(manifest_path: Path, container_name: str, image: str):
    original = manifest_path.read_text()
    lines = original.splitlines(keepends=True)

    containers_indent = None
    current_container = None
    target_container_indent = None
    replaced = False

    for idx, line in enumerate(lines):
        stripped = line.strip()
        if not stripped or stripped.startswith("#"):
            continue

        indent = len(line) - len(line.lstrip(" "))

        if stripped == "containers:":
            containers_indent = indent
            current_container = None
            target_container_indent = None
            continue

        if containers_indent is not None and indent <= containers_indent:
            containers_indent = None
            current_container = None
            target_container_indent = None

        if containers_indent is None:
            continue

        if stripped.startswith("- name:") and indent == containers_indent + 2:
            current_container = stripped.split(":", 1)[1].strip()
            target_container_indent = indent
            continue

        if current_container != container_name or target_container_indent is None:
            continue

        if indent == target_container_indent + 2 and stripped.startswith("image:"):
            prefix = line[: len(line) - len(line.lstrip(" "))]
            suffix = "\n" if line.endswith("\n") else ""
            existing = stripped.split(":", 1)[1].strip()
            if existing == image:
                return False, original, original
            lines[idx] = f"{prefix}image: {image}{suffix}"
            replaced = True
            break

    if not replaced:
        raise ValueError(
            f"could not find image field for container {container_name!r} in {manifest_path}"
        )

    updated = "".join(lines)
    manifest_path.write_text(updated)
    return True, original, updated


def ensure_git_identity(repo_dir: Path):
    author_name = os.environ.get("GITOPS_PR_GIT_AUTHOR_NAME", "github-actions[bot]")
    author_email = os.environ.get(
        "GITOPS_PR_GIT_AUTHOR_EMAIL",
        "41898282+github-actions[bot]@users.noreply.github.com",
    )
    run(["git", "config", "user.name", author_name], cwd=repo_dir)
    run(["git", "config", "user.email", author_email], cwd=repo_dir)


def open_or_update_pr(repo_dir: Path, target: dict, branch_name: str, title: str, body: str):
    repo = target["gitops_repo"]
    existing = run(
        [
            "gh",
            "pr",
            "list",
            "--repo",
            repo,
            "--head",
            branch_name,
            "--state",
            "open",
            "--json",
            "number",
            "--jq",
            ".[0].number // empty",
        ],
        cwd=repo_dir,
        capture_output=True,
    ).stdout.strip()
    if existing:
        print(f"PR already exists for {target['name']}: #{existing}")
        return

    run(
        [
            "gh",
            "pr",
            "create",
            "--repo",
            repo,
            "--base",
            target["gitops_branch"],
            "--head",
            branch_name,
            "--title",
            title,
            "--body",
            body,
        ],
        cwd=repo_dir,
    )


def clone_repo(target: dict, token: str):
    temp_dir = Path(tempfile.mkdtemp(prefix=f"gitops-{target['name']}-"))
    remote_url = f"https://x-access-token:{token}@github.com/{target['gitops_repo']}.git"
    run(
        [
            "git",
            "clone",
            "--depth",
            "1",
            "--branch",
            target["gitops_branch"],
            remote_url,
            str(temp_dir),
        ]
    )
    return temp_dir


def render_diff(path: Path, original: str, updated: str):
    diff = difflib.unified_diff(
        original.splitlines(),
        updated.splitlines(),
        fromfile=f"{path} (before)",
        tofile=f"{path} (after)",
        lineterm="",
    )
    return "\n".join(diff)


def build_pr_body(target: dict, image: str):
    source_repo = os.environ.get("GITHUB_REPOSITORY", "")
    source_sha = os.environ.get("GITHUB_SHA", "")
    workflow_run = os.environ.get("GITHUB_SERVER_URL", "https://github.com")
    run_id = os.environ.get("GITHUB_RUN_ID", "")

    lines = [
        f"Automated image bump for `{target['name']}`.",
        "",
        f"- Image: `{image}`",
        f"- Target manifest: `{target['manifest_path']}`",
    ]
    if source_repo and source_sha:
        lines.append(f"- Source commit: `{source_repo}@{source_sha}`")
    if source_repo and run_id:
        lines.append(f"- Workflow run: {workflow_run}/{source_repo}/actions/runs/{run_id}")
    lines.extend(
        [
            "",
            "Rollback:",
            f"- revert this PR merge, or open a new PR that resets `{target['container_name']}` to the previous immutable SHA tag",
        ]
    )
    return "\n".join(lines)


def main():
    parser = argparse.ArgumentParser(description="Open GitOps PRs for published images")
    parser.add_argument("--config", default="deploy/gitops-targets.json")
    parser.add_argument("--target")
    parser.add_argument("--all-targets", action="store_true")
    parser.add_argument("--image", required=True)
    parser.add_argument("--gitops-repo-dir", help="Use an existing local GitOps checkout for dry-run validation")
    parser.add_argument("--dry-run", action="store_true")
    parser.add_argument("--push", action="store_true")
    parser.add_argument("--open-pr", action="store_true")
    args = parser.parse_args()

    config_path = Path(args.config).resolve()
    targets = select_targets(load_targets(config_path), args.target, args.all_targets)

    if args.gitops_repo_dir and len(targets) != 1:
        raise ValueError("--gitops-repo-dir currently supports exactly one target")

    token = os.environ.get("GH_TOKEN", "")
    if (args.push or args.open_pr) and not token and not args.gitops_repo_dir:
        raise ValueError("GH_TOKEN is required when --push or --open-pr is used")
    if args.open_pr and not args.push:
        raise ValueError("--open-pr requires --push")

    app_name = image_repo_fragment(args.image)
    image_tag = image_tag_fragment(args.image)

    for target in targets:
        cleanup_dir = None
        if args.gitops_repo_dir:
            repo_dir = Path(args.gitops_repo_dir).resolve()
        else:
            repo_dir = clone_repo(target, token)
            cleanup_dir = repo_dir

        try:
            manifest_path = repo_dir / target["manifest_path"]
            if not manifest_path.exists():
                raise FileNotFoundError(f"manifest {manifest_path} does not exist")

            changed, original, updated = patch_manifest_image(
                manifest_path,
                target["container_name"],
                args.image,
            )
            if not changed:
                print(f"{target['name']}: manifest already uses {args.image}")
                continue

            print(f"{target['name']}: patched {target['manifest_path']}")
            print(render_diff(manifest_path, original, updated))

            if args.dry_run:
                continue

            branch_name = sanitize_ref_fragment(
                f"ci/{app_name}/{target['name']}/{image_tag}"
            )
            commit_title = f"chore({app_name}): bump {target['name']} image to {image_tag}"
            pr_title = commit_title
            pr_body = build_pr_body(target, args.image)

            ensure_git_identity(repo_dir)
            run(["git", "checkout", "-b", branch_name], cwd=repo_dir)
            run(["git", "add", target["manifest_path"]], cwd=repo_dir)
            run(["git", "commit", "-m", commit_title], cwd=repo_dir)

            if args.push:
                run(["git", "push", "--set-upstream", "origin", branch_name], cwd=repo_dir)
            if args.open_pr:
                open_or_update_pr(repo_dir, target, branch_name, pr_title, pr_body)
        finally:
            if cleanup_dir is not None:
                shutil.rmtree(cleanup_dir, ignore_errors=True)


if __name__ == "__main__":
    try:
        main()
    except Exception as exc:
        print(f"error: {exc}", file=sys.stderr)
        sys.exit(1)
