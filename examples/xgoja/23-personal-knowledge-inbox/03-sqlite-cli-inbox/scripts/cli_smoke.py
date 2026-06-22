#!/usr/bin/env python3
import json
import subprocess
import sys
from pathlib import Path


def main() -> int:
    if len(sys.argv) != 6:
        raise SystemExit("usage: cli_smoke.py <capture1> <list1> <db> <binary> <list2>")
    capture_path, list_path, db, binary, list2_path = sys.argv[1:]
    created = json.load(open(capture_path))
    listed = json.load(open(list_path))
    assert created["ok"] is True, created
    assert len(listed["items"]) == 2, listed
    first_id = created["item"]["id"]
    subprocess.check_call([
        binary,
        "verbs",
        "inbox",
        "archive",
        "--db",
        db,
        "--id",
        first_id,
    ], stdout=subprocess.DEVNULL)
    with open(list2_path, "w") as f:
        subprocess.check_call([
            binary,
            "verbs",
            "inbox",
            "list",
            "--db",
            db,
        ], stdout=f)
    after = json.load(open(list2_path))
    assert len(after["items"]) == 1, after
    assert after["items"][0]["title"] == "Second item", after
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
