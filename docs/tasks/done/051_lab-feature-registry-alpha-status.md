<!-- DOCTOC SKIP -->

# [FEATURE]: Lab Feature Registry with Status Tiers (Alpha/Beta)

**Target Repo:** `srat`
**Status:** ✅ Complete
**Issue Link:** [Optional GH Issue URL]

## 🎯 Objective

Replace the single flat `experimental_lab_mode` boolean gating with a
maintainable lab-feature registry — single source of truth in the Go
backend — that assigns a maturity tier to every lab feature and enforces
availability server-side.

- **Beta** features: visible when `experimental_lab_mode` is enabled (any build).
- **Alpha** features: only available in development/prerelease builds
  (`config.Environment() != "production"`); completely unreachable in
  release builds regardless of Lab Mode.

## 🛠️ Technical Specifications

- **Inputs:** `config.Environment()` (existing), `experimental_lab_mode` setting
- **Outputs:** new `GET /lab_features` endpoint returning the registry with
  a computed `available` flag per feature
- **Dependencies:** existing handler/fx patterns (`backend/src/api/system.go`),
  existing env classification (`backend/src/config/version.go`)

## 📝 Task List

- [x] Task 1: Add `backend/src/service/lab_features.go` — `LabFeatureStatus`
  (`alpha`/`beta`) + `LabFeature{Key,Name,Description,Status}` + static
  registry table + `Get(key)`/`All()` lookups
- [x] Task 2: Add `backend/src/api/lab_features_handler.go` — `GET /lab_features`
  returning the registry filtered by env (alpha omitted when
  `config.Environment() == "production"`), each entry with computed
  `available` (alpha → env check; beta → `experimental_lab_mode`)
- [x] Task 3: Add `LabFeature` DTO (`backend/src/dto/lab_features.go`),
  wire handler into fx (`cmd/srat-server/main-server.go`,
  `cmd/srat-openapi/main-openapi.go`)
- [x] Task 4: Gating helper in `hdidle_handler.go` — `requireLabFeature(key)`
  (404 unknown key, 403 alpha-in-production, beta delegates to
  `requireLabMode`); all five hdidle handlers now gate on `"hdidle"`
- [x] Task 5: Frontend `useLabFeatures()` hook (`frontend/src/hooks/useLabFeatures.ts`)
- [x] Task 6: Migrate surfaces to `isAvailable(...)` — NavBar `isLabOnly` →
  feature-key lookup, HomeAssistantPanel/NetworkDevicesPanel conditionals,
  HDIdle badge/metrics (keep `useLabMode` deps or fold in)
- [x] Task 7: Regenerate OpenAPI + `frontend/src/store/sratApi.ts`
  (`mise run //frontend:gen`)
- [x] Task 8: Backend tests — registry uniqueness/lookup, handler env-filtering
  (production vs `-rc.1`), alpha/beta/404 gating paths (≥70% coverage for
  new code)
- [x] Task 9: Frontend tests — hook fail-closed behavior + gating per panel
- [x] Task 10: Docs — tier lifecycle (alpha → beta → GA = removed from
  registry) in `docs/SETTINGS_DOCUMENTATION.md`; run `docs-validate`
- [x] Task 11: Update CHANGELOG.md; run `mise run //backend:test` and
  `mise run //frontend:test`

## 🧠 Implementation Notes (Copilot Context)

### Registry tiers (current assignment)
| Key | Feature | Tier |
|-----|---------|------|
| `hdidle` | HDIdle per-disk control | beta |
| `smb_conf` | smb.conf view | beta |
| `ha_use_nfs` | Use NFS for HA | beta |
| `ha_custom_component` | HA custom component tools | **alpha** |
| `smb_over_quic` | SMB over QUIC | beta |
| `addon_mdns` | Addon-side mDNS (zeroconf) | beta |

### Status lifecycle
alpha → beta (lab mode) → GA (drop from registry). Alpha entries are never
shipped in release builds — enforcement is server-side only, no frontend env
logic needed for correctness.

### Registry shape (Go)
```go
type LabFeatureStatus string
const (
    StatusAlpha LabFeatureStatus = "alpha" // dev + prerelease only
    StatusBeta  LabFeatureStatus = "beta"  // behind experimental_lab_mode
)
type LabFeature struct {
    Key, Name, Description string
    Status LabFeatureStatus
}
```

### Gating helper sketch
```go
func (h *HDIdleHandler) requireLabFeature(featureKey string) error {
    feature, ok := h.labRegistry.Get(featureKey)
    if !ok { return huma.Error404NotFound("unknown lab feature", nil) }
    if feature.Status == service.StatusAlpha && config.Environment() == "production" {
        return huma.Error403Forbidden("alpha feature not available in release builds", nil)
    }
    return h.requireLabMode()
}
```

## 🔗 Code References & TODOs

- [x] `backend/src/api/hdidle_handler.go` — `requireLabFeature` (was `requireLabMode`)
- [x] `backend/src/service/lab_features.go` — registry (single source of truth)
- [x] `backend/src/api/lab_features_handler.go` — GET /lab_features
- [x] `backend/src/dto/lab_features.go` — LabFeature DTO
- [x] `frontend/src/hooks/useLabFeatures.ts` — hook
- [x] `frontend/src/components/NavBar.tsx` — `isLabOnly` → feature key
- [x] `frontend/src/pages/settings/panels/HomeAssistantPanel.tsx` — swap gates
- [x] `frontend/src/pages/settings/panels/NetworkDevicesPanel.tsx` — SMB over QUIC gate
