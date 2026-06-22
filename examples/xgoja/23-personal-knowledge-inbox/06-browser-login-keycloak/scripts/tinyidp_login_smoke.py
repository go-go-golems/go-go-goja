#!/usr/bin/env python3
"""Drive Step 06 login against tinyidp and assert an app session.

The script intentionally uses only Python's standard library. It follows the
same browser-facing OIDC path as a human: /auth/login, tinyidp's login form,
/auth/callback, then /auth/session and /api/inbox with the resulting app cookie.
"""

from __future__ import annotations

import argparse
import http.cookiejar
import html.parser
import json
import urllib.parse
import urllib.request


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


def main() -> None:
    parser = argparse.ArgumentParser(description="Smoke-test Step 06 against tinyidp")
    parser.add_argument("--base-url", required=True, help="Generated app base URL")
    parser.add_argument("--login", default="alice", help="tinyidp login name")
    parser.add_argument("--expected-email", default="alice@example.test")
    args = parser.parse_args()

    base_url = args.base_url.rstrip("/")
    jar = http.cookiejar.CookieJar()
    opener = urllib.request.build_opener(
        urllib.request.HTTPCookieProcessor(jar),
        urllib.request.HTTPRedirectHandler(),
    )
    opener.addheaders = [("User-Agent", "go-go-goja-tinyidp-smoke/1.0")]

    response = opener.open(base_url + "/auth/login")
    body = response.read().decode("utf-8", "replace")
    form_parser = LoginFormParser()
    form_parser.feed(body)
    if not form_parser.action:
        raise SystemExit("tinyidp login page did not contain a form action")

    form = dict(form_parser.inputs)
    form["login"] = args.login
    request = urllib.request.Request(
        urllib.parse.urljoin(response.geturl(), form_parser.action),
        data=urllib.parse.urlencode(form).encode(),
        method="POST",
        headers={"Content-Type": "application/x-www-form-urlencoded"},
    )
    response = opener.open(request)
    response.read()

    session_response = opener.open(base_url + "/auth/session")
    session = json.loads(session_response.read().decode())
    if session.get("email") != args.expected_email:
        raise SystemExit(f"unexpected session email: {session!r}")
    if not session.get("csrfToken"):
        raise SystemExit(f"session did not include csrfToken: {session!r}")

    api_response = opener.open(base_url + "/api/inbox")
    if api_response.status != 200:
        raise SystemExit(f"/api/inbox returned {api_response.status}")

    print(f"ok tinyidp step06 full login smoke; session email={session.get('email')}")


if __name__ == "__main__":
    main()
