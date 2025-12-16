#!/bin/bash

# Pre-commit hook for validating macro imports
# Copy this script to .git/hooks/pre-commit and make it executable:
# cp scripts/pre-commit-macro-validate.sh .git/hooks/pre-commit
# chmod +x .git/hooks/pre-commit

# Validate macro imports before commit
echo "🔍 Validating macro imports..."

cd frontend
bun run validate-macros

if [ $? -ne 0 ]; then
    echo "❌ Macro import validation failed!"
    echo "Please fix the macro imports and try again."
    exit 1
fi

echo "✅ Macro imports are valid"
exit 0
