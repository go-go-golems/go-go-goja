#!/usr/bin/env python3
"""Assert Step 08 device-token captures stay scoped to the approving user.

The script uses tinyidp only for browser login. Device authorization remains
xgoja-owned: /auth/device/start, /auth/device/approve, /auth/device/token, and
/api/programmatic/capture all belong to the generated xgoja host.
"""

from __future__ import annotations

import argparse
import http.cookiejar
import html.parser
import json
import ssl
import urllib.error
import urllib.parse
import urllib.request
from dataclasses import dataclass


class LoginFormParser(html.parser.HTMLParser):
    def __init__(self) -> None:
        super().__init__()
        self.action: str | None = None
        self.inputs: dict[str, str] = {}
        self.in_form = False

    def handle_starttag(self, tag: str, attrs: list[tuple[str, str | None]]) -> None:
        attrs_dict = {key: value or "" for key, value in attrs}
        if tag == "form":
            self.in_form = True
            self.action = attrs_dict.get("action")
        if self.in_form and tag == "input":
            name = attrs_dict.get("name")
            if name:
                self.inputs[name] = attrs_dict.get("value", "")

    def handle_endtag(self, tag: str) -> None:
        if tag == "form":
            self.in_form = False


@dataclass
class BrowserSession:
    login: str
    expected_email: str
    expected_user_id: str
    opener: urllib.request.OpenerDirector
    csrf_token: str
    user_id: str


def request_json(
    opener: urllib.request.OpenerDirector,
    url: str,
    *,
    method: str = "GET",
    data: dict | None = None,
    headers: dict[str, str] | None = None,
    expected: int | None = None,
) -> tuple[int, dict]:
    body = None
    req_headers = dict(headers or {})
    if data is not None:
        body = json.dumps(data).encode()
        req_headers.setdefault("Content-Type", "application/json")
    req = urllib.request.Request(url, data=body, headers=req_headers, method=method)
    try:
        with opener.open(req, timeout=30) as resp:
            raw = resp.read().decode("utf-8", "replace")
            parsed = json.loads(raw) if raw else {}
            if expected is not None and resp.status != expected:
                raise RuntimeError(f"{url} returned {resp.status}, want {expected}: {parsed!r}")
            return resp.status, parsed
    except urllib.error.HTTPError as exc:
        raw = exc.read().decode("utf-8", "replace")
        try:
            parsed = json.loads(raw) if raw else {}
        except json.JSONDecodeError:
            parsed = {"error": raw}
        if expected is not None and exc.code != expected:
            raise RuntimeError(f"{url} returned {exc.code}, want {expected}: {parsed!r}")
        return exc.code, parsed


def login(base_url: str, username: str, password: str, expected_email: str, expected_user_id: str | None, ssl_context: ssl.SSLContext | None) -> BrowserSession:
    jar = http.cookiejar.CookieJar()
    opener = urllib.request.build_opener(
        urllib.request.HTTPCookieProcessor(jar),
        urllib.request.HTTPRedirectHandler(),
        urllib.request.HTTPSHandler(context=ssl_context) if ssl_context else urllib.request.HTTPSHandler(),
    )
    opener.addheaders = [("User-Agent", "go-go-goja-tinyidp-device-isolation-smoke/1.0")]

    response = opener.open(base_url + "/auth/login", timeout=30)
    body = response.read().decode("utf-8", "replace")
    form_parser = LoginFormParser()
    form_parser.feed(body)
    if not form_parser.action:
        raise RuntimeError("tinyidp login page did not contain a form action")

    form = dict(form_parser.inputs)
    form["login"] = username
    form["password"] = password
    # tiny-idp exposes approve and deny as submit buttons. HTML form submission
    # includes the selected button; urllib builds the payload itself, so state
    # the approved action explicitly rather than relying on browser behavior.
    form["action"] = "approve"
    request = urllib.request.Request(
        urllib.parse.urljoin(response.geturl(), form_parser.action),
        data=urllib.parse.urlencode(form).encode(),
        method="POST",
        headers={"Content-Type": "application/x-www-form-urlencoded"},
    )
    response = opener.open(request, timeout=30)
    response.read()

    _, session = request_json(opener, base_url + "/auth/session", expected=200)
    if session.get("email") != expected_email:
        raise RuntimeError(f"unexpected {username} email: {session!r}")
    if expected_user_id and session.get("userId") != expected_user_id:
        raise RuntimeError(f"unexpected {username} userId: {session!r}")
    if not session.get("csrfToken"):
        raise RuntimeError(f"session for {username} did not include csrfToken: {session!r}")

    return BrowserSession(
        login=username,
        expected_email=expected_email,
        expected_user_id=str(session["userId"]),
        opener=opener,
        csrf_token=session["csrfToken"],
        user_id=session["userId"],
    )


def start_device(base_url: str, client_name: str) -> dict:
    opener = urllib.request.build_opener()
    _, body = request_json(
        opener,
        base_url + "/auth/device/start",
        method="POST",
        data={"clientName": client_name, "actions": ["user.self.read"]},
        expected=200,
    )
    if not str(body.get("device_code", "")).startswith("ggdc_"):
        raise RuntimeError(f"unexpected device start response: {body!r}")
    if not body.get("user_code"):
        raise RuntimeError(f"device start did not include user_code: {body!r}")
    return body


def approve_device(base_url: str, session: BrowserSession, user_code: str) -> None:
    request_json(
        session.opener,
        base_url + "/auth/device/approve",
        method="POST",
        data={"user_code": user_code, "actions": ["user.self.read"]},
        headers={"X-CSRF-Token": session.csrf_token},
        expected=200,
    )


def poll_token(base_url: str, device_code: str) -> dict:
    opener = urllib.request.build_opener()
    _, body = request_json(
        opener,
        base_url + "/auth/device/token",
        method="POST",
        data={"grant_type": "urn:ietf:params:oauth:grant-type:device_code", "device_code": device_code},
        expected=200,
    )
    access_token = body.get("access_token", "")
    if not str(access_token).startswith("ggat_"):
        raise RuntimeError(f"token response did not include access token: {body!r}")
    if not body.get("refresh_token"):
        raise RuntimeError(f"token response did not include refresh token: {body!r}")
    return body


def refresh_token(base_url: str, refresh_value: str) -> dict:
    opener = urllib.request.build_opener()
    _, body = request_json(
        opener,
        base_url + "/auth/device/refresh",
        method="POST",
        data={"grant_type": "refresh_token", "refresh_token": refresh_value},
        expected=200,
    )
    if not str(body.get("access_token", "")).startswith("ggat_") or not str(body.get("refresh_token", "")).startswith("ggrt_"):
        raise RuntimeError(f"refresh response did not contain a rotated token pair: {body!r}")
    return body


def revoke_refresh_token(base_url: str, refresh_value: str) -> None:
    opener = urllib.request.build_opener()
    _, body = request_json(
        opener,
        base_url + "/auth/device/revoke",
        method="POST",
        data={"refresh_token": refresh_value},
        expected=200,
    )
    if body.get("ok") is not True:
        raise RuntimeError(f"refresh-token revocation returned unexpected body: {body!r}")


def token_capture(base_url: str, access_token: str, title: str, login: str) -> dict:
    opener = urllib.request.build_opener()
    _, body = request_json(
        opener,
        base_url + "/api/programmatic/capture",
        method="POST",
        data={"title": title, "url": f"https://example.com/device/{login}", "source": "tinyidp-device-isolation-smoke"},
        headers={"Authorization": f"Bearer {access_token}"},
        expected=201,
    )
    if body.get("ok") is not True:
        raise RuntimeError(f"programmatic capture returned unexpected body: {body!r}")
    return body


def approve_and_capture(base_url: str, session: BrowserSession, title: str) -> None:
    started = start_device(base_url, f"{session.login}-device-cli")
    approve_device(base_url, session, str(started["user_code"]))
    issued = poll_token(base_url, str(started["device_code"]))
    token_capture(base_url, str(issued["access_token"]), title, session.login)
    refreshed = refresh_token(base_url, str(issued["refresh_token"]))
    if refreshed["refresh_token"] == issued["refresh_token"]:
        raise RuntimeError("refresh endpoint did not rotate the refresh credential")
    revoke_refresh_token(base_url, str(refreshed["refresh_token"]))
    opener = urllib.request.build_opener()
    status, body = request_json(
        opener,
        base_url + "/auth/device/refresh",
        method="POST",
        data={"grant_type": "refresh_token", "refresh_token": refreshed["refresh_token"]},
        expected=400,
    )
    if status != 400 or body.get("error") != "invalid_grant":
        raise RuntimeError(f"revoked refresh token was accepted: {body!r}")


def list_titles(base_url: str, session: BrowserSession) -> list[str]:
    _, body = request_json(session.opener, base_url + "/api/inbox", expected=200)
    if body.get("ownerUserId") != session.user_id:
        raise RuntimeError(f"list owner mismatch for {session.login}: {body!r}")
    return [item.get("title", "") for item in body.get("items", [])]


def assert_titles(label: str, titles: list[str], *, present: str, absent: str) -> None:
    if present not in titles:
        raise RuntimeError(f"{label}: expected {present!r} in {titles!r}")
    if absent in titles:
        raise RuntimeError(f"{label}: did not expect {absent!r} in {titles!r}")


def main() -> None:
    parser = argparse.ArgumentParser(description="Assert tinyidp-backed device capture isolation")
    parser.add_argument("--base-url", required=True, help="Generated app base URL")
    parser.add_argument("--alice-login", default="alice")
    parser.add_argument("--alice-password", default="alice-password")
    parser.add_argument("--alice-email", default="alice@example.test")
    parser.add_argument("--bob-login", default="bob")
    parser.add_argument("--bob-password", default="bob-password")
    parser.add_argument("--bob-email", default="bob@example.test")
    parser.add_argument("--alice-user-id", default="")
    parser.add_argument("--bob-user-id", default="")
    parser.add_argument("--ca-file", default="", help="PEM CA file for a self-signed strict tinyidp issuer")
    args = parser.parse_args()

    base_url = args.base_url.rstrip("/")
    ssl_context = ssl.create_default_context(cafile=args.ca_file) if args.ca_file else None
    alice = login(base_url, args.alice_login, args.alice_password, args.alice_email, args.alice_user_id or None, ssl_context)
    bob = login(base_url, args.bob_login, args.bob_password, args.bob_email, args.bob_user_id or None, ssl_context)

    alice_title = "Alice device token item"
    bob_title = "Bob device token item"
    approve_and_capture(base_url, alice, alice_title)
    approve_and_capture(base_url, bob, bob_title)

    assert_titles("alice inbox", list_titles(base_url, alice), present=alice_title, absent=bob_title)
    assert_titles("bob inbox", list_titles(base_url, bob), present=bob_title, absent=alice_title)
    print("ok tinyidp device capture isolation")


if __name__ == "__main__":
    main()
