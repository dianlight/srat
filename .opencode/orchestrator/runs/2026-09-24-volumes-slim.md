# run: volumes-slim   status: done
request: Slim Volumes God Page via hooks plus codegen-type and util dedupe — acceptance: Volumes.tsx <400 lines, TreeView <=4 props, DetailPanel <=3 props, no local SMART types, single decodeEscapeSequence def, tsc clean, touched tests green plus rerun-each 10 stable
base: issue-1235-artful-narwhal @ 2f6ff0acae96a5fcfc8250cfa2041b495c5be7c3
| id | kind | status | complexity | model | session | worktree/branch | validated_commit | rounds | depends_on |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| mount-hook | implementation | validated, integrated, cleaned | normal | session-default | ses_f2c33edbeffeg7oGKX5jdmEyV6 | integrated into ee8127d38, worktree removed | 03c873aac5e7f1b360cb2b4704547a9d1f867d77 | 0 | - |
| selection-hook | implementation | validated, integrated, cleaned | normal | session-default | ses_f2c33edbdffedhgZH77PP6tPcE | integrated into ee8127d38, worktree removed | 34f39db5a0b31f37bc80ec27296215dd94f00ab1 | 1 | - |
| volumes-integration | implementation | validated, integrated, cleaned | complex | session-default | ses_f2c2ca453ffeCws7r53gphIlCG | integrated into ee8127d38, worktree removed | 0f958baf4827322dc06043cbcafec3084f6b5d84 | 1 | mount-hook, selection-hook |
| e2e-verify | verification | passed | normal | session-default | ses_f2c1a6b72ffeuoivLG4nhvIf47 | /Users/ltarantino/.local/share/opencode/worktree/b8cab8da09f72eaeef8e46904c6d2bb008d25ec9/issue-1235-verify / issue-1235-verify | ee8127d38 | 0 | volumes-integration |
| simplify | simplification | done, committed 75ec300f | normal | session-default | ses_f2c148003ffer4Tv9N4YrOBGZd | base checkout | - | 0 | e2e-verify |
## Decisions
- 2026-09-24: Prop-gate counts call-site JSX attributes; deprecated flat optionals retained per issue migration note — reviewer advisory accepted (rejected: removing legacy props and expanding scope to legacy tests)
## Decisions
- 2026-09-24: Volumes-only scope, single PR, hard <400 lines and prop-count gate — user confirmed (rejected: multi-PR split, aspirational gate)
- 2026-09-24: Implementer chooses surviving props; mount hook owns toast/confirm/retry/clear, selection hook owns dialog states plus localStorage and openDialogForPartition — user confirmed
- 2026-09-24: Shared decodeEscapeSequence under frontend/src/utils with all call sites updated; SMART locals replaced by sratApi imports keeping wrapper/Inner split; MSW pin mount plus SMART only, no full handlers.ts cleanup — user confirmed
## Findings & risks
- Volumes.tsx 824 lines, onSubmitMountVolume Volumes.tsx:352-431; props VolumesTreeView.tsx:31, VolumeDetailsPanel.tsx:37; renderDiskIcon dup VolumesTreeView.tsx:106 / VolumeDetailsPanel.tsx:95; isSynthesizedWholeDiskPartition helper volumes/utils.ts:103; decodeEscapeSequence dup volumes/utils.ts:3 and dashboard/metrics/utils.ts:1 plus extra call sites; SMART locals SmartStatusPanel.tsx:38-49 shadow sratApi.ts:1404,1437,1453; mocks/handlers.ts 2757 lines auto-generated ts-nocheck
