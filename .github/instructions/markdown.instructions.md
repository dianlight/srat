<!-- DOCTOC SKIP -->

---

description: "Markdown documentation standards for the SRAT project"
applyTo: "\**/*.md"

---

# Markdown Documentation Instructions

Use GitHub Flavored Markdown (GFM) and follow SRAT documentation standards. Keep guidance clear, consistent, and easy to scan.

## Structure

- Use a single `#` H1 title, then follow a strict hierarchy (`##` → `###` → `####`).
- Add a table of contents for documents with more than 10 sections, using `doctoc` comments if already present.
- Separate sections with blank lines.

## Formatting

- Bullets: use `-` for lists; indent nested lists by two spaces.
- Numbered lists: use `1.` for each item.
- Code: use fenced blocks with a language tag (for example, `bash,`go, ```tsx).
- Links: use `[descriptive text](https://example.com)` and avoid bare URLs unless an autolink is the best choice.
- Images: use `![alt text](url)` with meaningful alt text.
- Tables: use GFM pipe tables with header rows. Ensure each pipe (`|`) has exactly one space on each side (compact style) to comply with MD060.
- Task lists: use `- [ ]` and `- [x]`.
- End every file with exactly one newline.

## Content quality

- Keep content current with code changes.
- Prefer concrete examples and show expected output where useful.
- Use clear, concise language and SRAT terminology (see the Vale vocabulary).
- Avoid vague words like “simply” or “obviously.”

## Validation

- Run documentation checks when changing docs:
  - `make docs-validate`
  - `make docs-fix` (to auto-fix formatting)
- Address markdownlint, Vale, cspell, and link-check findings.

## Common pitfalls to avoid

- Skipping language tags on code blocks.
- Breaking heading hierarchy (for example, jumping from `##` to `####`).
- Leaving stale examples or mismatched behavior descriptions.
- Using inconsistent list markers or missing blank lines between sections.
