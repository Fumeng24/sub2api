#!/usr/bin/env python3
"""Prepare and verify upstream-first merge worktrees."""

from __future__ import annotations

import argparse
import datetime as dt
import os
import subprocess
import sys


def git(args: list[str], *, cwd: str | None = None, check: bool = True) -> str:
    proc = subprocess.run(
        ["git", *args],
        cwd=cwd,
        check=check,
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
    )
    if not check and proc.returncode != 0:
        return ""
    return proc.stdout.strip()


def git_ok(args: list[str], *, cwd: str | None = None) -> bool:
    return subprocess.run(
        ["git", *args],
        cwd=cwd,
        check=False,
        stdout=subprocess.DEVNULL,
        stderr=subprocess.DEVNULL,
    ).returncode == 0


def topology_failures(
    parents: list[str],
    *,
    upstream_sha: str,
    local_sha: str,
    upstream_ref: str,
) -> list[str]:
    failures: list[str] = []
    if len(parents) != 2:
        return [f"candidate must have exactly two parents, found {len(parents)}"]
    if parents[0] != upstream_sha:
        failures.append(
            f"first parent must be current {upstream_ref} ({upstream_sha}), found {parents[0]}"
        )
    if parents[1] != local_sha:
        failures.append(
            f"second parent must be pre-sync local ref ({local_sha}), found {parents[1]}"
        )
    return failures


def require_clean(repo: str) -> None:
    if git(["status", "--porcelain"], cwd=repo):
        raise RuntimeError("current worktree is not clean")


def fetch_upstream(repo: str, upstream_ref: str) -> None:
    if "/" not in upstream_ref:
        raise RuntimeError("--upstream must be a remote-tracking ref such as upstream/main")
    remote = upstream_ref.split("/", 1)[0]
    subprocess.run(["git", "fetch", remote, "--prune"], cwd=repo, check=True)


def prepare(args: argparse.Namespace) -> int:
    repo = git(["rev-parse", "--show-toplevel"])
    require_clean(repo)
    if args.fetch:
        fetch_upstream(repo, args.upstream)

    local_sha = git(["rev-parse", args.local], cwd=repo)
    upstream_sha = git(["rev-parse", args.upstream], cwd=repo)
    upstream_only = int(git(["rev-list", "--count", f"{args.local}..{args.upstream}"], cwd=repo))
    if upstream_only == 0:
        print(f"{args.local} already contains {args.upstream}; no sync worktree created")
        return 0

    stamp = dt.datetime.now(dt.timezone.utc).strftime("%Y%m%d-%H%M%S")
    branch = args.branch or f"sync/upstream-{stamp}"
    worktree = os.path.abspath(args.worktree or f"/tmp/sub2api-upstream-{stamp}")
    if os.path.exists(worktree):
        raise RuntimeError(f"worktree path already exists: {worktree}")
    if git_ok(["show-ref", "--verify", f"refs/heads/{branch}"], cwd=repo):
        raise RuntimeError(f"branch already exists: {branch}")

    subprocess.run(
        ["git", "worktree", "add", "-b", branch, worktree, upstream_sha],
        cwd=repo,
        check=True,
    )
    merge = subprocess.run(
        ["git", "merge", "--no-ff", "--no-commit", local_sha],
        cwd=worktree,
        check=False,
    )
    print(f"worktree: {worktree}")
    print(f"branch: {branch}")
    print(f"official first parent: {upstream_sha}")
    print(f"site second parent: {local_sha}")
    if merge.returncode == 0:
        print("merge staged without conflicts; run the full verification suite before committing")
    else:
        print("merge has conflicts; resolve them in the worktree without aborting the merge")
    return merge.returncode


def verify(args: argparse.Namespace) -> int:
    repo = git(["rev-parse", "--show-toplevel"])
    candidate = git(["rev-parse", args.candidate], cwd=repo)
    local_sha = git(["rev-parse", args.local_before], cwd=repo)
    upstream_sha = git(["rev-parse", args.upstream], cwd=repo)
    parents = git(["show", "-s", "--format=%P", candidate], cwd=repo).split()

    failures = topology_failures(
        parents,
        upstream_sha=upstream_sha,
        local_sha=local_sha,
        upstream_ref=args.upstream,
    )
    if not git_ok(["merge-base", "--is-ancestor", upstream_sha, candidate], cwd=repo):
        failures.append("upstream ref is not an ancestor of the candidate")
    if not git_ok(["merge-base", "--is-ancestor", local_sha, candidate], cwd=repo):
        failures.append("pre-sync local ref is not an ancestor of the candidate")
    if git(["diff", "--name-only", "--diff-filter=U", candidate], cwd=repo):
        failures.append("candidate contains unresolved paths")

    if failures:
        print("upstream merge verification failed:", file=sys.stderr)
        for failure in failures:
            print(f"- {failure}", file=sys.stderr)
        return 2

    print(f"candidate: {candidate}")
    print(f"parents: {parents[0]} {parents[1]}")
    print("upstream-first two-parent merge verified")
    return 0


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description=__doc__)
    subparsers = parser.add_subparsers(dest="command", required=True)

    prepare_parser = subparsers.add_parser("prepare", help="create a worktree and start a real merge")
    prepare_parser.add_argument("--local", default="HEAD", help="pre-sync local ref")
    prepare_parser.add_argument("--upstream", default="upstream/main", help="official remote ref")
    prepare_parser.add_argument("--branch", help="sync branch name")
    prepare_parser.add_argument("--worktree", help="sync worktree path")
    prepare_parser.add_argument("--fetch", action="store_true", help="fetch the upstream remote first")
    prepare_parser.set_defaults(func=prepare)

    verify_parser = subparsers.add_parser("verify", help="verify a completed upstream-first merge")
    verify_parser.add_argument("--candidate", default="HEAD", help="merge commit to verify")
    verify_parser.add_argument("--local-before", required=True, help="pre-sync local commit/ref")
    verify_parser.add_argument("--upstream", default="upstream/main", help="official remote ref")
    verify_parser.set_defaults(func=verify)
    return parser


def main() -> int:
    args = build_parser().parse_args()
    try:
        return args.func(args)
    except (RuntimeError, subprocess.CalledProcessError, ValueError) as exc:
        print(f"upstream sync failed: {exc}", file=sys.stderr)
        return 2


if __name__ == "__main__":
    raise SystemExit(main())
