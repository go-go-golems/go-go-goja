#!/usr/bin/env python3
"""Drive tinyidp browser login for Alice and Bob and assert inbox isolation.

This script uses two independent cookie jars. Each user logs in through the
normal browser OIDC path, captures one item through the session/CSRF-protected
browser API, and then lists the inbox. The assertions prove that Step 07's
server-side owner filter is based on the authenticated actor, not on global
inbox state.
"""

from __future__ import annotations

import argparse
import http.cookiejar
import html.parser
import json
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


def request_json(opener: urllib.request.OpenerDirector, url: str, *, method: str = "GET", data: dict | None = None, headers: dict[str, str] | None = None) -> tuple[int, dict]:
    body = None
    req_headers = dict(headers or {})
    if data is not None:
        body = json.dumps(data).encode()
        req_headers.setdefault("Content-Type", "application/json")
    req = urllib.request.Request(url, data=body, headers=req_headers, method=method)
    try:
        with opener.open(req, timeout=30) as resp:
            raw = resp.read().decode("utf-8", "replace")
            return resp.status, json.loads(raw) if raw else {}
    except urllib.error.HTTPError as exc:
        raw = exc.read().decode("utf-8", "replace")
        try:
            parsed = json.loads(raw) if raw else {}
        except json.JSONDecodeError:
            parsed = {"error": raw}
        return exc.code, parsed


def login(base_url: str, username: str, expected_email: str, expected_user_id: str) -> BrowserSession:
    jar = http.cookiejar.CookieJar()
    opener = urllib.request.build_opener(
        urllib.request.HTTPCookieProcessor(jar),
        urllib.request.HTTPRedirectHandler(),
    )
    opener.addheaders = [("User-Agent", "go-go-goja-tinyidp-isolation-smoke/1.0")]

    response = opener.open(base_url + "/auth/login", timeout=30)
    body = response.read().decode("utf-8", "replace")
    form_parser = LoginFormParser()
    form_parser.feed(body)
    if not form_parser.action:
        raise RuntimeError("tinyidp login page did not contain a form action")

    form = dict(form_parser.inputs)
    form["login"] = username
    request = urllib.request.Request(
        urllib.parse.urljoin(response.geturl(), form_parser.action),
        data=urllib.parse.urlencode(form).encode(),
        method="POST",
        headers={"Content-Type": "application/x-www-form-urlencoded"},
    )
    response = opener.open(request, timeout=30)
    response.read()

    status, session = request_json(opener, base_url + "/auth/session")
    if status != 200:
        raise RuntimeError(f"/auth/session for {username} returned {status}: {session!r}")
    if session.get("email") != expected_email:
        raise RuntimeError(f"unexpected {username} email: {session!r}")
    if session.get("userId") != expected_user_id:
        raise RuntimeError(f"unexpected {username} userId: {session!r}")
    if not session.get("csrfToken"):
        raise RuntimeError(f"session for {username} did not include csrfToken: {session!r}")

    return BrowserSession(
        login=username,
        expected_email=expected_email,
        expected_user_id=expected_user_id,
        opener=opener,
        csrf_token=session["csrfToken"],
        user_id=session["userId"],
    )


def capture(base_url: str, session: BrowserSession, title: str) -> dict:
    status, body = request_json(
        session.opener,
        base_url + "/api/capture",
        method="POST",
        data={"title": title, "url": f"https://example.com/{session.login}", "source": "tinyidp-isolation-smoke"},
        headers={"X-CSRF-Token": session.csrf_token},
    )
    if status != 201:
        raise RuntimeError(f"capture for {session.login} returned {status}: {body!r}")
    return body


def list_titles(base_url: str, session: BrowserSession) -> list[str]:
    status, body = request_json(session.opener, base_url + "/api/inbox")
    if status != 200:
        raise RuntimeError(f"list for {session.login} returned {status}: {body!r}")
    if body.get("ownerUserId") != session.user_id:
        raise RuntimeError(f"list owner mismatch for {session.login}: {body!r}")
    return [item.get("title", "") for item in body.get("items", [])]


def assert_titles(label: str, titles: list[str], *, present: str, absent: str) -> None:
    if present not in titles:
        raise RuntimeError(f"{label}: expected {present!r} in {titles!r}")
    if absent in titles:
        raise RuntimeError(f"{label}: did not expect {absent!r} in {titles!r}")


def main() -> None:
    parser = argparse.ArgumentParser(description="Assert tinyidp Alice/Bob inbox isolation")
    parser.add_argument("--base-url", required=True, help="Generated app base URL")
    parser.add_argument("--alice-email", default="alice@example.test")
    parser.add_argument("--bob-email", default="bob@example.test")
    parser.add_argument("--alice-user-id", default="user:user-alice-fixed")
    parser.add_argument("--bob-user-id", default="user:user-bob-fixed")
    args = parser.parse_args()

    base_url = args.base_url.rstrip("/")
    alice = login(base_url, "alice", args.alice_email, args.alice_user_id)
    bob = login(base_url, "bob", args.bob_email, args.bob_user_id)

    alice_title = "Alice tinyidp isolation item"
    bob_title = "Bob tinyidp isolation item"
    capture(base_url, alice, alice_title)
    capture(base_url, bob, bob_title)

    assert_titles("alice inbox", list_titles(base_url, alice), present=alice_title, absent=bob_title)
    assert_titles("bob inbox", list_titles(base_url, bob), present=bob_title, absent=alice_title)
    print("ok tinyidp alice/bob inbox isolation")


if __name__ == "__main__":
    main()
