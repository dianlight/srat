<!-- DOCTOC SKIP -->

# Memory Index: Stored Facts in Active Instructions

This index maps all 19 stored memory facts from prior sessions to their current locations in instruction files. Use this to understand what's documented, what's discoverable, and what may need future attention.

## Integration Status Summary

| Status                  | Count | Details                                                 |
| ----------------------- | ----- | ------------------------------------------------------- |
| ✅ Integrated (Round 2) | 8     | Actively documented in instructions                     |
| 🔍 Already Covered      | 6     | Exists in instructions but fact wasn't explicitly added |
| ⚠️ Pending Review       | 3     | May need integration or retirement                      |
| 📦 Archived             | 2     | Obsolete or superseded by new patterns                  |

---

## Integrated Facts (8/19) - Round 2 ✅

These facts were actively integrated into instruction files to prevent agent confusion and future bugs.

### Frontend Testing (5 facts)

| #   | Fact                                            | Location                                                          | Integrated | Details                                                                                          |
| --- | ----------------------------------------------- | ----------------------------------------------------------------- | :--------: | ------------------------------------------------------------------------------------------------ |
| 1   | MSW request body consumption (`clone().json()`) | `.github/instructions/fontend_test.instructions.md` lines 106–130 |     ✅     | Prevents `InvalidStateError: Body has already been used` during test reruns with `--rerun-each`  |
| 2   | MSW handlers preferred over cache seeding       | `.github/instructions/fontend_test.instructions.md` lines 131–147 |     ✅     | Live queries can overwrite RTK cache before assertions run                                       |
| 3   | Mock real source of truth (not network)         | `.github/instructions/fontend_test.instructions.md` lines 148–155 |     ✅     | For deterministic tests, set fixture props not network responses                                 |
| 5   | Avoid duplicate test cleanup                    | `.github/instructions/fontend_test.instructions.md` lines 12–19   |     ✅     | `bun-setup.ts` already calls `cleanup()` in `afterEach`; manual cleanup triggers MUI timing bugs |
| 4   | MUI v9 Dialog animation timing                  | `.github/instructions/fontend_test.instructions.md` lines 50–65   |     ✅     | Uses `createTheme({ components: { MuiDialog: { defaultProps: { transitionDuration: 0 } } } })`   |

### Backend Testing (2 facts)

| #   | Fact                                  | Location                                                          | Integrated | Details                                                                          |
| --- | ------------------------------------- | ----------------------------------------------------------------- | :--------: | -------------------------------------------------------------------------------- |
| 9   | DTO type safety (enum families)       | `.github/instructions/backend_test.instructions.md` lines 709–724 |     ✅     | Use `dto.ProblemSeverities` not `IssueSeverity`; enum families are distinct      |
| 10  | CanUpgrade must use semver comparison | `.github/instructions/backend_test.instructions.md` lines 725–732 |     ✅     | Use `github.com/Masterminds/semver/v3` for version comparison, not simple checks |

### Backend Services (1 fact)

| #   | Fact                            | Location                                      | Integrated | Details                                                                                      |
| --- | ------------------------------- | --------------------------------------------- | :--------: | -------------------------------------------------------------------------------------------- |
| 19  | HA Supervisor API vs REST proxy | `.github/copilot-instructions.md` lines 61–65 |     ✅     | For lifecycle ops (restart/start/stop), use Supervisor API; REST proxy causes 504 on restart |

---

## Already Covered Facts (6/19) - No New Integration Needed ✅

These facts are documented in active instructions but weren't explicitly added in Round 2 because they're already discoverable elsewhere.

| #   | Fact                                  | Primary Location                                                       | Status | Why Not Added                                   |
| --- | ------------------------------------- | ---------------------------------------------------------------------- | :----: | ----------------------------------------------- |
| 11  | RTK Query lazy hooks pattern          | `.github/instructions/reactjs.instructions.md` lines 180–186           |   🔍   | Clearly documented; agents discover via search  |
| 12  | IssueCard ignored-state keying        | `.github/instructions/frontend_test.instructions.md` (implicit)        |   🔍   | Logic is in code; would be over-documentation   |
| 13  | HA issues schema fixable exclusion    | `.github/instructions/python.instructions.md` (Home Assistant section) |   🔍   | Covered in HA custom component repairs section  |
| 14  | App.commandEvents test cleanup        | `.github/instructions/fontend_test.instructions.md` lines 12–19        |   🔍   | Generalized under "Avoid duplicate cleanup"     |
| 15  | Problem Severity field enum type      | `.github/instructions/backend_test.instructions.md` lines 709–724      |   🔍   | Integrated as "DTO type safety"                 |
| 16  | Repair command broadcasting ownership | `.github/copilot-instructions.md` lines 93–110                         |   🔍   | Covered in "Core Service Architecture Patterns" |

---

## Pending Review (3/19) - Needs Decision ⚠️

These facts may still be relevant but weren't integrated due to time constraints or unclear current state. Consider for future reviews.

| #   | Fact                                      |  Current Status   | Recommendation                                                                                                                                                        |
| --- | ----------------------------------------- | :---------------: | --------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 6   | HDIdleDiskSettings test determinism       |  Not integrated   | ✋ **Specific to one component**; may not be general enough for instruction docs. Keep in stored memory for future HDIdle work.                                       |
| 7   | Filesystem support preflight pattern      |  Not integrated   | ✋ **Specific to dialog tests**; generalization to "Don't over-seed RTK cache" covered under MSW handlers fact #2. Consider retiring or marking as super-specialized. |
| 8   | Happy-dom/Bun event constructor mismatch  |  Not integrated   | ✋ **Very specific setup issue**; mostly resolved by current `test/happy-dom-setup.ts`. Review whether this is still a problem in current Bun version.                          |
| 17  | RepairService compatibility bridge        | Partially covered | ✋ **Covered in service architecture**; consider adding explicit "compatibility bridge pattern" subsection for future API migrations.                                 |
| 18  | Custom component lifecycle restart repair | Partially covered | ✋ **Covered in settings API section**; generalization: "Lifecycle events via service delegation" could be a pattern.                                                 |

---

## Archived Facts (2/19) - Obsolete ❌

These facts have been superseded by newer patterns or are no longer applicable.

| #   | Fact                                | Why Archived | Alternative                                                                                         |
| --- | ----------------------------------- | :----------: | --------------------------------------------------------------------------------------------------- |
| 2   | Go 1.25 WaitGroup.Go vs Add/Done    |  Superseded  | Go 1.26+ requires `WaitGroup.Go`; Add/Done is legacy, see `.github/instructions/go.instructions.md` |
| 3   | TypeScript 5 unions vs type aliases |  Superseded  | TypeScript 6.0+ standard; see `.github/instructions/typescript-6-es2022.instructions.md`            |

---

## Recommendations for Future Rounds

### Quick Wins (Low Effort, High Value)

1. **Review Pending #6–8**: Decide whether these are component-specific (archive) or generalizable (integrate)
2. **Create pattern doc**: "Compatibility Bridge Pattern" for API migrations (would cover RepairService + future patterns)

### Medium Effort

1. **Retire archived facts**: After confirming Go 1.26 + TS 6.0 are stable, remove from stored memory
2. **Create decision tree**: "When to mock network vs props" to consolidate facts #3, #7, #12

### Best Practice: Memory Maintenance Cycle

- **After each optimization round**: Scan stored facts
- **Categorize**: Integrated / Already Covered / Pending / Archived
- **Action**: Integrate high-value ones, retire obsolete ones
- **Result**: Keep 15–20 actionable facts; retire 5–10 annually

---

## Cross-Reference: Fact → Instruction File

Sorted by instruction file for quick lookup:

### `.github/copilot-instructions.md`

- Fact #19: HA Supervisor API vs REST proxy (lines 61–65)
- Fact #16: Repair command broadcasting ownership (lines 93–110)

### `.github/instructions/backend_test.instructions.md`

- Fact #9: DTO type safety - enum families (lines 709–724)
- Fact #10: CanUpgrade semver comparison (lines 725–732)
- Fact #15: Problem Severity field enum type (lines 709–724, covered by #9)

### `.github/instructions/fontend_test.instructions.md`

- Fact #1: MSW body clone pattern (lines 106–130)
- Fact #2: MSW handlers over cache seeding (lines 131–147)
- Fact #3: Mock real source of truth (lines 148–155)
- Fact #4: MUI v9 Dialog animation (lines 50–65)
- Fact #5: Avoid duplicate cleanup (lines 12–19)
- Fact #14: App.commandEvents cleanup (lines 12–19, covered by #5)

### `.github/instructions/reactjs.instructions.md`

- Fact #11: RTK Query lazy hooks (lines 180–186)

### `docs/shared-principles.md`

- Fact #20: Cross-language testing lifecycle (referenced in link from multiple files)

---

## Next Steps

- [ ] **Round 3.1**: Review Pending facts #6–8; decide integrate/retire
- [ ] **Round 3.2**: Create pattern docs for new insights (Compatibility Bridge, Network vs Props)
- [ ] **Round 3.3**: Schedule annual memory maintenance to retire obsolete facts

---

**Last Updated**: 2026-04-25  
**Facts Tracked**: 19 total (8 integrated, 6 covered, 3 pending, 2 archived)  
**Maintainer**: Copilot optimization workflow
