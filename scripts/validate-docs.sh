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

# Check if required tools are installed
check_dependencies() {
    print_status "info" "Checking dependencies..."
    
    local missing_deps=()
    
    # Check for JavaScript runtime (prefer bun over Node.js)
    if command -v bun &> /dev/null; then
        NODE_RUNTIME="bun"
        print_status "info" "Using bun as JavaScript runtime (Node.js alternative)"
    elif command -v node &> /dev/null; then
        NODE_RUNTIME="node"
        print_status "info" "Using Node.js as JavaScript runtime"
    else
        missing_deps+=("node or bun (JavaScript runtime)")
    fi
    
    # Check for package manager (prefer bun, fallback to npm)
    if command -v bun &> /dev/null; then
        PACKAGE_MANAGER="bun"
        print_status "info" "Using bun as package manager"
    elif command -v npm &> /dev/null; then
        PACKAGE_MANAGER="npm"
        print_status "info" "Using npm as package manager"
    else
        missing_deps+=("npm or bun (package manager)")
    fi
    
    # Check for pre-commit
    if ! command -v pre-commit &> /dev/null; then
        missing_deps+=("pre-commit")
    fi
    
    if [ ${#missing_deps[@]} -ne 0 ]; then
        print_status "error" "Missing dependencies: ${missing_deps[*]}"
        echo "Please install the missing dependencies and try again."
        echo "For JavaScript runtime, install either Node.js or bun."
        echo "For package manager, install either npm or bun."
        exit 1
    fi
    
    print_status "success" "All dependencies are installed"
    print_status "info" "JavaScript runtime: $NODE_RUNTIME, Package manager: $PACKAGE_MANAGER"
}

# Install Node.js packages if needed
install_packages() {
    print_status "info" "Installing Node.js packages using $PACKAGE_MANAGER..."
    
    local packages=(
        "markdownlint-cli2"
        "markdown-link-check"
        "cspell"
        "prettier"
    )
    
    for package in "${packages[@]}"; do
        # Check if package is installed globally
        local is_installed=false
        
        if [ "$PACKAGE_MANAGER" = "bun" ]; then
            # For bun, check if the binary is available
            if command -v "${package}" &> /dev/null; then
                is_installed=true
            fi
        else
            # For npm, use npm list
            if npm list -g "$package" &> /dev/null; then
                is_installed=true
            fi
        fi
        
        if [ "$is_installed" = false ]; then
            print_status "info" "Installing $package with $PACKAGE_MANAGER..."
            
            if [ "$PACKAGE_MANAGER" = "bun" ]; then
                bun add -g "$package"
            else
                npm install -g "$package"
            fi
        fi
    done
    
    print_status "success" "Node.js packages installed with $PACKAGE_MANAGER"
}

# Run markdownlint
run_markdownlint() {
    print_status "info" "Running markdownlint..."
    
    if markdownlint-cli2 "**/*.md"; then
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
    
    # Create link check config if it doesn't exist
    if [ ! -f ".markdown-link-check.json" ]; then
        cat > .markdown-link-check.json << 'EOF'
{
  "timeout": "20s",
  "retryOn429": true,
  "retryCount": 3,
  "fallbackProtocols": ["http", "https"],
  "aliveStatusCodes": [200, 206, 301, 302, 307, 308],
  "ignorePatterns": [
    {
      "pattern": "^https://my.home-assistant.io"
    },
    {
      "pattern": "^mailto:"
    },
    {
      "pattern": "^#"
    }
  ]
}
EOF
    fi
    
    # Check links in all markdown files
    find . -name "*.md" -not -path "./node_modules/*" -not -path "./.git/*" | while read -r file; do
        if ! markdown-link-check "$file" --config .markdown-link-check.json; then
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
    
    # Create cspell config if it doesn't exist
    if [ ! -f ".cspell.json" ]; then
        cat > .cspell.json << 'EOF'
{
  "version": "0.2",
  "language": "en",
  "words": [
    "SRAT",
    "SambaNAS",
    "Hass",
    "addon",
    "addons",
    "hassio",
    "dianlight"
  ],
  "flagWords": [],
  "ignorePaths": [
    "node_modules/**",
    ".git/**",
    "*.log",
    "*.lock",
    "*.sum",
    "*.mod"
  ]
}
EOF
    fi
    
    if cspell "**/*.md"; then
        print_status "success" "Spell check passed"
        return 0
    else
        print_status "error" "Spell check failed"
        return 1
    fi
}

# Run prettier format check
run_format_check() {
    print_status "info" "Running format check..."
    
    if prettier --check "**/*.md"; then
        print_status "success" "Format check passed"
        return 0
    else
        print_status "warning" "Format check failed - run 'prettier --write \"**/*.md\"' to fix"
        return 1
    fi
}

# Run custom validation checks
run_custom_checks() {
    print_status "info" "Running custom validation checks..."
    
    local failed=0
    
    # Check for TOC in long documents
    find . -name "*.md" -not -path "./node_modules/*" -not -path "./.git/*" | while read -r file; do
        lines=$(wc -l < "$file")
        if [ "$lines" -gt 200 ]; then
            if ! grep -q -i "table of contents\|toc\|- \[.*\](#.*)" "$file"; then
                print_status "warning" "$file is $lines lines long but may be missing a Table of Contents"
            fi
        fi
    done
    
    # Check README structure
    if [ -f "README.md" ]; then
        required_sections=("Installation" "Usage" "License")
        
        for section in "${required_sections[@]}"; do
            if ! grep -q "## $section\|# $section" README.md; then
                print_status "warning" "README.md is missing required section: $section"
                failed=1
            fi
        done
    fi
    
    # Check CHANGELOG format
    if [ -f "CHANGELOG.md" ]; then
        if ! grep -q "## \[.*\] - [0-9]" CHANGELOG.md; then
            print_status "warning" "CHANGELOG.md may not follow proper version format"
            failed=1
        fi
    fi
    
    if [ $failed -eq 0 ]; then
        print_status "success" "Custom validation checks passed"
        return 0
    else
        print_status "warning" "Some custom validation checks failed"
        return 0  # Don't fail the entire script for warnings
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
    run_format_check || exit_code=1
    run_custom_checks || exit_code=1
    
    echo
    if [ $exit_code -eq 0 ]; then
        print_status "success" "All documentation validation checks passed!"
    else
        print_status "error" "Some documentation validation checks failed"
        echo
        echo "💡 Tips:"
        echo "   - Run 'prettier --write \"**/*.md\"' to fix formatting issues"
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
        
        # Determine JavaScript runtime (prefer bun over node)
        if command -v bun &> /dev/null; then
            NODE_RUNTIME="bun"
            print_status "info" "Using bun as JavaScript runtime"
        elif command -v node &> /dev/null; then
            NODE_RUNTIME="node"
        else
            print_status "error" "No JavaScript runtime found (Node.js or bun required)"
            exit 1
        fi
        
        # Determine package manager for prettier (prefer bun)
        if command -v bun &> /dev/null; then
            PACKAGE_MANAGER="bun"
        elif command -v npm &> /dev/null; then
            PACKAGE_MANAGER="npm"
        else
            print_status "error" "No package manager found (bun or npm required)"
            exit 1
        fi
        
        # Check if prettier is available, install if needed
        if [ "$PACKAGE_MANAGER" = "bun" ]; then
            # For bun, check if prettier is installed globally
            if ! bun pm -g ls | grep -q "prettier"; then
                print_status "info" "Installing prettier with $PACKAGE_MANAGER..."
                bun add -g prettier
            fi
            # Use bun to run prettier directly
            bun /root/.bun/install/global/node_modules/prettier/bin/prettier.cjs --write "**/*.md"
        else
            # For npm, use traditional approach
            if ! command -v prettier &> /dev/null; then
                print_status "info" "Installing prettier with $PACKAGE_MANAGER..."
                npm install -g prettier
            fi
            prettier --write "**/*.md"
        fi
        print_status "success" "Auto-fix completed"
        exit 0
        ;;
    *)
        main
        ;;
esac
