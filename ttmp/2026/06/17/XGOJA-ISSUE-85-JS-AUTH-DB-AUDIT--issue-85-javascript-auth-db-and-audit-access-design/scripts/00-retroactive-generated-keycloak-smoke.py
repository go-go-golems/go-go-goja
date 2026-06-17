#!/usr/bin/env python3
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

HELPER = Path('/home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/examples/xgoja/19-express-keycloak-auth-host/scripts/keycloak_smoke.py')
spec = importlib.util.spec_from_file_location('keycloak_smoke_helper', HELPER)
helper = importlib.util.module_from_spec(spec)
assert spec and spec.loader
spec.loader.exec_module(helper)


def psql(sql: str) -> str:
    cmd = [
        'docker', 'compose',
        '-f', '/home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/examples/xgoja/19-express-keycloak-auth-host/docker-compose.yml',
        'exec', '-T', 'postgres', 'psql', '-U', 'goja', '-d', 'goja_auth', '-tAc', sql,
    ]
    return subprocess.check_output(cmd, text=True).strip()


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument('--base-url', default='http://127.0.0.1:8790')
    parser.add_argument('--username', default='demo@example.test')
    parser.add_argument('--password', default='demo-password')
    args = parser.parse_args()
    base_url = args.base_url.rstrip('/')

    jar = CookieJar()
    opener = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(jar))
    opener.addheaders = [('User-Agent', 'go-go-goja-generated-keycloak-smoke/1.0')]

    status, body, _ = helper.request(opener, urllib.parse.urljoin(base_url, '/healthz'))
    helper.require_status('public health', status, 200, body)

    status, body, _ = helper.request(opener, urllib.parse.urljoin(base_url, '/me'))
    helper.require_status('me before login', status, 401, body)

    helper.login(opener, jar, base_url, args.username, args.password)

    status, body, _ = helper.request(opener, urllib.parse.urljoin(base_url, '/me'))
    helper.require_status('me after login', status, 200, body)
    me = helper.parse_json('me after login', body)
    actor_id = me.get('id')
    if not actor_id:
        raise RuntimeError(f'/me did not include actor id: {me!r}')

    # The generated host intentionally does not grant demo memberships in its generic OIDC normalizer.
    # Seed the demo app data externally, the same way a real app migration/admin process would.
    psql("INSERT INTO auth_app_tenants (id, slug, name) VALUES ('o1','demo','Demo Org') ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name")
    psql("INSERT INTO auth_app_resources (type, id, name, tenant_id, owner_id, claims_json) VALUES ('org','o1','Demo Org','o1','', '{}'), ('project','p1','Docker Compose Project','o1','', '{}') ON CONFLICT (type, id) DO UPDATE SET name = EXCLUDED.name, tenant_id = EXCLUDED.tenant_id")
    psql(f"INSERT INTO auth_app_memberships (user_id, tenant_id, role) VALUES ('{actor_id}', 'o1', 'admin') ON CONFLICT (user_id, tenant_id, role) DO NOTHING")
    print(f'ok seeded demo appauth rows for actor {actor_id}')

    status, body, _ = helper.request(opener, urllib.parse.urljoin(base_url, '/auth/session'))
    helper.require_status('session after login', status, 200, body)
    session = helper.parse_json('session after login', body)
    csrf = session.get('csrfToken')
    if not csrf:
        raise RuntimeError(f'/auth/session did not include csrfToken: {session!r}')

    project_url = urllib.parse.urljoin(base_url, '/orgs/o1/projects/p1')
    status, body, _ = helper.request(opener, project_url, method='PATCH')
    helper.require_status('project missing csrf', status, 403, body)
    status, body, _ = helper.request(opener, project_url, method='PATCH', headers={'X-CSRF-Token': csrf})
    helper.require_status('project update', status, 200, body)
    update = helper.parse_json('project update', body)
    if update.get('updated') != 'p1':
        raise RuntimeError(f'project update returned unexpected payload: {update!r}')

    status, body, _ = helper.request(opener, urllib.parse.urljoin(base_url, '/orgs/o1/audit'))
    helper.require_status('audit read', status, 200, body)
    audit = helper.parse_json('audit read', body)
    if audit.get('count', 0) < 1:
        raise RuntimeError(f'audit read returned no records: {audit!r}')

    invite_body = json.dumps({'email': 'invitee@example.test', 'role': 'viewer'}).encode()
    status, body, _ = helper.request(opener, urllib.parse.urljoin(base_url, '/orgs/o1/invites'), method='POST', data=invite_body, headers={'Content-Type': 'application/json', 'X-CSRF-Token': csrf})
    helper.require_status('invite issue', status, 200, body)
    invite = helper.parse_json('invite issue', body)
    token = invite.get('token')
    if not token:
        raise RuntimeError(f'invite issue did not include token: {invite!r}')

    accept_body = json.dumps({'token': token}).encode()
    status, body, _ = helper.request(opener, urllib.parse.urljoin(base_url, '/org-invites/accept'), method='POST', data=accept_body, headers={'Content-Type': 'application/json'})
    helper.require_status('invite accept', status, 200, body)
    accepted = helper.parse_json('invite accept', body)
    if accepted.get('orgId') != 'o1' or accepted.get('email') != 'invitee@example.test':
        raise RuntimeError(f'invite accept returned unexpected payload: {accepted!r}')

    status, body, _ = helper.request(opener, urllib.parse.urljoin(base_url, '/org-invites/accept'), method='POST', data=accept_body, headers={'Content-Type': 'application/json'})
    if status == 200:
        raise RuntimeError(f'invite reuse unexpectedly succeeded: {body!r}')
    print(f'ok invite reuse rejected with status {status}')

    cap_count = psql("SELECT count(*) FROM auth_capabilities WHERE purpose = 'org-invite' AND used_at IS NOT NULL")
    if not cap_count or int(cap_count) < 1:
        raise RuntimeError(f'expected used org-invite capability row, got {cap_count!r}')
    print(f'ok persisted used org-invite capability rows {cap_count}')

    print(json.dumps({'status': 'PASS', 'actorId': actor_id, 'csrfChecked': True, 'auditChecked': True, 'inviteChecked': True}))
    return 0


if __name__ == '__main__':
    try:
        raise SystemExit(main())
    except Exception as exc:
        print(f'FAIL: {exc}', file=sys.stderr)
        raise SystemExit(1)
