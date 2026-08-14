#!/usr/bin/env python3
"""Audit custom frontend ownership and both sides of reviewed fork blobs."""

from __future__ import annotations

import argparse
import json
import os
import re
import subprocess
import sys
from collections.abc import Callable
from typing import Any


REPO_ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
DEFAULT_POLICY = os.path.join(REPO_ROOT, "tools", "upstream_overlay_policy.json")
CUSTOM_ROOT = "frontend/src/custom/"
HEX_OBJECT_ID = re.compile(r"^[0-9a-f]{40}$")


def git(args: list[str], *, cwd: str = REPO_ROOT) -> str:
    proc = subprocess.run(
        ["git", *args],
        cwd=cwd,
        check=False,
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
    )
    if proc.returncode != 0:
        detail = proc.stderr.strip() or proc.stdout.strip()
        raise RuntimeError(detail or f"git {' '.join(args)} failed")
    return proc.stdout.strip()


def production_custom_paths(local_ref: str) -> set[str]:
    output = git(["ls-tree", "-r", "--name-only", local_ref, "--", CUSTOM_ROOT])
    return {
        path
        for path in output.splitlines()
        if path and "/__tests__/" not in path and not path.endswith(".spec.ts")
    }


def validate_policy(
    policy: dict[str, Any],
    *,
    actual_custom_paths: set[str],
    resolve_upstream_blob: Callable[[str], str],
    resolve_local_blob: Callable[[str], str],
) -> list[str]:
    failures: list[str] = []
    forks = policy.get("forks")
    local_only = policy.get("local_only_paths")
    if policy.get("version") != 2:
        failures.append("policy version must be 2")
    if not isinstance(forks, list):
        failures.append("policy forks must be a list")
        forks = []
    if not isinstance(local_only, list):
        failures.append("policy local_only_paths must be a list")
        local_only = []

    classified: set[str] = set()
    seen_sources: dict[str, str] = {}
    for index, entry in enumerate(forks):
        if not isinstance(entry, dict):
            failures.append(f"forks[{index}] must be an object")
            continue
        custom_path = entry.get("custom_path")
        upstream_path = entry.get("upstream_path")
        reviewed_blob = entry.get("reviewed_blob")
        reviewed_local_blob = entry.get("reviewed_local_blob")
        if not all(
            isinstance(value, str) and value
            for value in (custom_path, upstream_path, reviewed_blob, reviewed_local_blob)
        ):
            failures.append(
                f"forks[{index}] requires custom_path, upstream_path, reviewed_blob, "
                "and reviewed_local_blob"
            )
            continue
        if not custom_path.startswith(CUSTOM_ROOT):
            failures.append(f"fork custom path is outside {CUSTOM_ROOT}: {custom_path}")
        if custom_path in classified:
            failures.append(f"custom path is classified more than once: {custom_path}")
        classified.add(custom_path)
        if upstream_path in seen_sources:
            failures.append(
                f"upstream source is mapped more than once: {upstream_path} "
                f"({seen_sources[upstream_path]}, {custom_path})"
            )
        else:
            seen_sources[upstream_path] = custom_path
        if not HEX_OBJECT_ID.fullmatch(reviewed_blob):
            failures.append(f"reviewed blob is not a full object id for {custom_path}: {reviewed_blob}")
            continue
        if not HEX_OBJECT_ID.fullmatch(reviewed_local_blob):
            failures.append(
                f"reviewed local blob is not a full object id for {custom_path}: "
                f"{reviewed_local_blob}"
            )
            continue
        try:
            current_blob = resolve_upstream_blob(upstream_path)
        except RuntimeError as exc:
            failures.append(f"upstream source is missing for {custom_path}: {upstream_path} ({exc})")
            continue
        if current_blob != reviewed_blob:
            failures.append(
                f"upstream source changed for {custom_path}: {upstream_path} "
                f"reviewed={reviewed_blob} current={current_blob}"
            )
        try:
            current_local_blob = resolve_local_blob(custom_path)
        except RuntimeError as exc:
            failures.append(f"custom fork is missing for {custom_path} ({exc})")
            continue
        if current_local_blob != reviewed_local_blob:
            failures.append(
                f"custom fork changed for {custom_path}: "
                f"reviewed={reviewed_local_blob} current={current_local_blob}"
            )

    for index, path in enumerate(local_only):
        if not isinstance(path, str) or not path:
            failures.append(f"local_only_paths[{index}] must be a non-empty string")
            continue
        if not path.startswith(CUSTOM_ROOT):
            failures.append(f"local-only path is outside {CUSTOM_ROOT}: {path}")
        if path in classified:
            failures.append(f"custom path is classified more than once: {path}")
        classified.add(path)

    for path in sorted(actual_custom_paths - classified):
        failures.append(f"unclassified custom production file: {path}")
    for path in sorted(classified - actual_custom_paths):
        failures.append(f"classified custom production file is missing: {path}")
    return failures


def load_policy(path: str) -> dict[str, Any]:
    with open(path, encoding="utf-8") as source:
        loaded = json.load(source)
    if not isinstance(loaded, dict):
        raise RuntimeError("overlay policy root must be an object")
    return loaded


def refresh_policy(policy: dict[str, Any], upstream_ref: str, local_ref: str) -> dict[str, Any]:
    refreshed = json.loads(json.dumps(policy))
    refreshed["version"] = 2
    forks = refreshed.get("forks")
    if not isinstance(forks, list):
        raise RuntimeError("policy forks must be a list")
    for index, entry in enumerate(forks):
        if not isinstance(entry, dict):
            raise RuntimeError(f"forks[{index}] must be an object")
        custom_path = entry.get("custom_path")
        upstream_path = entry.get("upstream_path")
        if not isinstance(custom_path, str) or not isinstance(upstream_path, str):
            raise RuntimeError(f"forks[{index}] has invalid paths")
        entry["reviewed_blob"] = git(["rev-parse", f"{upstream_ref}:{upstream_path}"])
        entry["reviewed_local_blob"] = git(["rev-parse", f"{local_ref}:{custom_path}"])
    return refreshed


def write_policy(path: str, policy: dict[str, Any]) -> None:
    if git(["status", "--porcelain"]):
        raise RuntimeError("policy refresh requires a clean committed worktree")
    with open(path, "w", encoding="utf-8") as target:
        json.dump(policy, target, ensure_ascii=True, indent=2)
        target.write("\n")


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--policy", default=DEFAULT_POLICY)
    parser.add_argument("--upstream", default="upstream/main")
    parser.add_argument("--local", default="HEAD")
    parser.add_argument("--check", action="store_true")
    parser.add_argument("--write-policy", action="store_true")
    parser.add_argument("--reviewed", action="store_true")
    args = parser.parse_args()

    try:
        policy = load_policy(os.path.abspath(args.policy))
        if args.write_policy:
            if not args.reviewed:
                raise RuntimeError("--write-policy requires explicit --reviewed confirmation")
            write_policy(
                os.path.abspath(args.policy),
                refresh_policy(policy, args.upstream, args.local),
            )
            print(f"wrote reviewed custom fork policy: forks={len(policy.get('forks', []))}")
            return 0
        actual_paths = production_custom_paths(args.local)
        failures = validate_policy(
            policy,
            actual_custom_paths=actual_paths,
            resolve_upstream_blob=lambda path: git(["rev-parse", f"{args.upstream}:{path}"]),
            resolve_local_blob=lambda path: git(["rev-parse", f"{args.local}:{path}"]),
        )
    except (OSError, ValueError, RuntimeError, json.JSONDecodeError) as exc:
        print(f"upstream overlay audit failed: {exc}", file=sys.stderr)
        return 2

    print(
        f"custom production files={len(actual_paths)} "
        f"forks={len(policy.get('forks', []))} "
        f"local_only={len(policy.get('local_only_paths', []))}"
    )
    if failures:
        for failure in failures:
            print(f"- {failure}", file=sys.stderr)
        if args.check:
            return 2
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
