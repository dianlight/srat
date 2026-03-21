package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/dianlight/srat/config"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	"github.com/dianlight/srat/homeassistant/websocket"
	"github.com/dianlight/srat/internal/ctxkeys"
	"github.com/fsnotify/fsnotify"
	"go.uber.org/fx"
)

// AddonConfigWatcherServiceInterface is the public interface for the addon config watcher service.
// Lifecycle is managed via FX hooks; no additional public methods are required for this task.
type AddonConfigWatcherServiceInterface interface{}

// AddonConfigWatcherService watches for externally-initiated changes to the addon options file.
//
// Detection priority (all three paths run concurrently; hash dedup prevents duplicate notifications):
//  1. HA Supervisor WebSocket event (event_type == "supervisor_event", data.event == "addon_config_changed")
//  2. fsnotify watcher on optionsFilePath with 500 ms debounce
//  3. Interval ticker (default 60 s) as a safety net for NFS / overlay-FS environments
//
// onChanged is invoked at most once per unique SHA-256 content hash.
// It emits an AppConfigEvent on the event bus with the detected path and hash.
type AddonConfigWatcherService struct {
	ctx             context.Context
	wsClient        websocket.ClientInterface
	state           *dto.ContextState
	eventBus        events.EventBusInterface
	watchCtx        context.Context
	watchCancel     context.CancelFunc
	hashMu          sync.Mutex
	lastHash        string
	pollInterval    time.Duration
	optionsFilePath string // defaults to config.AddonOptionsFilePath; overridable in tests
	// onChanged is called once per unique options-file hash. Defaults to emitChanged.
	onChanged func(path, hash string)
}

// AddonConfigWatcherServiceParams holds all FX-injected dependencies.
type AddonConfigWatcherServiceParams struct {
	fx.In
	Ctx      context.Context
	State    *dto.ContextState
	EventBus events.EventBusInterface
	WsClient websocket.ClientInterface `optional:"true"`
}

// NewAddonConfigWatcherService creates the service and registers FX lifecycle hooks.
func NewAddonConfigWatcherService(lc fx.Lifecycle, params AddonConfigWatcherServiceParams) AddonConfigWatcherServiceInterface {
	s := &AddonConfigWatcherService{
		ctx:             params.Ctx,
		wsClient:        params.WsClient,
		state:           params.State,
		eventBus:        params.EventBus,
		pollInterval:    60 * time.Second,
		optionsFilePath: config.AddonOptionsFilePath,
	}
	s.onChanged = s.emitChanged
	s.watchCtx, s.watchCancel = context.WithCancel(params.Ctx)

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			// Seed initial hash so the first comparison starts from a known baseline.
			if hash, err := s.hashFile(s.optionsFilePath); err == nil {
				s.lastHash = hash
			}

			wg, _ := s.ctx.Value(ctxkeys.WaitGroup).(*sync.WaitGroup)

			// Attempt HA Supervisor event subscription (best-effort; failure is non-fatal).
			if s.wsClient != nil && wg != nil {
				wg.Go(func() { s.watchViaSupervisorEvents() })
			}

			// fsnotify watcher and interval ticker always run as fallback / safety net.
			if wg != nil {
				wg.Go(func() { s.watchViaFsnotify() })
				wg.Go(func() { s.watchViaTicker() })
			}

			slog.InfoContext(s.ctx, "addon_config_watcher: started",
				"supervisor_events", s.wsClient != nil,
				"options_path", s.optionsFilePath,
				"poll_interval", s.pollInterval)
			return nil
		},
		OnStop: func(ctx context.Context) error {
			s.watchCancel()
			return nil
		},
	})

	return s
}

// supervisorEventData is the shape of the data field inside a "supervisor_event" HA WS event.
type supervisorEventData struct {
	Data struct {
		Event string `json:"event"`
		Slug  string `json:"slug"`
	} `json:"data"`
}

// watchViaSupervisorEvents subscribes to "supervisor_event" on the HA Core WebSocket
// (proxied through ws://supervisor/core/websocket).
// It filters for events of type "addon_config_changed" and delegates to maybeNotify.
// If the subscription cannot be established the method returns silently; fsnotify covers detection.
func (s *AddonConfigWatcherService) watchViaSupervisorEvents() {
	unsub, err := s.wsClient.SubscribeEvents(s.watchCtx, "supervisor_event", func(msg json.RawMessage) {
		var ev supervisorEventData
		if err := json.Unmarshal(msg, &ev); err != nil {
			return
		}
		if ev.Data.Event != "addon_config_changed" {
			return
		}
		slog.DebugContext(s.ctx, "addon_config_watcher: supervisor addon_config_changed event received", "slug", ev.Data.Slug)
		hash, err := s.hashFile(s.optionsFilePath)
		if err != nil {
			slog.WarnContext(s.ctx, "addon_config_watcher: cannot hash options file after supervisor event", "err", err)
			return
		}
		s.maybeNotify(s.optionsFilePath, hash)
	})
	if err != nil {
		slog.WarnContext(s.ctx, "addon_config_watcher: supervisor_event subscription failed; fsnotify handles detection", "err", err)
		return
	}
	<-s.watchCtx.Done()
	_ = unsub()
}

// watchViaFsnotify watches the options file using fsnotify with a 500 ms debounce.
// It adds the file (not the directory) to the watcher and reacts to Write / Create / Rename events.
func (s *AddonConfigWatcherService) watchViaFsnotify() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		slog.WarnContext(s.ctx, "addon_config_watcher: cannot create fsnotify watcher", "err", err)
		return
	}
	defer watcher.Close()

	if err := watcher.Add(s.optionsFilePath); err != nil {
		slog.WarnContext(s.ctx, "addon_config_watcher: cannot watch options file (may not exist yet)", "path", s.optionsFilePath, "err", err)
		return
	}

	const debounceDelay = 500 * time.Millisecond
	var debounceTimer *time.Timer

	for {
		select {
		case <-s.watchCtx.Done():
			return
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename) == 0 {
				continue
			}
			if debounceTimer != nil {
				debounceTimer.Stop()
			}
			debounceTimer = time.AfterFunc(debounceDelay, func() {
				hash, err := s.hashFile(s.optionsFilePath)
				if err != nil {
					slog.WarnContext(s.ctx, "addon_config_watcher: cannot hash options file after fsnotify event", "err", err)
					return
				}
				s.maybeNotify(s.optionsFilePath, hash)
			})
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			slog.WarnContext(s.ctx, "addon_config_watcher: fsnotify error", "err", err)
		}
	}
}

// watchViaTicker polls the options file at pollInterval.
// This is a safety net for NFS / overlay-FS environments where inotify events may not fire.
func (s *AddonConfigWatcherService) watchViaTicker() {
	ticker := time.NewTicker(s.pollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-s.watchCtx.Done():
			return
		case <-ticker.C:
			hash, err := s.hashFile(s.optionsFilePath)
			if err != nil {
				// File absent is expected in dev / test environments without Supervisor.
				continue
			}
			s.maybeNotify(s.optionsFilePath, hash)
		}
	}
}

// maybeNotify compares the new hash against the last known hash.
// It updates lastHash and invokes onChanged only when the hash has changed.
// Thread-safe via hashMu.
func (s *AddonConfigWatcherService) maybeNotify(path, hash string) {
	s.hashMu.Lock()
	defer s.hashMu.Unlock()
	if hash == s.lastHash {
		return
	}
	s.lastHash = hash
	s.onChanged(path, hash)
}

// emitChanged is the default onChanged handler.
// It logs the detection and emits an AppConfigEvent on the event bus so that
// DirtyDataService, BroadcasterService, and other listeners are notified.
func (s *AddonConfigWatcherService) emitChanged(path, hash string) {
	slog.InfoContext(s.ctx, "addon_config_watcher: external addon config change detected",
		"path", path, "hash", hash)
	if s.eventBus != nil {
		s.eventBus.EmitAppConfig(events.AppConfigEvent{
			Event: events.Event{Type: events.EventTypes.UPDATE},
			Path:  path,
			Hash:  hash,
		})
	}
}

// hashFile computes the SHA-256 hex digest of the file at path.
func (s *AddonConfigWatcherService) hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
