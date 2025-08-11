#!/bin/bash

if [[ "${GIT_REFLOG_ACTION}" =~ "rebase".*"reword" ]]; then
    exit 0
fi

# This script is designed to be used as a pre-commit hook.
# It removes 'replace' directives from go.mod if go.mod is staged for commit.
# It saves the original go.mod to a temporary file for post-commit restoration.

GO_MOD_FILE="backend/src/go.mod"
TEMP_GO_MOD_ORIGINAL="/tmp/go.mod.original.$$" # Unique temp file for this run

echo "Running pre-commit hook to check for $GO_MOD_FILE replace lines..."

# Check if go.mod exists and is staged for commit
if git diff --cached --quiet "$GO_MOD_FILE"; then
    echo "$GO_MOD_FILE is not staged for commit or has no changes. Skipping go.mod modification."
    exit 0
fi

# Check if go.mod contains "replace" lines
if ! grep -q "replace" "$GO_MOD_FILE"; then
    echo "No 'replace' lines found in $GO_MOD_FILE. Skipping modification."
    exit 0
fi

echo "Found 'replace' lines in $GO_MOD_FILE. Processing..."

# Save the original go.mod content to a temporary file
cp "$GO_MOD_FILE" "$TEMP_GO_MOD_ORIGINAL"
if [ $? -ne 0 ]; then
    echo "Error: Failed to save original $GO_MOD_FILE to $TEMP_GO_MOD_ORIGINAL"
    exit 1
fi

# Remove 'replace' lines from go.mod
# We use sed's in-place edit (-i) to modify the file directly
sed -i '/^replace /d' "$GO_MOD_FILE"

# Stage the modified go.mod
git add "$GO_MOD_FILE"

echo "Successfully removed 'replace' lines from $GO_MOD_FILE and staged the changes."
echo "Original go.mod saved to $TEMP_GO_MOD_ORIGINAL for post-commit restoration."

exit 0
