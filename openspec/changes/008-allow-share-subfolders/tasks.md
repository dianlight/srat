# Tasks

## 1. Data Model and Persistence

- [x] 1.1 Add optional `Subfolder` field to `dto.SharedResource` and `dbom.ExportedShare`, plus DB migration defaulting to empty
- [x] 1.2 Verify backend build and model round-trip tests pass with the new column

## 2. Validation and Path Resolution

- [x] 2.1 Validate subfolder values: reject empty segments, absolute paths, and parent-traversal (`..`) — verified via API handler tests returning 422
- [x] 2.2 Resolve final share path via `filepath.Join(mountRoot, subfolder)` and create the directory when missing — verified via service unit tests

## 3. Samba Template and Frontend

- [x] 3.1 Update `smb.gtpl` to emit the resolved share path (`path = <resolved>`) — verified via template rendering checks
- [x] 3.2 Add optional "Subfolder" input with helper text to the frontend share form — verified via component test

## 4. Tests, API Surface, and Documentation

- [x] 4.1 Add backend unit and API handler tests covering resolution, default-to-root, and traversal rejection — verified via targeted backend tests
- [x] 4.2 Add frontend component test for the subfolder input — verified via targeted frontend test
- [x] 4.3 Update OpenAPI spec and regenerate frontend types — verified via type generation and typecheck
- [x] 4.4 Update share configuration documentation with the `subfolder` field — verified via docs validation
