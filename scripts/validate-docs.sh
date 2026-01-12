#!/bin/bash

# Documentation validation script for SRAT project
# This script runs all documentation validation checks locally
# Supports: markdownlint, Lychee (link checker), cspell, and Vale (prose linter)

set -e

echo "🔍 Running documentation validation checks..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    local status=$1
    local message=$2
    case $status in
        "success")
            echo -e "${GREEN}✅ $message${NC}"
            ;;
        "warning")
            echo -e "${YELLOW}⚠️  $message${NC}"
            ;;
        "error")
            echo -e "${RED}❌ $message${NC}"
            ;;
        "info")
            echo -e "ℹ️  $message"
            ;;
    esac
}

# Check if required tools are installed
check_dependencies() {
    print_status "info" "Checking dependencies..."
    local missing_deps=()
    RUNNER=""
    
    # Check for bunx or npmx (for JS tools)
    if command -v bunx &> /dev/null; then
        RUNNER="bunx"
        print_status "info" "Using bunx for running JS CLI tools"
    elif command -v npx &> /dev/null; then
        RUNNER="npx"
        print_status "info" "Using npx for running JS CLI tools"
    else
        missing_deps+=("bunx or npx (JS CLI runner)")
    fi
    
    # Check for Lychee (link checker)
    if ! command -v lychee &> /dev/null; then
        print_status "warning" "Lychee not found - link checking will be skipped"
        print_status "info" "Install Lychee: https://lychee.cli.rs/#/installation"
    fi
    
    # Check for Vale (prose linter)
    if ! command -v vale &> /dev/null; then
        print_status "warning" "Vale not found - prose linting will be skipped"
        print_status "info" "Install Vale: https://vale.sh/docs/vale-cli/installation/"
    fi
    
    if [ ${#missing_deps[@]} -ne 0 ]; then
        print_status "error" "Missing critical dependencies: ${missing_deps[*]}"
        echo "Please install the missing dependencies and try again."
        exit 1
    fi
    
    print_status "success" "Core dependencies are installed"
    print_status "info" "JS CLI runner: $RUNNER"
}

# Run markdownlint
run_markdownlint() {
    print_status "info" "Running markdownlint (GitHub Flavored Markdown)..."
    if $RUNNER markdownlint-cli2 "**/*.md" "#frontend/node_modules" "#backend/src/vendor" ; then
        print_status "success" "Markdownlint passed"
        return 0
    else
        print_status "error" "Markdownlint failed"
        return 1
    fi
}

# Run Lychee link checker
run_lychee() {
    if ! command -v lychee &> /dev/null; then
        print_status "warning" "Lychee not installed - skipping link check"
        return 0
    fi
    
    print_status "info" "Running Lychee link checker..."
    
    if lychee --config .lychee.toml . ; then
        print_status "success" "Lychee link check passed"
        return 0
    else
        print_status "error" "Lychee link check failed"
        return 1
    fi
}

# Run spell check
run_spell_check() {
    print_status "info" "Running spell check..."

    if $RUNNER cspell "**/*.md" --config .cspell.json; then
        print_status "success" "Spell check passed"
        return 0
    else
        print_status "error" "Spell check failed"
        return 1
    fi
}

# Run Vale prose linter
run_vale() {
    if ! command -v vale &> /dev/null; then
        print_status "warning" "Vale not installed - skipping prose linting"
        return 0
    fi
    
    print_status "info" "Running Vale prose linter (GitHub Flavored Markdown)..."
    
    # Sync Vale styles first
    if vale sync; then
        print_status "info" "Vale styles synced"
    else
        print_status "warning" "Could not sync Vale styles - continuing anyway"
    fi
    
    # Run Vale on all markdown files
    if find . -name "*.md" -not -path "./frontend/node_modules/*" -not -path "./.git/*" -not -path "./backend/src/vendor/*" -exec vale {} + ; then
        print_status "success" "Vale prose linting passed"
        return 0
    else
        print_status "warning" "Vale found style issues (not blocking)"
        return 0  # Vale warnings are non-blocking
    fi
}

# Main execution
main() {
    local exit_code=0

    echo "📚 SRAT Documentation Validation"
    echo "================================"
    echo "GitHub Flavored Markdown (GFM) Support"
    echo ""

    check_dependencies

    # Run all checks
    run_markdownlint || exit_code=1
    run_lychee || exit_code=1
    run_spell_check || exit_code=1
    run_vale || exit_code=1

    echo
    if [ $exit_code -eq 0 ]; then
        print_status "success" "All documentation validation checks passed!"
    else
        print_status "error" "Some documentation validation checks failed"
        echo
        echo "💡 Tips:"
        echo "   - Run '$0 --fix' to auto-fix formatting issues"
        echo "   - Run 'vale sync' to update Vale styles"
        echo "   - Check .lychee.toml for link checker configuration"
        echo "   - Check .vale.ini for prose linting configuration"
        echo "   - Check .markdownlint-cli2.jsonc for markdown linting rules"
    fi

    exit $exit_code
}

# Handle script arguments
case "${1:-}" in
    "--help"|"-h")
        echo "Documentation validation script for SRAT project"
        echo "GitHub Flavored Markdown (GFM) compatible"
        echo
        echo "Usage: $0 [option]"
        echo
        echo "Options:"
        echo "  --help, -h     Show this help message"
        echo "  --fix          Run auto-fix for formatting issues"
        echo
        echo "Dependencies:"
        echo "  Required:"
        echo "    - bunx or npx (JavaScript CLI runner)"
        echo "  Optional:"
        echo "    - Lychee (link and image checker)"
        echo "    - Vale (prose linter)"
        echo
        echo "Tools:"
        echo "  - markdownlint-cli2: Markdown syntax and formatting (GFM)"
        echo "  - Lychee: Link and image validation"
        echo "  - cspell: Spell checking"
        echo "  - Vale: Prose linting and style checking (GFM)"
        echo
        exit 0
        ;;
    "--fix")
        print_status "info" "Running auto-fix for formatting issues..."

        # Detect runner
        if command -v bunx &> /dev/null; then
            RUNNER="bunx"
        elif command -v npx &> /dev/null; then
            RUNNER="npx"
        else
            print_status "error" "No JS CLI runner found (bunx or npx required)"
            exit 1
        fi
        
        # Fix markdown formatting
        $RUNNER prettier --write "**/*.md" --ignore-path ".gitignore" --ignore-path "frontend/node_modules/**" --ignore-path "backend/src/vendor/**"
        
        # Fix markdownlint issues
        $RUNNER markdownlint-cli2 "**/*.md" "#frontend/node_modules" "#backend/src/vendor" --fix
        
        print_status "success" "Auto-fix completed"
        exit 0
        ;;
    *)
        main
        ;;
esac
