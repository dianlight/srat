#!/bin/bash
# Script to update coverage badges in README.md with actual test coverage values

set -e

# Get the repository root directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo "📊 Calculating test coverage..."
echo "Repository root: $REPO_ROOT"

# Run backend tests and capture coverage or error
echo "🔧 Running backend tests..."
cd "$REPO_ROOT/backend"
BACKEND_ERROR=false
if BACKEND_OUTPUT=$(make test 2>&1 | grep "Total coverage:" | tail -1); then
    # Extract coverage percentage from "Total coverage: XX.X%" format
    BACKEND_COVERAGE=$(echo "$BACKEND_OUTPUT" | awk '{gsub(/%/, "", $3); print $3}')
    # Validate coverage
    if [ -z "$BACKEND_COVERAGE" ] || ! [[ "$BACKEND_COVERAGE" =~ ^[0-9]+\.?[0-9]*$ ]]; then
        BACKEND_COVERAGE="0.0"
    fi
    echo -e "${GREEN}Backend Coverage: ${BACKEND_COVERAGE}%${NC}"
else
    echo -e "${RED}Error running backend tests:${NC}" >&2
    echo "$BACKEND_OUTPUT" >&2
    BACKEND_ERROR=true
    BACKEND_COVERAGE="0.0"
fi
cd "$REPO_ROOT"

# Run frontend tests and capture coverage or error
echo "🎨 Running frontend tests..."
cd "$REPO_ROOT/frontend"
FRONTEND_ERROR=false
if FRONTEND_OUTPUT=$(bun test --coverage 2>&1 | grep "All files" | tail -1); then
    # Extract first percentage (lines coverage)
    FRONTEND_COVERAGE=$(echo "$FRONTEND_OUTPUT" | awk -F'|' '{gsub(/^[ \t]+|[ \t]+$/, "", $2); print $2}')
    echo -e "${GREEN}Frontend Coverage: ${FRONTEND_COVERAGE}%${NC}"
else
    echo -e "${RED}Error running frontend tests:${NC}" >&2
    echo "$FRONTEND_OUTPUT" >&2
    FRONTEND_ERROR=true
    FRONTEND_COVERAGE="0.0"
fi
cd "$REPO_ROOT"

# Calculate global coverage (weighted average: 60% backend, 40% frontend)
GLOBAL_COVERAGE=$(awk "BEGIN {printf \"%.1f\", ($BACKEND_COVERAGE * 0.6) + ($FRONTEND_COVERAGE * 0.4)}")

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
# Update backend badge only if tests succeeded
if [ "$BACKEND_ERROR" = false ]; then
    sed -i.bak "s/Backend_Unit_Tests-[0-9.]*%25-[a-z]*/Backend_Unit_Tests-${BACKEND_ENCODED}-${BACKEND_COLOR}/g" "$REPO_ROOT/README.md"
else
    echo -e "${YELLOW}Skipping backend badge update due to test failure.${NC}"
fi
# Update frontend badge only if tests succeeded
if [ "$FRONTEND_ERROR" = false ]; then
    sed -i.bak "s/Frontend_Unit_Tests-[0-9.]*%25-[a-z]*/Frontend_Unit_Tests-${FRONTEND_ENCODED}-${FRONTEND_COLOR}/g" "$REPO_ROOT/README.md"
else
    echo -e "${YELLOW}Skipping frontend badge update due to test failure.${NC}"
fi
sed -i.bak "s/Global_Unit_Tests-[0-9.]*%25-[a-z]*/Global_Unit_Tests-${GLOBAL_ENCODED}-${GLOBAL_COLOR}/g" "$REPO_ROOT/README.md"

# Remove backup file
rm -f "$REPO_ROOT/README.md.bak"

echo "📈 Updating TEST_COVERAGE.md..."

# Get current date in ISO format
CURRENT_DATE=$(date +%Y-%m-%d)

# Update TEST_COVERAGE.md with new coverage data
TEST_COVERAGE_FILE="$REPO_ROOT/docs/TEST_COVERAGE.md"

if [ -f "$TEST_COVERAGE_FILE" ]; then
    # Update the coverage table
    sed -i.bak "s/| Backend (Go) | [0-9.]*% |/| Backend (Go) | ${BACKEND_COVERAGE}% |/" "$TEST_COVERAGE_FILE"
    sed -i.bak "s/| Frontend (TypeScript) | [0-9.]*% |/| Frontend (TypeScript) | ${FRONTEND_COVERAGE}% |/" "$TEST_COVERAGE_FILE"
    sed -i.bak "s/| Global (Weighted) | [0-9.]*% |/| Global (Weighted) | ${GLOBAL_COVERAGE}% |/" "$TEST_COVERAGE_FILE"
    
    # Update the last updated date
    sed -i.bak "s/\*Last updated: [0-9-]*\*/\*Last updated: ${CURRENT_DATE}\*/" "$TEST_COVERAGE_FILE"
    
    # Function to update or append coverage data in mermaid charts
    update_mermaid_chart() {
        local chart_title=$1
        local new_value=$2
        
        # Extract current x-axis dates and y-axis values
        local x_line=$(grep -A 1 "title \"${chart_title}\"" "$TEST_COVERAGE_FILE" | grep "x-axis")
        local y_line=$(grep -A 2 "title \"${chart_title}\"" "$TEST_COVERAGE_FILE" | grep "line")
        
        # Extract dates array content between brackets
        local dates=$(echo "$x_line" | sed 's/.*\[\(.*\)\].*/\1/')
        # Extract values array content between brackets
        local values=$(echo "$y_line" | sed 's/.*\[\(.*\)\].*/\1/')
        
        # Check if current date already exists
        if echo "$dates" | grep -q "$CURRENT_DATE"; then
            # Update existing date's value
            local date_array=(${dates//,/ })
            local value_array=(${values//,/ })
            local new_values=""
            local new_dates=""
            
            for i in "${!date_array[@]}"; do
                local clean_date=$(echo "${date_array[$i]}" | tr -d ' ')
                if [ "$clean_date" = "$CURRENT_DATE" ]; then
                    new_values+="${new_value}"
                else
                    new_values+="${value_array[$i]}"
                fi
                new_dates+="${date_array[$i]}"
                
                if [ $i -lt $((${#date_array[@]} - 1)) ]; then
                    new_values+=", "
                    new_dates+=", "
                fi
            done
            
            # Replace the lines
            sed -i.bak "/title \"${chart_title}\"/,/line \[/ {
                s|x-axis \[.*\]|x-axis [${new_dates}]|
                s|line \[.*\]|line [${new_values}]|
            }" "$TEST_COVERAGE_FILE"
        else
            # Append new date and value (keep only last 30 entries)
            local date_array=(${dates//,/ })
            local value_array=(${values//,/ })
            
            # Add new entry
            date_array+=("$CURRENT_DATE")
            value_array+=("$new_value")
            
            # Keep only last 30 entries
            if [ ${#date_array[@]} -gt 30 ]; then
                date_array=("${date_array[@]: -30}")
                value_array=("${value_array[@]: -30}")
            fi
            
            # Build new arrays as strings
            local new_dates="${date_array[0]}"
            local new_values="${value_array[0]}"
            for i in $(seq 1 $((${#date_array[@]} - 1))); do
                new_dates+=", ${date_array[$i]}"
                new_values+=", ${value_array[$i]}"
            done
            
            # Replace the lines
            sed -i.bak "/title \"${chart_title}\"/,/line \[/ {
                s|x-axis \[.*\]|x-axis [${new_dates}]|
                s|line \[.*\]|line [${new_values}]|
            }" "$TEST_COVERAGE_FILE"
        fi
    }
    
    # Update each chart
    update_mermaid_chart "Backend Test Coverage Over Time" "$BACKEND_COVERAGE"
    update_mermaid_chart "Frontend Test Coverage Over Time" "$FRONTEND_COVERAGE"
    update_mermaid_chart "Global Test Coverage Over Time" "$GLOBAL_COVERAGE"
    
    # Remove backup files
    rm -f "$TEST_COVERAGE_FILE.bak"
    
    echo -e "${GREEN}✅ TEST_COVERAGE.md updated successfully!${NC}"
else
    echo -e "${YELLOW}⚠️  TEST_COVERAGE.md not found, skipping history update${NC}"
fi

echo -e "${GREEN}✅ Coverage badges updated successfully!${NC}"
echo ""
echo "Updated badges:"
echo "  - Backend: ${BACKEND_COVERAGE}% (${BACKEND_COLOR})"
echo "  - Frontend: ${FRONTEND_COVERAGE}% (${FRONTEND_COLOR})"
echo "  - Global: ${GLOBAL_COVERAGE}% (${GLOBAL_COLOR})"
