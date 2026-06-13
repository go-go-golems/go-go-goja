#!/usr/bin/env python3
"""Drive the local Keycloak Authorization Code + PKCE flow for the example host.

The script intentionally uses only Python's standard library so the example does
not need Playwright, Selenium, or Node dependencies. It follows the host login
redirect, parses the Keycloak login form, submits the demo credentials, then uses
the resulting app session cookie to exercise planned Express routes.
"""

from __future__ import annotations

import argparse
import html
import json
import sys
import urllib.error
import urllib.parse
import urllib.request
from html.parser import HTMLParser
from http.cookiejar import CookieJar
from typing import Dict, Iterable, Optional, Tuple


class LoginFormParser(HTMLParser):
    def __init__(self) -> None:
        super().__init__()
        self.in_form = False
        self.form_action = ""
        self.form_method = "get"
        self.fields: Dict[str, str] = {}
        self._capture = False

    def handle_starttag(self, tag: str, attrs: Iterable[Tuple[str, Optional[str]]]) -> None:
        attrs_map = {name.lower(): value or "" for name, value in attrs}
        if tag.lower() == "form":
            action = attrs_map.get("action", "")
            method = attrs_map.get("method", "get").lower()
            form_id = attrs_map.get("id", "")
            # Keycloak's login form usually has id=kc-form-login and an action
            # URL containing /login-actions/authenticate. Accept either signal so
            # minor theme changes do not break the smoke.
            if method == "post" and (form_id == "kc-form-login" or "login-actions/authenticate" in action):
                self.in_form = True
                self.form_action = html.unescape(action)
                self.form_method = method
            return
        if self.in_form and tag.lower() == "input":
            name = attrs_map.get("name", "")
            if name:
                self.fields[name] = attrs_map.get("value", "")

    def handle_endtag(self, tag: str) -> None:
        if tag.lower() == "form" and self.in_form:
            self.in_form = False


def request(opener: urllib.request.OpenerDirector, url: str, *, method: str = "GET", data: Optional[bytes] = None, headers: Optional[Dict[str, str]] = None) -> Tuple[int, bytes, str]:
    req = urllib.request.Request(url, data=data, headers=headers or {}, method=method)
    try:
        with opener.open(req, timeout=30) as resp:
            return resp.status, resp.read(), resp.geturl()
    except urllib.error.HTTPError as exc:
        return exc.code, exc.read(), exc.geturl()


def require_status(label: str, actual: int, expected: int, body: bytes = b"") -> None:
    if actual != expected:
        excerpt = body[:500].decode("utf-8", "replace")
        raise RuntimeError(f"{label}: expected HTTP {expected}, got {actual}; body={excerpt!r}")
    print(f"ok {label:<28} {actual}")


def parse_json(label: str, body: bytes) -> dict:
    try:
        return json.loads(body.decode("utf-8"))
    except Exception as exc:  # noqa: BLE001 - smoke output should include body excerpt.
        excerpt = body[:500].decode("utf-8", "replace")
        raise RuntimeError(f"{label}: expected JSON, got {excerpt!r}") from exc


def cookie_header_for(jar: CookieJar, target_url: str) -> str:
    """Return cookies for a URL, including Secure localhost cookies over HTTP.

    Keycloak dev mode may set `Secure; SameSite=None` cookies even when it is
    served on `http://127.0.0.1`. Browsers commonly special-case localhost for
    secure contexts, but Python's CookieJar correctly refuses to send those
    cookies over plain HTTP. For this local-only smoke we add the Cookie header
    explicitly for the Keycloak form POST so the login session is preserved.
    """
    parsed = urllib.parse.urlparse(target_url)
    host = parsed.hostname or ""
    path = parsed.path or "/"
    pairs = []
    for cookie in jar:
        domain = cookie.domain.lstrip(".")
        if domain and domain != host and not host.endswith("." + domain):
            continue
        if not path.startswith(cookie.path):
            continue
        pairs.append(f"{cookie.name}={cookie.value}")
    return "; ".join(pairs)


def login(opener: urllib.request.OpenerDirector, jar: CookieJar, base_url: str, username: str, password: str) -> None:
    status, body, final_url = request(opener, urllib.parse.urljoin(base_url, "/auth/login"))
    require_status("login page", status, 200, body)

    parser = LoginFormParser()
    parser.feed(body.decode("utf-8", "replace"))
    if not parser.form_action:
        excerpt = body[:1000].decode("utf-8", "replace")
        raise RuntimeError(f"could not find Keycloak login form at {final_url}; body={excerpt!r}")

    form = dict(parser.fields)
    form["username"] = username
    form["password"] = password
    form.setdefault("credentialId", "")
    encoded = urllib.parse.urlencode(form).encode("utf-8")
    action = urllib.parse.urljoin(final_url, parser.form_action)
    headers = {"Content-Type": "application/x-www-form-urlencoded"}
    if cookie_header := cookie_header_for(jar, action):
        headers["Cookie"] = cookie_header
    status, body, callback_final_url = request(
        opener,
        action,
        method="POST",
        data=encoded,
        headers=headers,
    )
    require_status("keycloak form login", status, 200, body)
    if urllib.parse.urlparse(callback_final_url).netloc != urllib.parse.urlparse(base_url).netloc:
        raise RuntimeError(f"login did not return to host; final URL was {callback_final_url}")
    print(f"ok {'login redirected to host':<28} {callback_final_url}")


def main() -> int:
    parser = argparse.ArgumentParser(description="Smoke-test the Keycloak Express auth host example")
    parser.add_argument("--base-url", default="http://127.0.0.1:8790")
    parser.add_argument("--username", default="demo@example.test")
    parser.add_argument("--password", default="demo-password")
    args = parser.parse_args()

    base_url = args.base_url.rstrip("/")
    jar = CookieJar()
    opener = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(jar))
    opener.addheaders = [("User-Agent", "go-go-goja-keycloak-smoke/1.0")]

    status, body, _ = request(opener, urllib.parse.urljoin(base_url, "/healthz"))
    require_status("public health", status, 200, body)

    status, body, _ = request(opener, urllib.parse.urljoin(base_url, "/me"))
    require_status("me before login", status, 401, body)

    login(opener, jar, base_url, args.username, args.password)

    status, body, _ = request(opener, urllib.parse.urljoin(base_url, "/me"))
    require_status("me after login", status, 200, body)
    me = parse_json("me after login", body)
    if not me.get("id"):
        raise RuntimeError(f"/me did not include an actor id: {me!r}")

    status, body, _ = request(opener, urllib.parse.urljoin(base_url, "/auth/session"))
    require_status("session after login", status, 200, body)
    session = parse_json("session after login", body)
    csrf = session.get("csrfToken")
    if not csrf:
        raise RuntimeError(f"/auth/session did not include csrfToken: {session!r}")

    project_url = urllib.parse.urljoin(base_url, "/orgs/o1/projects/p1")
    status, body, _ = request(opener, project_url, method="PATCH")
    require_status("project missing csrf", status, 403, body)

    status, body, _ = request(opener, project_url, method="PATCH", headers={"X-CSRF-Token": csrf})
    require_status("project update", status, 200, body)
    update = parse_json("project update", body)
    if update.get("updated") != "p1":
        raise RuntimeError(f"project update returned unexpected payload: {update!r}")

    missing_url = urllib.parse.urljoin(base_url, "/orgs/o1/projects/missing")
    status, body, _ = request(opener, missing_url, method="PATCH", headers={"X-CSRF-Token": csrf})
    require_status("project missing", status, 404, body)

    status, body, _ = request(opener, urllib.parse.urljoin(base_url, "/auth/logout"), method="POST")
    require_status("logout", status, 204, body)

    status, body, _ = request(opener, urllib.parse.urljoin(base_url, "/me"))
    require_status("me after logout", status, 401, body)

    print(json.dumps({"status": "PASS", "actorId": me.get("id"), "csrfChecked": True}))
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except Exception as exc:  # noqa: BLE001 - smoke script should print concise failures.
        print(f"FAIL: {exc}", file=sys.stderr)
        raise SystemExit(1)
