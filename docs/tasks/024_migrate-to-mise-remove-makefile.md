# [REFACTOR]: Migrate to mise.jdx.dev and Remove Makefile

**Target Repo:** `srat`  **Status:** 📅 Planned  **Issue Link:** _TBD_

## 🎯 Objective
Migrate the entire monorepo to use [mise.jdx.dev](https://mise.jdx.dev) for toolchain and environment management, removing all Makefile-relative configurations. Clean up the Makefile by removing unused targets before migration. After migration, optimize all processes and update documentation accordingly.

## 🛠️ Technical Specifications
- **Inputs:** Existing Makefile, current toolchain configurations, documentation referencing Makefile
- **Outputs:** Monorepo managed by mise, Makefile removed, updated documentation, optimized workflows
- **Dependencies:** mise.jdx.dev, all current build/test/lint tools, documentation files referencing Makefile

## 📝 Task List
- [ ] Task 1: Audit and clean up Makefile, remove unused targets
- [ ] Task 2: Plan and document mise migration steps for all subprojects (backend, frontend, custom_components)
- [ ] Task 3: Implement mise configuration for all environments and workflows
- [ ] Task 4: Remove Makefile and all Makefile-relative configs
- [ ] Task 5: Update all documentation to reference mise workflows
- [ ] Task 6: Optimize and test all build, test, and lint processes under mise
- [ ] Task 7: Unit and integration testing
- [ ] Task 8: Code review, cleanup, and final validation
- [ ] Task 9: Ask to create a PR with the task implementation and link it here for tracking

## 🧠 Implementation Notes (Copilot Context)
- Remove unused Makefile targets before starting migration
- Use mise to manage all tool versions and scripts (Go, Node, Python, etc.)
- Ensure all CI/CD and local workflows are compatible with mise
- Update all developer and CI documentation to reference mise commands
- Optimize scripts and processes for performance and maintainability

## 🔗 Code References & TODOs
- [ ] TODO: Remove Makefile and related scripts
- [ ] TODO: Update docs/README.md, backend/README.md, frontend/README.md, custom_components/README.md
- [ ] FIXME: Any Makefile-specific logic in scripts/
