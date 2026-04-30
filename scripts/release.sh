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
	# SC2143: Use grep -q instead of checking for non-empty string output
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

# 2. Calculate next version (YYYY.MM.Patch)
log "Calculating next version..."
CURRENT_YEAR_MONTH=$(date +"%Y.%m")
LATEST_TAG=$(git tag --sort=-v:refname | head -n 1)

if [[ -z "$LATEST_TAG" ]]; then
	CALCULATED_VERSION="${CURRENT_YEAR_MONTH}.0"
else
	# Handle versions with suffixes like 2026.03.1-rc3
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

if [[ -n "$VERSION" ]]; then
	VERSION_BASE="${VERSION%-*}"
	CALC_BASE="${CALCULATED_VERSION%-*}"

	if [[ "$(printf '%s\n%s' "$VERSION_BASE" "$CALC_BASE" | sort -V | head -n1)" == "$VERSION_BASE" && "$VERSION_BASE" != "$CALC_BASE" ]]; then
		read -p "Warning: Provided version ($VERSION) is lower than suggested ($CALCULATED_VERSION). Proceed? (y/n) " -n 1 -r
		echo
		if [[ ! $REPLY =~ ^[Yy]$ ]]; then error "Aborted by user."; fi
	fi
	NEXT_VERSION=$VERSION
else
	NEXT_VERSION=$CALCULATED_VERSION
fi
log "Target version set to: $NEXT_VERSION"

# 3. Check for uncommitted files
if [[ "$IGNORE_UNCOMMITTED" == "false" ]]; then
	if [[ -n "$(git status --porcelain)" ]]; then
		error "Uncommitted changes found. Use --ignore-uncommitted to skip check."
	fi
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

# 8. Update CHANGELOG.md (Version only)
log "Checking CHANGELOG.md status..."
SKIP_CHANGELOG_COMMIT=false
if [[ -f "CHANGELOG.md" ]]; then
	if grep -q "## \[ 🚧 Unreleased \]" CHANGELOG.md; then
		log "Updating CHANGELOG.md with $NEXT_VERSION..."
		sed -i "s/## \[ 🚧 Unreleased \]/## $NEXT_VERSION/" CHANGELOG.md
	elif grep -q "## $NEXT_VERSION" CHANGELOG.md; then
		log "Version $NEXT_VERSION already found in CHANGELOG.md."
		read -p ">> Continue without updating CHANGELOG? [y/N]: " -n 1 -r
		echo
		if [[ ! $REPLY =~ ^[Yy]$ ]]; then error "Aborted by user."; fi
		SKIP_CHANGELOG_COMMIT=true
	else
		error "No 'Unreleased' section found and version $NEXT_VERSION is missing from CHANGELOG.md."
	fi
else
	error "CHANGELOG.md not found."
fi

# 9. Commit, Push and wait for CI to create assets
if [[ "$SKIP_CHANGELOG_COMMIT" == "false" ]]; then
	log "Committing version update to trigger build..."
	confirm "Commit and push CHANGELOG update for $NEXT_VERSION?"
	git add CHANGELOG.md
	git commit -m "chore: release $NEXT_VERSION"
	git push origin main
else
	log "Skipping CHANGELOG commit as version is already present."
fi

# 6. Check for Draft Release after last push
LAST_PUSH_DATE=$(git log -1 --format=%cI)
check_draft_release() {
	gh api repos/:owner/:repo/releases --jq ".[] | select(.draft == true and .updated_at > \"$LAST_PUSH_DATE\")"
}

log "Waiting for a draft release updated after $LAST_PUSH_DATE..."
while true; do
	DRAFT=$(check_draft_release)
	if [[ -n "$DRAFT" ]]; then
		DRAFT_ID=$(echo "$DRAFT" | jq -r '.id')
		log "Found Draft Release ID: $DRAFT_ID"
		break
	fi
	if [[ "$NO_WAIT" == "true" ]]; then error "No draft release found and --no-wait is set."; fi

	printf "  Draft not found yet. Waiting... "
	(sleep 30) &
	show_spinner $!
	echo ""
done

# 7. Check for running GitHub Actions on main
log "Checking for active workflow runs on main..."
while true; do
	RUNNING_ACTIONS=$(gh run list --branch main --status in_progress --status queued --limit 5 --json databaseId --jq '.[].databaseId')
	if [[ -z "$RUNNING_ACTIONS" ]]; then
		log "No active actions on main. Proceeding..."
		break
	fi
	if [[ "$NO_WAIT" == "true" ]]; then error "GitHub Actions are still running on main and --no-wait is set."; fi

	printf "  Workflows are still running. Waiting... "
	(sleep 30) &
	show_spinner $!
	echo ""
done

log "Waiting for CI build on main to update draft assets..."
(sleep 15) &
show_spinner $!
echo ""

while true; do
	STATUS=$(gh run list --branch main --limit 1 --json status,conclusion --jq '.[0]')
	RUN_STATUS=$(echo "$STATUS" | jq -r '.status')
	RUN_CONCLUSION=$(echo "$STATUS" | jq -r '.conclusion')

	if [[ "$RUN_STATUS" == "completed" ]]; then
		if [[ "$RUN_CONCLUSION" != "success" ]]; then
			error "CI Build on main failed with conclusion: $RUN_CONCLUSION"
		fi
		break
	fi

	# SC2059: Pass variables as arguments to printf format string
	printf "  Waiting for CI build (%s)... " "$RUN_STATUS"
	(sleep 30) &
	show_spinner $!
	echo ""
done

# 10. Finalize Draft Release
log "Publishing release $NEXT_VERSION..."
confirm "Publish draft release $DRAFT_ID as $NEXT_VERSION?"
gh release edit "$DRAFT_ID" --tag "$NEXT_VERSION" --title "$NEXT_VERSION" --draft=false

# 11. Prepend Unreleased and move Thanks/Notes
log "Resetting CHANGELOG.md for next cycle..."

# Use native Bash parameter expansion to escape dots for the awk pattern (Fixes SC2001)
ESCAPED_VERSION="${NEXT_VERSION//./\\.}"
THANKS_NOTES=$(awk "/## $ESCAPED_VERSION/{flag=1;next}/##/{flag=0}flag" CHANGELOG.md | grep -E "### (🙏 Thanks|🚨 Notes)" -A 10 || true)

if [[ -n "$THANKS_NOTES" ]]; then
	sed -i "/### 🙏 Thanks/d" CHANGELOG.md
	sed -i "/### 🚨 Notes/d" CHANGELOG.md
fi

NEW_UNRELEASED_HEADER=$(
	cat <<EOF
## [ 🚧 Unreleased ]

### ✨ Features

### 🐛 Bug Fixes

### 🏗 Chore

$THANKS_NOTES
EOF
)

echo "$NEW_UNRELEASED_HEADER" >CHANGELOG.md.tmp
echo "" >>CHANGELOG.md.tmp
cat CHANGELOG.md >>CHANGELOG.md.tmp
mv CHANGELOG.md.tmp CHANGELOG.md

# 12. Final commit
log "Finalizing release cycle..."
confirm "Commit and push CHANGELOG reset for the next cycle?"
git add CHANGELOG.md
git commit -m "chore: reset changelog for next development cycle"
git push origin main

log "Release $NEXT_VERSION complete!"
