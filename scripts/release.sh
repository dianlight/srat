#!/bin/bash

# release.sh - Script for automated release process
# Usage: ./release.sh [--version 2026.03.1-rc3] [--ignore-uncommitted] [--no-wait] [--interactive]

set -e

# --- Configuration & Defaults ---
VERSION=""
IGNORE_UNCOMMITTED=false
NO_WAIT=false
INTERACTIVE=false

# --- Helper Functions ---
log() { echo -e "\033[1;32m[RELEASE]\033[0m $1"; }
error() {
	echo -e "\033[1;31m[ERROR]\033[0m $1"
	exit 1
}

usage() {
	echo "Automated Release Script"
	echo ""
	echo "Usage: $0 [options]"
	echo ""
	echo "Options:"
	echo "  --version <v>           Specify the version to release (e.g., 2026.04.1-rc1)."
	echo "                          If omitted, the script calculates the next patch version."
	echo "  --ignore-uncommitted    Allow the script to run even if there are uncommitted changes."
	echo "  --no-wait               Exit with error instead of polling if draft/actions are pending."
	echo "  --interactive           Ask for confirmation before every commit, push, or release action."
	echo "  --help                  Display this help message."
	echo ""
	exit 0
}

confirm() {
	if [[ "$INTERACTIVE" == "true" ]]; then
		read -p ">> $1 [y/N]: " -n 1 -r
		echo
		if [[ ! $REPLY =~ ^[Yy]$ ]]; then
			error "Aborted by user."
		fi
	fi
}

show_spinner() {
	local pid=$1
	local delay=0.1
	# shellcheck disable=SC1003
	local spinstr='|/-\'
	while ps a | awk '{print $1}' | grep -q "$pid"; do
		local temp=${spinstr#?}
		printf " [%c]  " "$spinstr"
		spinstr=$temp${spinstr%"$temp"}
		sleep "$delay"
		printf "\b\b\b\b\b\b"
	done
	printf "    \b\b\b\b"
}

# Parse arguments
while [[ "$#" -gt 0 ]]; do
	case $1 in
	--version)
		VERSION="$2"
		shift
		;;
	--ignore-uncommitted) IGNORE_UNCOMMITTED=true ;;
	--no-wait) NO_WAIT=true ;;
	--interactive) INTERACTIVE=true ;;
	--help) usage ;;
	*)
		echo "Unknown option: $1"
		usage
		;;
	esac
	shift
done

# 1. Refresh all tags
log "Refreshing tags from remote..."
git fetch --tags --force

# 2. Calculate next version
log "Calculating next version..."
CURRENT_YEAR_MONTH=$(date +"%Y.%m")
LATEST_TAG=$(git tag --sort=-v:refname | head -n 1)

if [[ -z "$LATEST_TAG" ]]; then
	CALCULATED_VERSION="${CURRENT_YEAR_MONTH}.0"
else
	TAG_WITHOUT_SUFFIX="${LATEST_TAG%-*}"
	TAG_YM="${TAG_WITHOUT_SUFFIX%.*}"
	TAG_PATCH="${TAG_WITHOUT_SUFFIX##*.}"

	if [[ "$TAG_YM" == "$CURRENT_YEAR_MONTH" ]]; then
		if [[ "$TAG_PATCH" =~ ^[0-9]+$ ]]; then
			CALCULATED_VERSION="${TAG_YM}.$((TAG_PATCH + 1))"
		else
			CALCULATED_VERSION="${TAG_YM}.1"
		fi
	else
		CALCULATED_VERSION="${CURRENT_YEAR_MONTH}.0"
	fi
fi

NEXT_VERSION=${VERSION:-$CALCULATED_VERSION}
log "Target version set to: $NEXT_VERSION"

# 3. Check for uncommitted files
if [[ "$IGNORE_UNCOMMITTED" == "false" && -n "$(git status --porcelain)" ]]; then
	error "Uncommitted changes found. Use --ignore-uncommitted to skip check."
fi

# 4. Switch to main
CURRENT_BRANCH=$(git branch --show-current)
if [[ "$CURRENT_BRANCH" != "main" ]]; then
	log "Switching to main branch..."
	git checkout main
fi

# 5. Push unpushed commits
log "Ensuring main is synced with remote..."
confirm "Push existing commits to origin main?"
git push origin main

# 6. Update CHANGELOG.md (initial version header)
log "Checking CHANGELOG.md status..."
SKIP_CHANGELOG_COMMIT=false
if [[ -f "CHANGELOG.md" ]]; then
	if grep -q "## \[ 🚧 Unreleased \]" CHANGELOG.md; then
		log "Updating CHANGELOG.md with $NEXT_VERSION..."
		sed -i "s/## \[ 🚧 Unreleased \]/## $NEXT_VERSION/" CHANGELOG.md
	elif grep -q "## $NEXT_VERSION" CHANGELOG.md; then
		log "Version $NEXT_VERSION already found in CHANGELOG.md."
		confirm "Continue without updating CHANGELOG?"
		SKIP_CHANGELOG_COMMIT=true
	else
		error "No 'Unreleased' section found and version $NEXT_VERSION is missing."
	fi
else
	error "CHANGELOG.md not found."
fi

# 7. Commit and Push to trigger CI
if [[ "$SKIP_CHANGELOG_COMMIT" == "false" ]]; then
	log "Committing version update..."
	confirm "Commit and push CHANGELOG update for $NEXT_VERSION?"
	git add CHANGELOG.md
	git commit -m "chore: release $NEXT_VERSION"
	git push origin main
fi

# 8. Find Draft Release
LAST_PUSH_DATE=$(git log -1 --format=%cI)
log "Waiting for a draft release updated after $LAST_PUSH_DATE..."
while true; do
	DRAFT=$(gh api repos/:owner/:repo/releases --jq ".[] | select(.draft == true and .updated_at > \"$LAST_PUSH_DATE\")")
	if [[ -n "$DRAFT" ]]; then
		DRAFT_ID=$(echo "$DRAFT" | jq -r '.id')
		log "Found Draft Release ID: $DRAFT_ID"
		break
	fi
	if [[ "$NO_WAIT" == "true" ]]; then error "No draft release found."; fi
	printf "  Waiting for draft... "
	(sleep 15) &
	show_spinner $!
	echo ""
done

# --- 10. PUBLISHING PROCESS (3-STEP API APPROACH) ---

# STEP 1: Update metadata (tag/title) but keep as draft
log "[PUBLISH STEP 1] Updating draft metadata for $NEXT_VERSION..."
gh api -X PATCH "repos/:owner/:repo/releases/$DRAFT_ID" \
	-f tag_name="$NEXT_VERSION" \
	-f name="$NEXT_VERSION" \
	-F draft=true >/dev/null

# STEP 2: Delete existing assets to ensure clean CI upload
log "[PUBLISH STEP 2] Clearing existing assets from release..."
ASSET_IDS=$(gh api "repos/:owner/:repo/releases/$DRAFT_ID/assets" --jq '.[].id')
for asset_id in $ASSET_IDS; do
	gh api -X DELETE "repos/:owner/:repo/releases/assets/$asset_id"
done

# STEP 3: Trigger CI manual execution for build.yaml to produce release assets
log "Triggering CI workflow for release assets..."
gh workflow run build.yaml --ref main -f release=true
sleep 5 # Short delay to ensure workflow is registered

# INTERMEDIATE: Wait for CI and Check Workflows
log "Checking for active workflow runs on main before final publish..."
while true; do
	RUNNING_ACTIONS=$(gh run list --branch main --status in_progress --status queued --limit 5 --json databaseId --jq '.[].databaseId')
	if [[ -z "$RUNNING_ACTIONS" ]]; then
		log "No active actions on main."
		break
	fi
	if [[ "$NO_WAIT" == "true" ]]; then error "GitHub Actions are still running."; fi
	printf "  Workflows running. Waiting... "
	(sleep 30) &
	show_spinner $!
	echo ""
done

confirm "Proceed to STEP 3: Finalize and Publish release $NEXT_VERSION?"

# STEP 3: Remove draft state and publish
log "[PUBLISH STEP 3] Publishing release..."
MAX_RETRIES=5
RETRY_COUNT=0
while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
	if gh api -X PATCH "repos/:owner/:repo/releases/$DRAFT_ID" -F draft=false >/dev/null 2>/tmp/gh_error; then
		log "Successfully published release $NEXT_VERSION."
		break
	else
		RETRY_COUNT=$((RETRY_COUNT + 1))
		ERROR_MSG=$(cat /tmp/gh_error)
		log "Warning: API Error: $ERROR_MSG. Retrying ($RETRY_COUNT/$MAX_RETRIES)..."
		sleep 5
	fi
done

# 11. Reset CHANGELOG.md for next cycle
log "Resetting CHANGELOG.md for next cycle..."
ESCAPED_VERSION="${NEXT_VERSION//./\\.}"

# Capture Thanks/Notes from the version we just released
THANKS_NOTES=$(awk "/## $ESCAPED_VERSION/{flag=1;next}/##/{flag=0}flag" CHANGELOG.md | awk '/### (🙏 Thanks|🚨 Notes)/{flag=1}flag' || true)

TEMP_CHANGELOG="CHANGELOG.md.tmp"

# Reconstruction logic with grouped redirects for style/efficiency (SC2129)
{
	# 1. Header
	head -n 1 CHANGELOG.md
	echo ""
	# 2. New Unreleased section with migrated Thanks/Notes
	cat <<EOF
## [ 🚧 Unreleased ]

### ✨ Features

### 🐛 Bug Fixes

### 🏗 Chore

$THANKS_NOTES
EOF
	echo ""
} >"$TEMP_CHANGELOG"

# 3. Rest of the file, stripping migrated sections from the old version block
awk "
  /## $ESCAPED_VERSION/ { in_ver=1; print; next }
  in_ver && /### (🙏 Thanks|🚨 Notes)/ { skip=1; next }
  in_ver && /^## / { in_ver=0; skip=0 }
  !skip { if (NR > 1) print }
" CHANGELOG.md >>"$TEMP_CHANGELOG"

mv "$TEMP_CHANGELOG" CHANGELOG.md

# 12. Final commit
log "Finalizing release cycle..."
confirm "Commit and push CHANGELOG reset for the next cycle?"
git add CHANGELOG.md
git commit -m "chore: reset changelog for next development cycle"
git push origin main

log "Release $NEXT_VERSION complete!"
