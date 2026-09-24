// Package volume splits the former VolumeService God Object into focused
// collaborators: MountRepository (mount row persistence, reconciliation and
// eviction), MountOrchestrator (mount/unmount operations and automount guard)
// and UdevHandler (udev and volume event reactions).
package volume

import (
	"context"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/dianlight/srat/converter"
	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	"github.com/dianlight/srat/repository"
	"github.com/dianlight/tlog"
	"gitlab.com/tozd/go/errors"
)

// MountRepository loads, reconciles and evicts persisted mount point rows.
// It is the only volume-layer type that talks to mount point storage, via
// repository.MountPointPathRepositoryInterface; callers never touch *gorm.DB.
type MountRepository struct {
	ctx      context.Context
	repo     repository.MountPointPathRepositoryInterface
	disks    *dto.DiskMap
	eventBus events.EventBusInterface
	conv     converter.DtoToDbomConverterImpl
}

// NewMountRepository creates a MountRepository over shared volume state.
func NewMountRepository(ctx context.Context, repo repository.MountPointPathRepositoryInterface, disks *dto.DiskMap, eventBus events.EventBusInterface) *MountRepository {
	return &MountRepository{ctx: ctx, repo: repo, disks: disks, eventBus: eventBus}
}

// Persist stores one mount point without wiping configuration absent from
// the payload; the repository merges into the existing record.
func (m *MountRepository) Persist(md *dto.MountPointData) errors.E {
	tlog.TraceContext(m.ctx, "Mount point data", "data", spew.Sdump(md))
	return m.repo.Upsert(m.ctx, *md)
}

// LoadByDevice loads mount point data from the database for a partition.
// Before converting it reconciles legacy rows (#1073): dashes-path rows that
// normalize to an already-persisted underscores sibling for the same device
// are collision-merged onto the live row, and rows that are unmounted, not
// startup-mounted, share-less and missing on disk are hard-deleted as stale
// orphans. The returned map contains only the surviving rows.
func (m *MountRepository) LoadByDevice(part *dto.Partition) (map[string]*dto.MountPointData, errors.E) {
	if part.Id == nil || *part.Id == "" {
		return nil, nil
	}

	rows, err := m.repo.FindByDeviceID(m.ctx, *part.Id)
	if err != nil {
		return nil, err
	}

	if len(rows) == 0 {
		tlog.TraceContext(m.ctx, "No mount point records found in DB for device", "device", *part.Id, "name", part.Name)
		return make(map[string]*dto.MountPointData), nil
	}

	tlog.TraceContext(m.ctx, "Found mount point records in DB for device", "device", *part.Id, "name", part.Name, "count", len(rows))
	rows = m.Reconcile(part, rows)
	if len(rows) == 0 {
		return make(map[string]*dto.MountPointData), nil
	}
	mountData, convErr := m.conv.MountPointPathsToMountPointDataMap(rows)
	if convErr != nil {
		slog.ErrorContext(m.ctx, "Failed to convert mount point data", "device", *part.Id, "err", convErr)
		return nil, errors.WithStack(convErr)
	}

	tlog.TraceContext(m.ctx, "Loaded mount point from repository", "device", *part.Id, "mountData", mountData)
	return mountData, nil
}

// LoadByPath loads a single mount point configuration by its primary key
// (path, root), independently of the partition device id. The second return
// value reports whether a record was found.
func (m *MountRepository) LoadByPath(path string, root string) (*dto.MountPointData, bool, errors.E) {
	if path == "" {
		return nil, false, nil
	}
	rows, err := m.repo.FindByPath(m.ctx, path, root)
	if err != nil {
		return nil, false, err
	}
	if len(rows) == 0 {
		return nil, false, nil
	}
	dtoMP := dto.MountPointData{}
	if convErr := m.conv.MountPointPathToMountPointData(rows[0], &dtoMP, nil); convErr != nil {
		return nil, false, errors.WithStack(convErr)
	}
	return &dtoMP, true, nil
}

// normalizeMountPath collapses legacy dash-separated mount basenames to the
// underscore form produced by the current frontend suggestion
// (VolumeMountDialog replaces only whitespace-ish runes with "_", while the
// older wizard helper replaced every [^a-zA-Z0-9]+ run with "_"). Only the
// basename is normalized: "/mnt/ata-WD-part2" becomes "/mnt/ata_WD_part2".
// Non-/mnt paths and names without dashes pass through unchanged.
func normalizeMountPath(path string) string {
	base := filepath.Base(path)
	if !strings.Contains(base, "-") || !strings.HasPrefix(path, "/mnt/") {
		return path
	}
	return filepath.Join("/mnt", strings.ReplaceAll(base, "-", "_"))
}

// mountRowIsStaleOrphan reports whether a persisted row is safe to hard-delete:
// unmounted, not wanted at startup, share-less (checked via the preloaded
// ExportedShare — nil or zero Name), missing on disk (IsInvalid is derived
// from isPathDirNotExists during DTO conversion) and an ADDON mount (HOST
// rows are never pruned here).
func mountRowIsStaleOrphan(md *dto.MountPointData) bool {
	if md == nil || md.IsMounted || md.Share != nil {
		return false
	}
	if md.Type != "" && md.Type != "ADDON" {
		return false
	}
	if md.IsToMountAtStartup != nil && *md.IsToMountAtStartup {
		return false
	}
	return md.IsInvalid
}

// Reconcile merges dash/underscore collisions and prunes stale orphans for
// one partition's persisted rows (#1073). Only duplicate rows are touched: a
// singleton row is always kept, so a user-created but not-yet-mounted config
// is never deleted out from under the mount dialog. Share-bearing rows are
// never merged or pruned here — deleting them would orphan the
// exported_shares FK instead. Every eviction also drops the row from the
// in-memory DiskMap cache and emits a Disk UPDATE event so subscribers
// broadcast a fresh snapshot. The surviving rows are returned.
func (m *MountRepository) Reconcile(part *dto.Partition, rows []dbom.MountPointPath) []dbom.MountPointPath {
	if len(rows) <= 1 || part.Id == nil {
		return rows
	}
	mountData, convErr := m.conv.MountPointPathsToMountPointDataMap(rows)
	if convErr != nil {
		slog.WarnContext(m.ctx, "Skipping mount point reconciliation, conversion failed", "device", *part.Id, "err", convErr)
		return rows
	}
	// Index raw rows by path for O(1) survivor/orphan lookups below.
	byPath := make(map[string]int, len(rows))
	for i := range rows {
		byPath[rows[i].Path] = i
	}
	evicted := make(map[string]bool, len(rows))
	mergeInto := func(stalePath, livePath string) {
		stale, ok := mountData[stalePath]
		if !ok || evicted[stalePath] {
			return
		}
		live, ok := mountData[livePath]
		if !ok || evicted[livePath] {
			return
		}
		// Preserve automount intent: if the survivor is not startup-mounted
		// but the legacy row was, carry the flag over. The survivor keeps
		// its own path, share FK and DeviceId; only the flag migrates.
		// (DB round-trips an unset flag as false, so compare by value, not
		// by nilness.)
		liveStartup := live.IsToMountAtStartup != nil && *live.IsToMountAtStartup
		staleStartup := stale.IsToMountAtStartup != nil && *stale.IsToMountAtStartup
		if !liveStartup && staleStartup {
			t := true
			live.IsToMountAtStartup = &t
			dbLive := rows[byPath[livePath]]
			dbLive.IsToMountAtStartup = &t
			if err := m.repo.Save(m.ctx, &dbLive); err != nil {
				slog.WarnContext(m.ctx, "Failed to persist merged mount config", "live_path", livePath, "err", err)
			} else {
				rows[byPath[livePath]] = dbLive
			}
		}
		m.Evict(part, stalePath)
		evicted[stalePath] = true
	}
	// Collision merge: legacy dashes row normalizes onto the live sibling.
	// Only unmounted, share-less losers are evicted — a mounted row is live
	// by definition and must never be auto-deleted here.
	for _, row := range rows {
		if row.ExportedShare != nil && row.ExportedShare.Name != "" {
			continue
		}
		if normalized := normalizeMountPath(row.Path); normalized != row.Path {
			if live, ok := byPath[normalized]; ok && live != byPath[row.Path] &&
				(rows[live].ExportedShare == nil || rows[live].ExportedShare.Name == "") {
				if staleMD, ok := mountData[row.Path]; ok && staleMD != nil && staleMD.IsMounted {
					slog.WarnContext(m.ctx, "Keeping mounted dash-path row despite underscore collision", "path", row.Path, "live_path", normalized)
					continue
				}
				mergeInto(row.Path, normalized)
			}
		}
	}
	// Stale orphans: anything left that is unmounted, unwanted at startup,
	// share-less and missing on disk goes. Only runs when duplicates exist
	// (guarded above), so a singleton pending config is never deleted.
	for path, md := range mountData {
		if evicted[path] || !mountRowIsStaleOrphan(md) {
			continue
		}
		m.Evict(part, path)
		evicted[path] = true
	}
	if len(evicted) == 0 {
		return rows
	}
	survivors := rows[:0]
	for _, row := range rows {
		if !evicted[row.Path] {
			survivors = append(survivors, row)
		}
	}
	return survivors
}

// Evict hard-deletes one persisted mount row and drops it from the
// in-memory cache. It emits a Disk UPDATE (never a MountPoint event:
// HandleMountPointEvent persists every MountPoint event it sees, so emitting
// one here would resurrect the deleted row via upsert) so subscribers
// broadcast a fresh snapshot without the phantom entry. State is mutated
// before the emit: the bus dispatches synchronously, so listeners must read
// the already-evicted cache (#971).
func (m *MountRepository) Evict(part *dto.Partition, path string) {
	root := "/"
	if err := m.repo.DeleteHard(m.ctx, path, root); err != nil {
		slog.WarnContext(m.ctx, "Failed to delete stale mount point row", "device", *part.Id, "path", path)
		return
	}
	slog.InfoContext(m.ctx, "Pruned stale mount point row", "device", *part.Id, "path", path)
	if part.DiskId != nil {
		m.disks.RemoveMountPoint(*part.DiskId, *part.Id, path)
		if disk, ok := m.disks.Get(*part.DiskId); ok && disk != nil {
			_ = m.eventBus.EmitDisk(events.DiskEvent{
				Event: events.Event{Type: events.EventTypes.UPDATE},
				Disk:  disk,
			})
		}
	}
}
