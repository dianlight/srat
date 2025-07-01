#!/bin/bash
set -e

if [[ "${GIT_REFLOG_ACTION}" =~ "rebase".*"reword" ]]; then
    exit 0
fi

PACKAGE_JSON_PATH="frontend/package.json"

# Ensure the script is run from the repository root
if ! GIT_ROOT=$(git rev-parse --show-toplevel 2>/dev/null); then
    echo "Error: Not a git repository."
    exit 1
fi
cd "$GIT_ROOT" || exit 1

# 1. Get describe output to find the base tag and commit count
# The pattern '[0-9]*.[0-9]*.[0-9]*' matches tags like '1.2.3', '0.1.0', etc.
# We also want to match tags like '1.2.3-dev.1'.
# --long format: TAG-COMMITS_SINCE_TAG-gHASH
DESCRIBE_OUTPUT=$(git describe --tags --match='[0-9]*.[0-9]*.[0-9]*-dev.[0-9]*' --match='[0-9]*.[0-9]*.[0-9]*' --long 2>/dev/null)

if [ -z "$DESCRIBE_OUTPUT" ]; then
    echo "No matching tag found (patterns: 'X.Y.Z-dev.N', 'X.Y.Z'). Cannot determine version."
    echo "Please ensure you have a relevant tag like '1.0.0' or '1.0.0-dev.1' in your project history."
    exit 1
fi

# Parse the output (e.g., 1.2.3-5-gabcdef, 1.2.3-dev.1-0-gabcdef)
# Expected format: TAG-COMMITS_SINCE_TAG-gHASH
if [[ "$DESCRIBE_OUTPUT" =~ ^(.*)-([0-9]+)-g[0-9a-f]+$ ]]; then
    TAG_FROM_DESCRIBE=${BASH_REMATCH[1]} # This can be '1.2.3' or '1.2.3-dev.1'
    COMMIT_COUNT=${BASH_REMATCH[2]}
else
    echo "Error: Failed to parse git describe output: $DESCRIBE_OUTPUT"
    echo "Expected format like 'TAG-COMMITS-gHASH' (e.g., 1.2.3-0-gabcdef0 or 1.2.3-dev.1-0-gabcdef0)"
    exit 1
fi

# Validate the format of the tag part obtained from git describe
if ! [[ "$TAG_FROM_DESCRIBE" =~ ^[0-9]+\.[0-9]+\.[0-9]+(-dev\.[0-9]+)?$ ]]; then
    echo "Error: Parsed tag part '$TAG_FROM_DESCRIBE' from git describe output '$DESCRIBE_OUTPUT' is not in the expected format X.Y.Z or X.Y.Z-dev.N."
    exit 1
fi
# COMMIT_COUNT is validated by the regex ([0-9]+) to be a number during parsing.

# 2. Determine the new version string
if [ "$COMMIT_COUNT" -eq 0 ]; then
    # If on the tag directly, use the tag as is (e.g., '1.2.3' or '1.2.3-dev.1')
    NEW_VERSION="$TAG_FROM_DESCRIBE"
else
    # If there are commits since the tag, the version should be X.Y.Z-dev.COMMIT_COUNT.
    # Strip any existing -dev.N from TAG_FROM_DESCRIBE to get the core X.Y.Z part.
    CORE_VERSION="$TAG_FROM_DESCRIBE"
    if [[ "$TAG_FROM_DESCRIBE" =~ ^([0-9]+\.[0-9]+\.[0-9]+)-dev\.[0-9]+$ ]]; then
        CORE_VERSION=${BASH_REMATCH[1]}
    fi
    # CORE_VERSION should now be strictly X.Y.Z. This also serves as a final validation.
    if ! [[ "$CORE_VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
        echo "Error: Derived core version '$CORE_VERSION' (from tag '$TAG_FROM_DESCRIBE') is not in X.Y.Z format."
        exit 1
    fi
    NEW_VERSION="${CORE_VERSION}-dev.${COMMIT_COUNT}"
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
