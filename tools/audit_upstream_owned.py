#!/usr/bin/env python3
"""Lock every local modification of an upstream-owned path to reviewed blobs."""

from __future__ import annotations

import argparse
import json
import os
import subprocess
import sys
from dataclasses import dataclass
from typing import Callable


REPO_ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
DEFAULT_POLICY = os.path.join(REPO_ROOT, "tools", "upstream_owned_policy.json")


@dataclass(frozen=True)
class ModifiedPath:
    path: str
    upstream_blob: str
    local_blob: str


@dataclass(frozen=True)
class RenamedPath:
    upstream_path: str
    local_path: str
    upstream_blob: str
    local_blob: str


@dataclass(frozen=True)
class Snapshot:
    modified: tuple[ModifiedPath, ...]
    renamed: tuple[RenamedPath, ...]
    deleted: tuple[str, ...]


def git(args: list[str], *, cwd: str = REPO_ROOT) -> str:
    return subprocess.check_output(["git", *args], cwd=cwd, text=True).strip()


def blob(ref: str, path: str) -> str:
    return git(["rev-parse", f"{ref}:{path}"])


def collect_snapshot(local_ref: str, upstream_ref: str) -> Snapshot:
    raw = subprocess.check_output(
        [
            "git",
            "diff",
            "--name-status",
            "-z",
            "--find-renames",
            f"{upstream_ref}..{local_ref}",
        ],
        cwd=REPO_ROOT,
    ).decode("utf-8", errors="surrogateescape")
    fields = raw.split("\0")
    index = 0
    modified: list[ModifiedPath] = []
    renamed: list[RenamedPath] = []
    deleted: list[str] = []

    while index < len(fields) and fields[index]:
        status = fields[index]
        index += 1
        kind = status[0]
        if kind in {"R", "C"}:
            upstream_path, local_path = fields[index], fields[index + 1]
            index += 2
            renamed.append(
                RenamedPath(
                    upstream_path=upstream_path,
                    local_path=local_path,
                    upstream_blob=blob(upstream_ref, upstream_path),
                    local_blob=blob(local_ref, local_path),
                )
            )
            continue

        path = fields[index]
        index += 1
        if kind in {"M", "T"}:
            modified.append(
                ModifiedPath(
                    path=path,
                    upstream_blob=blob(upstream_ref, path),
                    local_blob=blob(local_ref, path),
                )
            )
        elif kind == "D":
            deleted.append(path)
        # Local-only additions are intentionally outside upstream ownership.

    return Snapshot(
        modified=tuple(sorted(modified, key=lambda item: item.path)),
        renamed=tuple(
            sorted(renamed, key=lambda item: (item.upstream_path, item.local_path))
        ),
        deleted=tuple(sorted(deleted)),
    )


def snapshot_policy(snapshot: Snapshot) -> dict:
    return {
        "version": 1,
        "modified": [item.__dict__ for item in snapshot.modified],
        "renamed": [item.__dict__ for item in snapshot.renamed],
    }


def _index_entries(
    entries: list[dict],
    key: Callable[[dict], str],
    label: str,
) -> tuple[dict[str, dict], list[str]]:
    indexed: dict[str, dict] = {}
    failures: list[str] = []
    for entry in entries:
        entry_key = key(entry)
        if entry_key in indexed:
            failures.append(f"duplicate {label} policy entry: {entry_key}")
        indexed[entry_key] = entry
    return indexed, failures


def validate_policy(policy: dict, snapshot: Snapshot) -> list[str]:
    failures: list[str] = []
    if not isinstance(policy, dict) or policy.get("version") != 1:
        return ["upstream-owned policy version must be 1"]
    modified_entries = policy.get("modified")
    renamed_entries = policy.get("renamed")
    if not isinstance(modified_entries, list) or not isinstance(renamed_entries, list):
        return ["upstream-owned policy must contain modified and renamed arrays"]

    expected_modified, duplicate_failures = _index_entries(
        modified_entries, lambda entry: str(entry.get("path", "")), "modified"
    )
    failures.extend(duplicate_failures)
    actual_modified = {item.path: item for item in snapshot.modified}

    for path in sorted(actual_modified.keys() - expected_modified.keys()):
        failures.append(f"unreviewed upstream-owned modification: {path}")
    for path in sorted(expected_modified.keys() - actual_modified.keys()):
        failures.append(f"stale upstream-owned modification policy entry: {path}")
    for path in sorted(actual_modified.keys() & expected_modified.keys()):
        expected = expected_modified[path]
        actual = actual_modified[path]
        if expected.get("upstream_blob") != actual.upstream_blob:
            failures.append(f"upstream source changed for reviewed override: {path}")
        if expected.get("local_blob") != actual.local_blob:
            failures.append(f"local reviewed override changed: {path}")

    rename_key = lambda entry: (
        f"{entry.get('upstream_path', '')} -> {entry.get('local_path', '')}"
    )
    expected_renamed, duplicate_failures = _index_entries(
        renamed_entries, rename_key, "rename"
    )
    failures.extend(duplicate_failures)
    actual_renamed = {
        f"{item.upstream_path} -> {item.local_path}": item for item in snapshot.renamed
    }
    for path in sorted(actual_renamed.keys() - expected_renamed.keys()):
        failures.append(f"unreviewed upstream-owned rename: {path}")
    for path in sorted(expected_renamed.keys() - actual_renamed.keys()):
        failures.append(f"stale upstream-owned rename policy entry: {path}")
    for path in sorted(actual_renamed.keys() & expected_renamed.keys()):
        expected = expected_renamed[path]
        actual = actual_renamed[path]
        if expected.get("upstream_blob") != actual.upstream_blob:
            failures.append(f"upstream source changed for reviewed rename: {path}")
        if expected.get("local_blob") != actual.local_blob:
            failures.append(f"local reviewed rename changed: {path}")

    for path in snapshot.deleted:
        failures.append(f"upstream-owned path deleted locally: {path}")
    return failures


def load_policy(path: str) -> dict:
    with open(path, encoding="utf-8") as source:
        return json.load(source)


def write_policy(path: str, snapshot: Snapshot) -> None:
    if snapshot.deleted:
        raise RuntimeError("cannot snapshot while upstream-owned paths are deleted")
    if git(["status", "--porcelain"]):
        raise RuntimeError("policy refresh requires a clean committed worktree")
    with open(path, "w", encoding="utf-8") as target:
        json.dump(snapshot_policy(snapshot), target, ensure_ascii=True, indent=2)
        target.write("\n")


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--local", default="HEAD")
    parser.add_argument("--upstream", default="upstream/main")
    parser.add_argument("--policy", default=DEFAULT_POLICY)
    parser.add_argument("--check", action="store_true")
    parser.add_argument("--write-policy", action="store_true")
    parser.add_argument(
        "--reviewed",
        action="store_true",
        help="confirm the full upstream diff was reviewed before refreshing policy",
    )
    return parser


def main() -> int:
    args = build_parser().parse_args()
    try:
        snapshot = collect_snapshot(args.local, args.upstream)
        if args.write_policy:
            if not args.reviewed:
                raise RuntimeError("--write-policy requires explicit --reviewed confirmation")
            write_policy(os.path.abspath(args.policy), snapshot)
            print(
                f"wrote reviewed upstream-owned policy: "
                f"modified={len(snapshot.modified)} renamed={len(snapshot.renamed)}"
            )
            return 0

        failures = validate_policy(load_policy(os.path.abspath(args.policy)), snapshot)
        print(
            f"upstream-owned paths: modified={len(snapshot.modified)} "
            f"renamed={len(snapshot.renamed)} deleted={len(snapshot.deleted)}"
        )
        if failures:
            print("upstream-owned path audit failed:", file=sys.stderr)
            for failure in failures:
                print(f"- {failure}", file=sys.stderr)
            return 2 if args.check else 0
        return 0
    except (OSError, RuntimeError, ValueError, subprocess.CalledProcessError) as exc:
        print(f"upstream-owned path audit failed: {exc}", file=sys.stderr)
        return 2


if __name__ == "__main__":
    raise SystemExit(main())
