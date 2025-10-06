#!/bin/bash
# Script to update coverage badges in README.md with actual test coverage values

set -e

# Get the repository root directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "📊 Calculating test coverage..."
echo "Repository root: $REPO_ROOT"

# Get backend coverage
echo "🔧 Running backend tests..."
cd "$REPO_ROOT/backend"
BACKEND_OUTPUT=$(make test 2>&1 | grep "Total coverage:" | tail -1)
# Extract coverage percentage from "Total coverage: XX.X%" format
BACKEND_COVERAGE=$(echo "$BACKEND_OUTPUT" | awk '{gsub(/%/, "", $3); print $3}')
# If empty or invalid, default to 0.0
if [ -z "$BACKEND_COVERAGE" ] || ! [[ "$BACKEND_COVERAGE" =~ ^[0-9]+\.?[0-9]*$ ]]; then
    BACKEND_COVERAGE="0.0"
fi
cd "$REPO_ROOT"

# Get frontend coverage
echo "🎨 Running frontend tests..."
cd "$REPO_ROOT/frontend"
FRONTEND_OUTPUT=$(bun test --coverage 2>&1 | grep "All files" | tail -1)
# Extract first percentage (lines coverage)
FRONTEND_COVERAGE=$(echo "$FRONTEND_OUTPUT" | awk -F'|' '{gsub(/^[ \t]+|[ \t]+$/, "", $2); print $2}')
cd "$REPO_ROOT"

# Calculate global coverage (weighted average: 60% backend, 40% frontend)
GLOBAL_COVERAGE=$(awk "BEGIN {printf \"%.1f\", ($BACKEND_COVERAGE * 0.6) + ($FRONTEND_COVERAGE * 0.4)}")

echo -e "${GREEN}Backend Coverage: ${BACKEND_COVERAGE}%${NC}"
echo -e "${GREEN}Frontend Coverage: ${FRONTEND_COVERAGE}%${NC}"
echo -e "${GREEN}Global Coverage: ${GLOBAL_COVERAGE}%${NC}"

# Determine badge colors based on coverage thresholds
get_color() {
    local coverage=$1
    if (( $(echo "$coverage >= 80" | bc -l) )); then
        echo "brightgreen"
    elif (( $(echo "$coverage >= 60" | bc -l) )); then
        echo "green"
    elif (( $(echo "$coverage >= 40" | bc -l) )); then
        echo "yellow"
    elif (( $(echo "$coverage >= 20" | bc -l) )); then
        echo "orange"
    else
        echo "red"
    fi
}

BACKEND_COLOR=$(get_color "$BACKEND_COVERAGE")
FRONTEND_COLOR=$(get_color "$FRONTEND_COVERAGE")
GLOBAL_COLOR=$(get_color "$GLOBAL_COVERAGE")

# URL encode the percentage sign
BACKEND_ENCODED="${BACKEND_COVERAGE}%25"
FRONTEND_ENCODED="${FRONTEND_COVERAGE}%25"
GLOBAL_ENCODED="${GLOBAL_COVERAGE}%25"

echo "📝 Updating README.md..."

# Update README.md with new coverage values
sed -i.bak "s/Backend_Unit_Tests-[0-9.]*%25-[a-z]*/Backend_Unit_Tests-${BACKEND_ENCODED}-${BACKEND_COLOR}/g" "$REPO_ROOT/README.md"
sed -i.bak "s/Frontend_Unit_Tests-[0-9.]*%25-[a-z]*/Frontend_Unit_Tests-${FRONTEND_ENCODED}-${FRONTEND_COLOR}/g" "$REPO_ROOT/README.md"
sed -i.bak "s/Global_Unit_Tests-[0-9.]*%25-[a-z]*/Global_Unit_Tests-${GLOBAL_ENCODED}-${GLOBAL_COLOR}/g" "$REPO_ROOT/README.md"

# Remove backup file
rm -f "$REPO_ROOT/README.md.bak"

echo -e "${GREEN}✅ Coverage badges updated successfully!${NC}"
echo ""
echo "Updated badges:"
echo "  - Backend: ${BACKEND_COVERAGE}% (${BACKEND_COLOR})"
echo "  - Frontend: ${FRONTEND_COVERAGE}% (${FRONTEND_COLOR})"
echo "  - Global: ${GLOBAL_COVERAGE}% (${GLOBAL_COLOR})"
