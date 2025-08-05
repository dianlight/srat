# GitHub Copilot Rules for SRAT Project

## Markdown Documentation Rules

### File Updates

- **Always update CHANGELOG.md** when making significant changes to features, bug fixes, or breaking changes
- **Update version-specific documentation** when API changes are made
- **Maintain README.md accuracy** when project structure, installation, or usage changes
- **Update implementation documentation** (IMPLEMENTATION\_\*.md files) when architectural changes are made

### Content Standards

- Use proper heading hierarchy (# > ## > ### > ####)
- Include code examples in triple backticks with language specification
- Add table of contents for documents longer than 10 sections
- Use consistent bullet points (- instead of \* or +)
- Include links with descriptive text rather than raw URLs
- End files with a single newline character

### Documentation Quality

- **Keep documentation current**: Update docs when code changes
- **Be specific**: Use concrete examples rather than vague descriptions
- **Include context**: Explain why changes were made, not just what changed
- **Test examples**: Ensure all code examples are functional and accurate
- **Cross-reference**: Link related documentation appropriately

## Code Documentation Rules

### API Documentation

- Update OpenAPI specs when API endpoints change
- Include request/response examples in API documentation
- Document error codes and their meanings
- Update Huma v2 documentation when handlers change

### Go Code Documentation

- Follow Go documentation conventions with proper godoc comments
- Document exported functions, types, and variables
- Include usage examples for complex functions
- Update test documentation when test patterns change

### Frontend Documentation

- Update component documentation when React components change
- Document TypeScript interfaces and types
- Include prop documentation for components
- Update build/deployment documentation when scripts change

## Workflow and CI/CD Rules

### Validation Requirements

- All Markdown files must pass linting (markdownlint)
- Links in documentation must be valid and accessible
- Code examples in documentation must be syntactically correct
- Documentation must follow project style guide

### Automated Updates

- Version numbers in documentation should match release versions
- Badge URLs and status indicators should be current
- Example configurations should reflect current schema
- Installation instructions should match current process

## Project-Specific Rules

### SRAT Architecture

- Update backend documentation when Go services change
- Update frontend documentation when React/TypeScript changes occur
- Maintain Home Assistant integration documentation
- Keep Docker configuration documentation current

### File-Specific Rules

#### README.md

- Update badges when repository status changes
- Update installation instructions for version changes
- Update feature list when capabilities change
- Update sponsor information as needed

#### CHANGELOG.md

- Follow Semantic Versioning principles
- Include migration guides for breaking changes
- Group changes by type (Added, Changed, Deprecated, Removed, Fixed, Security)
- Include issue/PR references where applicable

#### Implementation Docs

- Update IMPLEMENTATION\_\*.md when architectural decisions change
- Include reasoning for technical choices
- Maintain decision records for future reference
- Update when dependencies or integrations change

## Quality Gates

### Before Merge

- All documentation must be spell-checked
- Links must be validated
- Code examples must be tested
- Version references must be current

### Review Requirements

- Documentation changes require review from maintainers
- Breaking changes must include migration documentation
- New features must include user-facing documentation
- API changes must include updated OpenAPI specs

## Automation Guidelines

### GitHub Actions Integration

- Documentation validation runs on all PR branches
- Link checking occurs on schedule and PR events
- Version consistency is verified across all files
- Style guide compliance is enforced automatically

### Tool Configuration

- Use markdownlint for consistent formatting
- Use link-check for URL validation
- Use spell-check for content quality
- Use prettier for consistent formatting where applicable
