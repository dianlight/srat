#!/bin/bash
set -e

PACKAGE_JSON_PATH="frontend/package.json"

# Ensure the script is run from the repository root
if ! GIT_ROOT=$(git rev-parse --show-toplevel 2>/dev/null); then
    echo "Error: Not a git repository."
    exit 1
fi
cd "$GIT_ROOT" || exit 1

# 1. Get describe output to find the base tag and commit count
# The pattern '[0-9]*.[0-9]*.[0-9]*' matches tags like '1.2.3', '0.1.0', etc.
# --long format: TAG-COMMITS_SINCE_TAG-gHASH
DESCRIBE_OUTPUT=$(git describe --tags --match='[0-9]*.[0-9]*.[0-9]*' --long 2>/dev/null)

if [ -z "$DESCRIBE_OUTPUT" ]; then
    echo "No matching tag found (pattern: '[0-9]*.[0-9]*.[0-9]*'). Cannot determine version."
    echo "Please ensure you have a relevant tag like '1.0.0' in your project history."
    exit 1
fi

# Parse the output (e.g., 1.2.3-5-gabcdef or 1.2.3-0-gabcdef)
# $1 = TAG, $2 = COMMITS_SINCE_TAG
BASE_TAG=$(echo "$DESCRIBE_OUTPUT" | awk -F- '{print $1}')
COMMIT_COUNT=$(echo "$DESCRIBE_OUTPUT" | awk -F- '{print $2}')

# Validate parsing
if ! [[ "$BASE_TAG" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || ! [[ "$COMMIT_COUNT" =~ ^[0-9]+$ ]]; then
    echo "Error: Failed to parse git describe output: $DESCRIBE_OUTPUT"
    echo "Expected format: TAG-COUNT-gHASH (e.g., 1.2.3-0-gabcdef0)"
    exit 1
fi

# 2. Determine the new version string
NEW_VERSION="$BASE_TAG"
if [ "$COMMIT_COUNT" -gt 0 ]; then
    NEW_VERSION="${BASE_TAG}.dev${COMMIT_COUNT}"
fi

echo "Determined version for $PACKAGE_JSON_PATH: $NEW_VERSION"

# 3. Update package.json
if ! command -v jq &> /dev/null; then
    echo "Error: jq command could not be found. Please install jq."
    exit 1
fi

# Check if package.json exists and is readable
if [ ! -f "$PACKAGE_JSON_PATH" ]; then
    echo "Error: $PACKAGE_JSON_PATH not found."
    exit 1
fi
if [ ! -r "$PACKAGE_JSON_PATH" ]; then
    echo "Error: Cannot read $PACKAGE_JSON_PATH."
    exit 1
fi

CURRENT_VERSION=$(jq -r '.version' "$PACKAGE_JSON_PATH")

if [ "$CURRENT_VERSION" == "$NEW_VERSION" ]; then
    echo "$PACKAGE_JSON_PATH version is already $NEW_VERSION. No changes needed."
    exit 0
fi

TMP_PACKAGE_JSON=$(mktemp)
if jq ".version = \"$NEW_VERSION\"" "$PACKAGE_JSON_PATH" > "$TMP_PACKAGE_JSON"; then
    mv "$TMP_PACKAGE_JSON" "$PACKAGE_JSON_PATH"
    echo "Updated $PACKAGE_JSON_PATH version to $NEW_VERSION"
else
    echo "Error: Failed to update version in $PACKAGE_JSON_PATH using jq."
    rm -f "$TMP_PACKAGE_JSON"
    exit 1
fi

# 4. Stage the changes to package.json
git add "$PACKAGE_JSON_PATH"

exit 0
