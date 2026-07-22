#!/usr/bin/env python3
"""Drive generated example 21 against the local Keycloak/Postgres compose stack.

This script intentionally seeds demo appauth rows after login. The generated
host's generic OIDC normalizer upserts the user and projects existing
memberships; it must not grant the demo org membership itself.
"""

from __future__ import annotations

import argparse
import importlib.util
import json
import subprocess
import sys
import urllib.parse
import urllib.request
from http.cookiejar import CookieJar
from pathlib import Path


def load_helper(repo_root: Path):
    helper_path = repo_root / "examples/xgoja/19-express-keycloak-auth-host/scripts/keycloak_smoke.py"
    spec = importlib.util.spec_from_file_location("keycloak_smoke_helper", helper_path)
    if spec is None or spec.loader is None:
        raise RuntimeError(f"could not load helper from {helper_path}")
    helper = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(helper)
    return helper


def sql_literal(value: str) -> str:
    return "'" + value.replace("'", "''") + "'"


def psql(compose_file: Path, sql: str) -> str:
    cmd = [
        "docker",
        "compose",
        "-f",
        str(compose_file),
        "exec",
        "-T",
        "postgres",
        "psql",
        "-U",
        "goja",
        "-d",
        "goja_auth",
        "-tAc",
        sql,
    ]
    return subprocess.check_output(cmd, text=True).strip()


def main() -> int:
    parser = argparse.ArgumentParser(description="Smoke-test generated example 21 against local compose Keycloak")
    parser.add_argument("--repo-root", default=str(Path(__file__).resolve().parents[3]))
    parser.add_argument("--compose-file", default="")
    parser.add_argument("--base-url", default="http://127.0.0.1:8790")
    parser.add_argument("--username", default="demo@example.test")
    parser.add_argument("--password", default="demo-password")
    args = parser.parse_args()

    repo_root = Path(args.repo_root).resolve()
    compose_file = Path(args.compose_file).resolve() if args.compose_file else repo_root / "examples/xgoja/19-express-keycloak-auth-host/docker-compose.yml"
    helper = load_helper(repo_root)
    base_url = args.base_url.rstrip("/")

    jar = CookieJar()
    opener = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(jar))
    opener.addheaders = [("User-Agent", "go-go-goja-generated-keycloak-smoke/1.0")]

    status, body, _ = helper.request(opener, urllib.parse.urljoin(base_url, "/healthz"))
    helper.require_status("public health", status, 200, body)

    status, body, _ = helper.request(opener, urllib.parse.urljoin(base_url, "/me"))
    helper.require_status("me before login", status, 401, body)

    helper.login(opener, jar, base_url, args.username, args.password)

    status, body, _ = helper.request(opener, urllib.parse.urljoin(base_url, "/me"))
    helper.require_status("me after login", status, 200, body)
    me = helper.parse_json("me after login", body)
    actor_id = me.get("id")
    if not actor_id:
        raise RuntimeError(f"/me did not include actor id: {me!r}")

    # Seed demo authorization data externally; generic hostauth deliberately
    # does not grant demo memberships during OIDC normalization.
    psql(compose_file, "INSERT INTO auth_app_tenants (id, slug, name) VALUES ('o1','demo','Demo Org') ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name")
    psql(compose_file, "INSERT INTO auth_app_resources (type, id, name, tenant_id, owner_id, claims_json) VALUES ('org','o1','Demo Org','o1','', '{}'), ('project','p1','Docker Compose Project','o1','', '{}') ON CONFLICT (type, id) DO UPDATE SET name = EXCLUDED.name, tenant_id = EXCLUDED.tenant_id")
    psql(compose_file, f"INSERT INTO auth_app_memberships (user_id, tenant_id, role) VALUES ({sql_literal(actor_id)}, 'o1', 'admin') ON CONFLICT (user_id, tenant_id, role) DO NOTHING")
    print(f"ok seeded demo appauth rows for actor {actor_id}")

    status, body, _ = helper.request(opener, urllib.parse.urljoin(base_url, "/auth/session"))
    helper.require_status("session after login", status, 200, body)
    session = helper.parse_json("session after login", body)
    csrf = session.get("csrfToken")
    if not csrf:
        raise RuntimeError(f"/auth/session did not include csrfToken: {session!r}")

    project_url = urllib.parse.urljoin(base_url, "/orgs/o1/projects/p1")
    status, body, _ = helper.request(opener, project_url, method="PATCH")
    helper.require_status("project missing csrf", status, 403, body)

    status, body, _ = helper.request(opener, project_url, method="PATCH", headers={"X-CSRF-Token": csrf})
    helper.require_status("project update", status, 200, body)
    update = helper.parse_json("project update", body)
    if update.get("updated") != "p1":
        raise RuntimeError(f"project update returned unexpected payload: {update!r}")

    status, body, _ = helper.request(opener, urllib.parse.urljoin(base_url, "/orgs/o1/audit"))
    helper.require_status("audit read", status, 200, body)
    audit = helper.parse_json("audit read", body)
    if audit.get("count", 0) < 1:
        raise RuntimeError(f"audit read returned no records: {audit!r}")

    invite_body = json.dumps({"email": args.username, "role": "viewer"}).encode("utf-8")
    status, body, _ = helper.request(
        opener,
        urllib.parse.urljoin(base_url, "/orgs/o1/invites"),
        method="POST",
        data=invite_body,
        headers={"Content-Type": "application/json", "X-CSRF-Token": csrf},
    )
    helper.require_status("invite issue", status, 200, body)
    invite = helper.parse_json("invite issue", body)
    token = invite.get("token")
    if not token:
        raise RuntimeError(f"invite issue did not include token: {invite!r}")

    begin_body = json.dumps({"token": token}).encode("utf-8")
    status, body, _ = helper.request(
        opener,
        urllib.parse.urljoin(base_url, "/org-invites/begin"),
        method="POST",
        data=begin_body,
        headers={"Content-Type": "application/json"},
    )
    helper.require_status("invite begin", status, 200, body)
    begun = helper.parse_json("invite begin", body)
    registration_url = begun.get("registrationUrl", "")
    return_to = urllib.parse.parse_qs(urllib.parse.urlsplit(registration_url).query).get("return_to", [""])[0]
    pending = urllib.parse.parse_qs(urllib.parse.urlsplit(return_to).query).get("pending", [""])[0]
    if not pending:
        raise RuntimeError(f"invite begin did not return an opaque continuation: {begun!r}")

    accept_body = json.dumps({"pending": pending}).encode("utf-8")
    status, body, _ = helper.request(
        opener,
        urllib.parse.urljoin(base_url, "/org-invites/accept"),
        method="POST",
        data=accept_body,
        headers={"Content-Type": "application/json", "X-CSRF-Token": csrf},
    )
    helper.require_status("invite accept", status, 200, body)
    accepted = helper.parse_json("invite accept", body)
    if accepted.get("orgId") != "o1" or accepted.get("role") != "viewer":
        raise RuntimeError(f"invite accept returned unexpected payload: {accepted!r}")

    status, body, _ = helper.request(
        opener,
        urllib.parse.urljoin(base_url, "/org-invites/accept"),
        method="POST",
        data=accept_body,
        headers={"Content-Type": "application/json", "X-CSRF-Token": csrf},
    )
    helper.require_status("invite accept reused", status, 409, body)
    reused = helper.parse_json("invite accept reused", body)
    if "already used" not in str(reused.get("error", "")):
        raise RuntimeError(f"invite reuse returned unexpected payload: {reused!r}")

    cap_count = psql(compose_file, "SELECT count(*) FROM auth_capabilities WHERE purpose = 'org.invite.accept' AND used_at IS NOT NULL")
    if not cap_count or int(cap_count) < 1:
        raise RuntimeError(f"expected used org-invite capability row, got {cap_count!r}")
    print(f"ok persisted used org-invite capability rows {cap_count}")

    print(json.dumps({"status": "PASS", "actorId": actor_id, "csrfChecked": True, "auditChecked": True, "inviteChecked": True}))
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except Exception as exc:  # noqa: BLE001 - smoke scripts should print concise failures.
        print(f"FAIL: {exc}", file=sys.stderr)
        raise SystemExit(1)
