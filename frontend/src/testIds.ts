/**
 * Single source of truth for UI test ids.
 *
 * Rules:
 * - Every `data-testid` on a product component MUST be registered here.
 * - Naming: kebab-case, hierarchical `page-component-element`.
 * - The `testFixtures` section holds ids used only by test mock renderers;
 *   they are registered so the ESLint rule can tell deliberate ids from typos.
 *
 * Library-generated ids (e.g. MUI icon auto-testids like "AddIcon") and
 * throwaway `mock-*` fixture ids are exempt from this registry.
 */
export const testIds = {
  footer: {
    donationButton: "footer-donation-button",
  },
  dashboard: {
    hdidleSuggestionBadge: "dashboard-hdidle-suggestion-badge",
    smartIcon: "disk-health-smart-icon",
  },
  volumes: {
    partitionActionIcon: "partition-action-icon",
    actionsGrid: "partition-actions-grid",
    partitionActionsRoot: "partition-actions-root",
  },
  shares: {
    legacyBadge: "shares-legacy-badge",
  },
  smbConf: {
    codeViewer: "smbconf-code-viewer",
  },
  swagger: {
    explorer: "openapi-explorer",
  },
  fontawesome: {
    icon: "fontawesome-svg-icon",
    path: "fontawesome-icon-path",
  },
  testFixtures: {
    shareEditFormSubmit: "submit-form-test",
    shares: {
      select: "select-share",
      triggerUpdate: "trigger-update",
      triggerDelete: "trigger-delete",
    },
    userDetailsEditForm: "edit-form",
    syntaxHighlighter: "syntax-highlighter",
  },
} as const;

/** Union of every registered id, e.g. "footer-donation-button". */
export type TestId = Flatten<typeof testIds>;

type Flatten<T> =
  T extends Record<string, infer V>
    ? V extends string
      ? V
      : Flatten<V>
    : never;
