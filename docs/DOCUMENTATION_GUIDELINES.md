# Documentation Guidelines for SRAT Project

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->

- [Overview](#overview)
- [GitHub Flavored Markdown Support](#github-flavored-markdown-support)
- [Documentation Structure](#documentation-structure)
  - [Core Documentation Files](#core-documentation-files)
  - [Documentation Standards](#documentation-standards)
    - [Formatting Requirements](#formatting-requirements)
    - [Content Requirements](#content-requirements)
    - [Style Guidelines](#style-guidelines)
- [Validation Tools](#validation-tools)
  - [Automated Checks](#automated-checks)
  - [Running Validation Locally](#running-validation-locally)
- [Pre-commit Hooks](#pre-commit-hooks)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

This document provides guidelines for contributing to and maintaining documentation in the SRAT project with **GitHub Flavored Markdown (GFM)** support.

## Overview

The SRAT project uses automated validation to ensure documentation quality and consistency. All documentation changes are validated through GitHub Actions workflows and pre-commit hooks. The validation system is fully compatible with **GitHub Flavored Markdown (GFM)**, supporting tables, task lists, strikethrough, autolinks, and emoji.

## GitHub Flavored Markdown Support

SRAT documentation fully supports GitHub Flavored Markdown (GFM) features:

- ✅ **Tables**: Use GFM table syntax with alignment
- ✅ **Task Lists**: Use `- [ ]` and `- [x]` for checkboxes
- ✅ **Strikethrough**: Use `~~text~~` for strikethrough
- ✅ **Autolinks**: URLs are automatically linked
- ✅ **Emoji**: Use `:emoji_name:` syntax
- ✅ **Fenced Code Blocks**: Use triple backticks with language specification
- ✅ **HTML Elements**: Allowed elements include `<img>`, `<a>`, `<table>`, `<kbd>`, etc.

## Documentation Structure

### Core Documentation Files

- **README.md** - Main project documentation with installation and usage
- **CHANGELOG.md** - Version history and change tracking (follows Keep a Changelog format)
- **IMPLEMENTATION\_\*.md** - Technical implementation details
- **docs/HOME_ASSISTANT_INTEGRATION.md** - Home Assistant specific documentation
- **.github/copilot-instructions.md** - Instructions for GitHub Copilot
- **docs/DOCUMENTATION_VALIDATION_SETUP.md** - Documentation validation system details
- **docs/DOCUMENTATION_GUIDELINES.md** - This file

### Documentation Standards

#### Formatting Requirements

- **Use proper heading hierarchy** (# > ## > ### > ####)
- **Include language specification** for all code blocks (for example, \`\`\`bash, \`\`\`python)
- **Use consistent bullet points** (- instead of \* or +) for better GFM compatibility
- **End files with a single newline character**
- **Format links as** `[description](url)` instead of raw URLs
- **GFM tables**: Use pipe syntax with proper alignment
- **Task lists**: Use `- [ ]` for unchecked, `- [x]` for checked items
- **Emoji**: Use `:emoji_name:` syntax when appropriate

#### Content Requirements

- Keep documentation current with code changes
- Include working code examples
- Add table of contents for documents longer than 10 sections
- Cross-reference related documentation
- Explain both what changed and why

#### Style Guidelines

- **Use clear, concise language**—avoid vague terms like "simply," "obviously," "just."
- **Include concrete examples** rather than vague descriptions
- **Document error codes** and their meanings
- **Follow semantic versioning** in CHANGELOG.md
- **Consistent terminology** - Use project vocabulary (see `.vale/styles/Vocab/SRAT/accept.txt`)
- **Proper prose style** - Vale checks for Microsoft Writing Style Guide and write-good rules

## Validation Tools

### Automated Checks

The project uses several tools to validate documentation with full **GitHub Flavored Markdown** support:

- **markdownlint-cli2** - Checks Markdown formatting and style (GFM-compatible)
  - Configuration: `.markdownlint-cli2.jsonc`
  - Supports GFM tables, task lists, HTML elements
- **Lychee** - Advanced link and image validation
  - Configuration: `.lychee.toml`
  - Fast, concurrent link checking
  - Smart caching and retry logic
  - Native GFM support
- **cspell** - Spell checking with project vocabulary
  - Configuration: `.cspell.json`
  - Custom word list for technical terms
  - Ignores code blocks and URLs
- **Vale** - Prose linting and style checking (GFM-aware)
  - Configuration: `.vale.ini`
  - Vocabulary: `.vale/styles/Vocab/SRAT/`
  - Microsoft Writing Style Guide
  - write-good rules for clear writing
  - Non-blocking warnings
- **prettier** - Consistent formatting
- **doctoc** - Automatic table of contents generation
- **Custom validators** - Project-specific checks

### Running Validation Locally

```bash
# Check dependencies (including Lychee and Vale)
make docs-check

# Install JS-based documentation tools
make docs-install

# Note: Lychee and Vale need separate installation
# Lychee: https://lychee.cli.rs/#/installation
# Vale: https://vale.sh/docs/vale-cli/installation/

# Run all documentation validation checks (GFM-aware)
make docs-validate

# Auto-fix formatting issues
make docs-fix

# Generate table of contents
make docs-toc

# Show all available documentation commands
make docs-help

# Run individual validation script
./scripts/validate-docs.sh

# Auto-fix with validation script
./scripts/validate-docs.sh --fix

# Sync Vale styles (if Vale is installed)
vale sync
```

**Package Manager Support**: The validation tools support both `bun` and `npm`. The scripts will automatically detect and use `bun` if available, otherwise fall back to `npm` + `npx`.

**Optional Tools**: Lychee and Vale are optional but highly recommended. The validation script will gracefully skip them if not installed while still running other checks.

## Pre-commit Hooks

Documentation validation runs automatically via pre-commit hooks:

````bash
# Install pre-commit hooks
make prepare

# Run pre-commit manually
pre-commit run --all-files

# Run only documentation hooks
pre-commit run --all-files markdownlint
pre-commit run --all-files prettier
```shell

## GitHub Workflows

### Documentation Validation Workflow

The `.github/workflows/documentation.yml` workflow runs on:

- Push to main branch (for documentation files)
- Pull requests (for documentation files)
- Weekly schedule (to check for broken links)

#### Validation Steps

1. **Markdown Linting** - Checks formatting and style
2. **Link Validation** - Verifies all URLs are accessible
3. **Spell Check** - Validates spelling and terminology
4. **Format Check** - Ensures consistent formatting
5. **Content Validation** - Checks structure and requirements
6. **Security Check** - Scans for sensitive information
7. **Auto-fix** - Automatically fixes common issues

### Integration with GitHub Copilot

The project includes Copilot rules (`.github/copilot-rules.md`) that guide automated updates:

- Always update CHANGELOG.md for significant changes
- Maintain version consistency across files
- Update API documentation when endpoints change
- Follow established documentation patterns
- Include migration guides for breaking changes

## Contributing Documentation

### For New Features

When adding new features:

1. Update relevant documentation files
2. Add examples and usage instructions
3. Update API documentation if applicable
4. Add entry to CHANGELOG.md
5. Run validation checks locally
6. Include documentation checklist in PR

### For Bug Fixes

When fixing bugs:

1. Update documentation if behavior changes
2. Add troubleshooting information if helpful
3. Update examples if they were incorrect
4. Add entry to CHANGELOG.md
5. Verify all existing links still work

### For Breaking Changes

When making breaking changes:

1. Update all affected documentation
2. Create migration guide
3. Update version references
4. Add prominent entry to CHANGELOG.md
5. Update installation instructions if needed

## File-Specific Guidelines

### README.md

- Keep installation instructions current
- Update feature list when capabilities change
- Maintain working badge URLs
- Include clear usage examples

### CHANGELOG.md

- Follow [Keep a Changelog](https://keepachangelog.com/) format
- Group changes by type (Added, Changed, Fixed, etc.)
- Include issue/PR references
- Add migration instructions for breaking changes

### Implementation Documentation

- Update when architectural decisions change
- Include reasoning for technical choices
- Maintain decision records
- Update when dependencies change

### API Documentation

- Keep OpenAPI specs current
- Include request/response examples
- Document error codes and messages
- Update when endpoints change

## Troubleshooting

### Common Issues

**Markdownlint failures:**

- Check heading hierarchy
- Ensure code blocks have language specification
- Remove trailing whitespace

**Link check failures:**

- Verify external URLs are accessible
- Update moved or deleted links
- Check for typos in URLs

**Spell check failures:**

- Add project-specific terms to `.cspell.json`
- Fix genuine spelling errors
- Check for proper capitalization

**Format check failures:**

- Run `prettier --write "**/*.md"` to auto-fix
- Check for consistent indentation
- Ensure proper line endings

### Getting Help

- Check the validation output for specific error messages
- Run validation tools individually to isolate issues
- Review the configuration files for tool settings
- Ask in GitHub issues or discussions for complex problems

## Configuration Files

### Validation Tool Configuration

- **`.markdownlint-cli2.jsonc`** - Markdownlint rules and configuration (GFM-compatible)
- **`.lychee.toml`** - Lychee link and image checker configuration
- **`.vale.ini`** - Vale prose linting configuration
- **`.vale/styles/Vocab/SRAT/`** - Project-specific vocabulary (accept.txt, reject.txt)
- **`.cspell.json`** - Spell check dictionary and rules
- **`.prettierrc`** - Code formatting configuration (if present)
- **`.pre-commit-config.yaml`** - Pre-commit hook configuration

### GitHub Configuration

- **`.github/workflows/documentation.yml`** - CI validation workflow with Lychee and Vale
- **`.github/copilot-instructions.md`** - Instructions for GitHub Copilot
- **`.github/ISSUE_TEMPLATE/documentation.md`** - Documentation issue template
- **`.github/pull_request_template.md`** - PR template with documentation checklist

## Best Practices

### Writing Effective Documentation

1. **Start with the user's perspective** - What do they need to know?
2. **Provide context** - Explain why, not just what
3. **Include examples** - Show don't just tell
4. **Keep it current** - Update with code changes
5. **Test your examples** - Ensure they actually work
6. **Cross-reference** - Link related information
7. **Use clear headings** - Make content scannable

### Maintaining Documentation Quality

1. **Review regularly** - Check for outdated information
2. **Validate automatically** - Use CI checks to catch issues
3. **Update proactively** - Don't wait for bug reports
4. **Get feedback** - Ask users what's missing or unclear
5. **Follow standards** - Consistency helps readability

## Resources

- [Markdown Guide](https://www.markdownguide.org/)
- [Keep a Changelog](https://keepachangelog.com/)
- [Semantic Versioning](https://semver.org/)
- [GitHub Flavored Markdown](https://github.github.com/gfm/)
- [Documentation Best Practices](https://www.writethedocs.org/guide/)
````
