/**
 * Single source of truth for decoding `\xHH` escape sequences in backend
 * device/partition names. All volumes and dashboard call sites import from
 * here; `pages/volumes/utils.ts` and `pages/dashboard/metrics/utils.ts`
 * re-export this definition for backward compatibility.
 */
export function decodeEscapeSequence(source: unknown): string {
  if (typeof source !== "string") return "";
  return source.replace(/\\x([0-9A-Fa-f]{2})/g, (_match, group1) =>
    String.fromCharCode(parseInt(String(group1), 16)),
  );
}
