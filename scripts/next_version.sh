#!/usr/bin/env bash
# ==============================================================================
# next-version.sh
#
# Computes the next CalVer/SemVer version from the latest git tag and writes
# it to /tmp/.env.tmp as VERSION=<value>.
#
# Version format:  YEAR.MONTH.PATCH[-pre.N]
#   YEAR  — 2-digit calendar year  (e.g. 25)
#   MONTH — month WITHOUT leading zero (e.g. 3 for March)
#   PATCH — patch counter
#   pre.N — optional pre-release segment
#
# Increment rules (applied after the year/month check):
#   • Tag has NO pre-release  → bump PATCH by 1, append -pre.0
#     e.g.  25.3.4  →  25.3.5-pre.0
#   • Tag HAS a pre-release   → bump the pre-release counter only
#     e.g.  25.3.5-pre.2  →  25.3.5-pre.3
#
# Year/month correction (checked first, takes priority):
#   If YEAR or MONTH in the tag differ from today's values:
#     • Reset to  YEAR.MONTH.0
#     • If the original tag had a pre-release, keep the label but reset to .0
#       e.g.  24.11.3-pre.7  →  25.3.0-pre.0   (pre-release was present)
#       e.g.  24.11.3        →  25.3.0           (no pre-release)
# ==============================================================================
set -euo pipefail

# ------------------------------------------------------------------------------
# 1. Fetch the latest matching tag
# ------------------------------------------------------------------------------
RAW_TAG=$(git describe --tags --always --abbrev=0 \
            --match='[0-9]*.[0-9]*.[0-9]*' 2>/dev/null || true)

CURRENT_YEAR=$(date +%Y)          # 4-digit year, e.g. "2025"
CURRENT_MONTH=$(date +%-m)        # month, no leading zero, e.g. "3"

# ------------------------------------------------------------------------------
# 2. Handle missing / non-semver tags (first run, shallow clone, etc.)
# ------------------------------------------------------------------------------
if [[ -z "$RAW_TAG" ]] || ! [[ "$RAW_TAG" =~ ^[0-9]+\.[0-9]+\.[0-9]+ ]]; then
    echo "[next-version] No matching semver tag found — initialising at ${CURRENT_YEAR}.${CURRENT_MONTH}.0-pre.0" >&2
    VERSION="${CURRENT_YEAR}.${CURRENT_MONTH}.0-pre.0"

    printf 'VERSION=%s\n' "$VERSION" > /tmp/.env.tmp
    echo "[next-version] VERSION=${VERSION}"
    exit 0
fi

# ------------------------------------------------------------------------------
# 3. Parse MAJOR.MINOR.PATCH[-PRERELEASE]
# ------------------------------------------------------------------------------
#   Captures:
#     [1] MAJOR
#     [2] MINOR
#     [3] PATCH
#     [5] pre-release string (without the leading dash), empty if absent
if ! [[ "$RAW_TAG" =~ ^([0-9]+)\.([0-9]+)\.([0-9]+)(-([^+]+))?(\+.*)?$ ]]; then
    echo "[next-version] ERROR: tag '${RAW_TAG}' could not be parsed as semver." >&2
    exit 1
fi

MAJOR="${BASH_REMATCH[1]}"
MINOR="${BASH_REMATCH[2]}"
PATCH="${BASH_REMATCH[3]}"
PRE="${BASH_REMATCH[5]}"    # empty string when no pre-release

echo "[next-version] Last tag : ${RAW_TAG}  (major=${MAJOR} minor=${MINOR} patch=${PATCH} pre='${PRE}')"
echo "[next-version] Today    : year=${CURRENT_YEAR} month=${CURRENT_MONTH}"

# ------------------------------------------------------------------------------
# 4. Year/month correction — takes priority over normal increment rules
# ------------------------------------------------------------------------------
if [[ "$MAJOR" != "$CURRENT_YEAR" || "$MINOR" != "$CURRENT_MONTH" ]]; then
    echo "[next-version] Year/month mismatch — resetting to ${CURRENT_YEAR}.${CURRENT_MONTH}.0"

    if [[ -n "$PRE" ]]; then
        # Preserve the pre-release label (e.g. "pre", "alpha") but reset counter
        PRE_LABEL=$(echo "$PRE" | sed -E 's/\.[0-9]+$//')  # strip trailing .N
        VERSION="${CURRENT_YEAR}.${CURRENT_MONTH}.0-${PRE_LABEL}.0"
    else
        VERSION="${CURRENT_YEAR}.${CURRENT_MONTH}.0"
    fi

# ------------------------------------------------------------------------------
# 5. Normal increment — year and month are already current
# ------------------------------------------------------------------------------
else
    if [[ -z "$PRE" ]]; then
        # ── No pre-release: bump patch, add -pre.0 ──────────────────────────
        NEW_PATCH=$(( PATCH + 1 ))
        VERSION="${MAJOR}.${MINOR}.${NEW_PATCH}-pre.0"

    else
        # ── Has pre-release: bump only the pre-release counter ───────────────
        if [[ "$PRE" =~ ^(.+)\.([0-9]+)$ ]]; then
            PRE_LABEL="${BASH_REMATCH[1]}"
            PRE_NUM="${BASH_REMATCH[2]}"
            VERSION="${MAJOR}.${MINOR}.${PATCH}-${PRE_LABEL}.$(( PRE_NUM + 1 ))"
        else
            # Pre-release has no numeric suffix — append .1
            VERSION="${MAJOR}.${MINOR}.${PATCH}-${PRE}.1"
        fi
    fi
fi

# ------------------------------------------------------------------------------
# 6. Persist and report
# ------------------------------------------------------------------------------
printf 'VERSION=%s\n' "$VERSION" > /tmp/.env.tmp
echo "[next-version] VERSION=${VERSION}  →  written to /tmp/.env.tmp"