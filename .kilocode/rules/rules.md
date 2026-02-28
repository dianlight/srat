<!-- DOCTOC SKIP -->

When making function calls using tools that accept array or object parameters ensure those are structured using JSON. For example:

```xml
<example_complex_tool>
  <parameter>[
    {"color": "orange", "options": {"option_key_1": true, "option_key_2": "value"}},
    {"color": "purple", "options": {"option_key_1": true, "option_key_2": "value"}}
  ]</parameter>
</example_complex_tool>
```

> **Note:** Never pass array or object parameters as plain strings or inline text — always use valid JSON syntax.

# Project Coding Instructions

Before generating or modifying any code in this project, you **must** read and follow the project's coding instructions:

1. Read [`.github/copilot-instructions.md`](.github/copilot-instructions.md) — contains the complete set of non-negotiable coding rules for this project.

2. Read the relevant language/task-specific instruction file(s) from [`.github/instructions/`](.github/instructions/):
   - [`go.instructions.md`](.github/instructions/go.instructions.md) — Go backend development rules
   - [`reactjs.instructions.md`](.github/instructions/reactjs.instructions.md) — React/TypeScript frontend rules
   - [`python.instructions.md`](.github/instructions/python.instructions.md) — Python / Home Assistant integration rules
   - [`backend_test.instructions.md`](.github/instructions/backend_test.instructions.md) — Backend testing rules
   - [`fontend_test.instructions.md`](.github/instructions/fontend_test.instructions.md) — Frontend testing rules
   - [`markdown.instructions.md`](.github/instructions/markdown.instructions.md) — Markdown documentation rules

These instructions are **mandatory** and take precedence over any general coding conventions. All generated code must conform to the patterns, naming conventions, tooling choices, and quality standards defined in those files.

3. If you have any questions about the project's coding standards, ask for clarification before proceeding.

4. Create commit messages following instructions in [`.github/.copilot-commit-message-instructions.md`](.github/.copilot-commit-message-instructions.md) to ensure clear and consistent commit history.
