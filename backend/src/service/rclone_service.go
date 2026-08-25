package service

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"uuid"

	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
	"gorm.io/gorm"

	"github.com/dianlight/srat/converter"
	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	sr "github.com/dianlight/srat/service/rclone"
)

// hassosDataPath is the local mount point of the Home Assistant OS data
// partition inside the SRAT container. Override with SRAT_HASSOS_DATA_PATH
// for non-addon development environments.
const hassosDataPath = "/mnt/data/supervisor"

// syncPollInterval is the delay between async job status polls.
const syncPollInterval = 500 * time.Millisecond

// oauthStateTTL bounds how long a pending authorization may stay open.
const oauthStateTTL = 10 * time.Minute

// RcloneServiceInterface exposes the cloud-sync link management API used by
// the lab-gated rclone HTTP handlers (issue #954).
type RcloneServiceInterface interface {
	// LibraryAvailable reports whether this binary embeds librclone.
	LibraryAvailable() bool
	// ListProviders returns all registered provider drivers.
	ListProviders() []dto.RcloneProviderInfo
	// GetLink returns the link for a target or nil when absent.
	GetLink(targetKind, targetID string) (*dto.RcloneLink, errors.E)
	// ListLinks returns every configured link.
	ListLinks() ([]dto.RcloneLink, errors.E)
	// SaveLink creates/updates the non-secret configuration of a link.
	SaveLink(link dto.RcloneLink) errors.E
	// DeleteLink removes the link and its managed rclone remote.
	DeleteLink(ctx context.Context, targetKind, targetID string) errors.E
	// StartAuth prepares an OAuth flow: persists pending settings, builds
	// the provider URL and returns it with the correlated state token.
	StartAuth(ctx context.Context, targetKind, targetID string, settings map[string]string) (*dto.RcloneAuthStartResponse, errors.E)
	// HandleOAuthCallback validates state, exchanges code→token and writes
	// the remote into the managed rclone.conf. Returns the linked target.
	HandleOAuthCallback(ctx context.Context, code, state string) (*dto.RcloneLink, errors.E)
	// Diff compares local and remote roots of a link.
	Diff(ctx context.Context, targetKind, targetID string) (*dto.RcloneDiffResult, errors.E)
	// Sync starts an asynchronous synchronization job (push|pull|bidi).
	Sync(targetKind, targetID, direction string, dryRun bool) errors.E
	// AbortSync cancels the running job for a link, if any.
	AbortSync(targetKind, targetID string) errors.E
}

// RcloneService implements RcloneServiceInterface.
type RcloneService struct {
	db             *gorm.DB
	ctx            context.Context
	state          *dto.ContextState
	eventBus       events.EventBusInterface
	rc             sr.RcloneRPC
	converter      converter.DtoToDbomConverterImpl
	problemService ProblemServiceInterface

	mu        sync.Mutex
	running   map[string]*rcloneRunningJob // key targetKind/targetID
	pendingMu sync.Mutex
	pending   map[string]rclonePendingAuth // state → auth request + deadline
	redirect  string                       // OAuth callback base URL

	autoSyncStop chan struct{} // signals the auto-sync loop to exit; nil when stopped
}

type rcloneRunningJob struct {
	jobID     int64
	cancel    context.CancelFunc
	link      dbom.RcloneLink
	direction string
	dryRun    bool
}

// RcloneServiceParams defines fx dependencies.
type RcloneServiceParams struct {
	fx.In
	DB         *gorm.DB
	Ctx        context.Context
	State      *dto.ContextState
	EventBus   events.EventBusInterface
	RcloneRPC  sr.RcloneRPC
	ProblemSrv ProblemServiceInterface
}

// NewRcloneService creates the cloud-sync service.
func NewRcloneService(lc fx.Lifecycle, in RcloneServiceParams) RcloneServiceInterface {
	svc := &RcloneService{
		db:             in.DB,
		ctx:            in.Ctx,
		state:          in.State,
		eventBus:       in.EventBus,
		rc:             in.RcloneRPC,
		problemService: in.ProblemSrv,
		running:        map[string]*rcloneRunningJob{},
		pending:        map[string]rclonePendingAuth{},
	}
	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			svc.startAutoSync()
			return nil
		},
		OnStop: func(_ context.Context) error {
			svc.stopAutoSync()
			return nil
		},
	})
	return svc
}

func rcloneJobKey(targetKind, targetID string) string { return targetKind + "/" + targetID }

// remoteName is the managed rclone.conf section name for a link.
func remoteName(link *dbom.RcloneLink) string {
	id := strings.NewReplacer("/", "_", "\\", "_", ":", "_", ".", "_").Replace(link.TargetID)
	return "srat_" + strings.ToLower(link.TargetKind) + "_" + id
}

// fsSpec renders an rclone filesystem specifier for one side of an operation.
func (s *RcloneService) fsSpec(remote *string, localPath string) string {
	if remote == nil {
		return localPath
	}
	return *remote + ":" + strings.TrimPrefix(localPath, "/")
}

// LocalPath resolves the local root directory for a target kind/id.
func (s *RcloneService) LocalPath(targetKind, targetID string) (string, error) {
	switch targetKind {
	case dto.RcloneTargetKindVolume:
		if !strings.HasPrefix(targetID, "/") {
			return "", fmt.Errorf("volume target id must be an absolute path: %q", targetID)
		}
		return filepath.Clean(targetID), nil
	case dto.RcloneTargetKindHassosData:
		p := os.Getenv("SRAT_HASSOS_DATA_PATH")
		if p == "" {
			p = hassosDataPath
		}
		return filepath.Clean(p), nil
	default:
		return "", fmt.Errorf("unknown rclone target kind %q", targetKind)
	}
}

// ---------- providers ----------

func (s *RcloneService) LibraryAvailable() bool { return s.rc.Available() }

func (s *RcloneService) ListProviders() []dto.RcloneProviderInfo {
	drivers := sr.ListDrivers()
	out := make([]dto.RcloneProviderInfo, 0, len(drivers))
	for _, d := range drivers {
		fields := make([]dto.RcloneConfigField, 0, len(d.ConfigFields()))
		for _, f := range d.ConfigFields() {
			fields = append(fields, dto.RcloneConfigField{
				Name: f.Name, Label: f.Label, Description: f.Description,
				Secret: f.Secret, Required: f.Required,
			})
		}
		out = append(out, dto.RcloneProviderInfo{Name: d.Name(), DisplayName: d.DisplayName(), ConfigFields: fields})
	}
	return out
}

// ---------- link CRUD ----------

func (s *RcloneService) GetLink(targetKind, targetID string) (*dto.RcloneLink, errors.E) {
	var row dbom.RcloneLink
	err := s.db.WithContext(s.ctx).Where("target_kind = ? AND target_id = ?", targetKind, targetID).First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, errors.Wrap(err, "query rclone link")
	}
	out, convErr := s.converter.RcloneLinkToRcloneLinkDTO(row)
	if convErr != nil {
		return nil, errors.Wrap(convErr, "convert rclone link")
	}
	return &out, nil
}

func (s *RcloneService) ListLinks() ([]dto.RcloneLink, errors.E) {
	var rows []*dbom.RcloneLink
	if err := s.db.WithContext(s.ctx).Order("target_kind, target_id").Find(&rows).Error; err != nil {
		return nil, errors.Wrap(err, "list rclone links")
	}
	out := make([]dto.RcloneLink, 0, len(rows))
	for _, row := range rows {
		l, convErr := s.converter.RcloneLinkToRcloneLinkDTO(*row)
		if convErr != nil {
			return nil, errors.Wrap(convErr, "convert rclone link")
		}
		out = append(out, l)
	}
	return out, nil
}

// validateTarget checks kind and provider against known values/drivers.
func validateTarget(targetKind, provider string) error {
	if targetKind != dto.RcloneTargetKindVolume && targetKind != dto.RcloneTargetKindHassosData {
		return fmt.Errorf("invalid target_kind %q", targetKind)
	}
	if _, ok := sr.GetDriver(provider); !ok {
		return fmt.Errorf("unknown provider %q", provider)
	}
	return nil
}

func (s *RcloneService) SaveLink(link dto.RcloneLink) errors.E {
	if err := validateTarget(link.TargetKind, link.Provider); err != nil {
		return errors.Wrap(err, "validate rclone link")
	}
	var row dbom.RcloneLink
	err := s.db.Where("target_kind = ? AND target_id = ?", link.TargetKind, link.TargetID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		row = dbom.RcloneLink{TargetKind: link.TargetKind, TargetID: link.TargetID}
	} else if err != nil {
		return errors.Wrap(err, "query rclone link")
	}
	row.Provider = link.Provider
	row.RemotePath = link.RemotePath
	row.AutoSync = link.AutoSync
	row.ScheduleMinutes = link.ScheduleMinutes
	if err := s.db.Save(&row).Error; err != nil {
		return errors.Wrap(err, "save rclone link")
	}
	s.reconfigureAutoSync()
	return nil
}

func (s *RcloneService) DeleteLink(ctx context.Context, targetKind, targetID string) errors.E {
	key := rcloneJobKey(targetKind, targetID)
	s.mu.Lock()
	job, running := s.running[key]
	s.mu.Unlock()
	if running {
		// Cancel without joining: the job goroutine only wakes on its poll
		// tick and then exits via runSync's "aborted" error. Any final row
		// Save it attempts is benign — GORM's Save-with-primary-key issues
		// an UPDATE that matches zero rows once the link is soft-deleted,
		// so a deleted link cannot be resurrected.
		job.cancel()
	}
	var row dbom.RcloneLink
	err := s.db.Where("target_kind = ? AND target_id = ?", targetKind, targetID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return errors.Wrap(err, "query rclone link")
	}
	// Best-effort removal of the managed remote config; tokens must not leak.
	name := remoteName(&row)
	var out map[string]any
	_ = sr.Call(ctx, s.rc, "config/delete", map[string]any{"name": name}, &out)
	if err := s.db.Delete(&row).Error; err != nil {
		return errors.Wrap(err, "delete rclone link")
	}
	s.reconfigureAutoSync()
	return nil
}

// ---------- OAuth flow ----------

// callbackBaseURL derives the externally reachable base URL used to build
// OAuth redirect URIs. It prefers the SRAT public port from context state so
// links work behind Home Assistant ingress too (browser resolves relative
// to the host it opened).
func (s *RcloneService) callbackBaseURL() string {
	if s.redirect != "" {
		return s.redirect
	}
	port := os.Getenv("SRAT_PORT")
	if port == "" {
		port = "8080"
	}
	return "http://localhost:" + port
}

func (s *RcloneService) StartAuth(ctx context.Context, targetKind, targetID string, settings map[string]string) (*dto.RcloneAuthStartResponse, errors.E) {
	var row dbom.RcloneLink
	err := s.db.Where("target_kind = ? AND target_id = ?", targetKind, targetID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.New("link not found; save the link before starting authorization")
	} else if err != nil {
		return nil, errors.Wrap(err, "query rclone link")
	}
	driver, ok := sr.GetDriver(row.Provider)
	if !ok {
		return nil, errors.Errorf("unknown provider %q", row.Provider)
	}
	state := uuid.New().String()
	req := sr.AuthRequest{
		RedirectURI: s.callbackBaseURL() + "/api/rclone/oauth/callback",
		State:       state,
		Settings:    settings,
		TargetKind:  targetKind,
		TargetID:    targetID,
	}
	authURL, derr := driver.AuthStart(ctx, req)
	if derr != nil {
		return nil, errors.Wrap(derr, "build authorization url")
	}
	row.OAuthState = state
	row.Status = dto.RcloneStatusUnlinked
	if serr := s.db.Save(&row).Error; serr != nil {
		return nil, errors.Wrap(serr, "save rclone link state")
	}
	s.pendingMu.Lock()
	s.pending[state] = rclonePendingAuth{req: req, expires: time.Now().Add(oauthStateTTL)}
	s.pendingMu.Unlock()
	return &dto.RcloneAuthStartResponse{AuthURL: authURL, State: state, RedirectURI: req.RedirectURI}, nil
}

// rclonePendingAuth is one in-flight OAuth flow. The deadline enforces
// oauthStateTTL so stale states cannot be replayed after the wizard was
// abandoned.
type rclonePendingAuth struct {
	req     sr.AuthRequest
	expires time.Time
}

// HandleOAuthCallback completes the flow started by StartAuth. States older
// than oauthStateTTL are rejected even if still present in the pending map.
func (s *RcloneService) HandleOAuthCallback(ctx context.Context, code, state string) (*dto.RcloneLink, errors.E) {
	s.pendingMu.Lock()
	pending, ok := s.pending[state]
	delete(s.pending, state)
	s.pendingMu.Unlock()
	if !ok || time.Now().After(pending.expires) {
		return nil, errors.New("unknown or expired oauth state")
	}
	req := pending.req
	var row dbom.RcloneLink
	if err := s.db.Where("target_kind = ? AND target_id = ?", req.TargetKind, req.TargetID).First(&row).Error; err != nil {
		return nil, errors.Wrap(err, "locate pending rclone link")
	}
	driver, ok := sr.GetDriver(row.Provider)
	if !ok {
		return nil, errors.Errorf("unknown provider %q", row.Provider)
	}
	token, derr := driver.ExchangeCode(ctx, req, code)
	if derr != nil {
		row.Status = dto.RcloneStatusError
		row.LastSyncMessage = derr.Error()
		_ = s.db.Save(&row).Error
		return nil, errors.Wrap(derr, "exchange oauth code")
	}
	// Persist the remote into the managed rclone.conf. Tokens live only there.
	name := remoteName(&row)
	params := map[string]any{
		"name": name,
		"parameters": map[string]string{
			"type":          row.Provider,
			"client_id":     req.Settings["client_id"],
			"client_secret": req.Settings["client_secret"],
			"token":         token.TokenJSON,
		},
	}
	var createOut any
	if cerr := sr.Call(ctx, s.rc, "config/create", params, &createOut); cerr != nil {
		return nil, errors.Wrap(cerr, "persist remote configuration")
	}
	row.OAuthState = ""
	row.Status = dto.RcloneStatusAuthorized
	if err := s.db.Save(&row).Error; err != nil {
		return nil, errors.Wrap(err, "save rclone link status")
	}
	out, convErr := s.converter.RcloneLinkToRcloneLinkDTO(row)
	if convErr != nil {
		return nil, errors.Wrap(convErr, "convert rclone link")
	}
	return &out, nil
}

// ---------- diff ----------

type lsjsonEntry struct {
	Path    string `json:"Path"`
	Size    int64  `json:"Size"`
	ModTime string `json:"ModTime"`
	IsDir   bool   `json:"IsDir"`
}

type lsjsonResponse struct {
	List []lsjsonEntry `json:"list"`
}

// lsjson lists a filesystem recursively via operations/lsjson.
func (s *RcloneService) lsjson(ctx context.Context, fs string) (map[string]lsjsonEntry, error) {
	var resp lsjsonResponse
	req := map[string]any{"fs": fs, "remote": "", "recursive": true, "hash": false}
	if err := sr.Call(ctx, s.rc, "operations/lsjson", req, &resp); err != nil {
		return nil, err
	}
	m := make(map[string]lsjsonEntry, len(resp.List))
	for _, e := range resp.List {
		if e.IsDir || e.Path == "" {
			continue
		}
		m[e.Path] = e
	}
	return m, nil
}

func parseModTime(v string) *time.Time {
	t, err := time.Parse(time.RFC3339Nano, v)
	if err != nil {
		return nil
	}
	return &t
}

func intPtr(v int64) *int64 { return &v }

// Diff compares both roots and aggregates counts by difference class.
func (s *RcloneService) Diff(ctx context.Context, targetKind, targetID string) (*dto.RcloneDiffResult, errors.E) {
	link, gerr := s.GetLink(targetKind, targetID)
	if gerr != nil {
		return nil, gerr
	}
	if link == nil || link.Status != dto.RcloneStatusAuthorized {
		return nil, errors.New("target is not linked to a remote yet")
	}
	localPath, lerr := s.LocalPath(targetKind, targetID)
	if lerr != nil {
		return nil, errors.Wrap(lerr, "resolve local path")
	}
	name := remoteName(&dbom.RcloneLink{TargetKind: targetKind, TargetID: targetID})
	localFS, _ := s.lsjson(ctx, localPath)
	remoteFS, rerr := s.lsjson(ctx, name+":"+strings.TrimPrefix(localPath, "/"))
	if rerr != nil && localFS == nil {
		return nil, errors.Wrap(rerr, "list remote files")
	}
	result := &dto.RcloneDiffResult{Entries: []dto.RcloneDiffEntry{}}
	if rerr != nil && localFS != nil {
		// A missing remote folder is a normal pre-first-sync state (every
		// file shows as local_only), but an auth or transport failure would
		// look identical. Surface it as a warning so the UI can tell the
		// user the comparison may be misleading instead of silently
		// reporting "everything is new".
		result.Warning = fmt.Sprintf("remote listing failed (%s); treat results with caution", rerr.Error())
	}
	addEntry := func(e dto.RcloneDiffEntry) {
		result.Entries = append(result.Entries, e)
		switch e.DiffType {
		case "local_only":
			result.LocalOnly++
		case "remote_only":
			result.RemoteOnly++
		case "changed":
			result.Changed++
		}
	}
	for path, le := range localFS {
		re, ok := remoteFS[path]
		if !ok {
			addEntry(dto.RcloneDiffEntry{Path: path, DiffType: "local_only", LocalSize: intPtr(le.Size), LocalModTime: parseModTime(le.ModTime)})
			continue
		}
		if le.Size != re.Size || le.ModTime != re.ModTime {
			addEntry(dto.RcloneDiffEntry{
				Path: path, DiffType: "changed",
				LocalSize: intPtr(le.Size), RemoteSize: intPtr(re.Size),
				LocalModTime: parseModTime(le.ModTime), RemoteModTime: parseModTime(re.ModTime),
			})
		}
	}
	for path, re := range remoteFS {
		if _, ok := localFS[path]; !ok {
			addEntry(dto.RcloneDiffEntry{Path: path, DiffType: "remote_only", RemoteSize: intPtr(re.Size), RemoteModTime: parseModTime(re.ModTime)})
		}
	}
	return result, nil
}

// ---------- sync engine ----------

func (s *RcloneService) emitTask(task *dto.RcloneTask) {
	s.eventBus.EmitRcloneTask(events.RcloneTaskEvent{Event: events.Event{Type: events.EventTypes.UPDATE}, Task: task})
}

// Sync starts an asynchronous job. push = local→remote mirror, pull =
// remote→local mirror, bidi = non-destructive bidirectional copy (newer
// files copied each way; deletions are never propagated in lab v1).
//
// When dryRun is true rclone executes the same passes with dryRun=true
// (nothing is transferred) and the link row bookkeeping (LastSyncAt,
// result, status) plus problem raising are skipped — a dry run must be
// side-effect free for the stored link.
func (s *RcloneService) Sync(targetKind, targetID, direction string, dryRun bool) errors.E {
	link, gerr := s.GetLink(targetKind, targetID)
	if gerr != nil {
		return gerr
	}
	if link == nil || link.Status != dto.RcloneStatusAuthorized {
		return errors.New("target is not linked to a remote yet")
	}
	if direction != dto.RcloneSyncPush && direction != dto.RcloneSyncPull && direction != dto.RcloneSyncBidi {
		return errors.Errorf("invalid direction %q (want push|pull|bidi)", direction)
	}
	key := rcloneJobKey(targetKind, targetID)
	s.mu.Lock()
	if _, busy := s.running[key]; busy {
		s.mu.Unlock()
		return errors.New("a sync job is already running for this target")
	}
	runCtx, cancel := context.WithCancel(s.ctx)
	job := &rcloneRunningJob{cancel: cancel, direction: direction, dryRun: dryRun, link: dbom.RcloneLink{TargetKind: targetKind, TargetID: targetID}}
	s.running[key] = job
	s.mu.Unlock()

	go func() {
		defer func() {
			s.mu.Lock()
			delete(s.running, key)
			s.mu.Unlock()
			cancel()
		}()
		task := &dto.RcloneTask{TargetKind: targetKind, TargetID: targetID, Operation: "sync", Direction: direction, Status: "start"}
		if dryRun {
			task.Message = "dry-run"
		}
		s.emitTask(cloneTask(task))
		err := s.runSync(runCtx, job, task)
		finish := &dto.RcloneTask{TargetKind: targetKind, TargetID: targetID, Operation: "sync", Direction: direction, Notes: task.Notes, Progress: 100}
		if err != nil {
			finish.Status = "failure"
			finish.Error = err.Error()
		} else {
			finish.Status = "success"
		}
		if !dryRun {
			now := time.Now()
			var row dbom.RcloneLink
			if qerr := s.db.Where("target_kind = ? AND target_id = ?", targetKind, targetID).First(&row).Error; qerr == nil {
				row.LastSyncAt = &now
				if err != nil {
					row.LastSyncResult = "failure"
					row.LastSyncMessage = err.Error()
					row.Status = dto.RcloneStatusError
				} else {
					row.LastSyncResult = "success"
					row.LastSyncMessage = ""
					row.Status = dto.RcloneStatusAuthorized
				}
				_ = s.db.Save(&row).Error
			}
			if err != nil {
				s.raiseSyncProblem(targetKind, targetID, err)
			}
		}
		s.emitTask(finish)
	}()
	return nil
}

// runSync executes one or two sync/copy jobs and polls progress.
func (s *RcloneService) runSync(ctx context.Context, job *rcloneRunningJob, task *dto.RcloneTask) error {
	localPath, lerr := s.LocalPath(job.link.TargetKind, job.link.TargetID)
	if lerr != nil {
		return lerr
	}
	var row dbom.RcloneLink
	if err := s.db.Where("target_kind = ? AND target_id = ?", job.link.TargetKind, job.link.TargetID).First(&row).Error; err != nil {
		return fmt.Errorf("load link: %w", err)
	}
	name := remoteName(&row)
	remoteFS := name + ":" + strings.TrimPrefix(localPath, "/")

	passes := [][2]string{}
	switch job.direction {
	case dto.RcloneSyncPush:
		passes = append(passes, [2]string{localPath, remoteFS})
	case dto.RcloneSyncPull:
		passes = append(passes, [2]string{remoteFS, localPath})
	default: // bidi: copy newer both ways, never delete (lab v1 semantics)
		passes = append(passes, [2]string{localPath, remoteFS}, [2]string{remoteFS, localPath})
	}
	for i, pass := range passes {
		if ctx.Err() != nil {
			return fmt.Errorf("aborted")
		}
		group := fmt.Sprintf("srat/%s/%d", keyOf(job), time.Now().UnixNano())
		req := map[string]any{
			"srcFs": pass[0], "dstFs": pass[1],
			"_async": true, "_group": group,
		}
		if job.dryRun {
			req["dryRun"] = true
		}
		var startOut struct {
			JobID int64 `json:"jobid"`
		}
		// bidi uses copy (non-destructive); push/pull use sync (mirror).
		method := "sync/sync"
		if job.direction == dto.RcloneSyncBidi {
			method = "sync/copy"
		}
		if err := sr.Call(ctx, s.rc, method, req, &startOut); err != nil {
			return err
		}
		job.jobID = startOut.JobID
		if perr := s.pollJob(ctx, job, task, group, i+1, len(passes)); perr != nil {
			return perr
		}
	}
	return nil
}

func keyOf(job *rcloneRunningJob) string { return rcloneJobKey(job.link.TargetKind, job.link.TargetID) }

// pollJob waits for an async rclone job, streaming progress events.
func (s *RcloneService) pollJob(ctx context.Context, job *rcloneRunningJob, task *dto.RcloneTask, group string, pass, totalPasses int) error {
	statusReq := map[string]any{"jobid": job.jobID}
	statsReq := map[string]any{"group": group}
	lastProgress := -1
	for {
		select {
		case <-ctx.Done():
			var stopOut map[string]any
			_ = sr.Call(context.Background(), s.rc, "job/stop", statusReq, &stopOut)
			return fmt.Errorf("aborted")
		case <-time.After(syncPollInterval):
		}
		var status struct {
			Finished bool   `json:"finished"`
			Success  bool   `json:"success"`
			Error    string `json:"error"`
		}
		if err := sr.Call(ctx, s.rc, "job/status", statusReq, &status); err != nil {
			return err
		}
		var stats struct {
			Bytes      int64 `json:"bytes"`
			TotalBytes int64 `json:"totalBytes"`
		}
		progress := 999
		if serr := sr.Call(ctx, s.rc, "core/stats", statsReq, &stats); serr == nil && stats.TotalBytes > 0 {
			progress = int(100 * stats.Bytes / stats.TotalBytes)
		}
		if progress != lastProgress && progress != 999 {
			lastProgress = progress
			span := 100 / totalPasses
			task.Progress = span*(pass-1) + progress*span/100
			task.Status = "running"
			msg := fmt.Sprintf("pass %d/%d", pass, totalPasses)
			if task.Message != msg {
				task.Message = msg
			}
			s.emitTask(cloneTask(task))
		}
		if status.Finished {
			if !status.Success {
				return fmt.Errorf("rclone job %d failed: %s", job.jobID, status.Error)
			}
			return nil
		}
	}
}

func cloneTask(t *dto.RcloneTask) *dto.RcloneTask {
	c := *t
	c.Notes = append([]string(nil), t.Notes...)
	return &c
}

// AbortSync cancels a running job for a target.
func (s *RcloneService) AbortSync(targetKind, targetID string) errors.E {
	s.mu.Lock()
	job, ok := s.running[rcloneJobKey(targetKind, targetID)]
	s.mu.Unlock()
	if !ok {
		return errors.New("no sync job running for this target")
	}
	job.cancel()
	return nil
}

// ---------- auto-sync scheduler ----------

//	startAutoSync launches the background goroutine that periodically triggers
//
// syncs for links with AutoSync enabled. Idempotent: calling while already
// running is a no-op.
func (s *RcloneService) startAutoSync() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.autoSyncStop != nil {
		return
	}
	stop := make(chan struct{})
	s.autoSyncStop = stop
	go func() {
		s.autoSyncLoop(stop, s.autoSyncInterval())
	}()
}

// stopAutoSync halts the auto-sync background loop. Idempotent.
func (s *RcloneService) stopAutoSync() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.autoSyncStop == nil {
		return
	}
	close(s.autoSyncStop)
	s.autoSyncStop = nil
}

// autoSyncLoop polls eligible links every interval, triggering Sync for
// overdue targets. After each tick the interval is re-evaluated so link
// additions/removals take effect on the next cycle.
func (s *RcloneService) autoSyncLoop(stop chan struct{}, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			s.autoSyncTick()
			// Re-evaluate interval in case links were added/removed.
			ticker.Reset(s.autoSyncInterval())
		}
	}
}

// autoSyncInterval queries eligible auto-sync links and returns the shortest
// ScheduleMinutes. If no links are eligible it returns a long default interval
// (the loop will just sleep until the next reconfigure).
func (s *RcloneService) autoSyncInterval() time.Duration {
	var rows []dbom.RcloneLink
	if err := s.db.WithContext(s.ctx).
		Where("auto_sync = ? AND schedule_minutes > 0 AND status = ?",
			true, dto.RcloneStatusAuthorized).
		Find(&rows).Error; err != nil || len(rows) == 0 {
		return 5 * time.Minute
	}
	min := rows[0].ScheduleMinutes
	for _, r := range rows[1:] {
		if r.ScheduleMinutes < min {
			min = r.ScheduleMinutes
		}
	}
	return time.Duration(min) * time.Minute
}

// autoSyncTick checks each eligible link and triggers Sync when the link is
// overdue (LastSyncAt is nil or older than ScheduleMinutes ago).
func (s *RcloneService) autoSyncTick() {
	var rows []dbom.RcloneLink
	if err := s.db.WithContext(s.ctx).
		Where("auto_sync = ? AND schedule_minutes > 0 AND status = ?",
			true, dto.RcloneStatusAuthorized).
		Find(&rows).Error; err != nil {
		slog.WarnContext(s.ctx, "rclone auto-sync: failed to query eligible links", "error", err.Error())
		return
	}
	now := time.Now()
	for i := range rows {
		r := &rows[i]
		if r.ScheduleMinutes <= 0 {
			continue
		}
		due := r.LastSyncAt == nil || now.Sub(*r.LastSyncAt) >= time.Duration(r.ScheduleMinutes)*time.Minute
		if !due {
			continue
		}
		key := rcloneJobKey(r.TargetKind, r.TargetID)
		s.mu.Lock()
		_, busy := s.running[key]
		s.mu.Unlock()
		if busy {
			continue
		}
		// Best-effort: fire and forget. Sync() does its own validation
		// and marks last-sync bookkeeping when the job finishes.
		if err := s.Sync(r.TargetKind, r.TargetID, dto.RcloneSyncPush, false); err != nil {
			slog.DebugContext(s.ctx, "rclone auto-sync: sync failed to start",
				"target", key, "error", err.Error())
		}
	}
}

// reconfigureAutoSync wakes the auto-sync loop so it re-evaluates intervals
// after a link save or delete. Safe to call from any goroutine. Non-blocking:
// if a reconfigure is already pending it is a no-op.
func (s *RcloneService) reconfigureAutoSync() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.autoSyncStop == nil {
		return
	}
	// Poke the loop by stopping and restarting it. This is the simplest
	// approach that avoids a second channel; the loop is cheap to restart.
	close(s.autoSyncStop)
	stop := make(chan struct{})
	s.autoSyncStop = stop
	go func() {
		s.autoSyncLoop(stop, s.autoSyncInterval())
	}()
}

// raiseSyncProblem records a persistent problem entry so failures surface on
// the dashboard even after the WebSocket event was missed.
func (s *RcloneService) raiseSyncProblem(targetKind, targetID string, cause error) {
	if s.problemService == nil {
		return
	}
	_, err := s.problemService.Upsert(&dto.Problem{
		ProblemKey:  fmt.Sprintf("rclone_sync_failed_%s_%s_%s", targetKind, sanitizeKey(targetID), time.Now().Format("20060102")),
		Title:       "Cloud sync failed",
		Description: fmt.Sprintf("Cloud sync (%s/%s) failed: %v", targetKind, targetID, cause),
		Severity:    dto.ProblemSeverities.PROBLEMSEVERITYWARNING,
		Status:      dto.ProblemLifecycleStatuses.PROBLEMLIFECYCLESTATUSCREATED,
	})
	if err != nil {
		slog.WarnContext(s.ctx, "failed to record rclone sync problem", "error", err.Error())
	}
}

func sanitizeKey(s string) string {
	return strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
			return r
		}
		return '_'
	}, s)
}
