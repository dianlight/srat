# Backend

## Security scanning

Run gosec to scan the backend codebase:

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
