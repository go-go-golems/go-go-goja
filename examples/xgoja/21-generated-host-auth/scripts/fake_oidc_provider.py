#!/usr/bin/env python3
"""Tiny OIDC discovery endpoint for generated-host auth smoke tests."""

from __future__ import annotations

import argparse
import json
from http.server import BaseHTTPRequestHandler, HTTPServer


class Handler(BaseHTTPRequestHandler):
    issuer = ""

    def do_GET(self) -> None:  # noqa: N802 - stdlib callback name
        if self.path == "/.well-known/openid-configuration":
            self._write_json(
                {
                    "issuer": self.issuer,
                    "authorization_endpoint": self.issuer + "/auth",
                    "token_endpoint": self.issuer + "/token",
                    "jwks_uri": self.issuer + "/jwks",
                    "id_token_signing_alg_values_supported": ["RS256"],
                }
            )
            return
        if self.path == "/jwks":
            self._write_json({"keys": []})
            return
        self.send_response(404)
        self.end_headers()

    def _write_json(self, payload: dict) -> None:
        body = json.dumps(payload).encode("utf-8")
        self.send_response(200)
        self.send_header("content-type", "application/json")
        self.send_header("content-length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def log_message(self, fmt: str, *args: object) -> None:
        return


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--host", default="127.0.0.1")
    parser.add_argument("--port", type=int, required=True)
    parser.add_argument("--issuer", required=True)
    args = parser.parse_args()
    Handler.issuer = args.issuer.rstrip("/")
    HTTPServer((args.host, args.port), Handler).serve_forever()


if __name__ == "__main__":
    main()
