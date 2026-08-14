#!/usr/bin/env python3
"""Summarize divergence from the official upstream branch.

This is intentionally read-only. It gives maintainers a repeatable snapshot
before reconciling or merging the official upstream history.
"""

from __future__ import annotations

import argparse
import collections
import json
import subprocess
import sys
from dataclasses import dataclass
from typing import Iterable


@dataclass(frozen=True)
class Commit:
    sha: str
    subject: str
    category: str


@dataclass(frozen=True)
class PathChange:
    status: str
    old_path: str | None
    new_path: str | None

    @property
    def path(self) -> str:
        return self.new_path or self.old_path or ""


GENERATED_EXACT_PATHS = {
    "backend/cmd/server/wire_gen.go",
}


CATEGORY_KEYWORDS: tuple[tuple[str, tuple[str, ...]], ...] = (
    (
        "security",
        (
            "security",
            "vuln",
            "sanitize",
            "escape",
            "auth bypass",
            "鉴权",
            "漏洞",
            "安全",
        ),
    ),
    (
        "billing-payment",
        (
            "billing",
            "payment",
            "refund",
            "invoice",
            "quota",
            "balance",
            "price",
            "计费",
            "支付",
            "退款",
            "开票",
            "余额",
        ),
    ),
    (
        "gateway-openai",
        (
            "openai",
            "codex",
            "compact",
            "responses",
            "response_format",
            "failover",
            "gpt-5.6",
            "ws",
            "http_bridge",
        ),
    ),
    (
        "scheduler-observability",
        (
            "scheduler",
            "monitor",
            "ops",
            "usage",
            "log",
            "concurrency",
            "redis",
            "调度",
            "监控",
            "用量",
        ),
    ),
    (
        "providers",
        (
            "grok",
            "anthropic",
            "claude",
            "gemini",
            "antigravity",
            "xai",
        ),
    ),
    (
        "frontend",
        (
            "frontend",
            "i18n",
            "ui",
            "keys",
            "layout",
            "前端",
            "语言包",
        ),
    ),
    (
        "batch-image",
        (
            "batch image",
            "batch-image",
            "batch_image",
            "image",
            "生图",
            "图片",
        ),
    ),
    (
        "refactor",
        (
            "refactor",
            "split",
            "拆分",
            "纯移动",
        ),
    ),
)


def git(args: list[str], *, check: bool = True) -> str:
    proc = subprocess.run(
        ["git", *args],
        check=check,
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
    )
    return proc.stdout.strip()


def git_lines(args: list[str]) -> list[str]:
    output = git(args)
    return [line for line in output.splitlines() if line.strip()]


def git_succeeds(args: list[str]) -> bool:
    return subprocess.run(
        ["git", *args],
        stdout=subprocess.DEVNULL,
        stderr=subprocess.DEVNULL,
        check=False,
    ).returncode == 0


def categorize(subject: str) -> str:
    lower = subject.lower()
    for category, keywords in CATEGORY_KEYWORDS:
        if any(keyword in lower for keyword in keywords):
            return category
    return "other"


def parse_cherry(local_ref: str, upstream_ref: str) -> tuple[list[Commit], list[Commit]]:
    equivalent: list[Commit] = []
    missing: list[Commit] = []
    for line in git_lines(["cherry", "-v", local_ref, upstream_ref]):
        marker, rest = line[0], line[2:]
        parts = rest.split(" ", 1)
        sha = parts[0]
        subject = parts[1] if len(parts) > 1 else ""
        commit = Commit(sha=sha, subject=subject, category=categorize(subject))
        if marker == "-":
            equivalent.append(commit)
        elif marker == "+":
            missing.append(commit)
    return equivalent, missing


def scope_for_path(path: str) -> str:
    parts = path.split("/")
    if not parts:
        return path
    if parts[0] in {"backend", "frontend"} and len(parts) >= 2:
        if parts[0] == "backend" and len(parts) >= 3 and parts[1] == "internal":
            return "/".join(parts[:3])
        if parts[0] == "frontend" and len(parts) >= 3 and parts[1] == "src":
            return "/".join(parts[:3])
        return "/".join(parts[:2])
    return parts[0]


def count_scopes(paths: Iterable[str]) -> collections.Counter[str]:
    counter: collections.Counter[str] = collections.Counter()
    for path in paths:
        counter[scope_for_path(path)] += 1
    return counter


def is_generated_path(path: str) -> bool:
    if path in GENERATED_EXACT_PATHS:
        return True
    return path.startswith("backend/ent/") and not path.startswith("backend/ent/schema/")


def parse_path_changes(local_ref: str, upstream_ref: str) -> list[PathChange]:
    changes: list[PathChange] = []
    for line in git_lines(
        ["diff", "--name-status", "--find-renames", f"{upstream_ref}..{local_ref}"]
    ):
        parts = line.split("\t")
        status = parts[0]
        kind = status[:1]
        if kind in {"R", "C"} and len(parts) >= 3:
            changes.append(PathChange(status=status, old_path=parts[1], new_path=parts[2]))
        elif len(parts) >= 2:
            old_path = parts[1] if kind == "D" else None
            new_path = None if kind == "D" else parts[1]
            changes.append(PathChange(status=status, old_path=old_path, new_path=new_path))
    return changes


def collect_churn(local_ref: str, upstream_ref: str) -> dict[str, int]:
    churn: dict[str, int] = {}
    for line in git_lines(
        ["diff", "--numstat", "--find-renames", f"{upstream_ref}..{local_ref}"]
    ):
        parts = line.split("\t", 2)
        if len(parts) != 3 or not parts[0].isdigit() or not parts[1].isdigit():
            continue
        churn[parts[2]] = int(parts[0]) + int(parts[1])
    return churn


def check_failures(
    report: dict,
    *,
    allow_upstream_deletions: bool = False,
    max_handwritten_overlap: int | None = None,
) -> list[str]:
    failures: list[str] = []
    if not report["upstream_tip_is_ancestor"]:
        failures.append("upstream tip is not a Git ancestor of the local ref")
    if report["upstream_only_commits"] != 0:
        failures.append(f"{report['upstream_only_commits']} upstream commit(s) are not merged")
    if report["unaccounted_missing_upstream_commits"] != 0:
        failures.append(
            f"{report['unaccounted_missing_upstream_commits']} upstream commit(s) are unaccounted"
        )
    if report["deleted_upstream_files"] and not allow_upstream_deletions:
        failures.append(
            f"{report['deleted_upstream_files']} upstream-owned file(s) are deleted locally"
        )
    overlap = report["handwritten_upstream_overlap_files"]
    if max_handwritten_overlap is not None and overlap > max_handwritten_overlap:
        failures.append(
            f"handwritten upstream overlap increased to {overlap}; "
            f"maximum allowed is {max_handwritten_overlap}"
        )
    return failures


def collect(local_ref: str, upstream_ref: str, limit: int) -> dict:
    merge_base = git(["merge-base", local_ref, upstream_ref])
    local_sha = git(["rev-parse", local_ref])
    upstream_sha = git(["rev-parse", upstream_ref])
    ahead_behind = git(["rev-list", "--left-right", "--count", f"{local_ref}...{upstream_ref}"]).split()
    local_only, upstream_only = int(ahead_behind[0]), int(ahead_behind[1])

    equivalent, missing = parse_cherry(local_ref, upstream_ref)
    unaccounted_missing = missing

    changes = parse_path_changes(local_ref, upstream_ref)
    paths = [change.path for change in changes]
    statuses = collections.Counter(change.status for change in changes)
    local_additions = [change.path for change in changes if change.status.startswith("A")]
    deleted_upstream_paths = [change.path for change in changes if change.status.startswith("D")]
    generated_paths = [path for path in paths if is_generated_path(path)]
    upstream_overlap_paths = [
        change.path
        for change in changes
        if change.status[:1] in {"M", "D", "R", "T"}
    ]
    handwritten_overlap_paths = [
        path for path in upstream_overlap_paths if not is_generated_path(path)
    ]
    churn = collect_churn(local_ref, upstream_ref)
    overlap_hotspots = sorted(
        (
            {"path": path, "churn": churn.get(path, 0)}
            for path in handwritten_overlap_paths
        ),
        key=lambda item: (-item["churn"], item["path"]),
    )
    scopes = count_scopes(paths)
    categories = collections.Counter(commit.category for commit in missing)
    unaccounted_categories = collections.Counter(commit.category for commit in unaccounted_missing)

    migrations = sorted(path for path in paths if path.startswith("backend/migrations/"))

    return {
        "local_ref": local_ref,
        "upstream_ref": upstream_ref,
        "merge_base": merge_base,
        "upstream_tip_is_ancestor": git_succeeds(["merge-base", "--is-ancestor", upstream_ref, local_ref]),
        "local_sha": local_sha,
        "upstream_sha": upstream_sha,
        "local_only_commits": local_only,
        "upstream_only_commits": upstream_only,
        "patch_equivalent_upstream_commits": len(equivalent),
        "missing_upstream_commits": len(missing),
        "unaccounted_missing_upstream_commits": len(unaccounted_missing),
        "changed_files": len(paths),
        "status_counts": dict(sorted(statuses.items())),
        "local_addition_files": len(local_additions),
        "generated_delta_files": len(generated_paths),
        "modified_upstream_files": len(upstream_overlap_paths),
        "handwritten_upstream_overlap_files": len(handwritten_overlap_paths),
        "deleted_upstream_files": len(deleted_upstream_paths),
        "deleted_upstream_paths": deleted_upstream_paths[:limit],
        "upstream_overlap_hotspots": overlap_hotspots[:limit],
        "top_scopes": scopes.most_common(limit),
        "missing_categories": categories.most_common(),
        "unaccounted_missing_categories": unaccounted_categories.most_common(),
        "missing_commits": [commit.__dict__ for commit in missing[:limit]],
        "unaccounted_missing_commits": [commit.__dict__ for commit in unaccounted_missing[:limit]],
        "patch_equivalent_commits": [commit.__dict__ for commit in equivalent[:limit]],
        "migration_files": migrations[:limit],
        "migration_file_count": len(migrations),
    }


def print_markdown(report: dict) -> None:
    print("# Upstream Delta Audit")
    print()
    print(f"- Local ref: `{report['local_ref']}` `{report['local_sha'][:10]}`")
    print(f"- Upstream ref: `{report['upstream_ref']}` `{report['upstream_sha'][:10]}`")
    print(f"- Merge base: `{report['merge_base'][:10]}`")
    print(f"- Upstream tip is a Git ancestor of local: `{report['upstream_tip_is_ancestor']}`")
    print(f"- Local-only commits: `{report['local_only_commits']}`")
    print(f"- Upstream-only commits: `{report['upstream_only_commits']}`")
    print(f"- Patch-equivalent upstream commits: `{report['patch_equivalent_upstream_commits']}`")
    print(f"- Non-patch-equivalent upstream commits requiring reconciliation: `{report['missing_upstream_commits']}`")
    print(f"- Unaccounted missing upstream commits: `{report['unaccounted_missing_upstream_commits']}`")
    print(f"- Changed files in `{report['upstream_ref']}..{report['local_ref']}`: `{report['changed_files']}`")
    print(f"- Name-status counts: `{report['status_counts']}`")
    print(f"- Local-only additions: `{report['local_addition_files']}`")
    print(f"- Generated delta files: `{report['generated_delta_files']}`")
    print(f"- Modified upstream-owned files: `{report['modified_upstream_files']}`")
    print(f"- Handwritten upstream overlap files: `{report['handwritten_upstream_overlap_files']}`")
    print(f"- Deleted upstream-owned files: `{report['deleted_upstream_files']}`")
    print()

    print("## Reconciliation Debt Categories")
    for category, count in report["missing_categories"]:
        print(f"- `{category}`: {count}")
    print()

    print("## Top Changed Scopes")
    for scope, count in report["top_scopes"]:
        print(f"- `{scope}`: {count}")
    print()

    print("## Migration Files")
    print(f"Total: `{report['migration_file_count']}`")
    for path in report["migration_files"]:
        print(f"- `{path}`")
    print()

    print("## Deleted Upstream-Owned Files")
    for path in report["deleted_upstream_paths"]:
        print(f"- `{path}`")
    print()

    print("## Handwritten Overlap Hotspots")
    for item in report["upstream_overlap_hotspots"]:
        print(f"- `{item['path']}`: {item['churn']} changed line(s)")
    print()

    print("## Unaccounted Missing Upstream Commits")
    for commit in report["unaccounted_missing_commits"]:
        print(f"- `{commit['sha'][:10]}` `{commit['category']}` {commit['subject']}")
def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--local", default="HEAD", help="local ref to compare from")
    parser.add_argument("--upstream", default="upstream/main", help="official upstream ref")
    parser.add_argument("--fetch", action="store_true", help="run git fetch for the upstream remote first")
    parser.add_argument("--json", action="store_true", help="emit JSON instead of Markdown")
    parser.add_argument(
        "--check",
        action="store_true",
        help="exit non-zero unless upstream ancestry is complete and no upstream file is deleted",
    )
    parser.add_argument(
        "--allow-upstream-deletions",
        action="store_true",
        help="do not fail --check for intentionally deleted upstream-owned files",
    )
    parser.add_argument(
        "--require-clean",
        action="store_true",
        help="fail when the current worktree has tracked or untracked changes",
    )
    parser.add_argument(
        "--max-handwritten-overlap",
        type=int,
        help="fail --check when modified handwritten upstream files exceed this count",
    )
    parser.add_argument("--limit", type=int, default=80, help="maximum list entries to print")
    args = parser.parse_args()

    if args.max_handwritten_overlap is not None and args.max_handwritten_overlap < 0:
        parser.error("--max-handwritten-overlap must be non-negative")

    if args.fetch:
        remote = args.upstream.split("/", 1)[0] if "/" in args.upstream else "upstream"
        subprocess.run(["git", "fetch", remote, "--prune"], check=True)

    try:
        report = collect(args.local, args.upstream, args.limit)
    except subprocess.CalledProcessError as exc:
        sys.stderr.write(exc.stderr or str(exc))
        return exc.returncode or 1

    if args.json:
        print(json.dumps(report, ensure_ascii=False, indent=2))
    else:
        print_markdown(report)

    failures = check_failures(
        report,
        allow_upstream_deletions=args.allow_upstream_deletions,
        max_handwritten_overlap=args.max_handwritten_overlap,
    ) if args.check else []
    if args.require_clean and git_lines(["status", "--porcelain"]):
        failures.append("worktree is not clean")
    if failures:
        sys.stderr.write("Upstream alignment check failed:\n")
        for failure in failures:
            sys.stderr.write(f"- {failure}\n")
        return 2
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
