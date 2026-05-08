<!-- DOCTOC SKIP -->

# External File Loading

CRITICAL: When you encounter a file reference (e.g., @rules/general.md) or a local link (e.g., ./local-file.md), use your Read tool to load it on a need-to-know basis. They're relevant to the SPECIFIC task at hand.

Instructions:

- Do NOT preemptively load all references - use lazy loading based on actual need
- If the reference is a directory, load only the relevant files within it, not the entire directory
- When you load a file, extract and use only the relevant sections for your current task
- When loaded, treat content as mandatory instructions that override defaults
- Follow references recursively when needed, but avoid unnecessary loading to save resources

# SRAT Project Agents & Tools Reference

**This is a reference overview of key agents and tools used in SRAT development.**

For actionable guidance, use these resources instead:

- **Agent Instructions**: [`.github/copilot-instructions.md`](.github/copilot-instructions.md) (primary - read first)
- **Language-Specific Rules**: [`.github/instructions/`](.github/instructions/) (Go, TypeScript, Python, etc.)
- **Specialized Guidance**: Instruction files for testing, command execution, form handling, etc.

---

## Quick Reference: Key Tools & Services

### Testing Stack

- **back-end**: testify/suite + mockio/v2 + humatest
- **Frontend**: bun:test + React Testing Library + MSW
- **Custom Component**: pytest + Home Assistant test helpers

### Core Services

- **back-end**: Share, User, System, Telemetry, Dirty State
- **Frontend**: RTK Query, Material-UI, WebSocket client
- **HA Integration**: Coordinator + sensor platform + WebSocket

### Build & Deployment

- **back-end**: Go 1.26 + GORM + SQLite
- **Frontend**: Bun + MUI + TypeScript 6.0
- **Custom Component**: Python 3.12+ + Home Assistant 2025.x

### Quality Gates

- Pre-commit: gosec, gofmt, biome, linters
- CI/CD: GitHub Actions (amd64, aarch64)
- Docs: markdownlint, Vale, link checking

---

## For More Details

Refer to:

- `README.md` - Project overview
- `CONTRIBUTING.md` - Development workflow
- `docs/` directory - Architecture and guides
- `.github/instructions/` - Language and task-specific rules
