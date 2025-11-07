#!/bin/bash
# Script to update coverage badges in README.md with actual test coverage values
# 
# Usage:
#   ./update-coverage-badges.sh                              # Run tests and update badges
#   ./update-coverage-badges.sh --backend 45.2 --frontend 78.5  # Use provided coverage values, skip tests

set -e

# Get the repository root directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Parse CLI arguments
BACKEND_COVERAGE=""
FRONTEND_COVERAGE=""
RUN_TESTS=true

while [[ $# -gt 0 ]]; do
    case $1 in
        --backend)
            BACKEND_COVERAGE="$2"
            shift 2
            ;;
        --frontend)
            FRONTEND_COVERAGE="$2"
            shift 2
            ;;
        *)
            echo "Unknown option: $1"
            echo "Usage: $0 [--backend COVERAGE] [--frontend COVERAGE]"
            exit 1
            ;;
    esac
done

# Determine if we should run tests
if [ -n "$BACKEND_COVERAGE" ] && [ -n "$FRONTEND_COVERAGE" ]; then
    RUN_TESTS=false
    echo "📊 Using provided coverage values (skipping tests)..."
else
    RUN_TESTS=true
    echo "📊 Calculating test coverage..."
fi

echo "Repository root: $REPO_ROOT"

# Run backend tests if not provided via CLI
if [ "$RUN_TESTS" = true ]; then
    echo "🔧 Running backend tests..."
    cd "$REPO_ROOT/backend"
    BACKEND_ERROR=false
    set +e
    BACKEND_FULL_OUTPUT=$(make test 2>&1)
    BACKEND_EXIT_CODE=$?
    set -e

    echo "---- Backend test output start ----"
    echo "$BACKEND_FULL_OUTPUT"
    echo "---- Backend test output end ----"

    # Try to extract coverage from a line like: "Total coverage: XX.X%"
    BACKEND_COVERAGE_LINE=$(echo "$BACKEND_FULL_OUTPUT" | grep "Total coverage:" | tail -1 || true)
    if [ -n "$BACKEND_COVERAGE_LINE" ]; then
        BACKEND_COVERAGE=$(echo "$BACKEND_COVERAGE_LINE" | awk '{gsub(/%/, "", $3); print $3}')
        # Normalize/validate
        if [ -z "$BACKEND_COVERAGE" ] || ! [[ "$BACKEND_COVERAGE" =~ ^[0-9]+\.?[0-9]*$ ]]; then
            BACKEND_COVERAGE="0.0"
        fi
    else
        BACKEND_ERROR=true
        BACKEND_COVERAGE="0.0"
    fi
    if [ $BACKEND_EXIT_CODE -ne 0 ]; then
        BACKEND_ERROR=true
    fi
    
    # Extract package-level coverage data
    # Format: "ok  github.com/dianlight/srat/package  (cached)  coverage: XX.X% of statements"
    BACKEND_PACKAGE_COVERAGE=$(echo "$BACKEND_FULL_OUTPUT" | grep -E "github.com/dianlight/srat/.+coverage:" | sed 's|.*github.com/dianlight/srat/||' | sed 's/\t\+/ /g' | awk '{gsub(/coverage:/, "", $0); gsub(/of statements/, "", $0); gsub(/\(cached\)/, "", $0); print $1, $NF}' | sed 's/%//g')
    
    cd "$REPO_ROOT"
else
    BACKEND_ERROR=false
    # Validate CLI-provided backend coverage
    if ! [[ "$BACKEND_COVERAGE" =~ ^[0-9]+\.?[0-9]*$ ]]; then
        echo -e "${RED}Invalid backend coverage value: $BACKEND_COVERAGE${NC}"
        exit 1
    fi
    BACKEND_PACKAGE_COVERAGE=""
fi
echo -e "${GREEN}Backend Coverage: ${BACKEND_COVERAGE}%${NC}"

# Run frontend tests if not provided via CLI
if [ "$RUN_TESTS" = true ]; then
    echo "🎨 Running frontend tests..."
    cd "$REPO_ROOT/frontend"
    FRONTEND_ERROR=false
    set +e
    FRONTEND_FULL_OUTPUT=$(bun test --coverage 2>&1)
    FRONTEND_EXIT_CODE=$?
    set -e

    echo "---- Frontend test output start ----"
    echo "$FRONTEND_FULL_OUTPUT"
    echo "---- Frontend test output end ----"

    # Try to extract lines coverage from a table row that contains "All files"
    FRONTEND_COVERAGE_LINE=$(echo "$FRONTEND_FULL_OUTPUT" | grep "All files" | tail -1 || true)
    if [ -n "$FRONTEND_COVERAGE_LINE" ]; then
        # Extract second column and strip spaces and percent sign
        FRONTEND_COVERAGE=$(echo "$FRONTEND_COVERAGE_LINE" | awk -F'|' '{gsub(/^[ \t]+|[ \t]+$/, "", $2); gsub(/%/, "", $2); print $2}')
        if [ -z "$FRONTEND_COVERAGE" ] || ! [[ "$FRONTEND_COVERAGE" =~ ^[0-9]+\.?[0-9]*$ ]]; then
            FRONTEND_COVERAGE="0.0"
        fi
    else
        FRONTEND_ERROR=true
        FRONTEND_COVERAGE="0.0"
    fi
    if [ $FRONTEND_EXIT_CODE -ne 0 ]; then
        FRONTEND_ERROR=true
    fi
    cd "$REPO_ROOT"
else
    FRONTEND_ERROR=false
    # Validate CLI-provided frontend coverage
    if ! [[ "$FRONTEND_COVERAGE" =~ ^[0-9]+\.?[0-9]*$ ]]; then
        echo -e "${RED}Invalid frontend coverage value: $FRONTEND_COVERAGE${NC}"
        exit 1
    fi
fi
echo -e "${GREEN}Frontend Coverage: ${FRONTEND_COVERAGE}%${NC}"

# Calculate global coverage (weighted average: 60% backend, 40% frontend)
GLOBAL_COVERAGE=$(awk "BEGIN {printf \"%.1f\", ($BACKEND_COVERAGE * 0.6) + ($FRONTEND_COVERAGE * 0.4)}")


echo -e "${GREEN}Global Coverage: ${GLOBAL_COVERAGE}%${NC}"

# If any coverage is 0.0 then skip updating files to avoid accidental overwrites
if [ "$BACKEND_COVERAGE" = "0.0" ] || [ "$FRONTEND_COVERAGE" = "0.0" ]; then
    echo -e "${YELLOW}One or more coverage values are 0.0 — skipping README.md and TEST_COVERAGE.md updates.${NC}"
    echo "Updated badges (skipped):"
    echo "  - Backend: ${BACKEND_COVERAGE}%"
    echo "  - Frontend: ${FRONTEND_COVERAGE}%"
    echo "  - Global: ${GLOBAL_COVERAGE}%"
    exit 0
fi

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
        local y_line=$(grep -A 3 "title \"${chart_title}\"" "$TEST_COVERAGE_FILE" | grep "line")
        
        # Extract dates array content between brackets
        local dates=$(echo "$x_line" | sed 's/.*\[\(.*\)\].*/\1/')
        # Extract values array content between brackets
        local values=$(echo "$y_line" | sed 's/.*\[\(.*\)\].*/\1/')

        # Parse arrays preserving empty entries
        local date_array=()
        local value_array=()
        
        # Split dates by comma, trim whitespace, but keep empty entries
        IFS=',' read -ra date_parts <<< "$dates"
        for d in "${date_parts[@]}"; do
            d=$(echo "$d" | xargs)  # Trim whitespace
            date_array+=("$d")
        done
        
        # Split values by comma, trim whitespace, but keep empty entries
        IFS=',' read -ra value_parts <<< "$values"
        for v in "${value_parts[@]}"; do
            v=$(echo "$v" | xargs)  # Trim whitespace
            value_array+=("$v")
        done

        # Ensure value_array has same length as date_array (fill with empty strings if needed)
        while [ ${#value_array[@]} -lt ${#date_array[@]} ]; do
            value_array+=("")
        done

        # Check if current date already exists
        local found=0
        for i in "${!date_array[@]}"; do
            if [ "${date_array[$i]}" = "$CURRENT_DATE" ]; then
                value_array[$i]="$new_value"
                found=1
                break
            fi
        done
        if [ $found -eq 0 ]; then
            date_array+=("$CURRENT_DATE")
            value_array+=("$new_value")
        fi

        # Keep only last 30 entries
        if [ ${#date_array[@]} -gt 30 ]; then
            date_array=("${date_array[@]: -30}")
            value_array=("${value_array[@]: -30}")
        fi

        # Build new arrays as strings (no leading/trailing commas, preserve empty values)
        local new_dates=""
        local new_values=""
        for i in "${!date_array[@]}"; do
            if [ -n "$new_dates" ]; then new_dates+=", "; fi
            if [ -n "$new_values" ]; then new_values+=", "; fi
            new_dates+="${date_array[$i]}"
            # For values, we want to keep the value even if empty
            new_values+="${value_array[$i]}"
        done

        # Replace the lines
        sed -i.bak "/title \"${chart_title}\"/,/line \[/ {
            s|x-axis \[.*\]|x-axis [${new_dates}]|
            s|line \[.*\]|line [${new_values}]|
        }" "$TEST_COVERAGE_FILE"
    }
    
    # Update each chart
    update_mermaid_chart "Backend Test Coverage Over Time" "$BACKEND_COVERAGE"
    update_mermaid_chart "Frontend Test Coverage Over Time" "$FRONTEND_COVERAGE"
    update_mermaid_chart "Global Test Coverage Over Time" "$GLOBAL_COVERAGE"
    
    # Function to get status emoji based on coverage
    get_status_emoji() {
        local coverage=$1
        if (( $(echo "$coverage >= 80" | bc -l) )); then
            echo "✅ Excellent"
        elif (( $(echo "$coverage >= 60" | bc -l) )); then
            echo "✅ Good"
        elif (( $(echo "$coverage >= 40" | bc -l) )); then
            echo "🟡 Yellow"
        elif (( $(echo "$coverage >= 20" | bc -l) )); then
            echo "🟠 Orange"
        else
            echo "🔴 Critical"
        fi
    }
    
    # Function to get status text based on coverage
    get_status_text() {
        local coverage=$1
        if (( $(echo "$coverage >= 80" | bc -l) )); then
            echo "✅ Excellent"
        elif (( $(echo "$coverage >= 60" | bc -l) )); then
            echo "✅ Good"
        elif (( $(echo "$coverage >= 20" | bc -l) )); then
            echo "🟠 Needs Work"
        else
            echo "🔴 Critical"
        fi
    }
    
    # Update Current Coverage table (lines 11-13)
    BACKEND_STATUS=$(get_status_emoji "$BACKEND_COVERAGE")
    FRONTEND_STATUS=$(get_status_emoji "$FRONTEND_COVERAGE")
    GLOBAL_STATUS=$(get_status_emoji "$GLOBAL_COVERAGE")
    
    sed -i.bak "s/| Backend (Go) *| [0-9.]*% *| .* |/| Backend (Go)          | ${BACKEND_COVERAGE}%    | ${BACKEND_STATUS} |/" "$TEST_COVERAGE_FILE"
    sed -i.bak "s/| Frontend (TypeScript) *| [0-9.]*% *| .* |/| Frontend (TypeScript) | ${FRONTEND_COVERAGE}%   | ${FRONTEND_STATUS} |/" "$TEST_COVERAGE_FILE"
    sed -i.bak "s/| Global (Weighted) *| [0-9.]*% *| .* |/| Global (Weighted)     | ${GLOBAL_COVERAGE}%    | ${GLOBAL_STATUS} |/" "$TEST_COVERAGE_FILE"
    
    # Update Backend Package-Level Coverage table
    if [ -n "$BACKEND_PACKAGE_COVERAGE" ]; then
        echo "📦 Updating backend package-level coverage..."
        
        # Create associative arrays for package coverage
        declare -A pkg_coverage
        while IFS=' ' read -r pkg cov; do
            if [ -n "$pkg" ] && [ -n "$cov" ]; then
                # Map package names to match TEST_COVERAGE.md format
                case "$pkg" in
                    "api")
                        pkg_coverage["api"]="$cov"
                        ;;
                    "cmd/srat-cli")
                        pkg_coverage["cmd/srat-cli"]="$cov"
                        ;;
                    "cmd/srat-openapi")
                        pkg_coverage["cmd/srat-openapi"]="$cov"
                        ;;
                    "cmd/srat-server")
                        pkg_coverage["cmd/srat-server"]="$cov"
                        ;;
                    "config")
                        pkg_coverage["config"]="$cov"
                        ;;
                    "converter")
                        pkg_coverage["converter"]="$cov"
                        ;;
                    "dbom")
                        pkg_coverage["dbom"]="$cov"
                        ;;
                    "dbom/migrations")
                        pkg_coverage["dbom/migrations"]="$cov"
                        ;;
                    "dto")
                        pkg_coverage["dto"]="$cov"
                        ;;
                    "homeassistant/addons")
                        pkg_coverage["homeassistant/addons"]="$cov"
                        ;;
                    "homeassistant/core")
                        pkg_coverage["homeassistant/core"]="$cov"
                        ;;
                    "homeassistant/core_api")
                        pkg_coverage["homeassistant/core_api"]="$cov"
                        ;;
                    "homeassistant/hardware")
                        pkg_coverage["homeassistant/hardware"]="$cov"
                        ;;
                    "homeassistant/host")
                        pkg_coverage["homeassistant/host"]="$cov"
                        ;;
                    "homeassistant/ingress")
                        pkg_coverage["homeassistant/ingress"]="$cov"
                        ;;
                    "homeassistant/mount")
                        pkg_coverage["homeassistant/mount"]="$cov"
                        ;;
                    "homeassistant/resolution")
                        pkg_coverage["homeassistant/resolution"]="$cov"
                        ;;
                    "homeassistant/root")
                        pkg_coverage["homeassistant/root"]="$cov"
                        ;;
                    "homeassistant/websocket")
                        pkg_coverage["homeassistant/websocket"]="$cov"
                        ;;
                    "internal")
                        pkg_coverage["internal"]="$cov"
                        ;;
                    "internal/appsetup")
                        pkg_coverage["internal/appsetup"]="$cov"
                        ;;
                    "internal/osutil")
                        pkg_coverage["internal/osutil"]="$cov"
                        ;;
                    "repository")
                        # Check if this is repository/dao or just repository
                        if [[ "$pkg" == "repository" ]]; then
                            pkg_coverage["repository"]="$cov"
                        fi
                        ;;
                    "repository/dao")
                        pkg_coverage["repository/dao"]="$cov"
                        ;;
                    "server")
                        pkg_coverage["server"]="$cov"
                        ;;
                    "service")
                        pkg_coverage["service"]="$cov"
                        ;;
                    "tempio")
                        pkg_coverage["tempio"]="$cov"
                        ;;
                    "tlog")
                        pkg_coverage["tlog"]="$cov"
                        ;;
                    "unixsamba")
                        pkg_coverage["unixsamba"]="$cov"
                        ;;
                esac
            fi
        done <<< "$BACKEND_PACKAGE_COVERAGE"
        
        # Update each package line in the table
        for pkg in "${!pkg_coverage[@]}"; do
            cov="${pkg_coverage[$pkg]}"
            status=$(get_status_text "$cov")
            
            # Escape special characters for sed
            pkg_escaped=$(echo "$pkg" | sed 's/\//\\\//g')
            
            # Update the line in the markdown table
            # Match lines like: | `package` | XX.X% | Status | Priority |
            # We need to preserve the Priority column
            sed -i.bak "/| \`${pkg_escaped}\` *|/ s/| [0-9.]*% *| [^|]* |/| ${cov}% | ${status} |/" "$TEST_COVERAGE_FILE"
        done
        
        # Update summary counts
        excellent_count=0
        good_count=0
        needs_work_count=0
        critical_count=0
        
        for pkg in "${!pkg_coverage[@]}"; do
            cov="${pkg_coverage[$pkg]}"
            if (( $(echo "$cov >= 80" | bc -l) )); then
                ((excellent_count++))
            elif (( $(echo "$cov >= 60" | bc -l) )); then
                ((good_count++))
            elif (( $(echo "$cov >= 20" | bc -l) )); then
                ((needs_work_count++))
            else
                ((critical_count++))
            fi
        done
        
        total_good=$((excellent_count + good_count))
        
        # Update summary section
        sed -i.bak "s/- ✅ \*\*[0-9]* packages\*\* already meet or exceed 60% threshold/- ✅ **${total_good} packages** already meet or exceed 60% threshold/" "$TEST_COVERAGE_FILE"
        sed -i.bak "s/- 🟠 \*\*[0-9]* packages\*\* need improvement (20-59% coverage)/- 🟠 **${needs_work_count} packages** need improvement (20-59% coverage)/" "$TEST_COVERAGE_FILE"
        sed -i.bak "s/- 🔴 \*\*[0-9]* packages\*\* critical (below 20% coverage)/- 🔴 **${critical_count} packages** critical (below 20% coverage)/" "$TEST_COVERAGE_FILE"
    fi
    
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
