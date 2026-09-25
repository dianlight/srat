# Orchestrator learnings
- 2026-09-24: Fresh frontend worktrees have hollow node_modules (tsc TS2688, missing vite plugin) — run bun install in frontend/ first; selection-hook and verify sessions both hit this.
- 2026-09-24: vitest 5 has no --rerun-each flag; stability repeats via shell-looped vitest runs instead.
- 2026-09-24: Narrowed useState setter types break future functional-update call sites — declare Dispatch<SetStateAction<..>> in extracted hooks from the start.
- 2026-09-24: Pure helpers moved out of updateQueryData recipes must be returned, not just called — Immer discards void recipes; existing MSW-refetch tests can mask the no-op.
- 2026-09-24: Biome useIterableCallbackReturn rejects expression-form forEach dispose callbacks — keep block form; pre-commit hk hooks catch it at commit time.
- 2026-09-24: Prop-count gates should count call-site JSX attributes; deprecated flat optionals per issue migration notes keep legacy tests green without violating the gate.
- 2026-09-24: Mount-hook seam that works: hook owns when (finally/clear timing, toast, confirm retry), parent owns what via onCleared callback; setError root as optional param keeps toast-only behavior default.
- 2026-09-24: volumes test scope green baseline: 186 files-scoped tests; full touched-area scope 202 with dashboard metrics plus ActionableItems.
