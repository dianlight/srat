# run: volumes-slim   status: active
request: Slim Volumes God Page via hooks plus codegen-type and util dedupe — acceptance: Volumes.tsx <400 lines, TreeView <=4 props, DetailPanel <=3 props, no local SMART types, single decodeEscapeSequence def, tsc clean, touched tests green plus rerun-each 10 stable
base: issue-1235-artful-narwhal @ 2f6ff0acae96a5fcfc8250cfa2041b495c5be7c3
| id | kind | status | complexity | model | session | worktree/branch | validated_commit | rounds | depends_on |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| mount-hook | implementation | pending | normal | session-default | - | - | - | 0 | - |
| selection-hook | implementation | pending | normal | session-default | - | - | - | 0 | - |
| volumes-integration | implementation | pending | complex | session-default | - | - | - | 0 | mount-hook, selection-hook |
## Decisions
- 2026-09-24: Volumes-only scope, single PR, hard <400 lines and prop-count gate — user confirmed (rejected: multi-PR split, aspirational gate)
- 2026-09-24: Implementer chooses surviving props; mount hook owns toast/confirm/retry/clear, selection hook owns dialog states plus localStorage and openDialogForPartition — user confirmed
- 2026-09-24: Shared decodeEscapeSequence under frontend/src/utils with all call sites updated; SMART locals replaced by sratApi imports keeping wrapper/Inner split; MSW pin mount plus SMART only, no full handlers.ts cleanup — user confirmed
## Findings & risks
- Volumes.tsx 824 lines, onSubmitMountVolume Volumes.tsx:352-431; props VolumesTreeView.tsx:31, VolumeDetailsPanel.tsx:37; renderDiskIcon dup VolumesTreeView.tsx:106 / VolumeDetailsPanel.tsx:95; isSynthesizedWholeDiskPartition helper volumes/utils.ts:103; decodeEscapeSequence dup volumes/utils.ts:3 and dashboard/metrics/utils.ts:1 plus extra call sites; SMART locals SmartStatusPanel.tsx:38-49 shadow sratApi.ts:1404,1437,1453; mocks/handlers.ts 2757 lines auto-generated ts-nocheck
