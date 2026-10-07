#!/usr/bin/env python3
"""Recompute SECURITY.md supported versions from GitHub Releases (API).

Rules:
  - The latest stable release is always supported.
  - One release per month (max patchset, ``YYYY.M.x``) inside the rolling
    6 calendar months ending at the latest stable release is supported.
  - RC/dev releases (tag suffix ``-rc.`` / ``-dev.`` or API prerelease flag)
    are never supported and never trigger an update.

Stable means: API ``draft == False``, ``prerelease == False`` AND tag matches
``YYYY.M.PATCH`` with no suffix. The suffix check is load-bearing: this repo
publishes ``-rc`` releases with ``prerelease == False`` (see build.yaml), so
the flag alone cannot be trusted.
"""

from __future__ import annotations

import argparse
import json
import os
import re
import subprocess
import sys
from datetime import datetime, timezone

STABLE_RE = re.compile(r"^(\d{4})\.(\d{1,2})\.(\d+)$")
# Matches -rc18, -rc.1, -dev.0, -dev0 (case-insensitive). Real tags in this
# repo use "-rc18" (no dot), so the dot must stay optional.
SUFFIX_RE = re.compile(r"-(rc|dev)", re.IGNORECASE)
TABLE_HEADER_RE = re.compile(r"^\|\s*Version\s*\|\s*Supported\s*\|")

SECURITY_MD = "SECURITY.md"
TABLE_HEADER = "| Version | Supported |"


def run(cmd: list[str], env: dict[str, str] | None = None) -> str:
    out = subprocess.run(cmd, capture_output=True, text=True, check=True, env=env)
    return out.stdout


def fetch_releases(repo: str, token: str | None) -> list[dict]:
    """Fetch non-draft releases via the GitHub REST API (respects API flags)."""
    env = dict(os.environ)
    if token:
        env["GH_TOKEN"] = token
    raw = run(
        [
            "gh",
            "api",
            "--paginate",
            f"/repos/{repo}/releases",
            "--jq",
            (
                "[.[] | select(.draft == false) "
                "| {tag: .tag_name, pre: .prerelease, "
                "pub: (.published_at // .created_at)}]"
            ),
        ],
        env=env,
    )
    # Paginated output is one JSON array per page; merge them.
    releases: list[dict] = []
    for line in raw.splitlines():
        line = line.strip()
        if not line:
            continue
        releases.extend(json.loads(line))
    return releases


def is_stable(rel: dict) -> bool:
    if rel.get("pre"):
        return False
    tag = str(rel.get("tag", ""))
    if SUFFIX_RE.search(tag):
        return False
    return STABLE_RE.match(tag) is not None


def parse_stable(rel: dict) -> tuple[int, int, int, datetime] | None:
    m = STABLE_RE.match(str(rel.get("tag", "")))
    if not m:
        return None
    try:
        pub = datetime.fromisoformat(str(rel["pub"]).replace("Z", "+00:00"))
    except (KeyError, ValueError):
        return None
    if pub.tzinfo is None:
        pub = pub.replace(tzinfo=timezone.utc)
    return (int(m.group(1)), int(m.group(2)), int(m.group(3)), pub)


def compute_supported(releases: list[dict]) -> list[str]:
    """Return supported ``YYYY.M.x`` series, newest first."""
    stable: list[tuple[int, int, int, datetime]] = []
    for rel in releases:
        if not is_stable(rel):
            continue
        parsed = parse_stable(rel)
        if parsed:
            stable.append(parsed)
    if not stable:
        return []

    latest = max(stable, key=lambda s: (s[0], s[1], s[2]))
    latest_idx = latest[0] * 12 + latest[1]

    # Max patchset per YYYY.M group.
    groups: dict[tuple[int, int], tuple[int, int, int, datetime]] = {}
    for year, month, patch, pub in stable:
        key = (year, month)
        if key not in groups or patch > groups[key][2]:
            groups[key] = (year, month, patch, pub)

    # Rolling 6 calendar months ending at the latest stable (inclusive).
    supported = [
        f"{y}.{m}.x"
        for (y, m), (_, _, _, _) in groups.items()
        if 0 <= latest_idx - (y * 12 + m) <= 5
    ]
    # Latest is always supported, even if its date is an outlier.
    latest_series = f"{latest[0]}.{latest[1]}.x"
    if latest_series not in supported:
        supported.append(latest_series)

    def key_fn(s: str) -> tuple[int, int]:
        y, m, _ = s.split(".")
        return (int(y), int(m))

    return sorted(supported, key=key_fn, reverse=True)


def render_table(supported: list[str]) -> list[str]:
    lines = [
        TABLE_HEADER,
        "| ------- | --------- |",
    ]
    for series in supported:
        lines.append(f"| {series} | :white_check_mark: |")
    oldest = supported[-1]  # ascending tail; list is newest-first
    base = oldest.replace(".x", ".0")
    lines.append(f"| < {base} | :x: |")
    return lines


def update_security_md(supported: list[str], path: str = SECURITY_MD) -> bool:
    new_table = render_table(supported)
    with open(path, encoding="utf-8") as fh:
        content = fh.read()
    lines = content.splitlines()
    try:
        start = next(
            i for i, line in enumerate(lines) if TABLE_HEADER_RE.match(line.strip())
        )
    except StopIteration:
        print(f"error: table header not found in {path}", file=sys.stderr)
        return False
    end = start + 1
    while end < len(lines) and lines[end].lstrip().startswith("|"):
        end += 1
    updated = lines[:start] + new_table + lines[end:]
    new_content = "\n".join(updated) + "\n"
    if new_content == content:
        return False
    with open(path, "w", encoding="utf-8") as fh:
        fh.write(new_content)
    return True


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--repo", default=os.environ.get("GITHUB_REPOSITORY", ""))
    parser.add_argument("--token", default=os.environ.get("GH_TOKEN", ""))
    parser.add_argument(
        "--releases-json",
        default="",
        help="JSON file with [{tag, pre, pub}] for offline testing.",
    )
    parser.add_argument("--security-md", default=SECURITY_MD)
    parser.add_argument(
        "--trigger-tag",
        default=os.environ.get("TRIGGER_TAG", ""),
        help="Release tag that triggered this run; RC/dev tags skip the update.",
    )
    parser.add_argument(
        "--trigger-prerelease",
        default=os.environ.get("TRIGGER_PRERELEASE", ""),
        help="API prerelease flag of the triggering release ('true' skips).",
    )
    args = parser.parse_args()

    # RC rule: an RC/dev trigger never updates the policy; the stable
    # release that follows it will.
    if args.trigger_tag and (
        SUFFIX_RE.search(args.trigger_tag) or args.trigger_prerelease.lower() == "true"
    ):
        print(f"Skipping: trigger {args.trigger_tag!r} is a pre-release.")
        return 0

    if args.releases_json:
        with open(args.releases_json, encoding="utf-8") as fh:
            releases = json.load(fh)
    else:
        if not args.repo:
            print("error: --repo or GITHUB_REPOSITORY is required", file=sys.stderr)
            return 2
        releases = fetch_releases(args.repo, args.token or None)

    supported = compute_supported(releases)
    if not supported:
        print("No stable releases found; leaving SECURITY.md unchanged.")
        return 0

    changed = update_security_md(supported, path=args.security_md)
    print(
        f"Supported: {', '.join(supported)} ({'updated' if changed else 'no change'})"
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
