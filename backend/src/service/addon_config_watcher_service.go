package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/dianlight/srat/config"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	"github.com/dianlight/srat/homeassistant/apps"
	"github.com/dianlight/srat/homeassistant/websocket"
	"github.com/dianlight/srat/internal/ctxkeys"
	"github.com/dianlight/tlog"
	"github.com/fsnotify/fsnotify"
	"go.uber.org/fx"
)

// AddonConfigWatcherServiceInterface is a nominal marker type used by the FX dependency injection container.
// It has no methods; callers never interact with it directly — FX resolves the concrete *AddonConfigWatcherService
// via fx.Invoke so the service lifecycle hooks are registered at startup.
type AddonConfigWatcherServiceInterface interface{}

// AddonConfigWatcherService watches for externally-initiated changes to the addon options file.
//
// Detection priority (all three paths run concurrently; hash dedup prevents duplicate notifications):
//  1. HA Supervisor WebSocket event (event_type == "supervisor_event", data.event == "addon_config_changed")
//  2. fsnotify watcher on optionsFilePath with 500 ms debounce
//  3. Interval ticker (default 60 s) as a safety net for NFS / overlay-FS environments
//
// onChanged is invoked at most once per unique SHA-256 content hash.
// It emits an AppConfigEvent on the event bus and upserts a Problem (or persistent notification fallback).
type AddonConfigWatcherService struct {
	ctx             context.Context
	addonsClient    addonInfoClient
	wsClient        websocket.ClientInterface
	eventBus        events.EventBusInterface
	problemService  ProblemServiceInterface
	haService       HomeAssistantServiceInterface
	watchCtx        context.Context
	watchCancel     context.CancelFunc
	hashMu          sync.Mutex
	lastHash        string
	pollInterval    time.Duration
	optionsFilePath string // defaults to config.AddonOptionsFilePath; overridable in tests
	newFsnotify     func() (fsnotifyWatcher, error)
	debounceAfter   func(time.Duration, func()) timerStopper
	debounceDelay   time.Duration
	retryDelay      time.Duration
	// onChanged is called once per unique options-file hash. Defaults to emitChanged.
	onChanged func(path, hash string)
}

type timerStopper interface {
	Stop() bool
}

type fsnotifyWatcher interface {
	Add(name string) error
	Close() error
	Events() <-chan fsnotify.Event
	Errors() <-chan error
}

type realFsnotifyWatcher struct {
	*fsnotify.Watcher
}

func (w *realFsnotifyWatcher) Events() <-chan fsnotify.Event { return w.Watcher.Events }

func (w *realFsnotifyWatcher) Errors() <-chan error { return w.Watcher.Errors }

// AddonConfigWatcherServiceParams holds all FX-injected dependencies.
type AddonConfigWatcherServiceParams struct {
	fx.In
	Ctx            context.Context
	AddonsClient   apps.ClientWithResponsesInterface `optional:"true"`
	EventBus       events.EventBusInterface
	ProblemService ProblemServiceInterface       `optional:"true"`
	HAService      HomeAssistantServiceInterface `optional:"true"`
	WsClient       websocket.ClientInterface     `optional:"true"`
}

type addonInfoClient interface {
	GetAppInfoWithResponse(ctx context.Context, addon string, reqEditors ...apps.RequestEditorFn) (*apps.GetAppInfoResponse, error)
}

const supervisorOptionsSource = "supervisor:addon_options"

// NewAddonConfigWatcherService creates the service and registers FX lifecycle hooks.
func NewAddonConfigWatcherService(lc fx.Lifecycle, params AddonConfigWatcherServiceParams) AddonConfigWatcherServiceInterface {
	s := &AddonConfigWatcherService{
		ctx:             params.Ctx,
		addonsClient:    params.AddonsClient,
		wsClient:        params.WsClient,
		eventBus:        params.EventBus,
		problemService:  params.ProblemService,
		haService:       params.HAService,
		pollInterval:    60 * time.Second,
		optionsFilePath: config.AddonOptionsFilePath,
		newFsnotify: func() (fsnotifyWatcher, error) {
			watcher, err := fsnotify.NewWatcher()
			if err != nil {
				return nil, err
			}
			return &realFsnotifyWatcher{Watcher: watcher}, nil
		},
		debounceAfter: func(d time.Duration, f func()) timerStopper {
			return time.AfterFunc(d, f)
		},
		debounceDelay: 500 * time.Millisecond,
		retryDelay:    5 * time.Second,
	}
	s.onChanged = s.emitChanged
	s.watchCtx, s.watchCancel = context.WithCancel(params.Ctx)

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			// Seed initial hash so the first comparison starts from a known baseline.
			if hash, err := s.hashObservedConfig(); err == nil {
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
				"hash_source", s.observedConfigPath(),
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
	Event     string `json:"event"`
	EventType string `json:"event_type"`
	Slug      string `json:"slug"`
	Addon     string `json:"addon"`
	Data      struct {
		Event     string `json:"event"`
		EventType string `json:"event_type"`
		Slug      string `json:"slug"`
		Addon     string `json:"addon"`
	} `json:"data"`
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func parseSupervisorAddonConfigChanged(raw json.RawMessage) (event string, slug string, ok bool, err error) {
	var ev supervisorEventData
	if err := json.Unmarshal(raw, &ev); err != nil {
		return "", "", false, err
	}

	event = firstNonEmpty(ev.Data.Event, ev.Data.EventType, ev.Event, ev.EventType)
	slug = firstNonEmpty(ev.Data.Slug, ev.Data.Addon, ev.Slug, ev.Addon)

	return event, slug, event == "addon_config_changed", nil
}

func shouldWarnSupervisorSubscriptionFailure(err error, attempt int) bool {
	if err == nil {
		return false
	}

	if attempt <= 1 && strings.Contains(strings.ToLower(err.Error()), "not connected") {
		return false
	}

	return true
}

// watchViaSupervisorEvents subscribes to "supervisor_event" on the HA Core WebSocket
// (proxied through ws://supervisor/core/websocket).
// It filters for events of type "addon_config_changed" and delegates to maybeNotify.
// If the subscription cannot be established the method returns silently; fsnotify covers detection.
func (s *AddonConfigWatcherService) watchViaSupervisorEvents() {
	retryDelay := s.retryDelay
	if retryDelay <= 0 {
		retryDelay = 5 * time.Second
	}

	attempt := 0
	for {
		attempt++
		unsub, err := s.wsClient.SubscribeEvents(s.watchCtx, "supervisor_event", func(msg json.RawMessage) {
			eventName, slug, isConfigChanged, err := parseSupervisorAddonConfigChanged(msg)
			if err != nil {
				slog.WarnContext(s.ctx, "addon_config_watcher: failed to decode supervisor event payload", "err", err, "payload", string(msg))
				return
			}

			slog.DebugContext(s.ctx, "addon_config_watcher: supervisor event received", "event", eventName, "slug", slug, "payload", string(msg))

			if !isConfigChanged {
				return
			}

			slog.InfoContext(s.ctx, "addon_config_watcher: supervisor addon_config_changed event matched", "slug", slug)
			hash, err := s.hashObservedConfig()
			if err != nil {
				slog.WarnContext(s.ctx, "addon_config_watcher: cannot hash observed config after supervisor event", "source", s.observedConfigPath(), "err", err)
				return
			}
			s.maybeNotify(s.observedConfigPath(), hash)
		})
		if err == nil {
			slog.InfoContext(s.ctx, "addon_config_watcher: supervisor_event subscription established", "attempt", attempt)
			<-s.watchCtx.Done()
			_ = unsub()
			return
		}

		if shouldWarnSupervisorSubscriptionFailure(err, attempt) {
			slog.WarnContext(s.ctx, "addon_config_watcher: supervisor_event subscription failed; retrying", "err", err, "retry_delay", retryDelay, "attempt", attempt)
		} else {
			slog.InfoContext(s.ctx, "addon_config_watcher: supervisor_event subscription not ready yet; retrying", "err", err, "retry_delay", retryDelay, "attempt", attempt)
		}

		select {
		case <-time.After(retryDelay):
		case <-s.watchCtx.Done():
			return
		}
	}
}

// watchViaFsnotify watches the options file using fsnotify with a 500 ms debounce.
// It adds the file (not the directory) to the watcher and reacts to Write / Create / Rename events.
func (s *AddonConfigWatcherService) watchViaFsnotify() {
	newWatcher := s.newFsnotify
	if newWatcher == nil {
		newWatcher = func() (fsnotifyWatcher, error) {
			watcher, err := fsnotify.NewWatcher()
			if err != nil {
				return nil, err
			}
			return &realFsnotifyWatcher{Watcher: watcher}, nil
		}
	}

	watcher, err := newWatcher()
	if err != nil {
		slog.WarnContext(s.ctx, "addon_config_watcher: cannot create fsnotify watcher", "err", err)
		return
	}
	defer watcher.Close()

	if err := watcher.Add(s.optionsFilePath); err != nil {
		slog.WarnContext(s.ctx, "addon_config_watcher: cannot watch options file (may not exist yet)", "path", s.optionsFilePath, "err", err)
		return
	}

	debounceDelay := s.debounceDelay
	if debounceDelay <= 0 {
		debounceDelay = 500 * time.Millisecond
	}
	debounceAfter := s.debounceAfter
	if debounceAfter == nil {
		debounceAfter = func(d time.Duration, f func()) timerStopper {
			return time.AfterFunc(d, f)
		}
	}
	var debounceTimer timerStopper

	for {
		select {
		case <-s.watchCtx.Done():
			return
		case event, ok := <-watcher.Events():
			if !ok {
				return
			}
			if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename) == 0 {
				continue
			}
			if debounceTimer != nil {
				debounceTimer.Stop()
			}
			debounceTimer = debounceAfter(debounceDelay, func() {
				hash, err := s.hashObservedConfig()
				if err != nil {
					slog.WarnContext(s.ctx, "addon_config_watcher: cannot hash observed config after fsnotify event", "source", s.observedConfigPath(), "err", err)
					return
				}
				s.maybeNotify(s.observedConfigPath(), hash)
			})
		case err, ok := <-watcher.Errors():
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

	var lastMtime time.Time

	for {
		select {
		case <-s.watchCtx.Done():
			return
		case <-ticker.C:
			// fast path: if we are watching a file locally, only read the file if mtime changed.
			if s.addonsClient == nil {
				if st, err := os.Stat(s.optionsFilePath); err == nil {
					currMtime := st.ModTime()
					if !lastMtime.IsZero() && currMtime.Equal(lastMtime) {
						// File hasn't changed, skip full read and hash
						continue
					}
					lastMtime = currMtime
				}
			}

			hash, err := s.hashObservedConfig()
			if err != nil {
				// Missing observed config is expected in dev / test environments without Supervisor.
				continue
			}
			s.maybeNotify(s.observedConfigPath(), hash)
		}
	}
}

func (s *AddonConfigWatcherService) observedConfigPath() string {
	if s.addonsClient != nil {
		return supervisorOptionsSource
	}

	return s.optionsFilePath
}

func (s *AddonConfigWatcherService) hashObservedConfig() (string, error) {
	if s.addonsClient != nil {
		return s.hashSupervisorOptions()
	}

	return s.hashFile(s.optionsFilePath)
}

func (s *AddonConfigWatcherService) hashSupervisorOptions() (string, error) {
	infoResp, err := s.addonsClient.GetAppInfoWithResponse(s.watchCtx, "self")
	if err != nil {
		return "", err
	}
	if infoResp.StatusCode() != 200 {
		return "", errors.New("unexpected supervisor addon info status")
	}
	if infoResp.JSON200 == nil {
		return "", errors.New("missing supervisor addon info payload")
	}

	options := map[string]any{}
	if infoResp.JSON200.Data.Options != nil {
		options = *infoResp.JSON200.Data.Options
	}

	payload, err := json.Marshal(options)
	if err != nil {
		return "", err
	}

	h := sha256.Sum256(payload)
	return hex.EncodeToString(h[:]), nil
}

// maybeNotify compares the new hash against the last known hash.
// It updates lastHash and invokes onChanged only when the hash has changed.
// Thread-safe via hashMu.
func (s *AddonConfigWatcherService) maybeNotify(path, hash string) {
	s.hashMu.Lock()
	defer s.hashMu.Unlock()
	if hash == s.lastHash {
		tlog.TraceContext(s.ctx, "addon_config_watcher: unchanged options hash; skipping notification", "path", path, "hash", hash)
		return
	}
	s.lastHash = hash
	tlog.DebugContext(s.ctx, "addon_config_watcher: options hash changed", "path", path, "hash", hash)
	s.onChanged(path, hash)
}

// emitChanged is the default onChanged handler.
// It logs the detection, emits an AppConfigEvent on the event bus, and upserts a Problem
// (or falls back to a HA persistent notification when ProblemService is unavailable).
func (s *AddonConfigWatcherService) emitChanged(path, hash string) {
	tlog.InfoContext(s.ctx, "addon_config_watcher: external addon config change detected",
		"path", path, "hash", hash)

	if s.eventBus != nil {
		s.eventBus.EmitAppConfig(events.AppConfigEvent{
			Event: events.Event{Type: events.EventTypes.UPDATE},
			Path:  path,
			Hash:  hash,
		})
	}

	const repairID = "addon_config_changed"

	if s.problemService != nil {
		_, err := s.problemService.Upsert(&dto.Problem{
			ProblemKey:     repairID,
			Title:          "Addon configuration changed externally",
			Description:    "The addon options file was modified outside of SRAT. Reload the configuration to apply the new settings.",
			Severity:       dto.ProblemSeverities.PROBLEMSEVERITYWARNING,
			Status:         dto.ProblemLifecycleStatuses.PROBLEMLIFECYCLESTATUSCREATED,
			TranslationKey: repairID,
			IsFixable:      false,
			IsPersistent:   true,
		})
		if err != nil {
			tlog.WarnContext(s.ctx, "addon_config_watcher: could not upsert addon-config problem", "err", err)
			return
		}
		return
	}

	// Fallback: create a HA persistent notification when ProblemService is not available.
	if s.haService != nil {
		err := s.haService.CreatePersistentNotification(
			repairID,
			"Addon configuration changed externally",
			"The addon options file was modified outside of SRAT. Reload the configuration to apply the new settings.",
		)
		if err != nil {
			tlog.WarnContext(s.ctx, "addon_config_watcher: could not create persistent notification", "err", err)
		}
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
