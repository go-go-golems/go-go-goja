#!/usr/bin/env python3
import json
import subprocess
import sys


def main() -> int:
    if len(sys.argv) != 6:
        raise SystemExit("usage: api_smoke.py <capture1> <list1> <binary> <base-url> <list2>")
    capture_path, list_path, binary, base_url, list2_path = sys.argv[1:]
    created = json.load(open(capture_path))
    listed = json.load(open(list_path))
    assert created["ok"] is True, created
    assert len(listed["items"]) == 2, listed
    first_id = created["item"]["id"]
    subprocess.check_call([
        binary,
        "verbs",
        "inboxctl",
        "archive",
        "--base-url",
        base_url,
        "--id",
        first_id,
    ], stdout=subprocess.DEVNULL)
    with open(list2_path, "w") as f:
        subprocess.check_call([
            binary,
            "verbs",
            "inboxctl",
            "list",
            "--base-url",
            base_url,
        ], stdout=f)
    after = json.load(open(list2_path))
    assert len(after["items"]) == 1, after
    assert after["items"][0]["title"] == "Second API item", after
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
