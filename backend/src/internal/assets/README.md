<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [Internal Embedded Assets](#internal-embedded-assets)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

# Internal Embedded Assets

This directory stores build-time generated assets embedded into back-end binaries for `embedallowed` builds.

- `srat.zip` is generated during `mise run //backend:build` via `//custom_components:package-hacs`.
- The zip is intentionally ignored in git and should not be committed.
