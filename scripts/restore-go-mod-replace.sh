#!/bin/bash

# This script is designed to be used as a post-commit hook.
# It restores the original go.mod content from a temporary file
# if it was modified by the pre-commit hook.

GO_MOD_FILE="backend/src/go.mod"
TEMP_GO_MOD_ORIGINAL_PREFIX="go.mod.original."

echo "Running post-commit hook to restore $GO_MOD_FILE..."

# Find the temporary file created by the pre-commit hook
# We need to be careful here, as the PID part ($$) in the temp filename
# might not be directly accessible to the post-commit hook if executed
# in a different shell context or if the pre-commit hook's PID is not
# directly passed.
# A more robust solution involves storing the temp file name in a known place
# or using a more predictable naming convention, or relying on `git stash`.

# For this specific case, let's assume the pre-commit hook has just run
# and we can find a recent temp file.
# A more robust approach might be to write the temp file path to a known,
# repository-specific temporary file during pre-commit.

echo $1

# Let's search for the temp file based on the prefix.
# This assumes there isn't another pre-commit hook running simultaneously
# or that older temp files are cleaned up.
TEMP_GO_MOD_ORIGINAL=$(find /tmp -name "${TEMP_GO_MOD_ORIGINAL_PREFIX}*" -maxdepth 1 -mmin -5 | head -n 1) # Look for files modified in last 5 minutes

if [ -z "$TEMP_GO_MOD_ORIGINAL" ]; then
    echo "No temporary original go.mod file found. Skipping restoration."
    exit 0
fi

if [ ! -f "$TEMP_GO_MOD_ORIGINAL" ]; then
    echo "Temporary original go.mod file '$TEMP_GO_MOD_ORIGINAL' does not exist. Skipping restoration."
    exit 0
fi

echo "Restoring $GO_MOD_FILE from $TEMP_GO_MOD_ORIGINAL..."

# Restore the original go.mod
mv "$TEMP_GO_MOD_ORIGINAL" "$GO_MOD_FILE"
if [ $? -ne 0 ]; then
    echo "Error: Failed to restore $GO_MOD_FILE from $TEMP_GO_MOD_ORIGINAL"
    exit 1
fi

echo "Successfully restored $GO_MOD_FILE."

exit 0