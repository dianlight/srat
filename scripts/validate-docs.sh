#!/bin/bash

# Documentation validation script for SRAT project
# This script runs all documentation validation checks locally

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

# Check if required tools are installed (bunx or npmx)
check_dependencies() {
    print_status "info" "Checking dependencies..."
    local missing_deps=()
    RUNNER=""
    if command -v bunx &> /dev/null; then
        RUNNER="bunx"
        print_status "info" "Using bunx for running JS CLI tools"
    elif command -v npmx &> /dev/null; then
        RUNNER="npmx"
        print_status "info" "Using npmx for running JS CLI tools"
    else
        missing_deps+=("bunx or npmx (JS CLI runner)")
    fi
    if ! command -v pre-commit &> /dev/null; then
        missing_deps+=("pre-commit")
    fi
    if [ ${#missing_deps[@]} -ne 0 ]; then
        print_status "error" "Missing dependencies: ${missing_deps[*]}"
        echo "Please install the missing dependencies and try again."
        echo "For JS CLI runner, install bunx or npmx."
        exit 1
    fi
    print_status "success" "All dependencies are installed"
    print_status "info" "JS CLI runner: $RUNNER"
}

 # Additional help text for JS CLI runner
 echo "All JS CLI tools (markdownlint, cspell, prettier, etc.) are run via bunx or npmx."

# No need to install packages with bunx/npmx
install_packages() {
    print_status "info" "No package installation needed with $RUNNER (bunx/npmx)"
}

# Run markdownlint
run_markdownlint() {
    print_status "info" "Running markdownlint..."
    if $RUNNER markdownlint-cli2 "**/*.md" "#frontend/node_modules" ; then
        print_status "success" "Markdownlint passed"
        return 0
    else
        print_status "error" "Markdownlint failed"
        return 1
    fi
}

# Run link check
run_link_check() {
    print_status "info" "Running link check..."

    local failed=0


    # Check links in all markdown files
    find . -name "*.md" -not -path "./frontend/node_modules/*" -not -path "./.git/*" | while read -r file; do
        if ! $RUNNER markdown-link-check "$file" --config .markdown-link-check.json; then
            print_status "error" "Link check failed for $file"
            failed=1
        fi
    done

    if [ $failed -eq 0 ]; then
        print_status "success" "Link check passed"
        return 0
    else
        print_status "error" "Link check failed"
        return 1
    fi
}

# Run spell check
run_spell_check() {
    print_status "info" "Running spell check..."

    if $RUNNER cspell "**/*.md"; then
        print_status "success" "Spell check passed"
        return 0
    else
        print_status "error" "Spell check failed"
        return 1
    fi
}



# Main execution
main() {
    local exit_code=0

    echo "📚 SRAT Documentation Validation"
    echo "================================"

    check_dependencies
    install_packages

    # Run all checks
    run_markdownlint || exit_code=1
    run_link_check || exit_code=1
    run_spell_check || exit_code=1

    echo
    if [ $exit_code -eq 0 ]; then
        print_status "success" "All documentation validation checks passed!"
    else
        print_status "error" "Some documentation validation checks failed"
        echo
        echo "💡 Tips:"
        echo "   - Run 'prettier --write \"**/*.md\" --ignore-path \"frontend/node_modules/**\"' to fix formatting issues"
        echo "   - Or use '$0 --fix' to auto-fix formatting"
        echo "   - Check the output above for specific issues to address"
        echo "   - You can run individual checks by examining this script"
        echo "   - This script supports both bun and npm package managers"
    fi

    exit $exit_code
}

# Handle script arguments
case "${1:-}" in
    "--help"|"-h")
        echo "Documentation validation script for SRAT project"
        echo
        echo "Usage: $0 [option]"
        echo
        echo "Options:"
        echo "  --help, -h     Show this help message"
        echo "  --fix          Run auto-fix for formatting issues"
        echo
        echo "Dependencies:"
        echo "  - Node.js or bun (JavaScript runtime)"
        echo "  - bun or npm (package manager)"
        echo "  - pre-commit (for git hooks)"
        echo
        echo "The script will automatically detect available tools:"
        echo "  - Prefers bun as both runtime and package manager"
        echo "  - Falls back to Node.js + npm if bun not available"
        echo "  - bun can serve as both JavaScript runtime and package manager"
        echo
        exit 0
        ;;
    "--fix")
        print_status "info" "Running auto-fix for formatting issues..."

        # Use bunx or npmx for prettier
        if command -v bunx &> /dev/null; then
            RUNNER="bunx"
            print_status "info" "Using bunx for running prettier"
        elif command -v npmx &> /dev/null; then
            RUNNER="npmx"
            print_status "info" "Using npmx for running prettier"
        else
            print_status "error" "No JS CLI runner found (bunx or npmx required)"
            exit 1
        fi
        $RUNNER prettier --write "**/*.md" --ignore-path "frontend/node_modules/**"
        print_status "success" "Auto-fix completed"
        exit 0
        ;;
    *)
        main
        ;;
esac
