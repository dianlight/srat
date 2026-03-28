# [REFACTOR]: Migrate to mise.jdx.dev and Remove Makefile

**Target Repo:** `srat`  **Status:** 🔄 In Progress  **Issue Link:** https://github.com/dianlight/srat/issues/532

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
- [ ] Task 7: Ensure CI/CD pipelines are updated and functional with mise https://mise.jdx.dev/continuous-integration.html#github-actions
- [ ] Task 8: Conduct code review, cleanup, and final validation
- [ ] Task 9: Evaluate the use of https://hk.jdx.dev/why-hk.html over prek for any relevant optimizations or improvements and better mise integration
- [ ] Task 10: Implement any necessary changes based on the evaluation of hk.jdx.dev and integrate it into the workflow if beneficial
- [ ] Task 11: Update documentation to reflect any changes made based on the hk.jdx.dev evaluation and integration
- [ ] Task 12: Use mise to manage all tool versions and scripts (Go, Node, Python, etc.) across the monorepo
- [ ] Task 13: Migrate all developer environment setup and devcontainer to use mise
- [ ] Task 14: Devcontainer environment upgrade with the use of https://mise.jdx.dev/mise-cookbook/shell-tricks.html and other mise features to optimize the development environment and workflow
- [ ] Task 15: Add mise MCP configuration for all relevant tools and scripts
- [ ] Task 16: Add vscode related to workspace config and plugins mise-vscode to devcontainern 
- [ ] Task 17: Code review, cleanup, and final validation
- [ ] Task 18: Check also renovate config if need changes
- [ ] Task 19: Ask to create a PR with the task implementation and link it here for tracking

## 🧠 Implementation Notes (Copilot Context)
**Branch:** `refactor/migrate-to-mise-remove-makefile` (feature branch created)

**Pre-implementation Plan:**
- Fully migrate the monorepo to use mise.jdx.dev for all toolchain and environment management.
- Remove all Makefile-relative configs and the Makefile itself.
- All build, test, lint, and dev workflows must work via mise.
- Documentation and CI/CD must reference mise, not Makefile.
- Impacted files: `/Makefile`, `/backend/Makefile`, `/custom_components/Makefile`, scripts referencing Makefile, all README.md/docs, devcontainer, CI/CD config.
- Step-by-step plan:
	1. Audit and clean up all Makefiles, removing unused targets.
	2. Plan and document mise migration steps for backend, frontend, and custom_components.
	3. Implement mise configuration for all environments and workflows.
	4. Remove Makefile and all Makefile-relative configs.
	5. Update all documentation to reference mise workflows.
	6. Optimize and test all build, test, and lint processes under mise.
	7. Update CI/CD pipelines for mise compatibility.
	8. Evaluate hk.jdx.dev for possible integration.
	9. Migrate devcontainer and developer setup to mise.
	10. Add mise MCP config for all tools/scripts.
	11. Add VSCode mise plugin/config to devcontainer.
	12. Update Renovate config if needed.
	13. Final code review and validation.
- Test/validation: All build, test, and lint commands must succeed using mise; onboarding/setup must work; CI/CD must pass; docs must be accurate.
- Risks: Missed Makefile logic, CI/CD or devcontainer breakage, stale Makefile references, incomplete migration of subprojects.

## 🔗 Code References & TODOs
- [ ] TODO: Remove Makefile and related scripts
- [ ] TODO: Update docs/README.md, backend/README.md, frontend/README.md, custom_components/README.md
- [ ] FIXME: Any Makefile-specific logic in scripts/
