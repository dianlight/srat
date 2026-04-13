# Internal Embedded Assets

This directory stores build-time generated assets embedded into back-end binaries for `embedallowed` builds.

- `srat.zip` is generated during `mise run //backend:build` via `//custom_components:package-hacs`.
- The zip is intentionally ignored in git and should not be committed.
