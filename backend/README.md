# back-end

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->

- [CLI Commands](#cli-commands)
  - [Database Requirements](#database-requirements)
    - [Examples](#examples)
  - [OpenAPI Generation](#openapi-generation)
- [Security scanning](#security-scanning)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## CLI Commands

The SRAT CLI (`srat-cli`) provides several commands for managing the Samba administration tool.

### Database Requirements

Different commands have different database requirements:

| Command   | Database Required | Default DB Path         | Notes                                              |
| --------- | ----------------- | ----------------------- | -------------------------------------------------- |
| `version` | **No**            | N/A                     | Outputs version information without any DB access  |
| `upgrade` | Yes (in-memory)   | `file::memory:`         | Uses in-memory database by default, no file needed |
| `start`   | Yes (file)        | Must specify with `-db` | Requires persistent database file                  |
| `stop`    | Yes (file)        | Must specify with `-db` | Requires persistent database file                  |

#### Examples

```bash
# Version command - no database needed
srat-cli version
srat-cli version -short

# Upgrade command - uses in-memory DB by default
srat-cli upgrade -channel release
srat-cli upgrade -channel prerelease

# Start command - requires database file
srat-cli -db /data/config.db start -out /etc/samba/smb.conf

# Stop command - requires database file
srat-cli -db /data/config.db stop
```

### OpenAPI Generation

The `srat-openapi` tool generates OpenAPI specification files. It requires database initialization but uses an in-memory database by default.

```bash
# Generate OpenAPI docs (uses in-memory DB)
srat-openapi -out ./docs/
```

## Security scanning

Run gosec to scan the back-end codebase:

- make security (alias of `make -C ./backend gosec`)
- Reports are limited to high severity and high confidence issues; generated files are excluded.
- To scan all severities locally, run: `make -C ./backend gosec_all`

Exit codes:

- 0: no issues (CI passes)
- non-zero: issues found; inspect the output and address or justify.

Tips:

- Prefer 0750 for directories and 0600 for sensitive files written by the app.
- Avoid invoking shells; when running commands, restrict binaries and validate arguments.
- Guard integer conversions between signed/unsigned types.
