#!/usr/bin/env bash
# Enforce context-aware slog/tlog usage where a ctx parameter is present.
# Fails if a function with a `ctx context.Context` parameter (or `ctx *context.Context`) contains
# non-context slog/tlog calls: slog.Info(...), tlog.Warn(...), etc, instead of slog.InfoContext(ctx,...)
# Suppress with `// nolint:contextlog` on the same line.
# Skip entirely by exporting SKIP_CONTEXT_LOGGING_LINT=1

set -euo pipefail

if [[ "${SKIP_CONTEXT_LOGGING_LINT:-}" == "1" ]]; then
  echo "[context-log] Skipped (SKIP_CONTEXT_LOGGING_LINT=1)"
  exit 0
fi

shopt -s nullglob
ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
GO_FILES=()
while IFS= read -r -d '' f; do GO_FILES+=("$f"); done < <(find "$ROOT_DIR/backend/src" -type f -name '*.go' -not -path '*/vendor/*' -print0)

violations=()

for file in "${GO_FILES[@]}"; do
  # Use awk state machine per file
  awk -v FILE="$file" '
    function report(l, txt) { printf("%s:%d: %s\n", FILE, l, txt); }
    /func[ \t].*\(/ {
      inFunc=1
      hasCtx=0
      # simple detection of ctx param
      if ($0 ~ /ctx[ \t]*,|ctx[ \t]*context\.Context|ctx[ \t]*\*context\.Context/) { hasCtx=1 }
    }
      inFunc && index($0,"{") { braceDepth++ }
      inFunc && index($0,"}") { braceDepth-- ; if (braceDepth<=0) { inFunc=0; hasCtx=0 } }
    {
      if (inFunc && hasCtx) {
        # match non-context slog/tlog
        if ($0 ~ /(slog|tlog)\.(Trace|Debug|Info|Warn|Error)\(/ && $0 !~ /(slog|tlog)\.(Trace|Debug|Info|Warn|Error)Context/ && $0 !~ /nolint:contextlog/) {
          report(NR, "Non-context logging inside ctx-enabled function: " $0)
        }
      }
    }
  ' "$file" >> /tmp/context_log_scan.$$ || true

done

if [[ -s /tmp/context_log_scan.$$ ]]; then
  mapfile -t violations < /tmp/context_log_scan.$$
  echo "Context logging violations detected:" >&2
  printf '%s\n' "${violations[@]}" >&2
  echo "Total: ${#violations[@]} (use // nolint:contextlog to suppress specific false positives)." >&2
  exit 1
else
  echo "[context-log] OK (no violations)"
fi
rm -f /tmp/context_log_scan.$$ || true
