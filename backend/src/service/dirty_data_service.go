package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	"github.com/dianlight/tlog"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
)

type DirtyDataServiceInterface interface {
	GetDirtyDataTracker() dto.DataDirtyTracker
	IsTimerRunning() bool
}

type DirtyDataService struct {
	ctx              context.Context
	dataDirtyTracker dto.DataDirtyTracker
	timer            *time.Timer
	eventBus         events.EventBusInterface
}

func NewDirtyDataService(lc fx.Lifecycle, ctx context.Context, eventBus events.EventBusInterface) DirtyDataServiceInterface {
	p := new(DirtyDataService)
	p.ctx = ctx
	p.dataDirtyTracker = dto.DataDirtyTracker{}
	p.eventBus = eventBus

	unsubscribe := make([]func(), 4)
	if eventBus != nil {
		unsubscribe[0] = eventBus.OnShare(func(ctx context.Context, event events.ShareEvent) errors.E {
			slog.DebugContext(ctx, "DirtyDataService received Share event", "share", event.Share.Name)
			p.setDirtyShares()
			return nil
		})
		unsubscribe[1] = eventBus.OnUser(func(ctx context.Context, event events.UserEvent) errors.E {
			slog.DebugContext(ctx, "DirtyDataService received User event", "user", event.User.Username)
			p.setDirtyUsers()
			return nil
		})
		unsubscribe[2] = eventBus.OnSetting(func(ctx context.Context, event events.SettingEvent) errors.E {
			slog.DebugContext(ctx, "DirtyDataService received Setting event", "setting", event.Setting)
			p.setDirtySettings()
			return nil
		})
		unsubscribe[3] = eventBus.OnSamba(func(ctx context.Context, event events.SambaEvent) errors.E {
			slog.DebugContext(ctx, "DirtyDataService received Samba event", "tracker", event.DataDirtyTracker)
			if event.Type == events.EventTypes.CLEAN {
				p.resetDirtyStatus()
			}
			return nil
		})
		/*
			unsubscribe[4] = eventBus.OnMountPoint(func(ctx context.Context, mpe events.MountPointEvent) {
				slog.DebugContext(ctx, "DirtyDataService received MountPoint event", "mountpoint", mpe.MountPoint.Path)
				p.setDirtyShares()
			})
		*/
	}

	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			tlog.DebugContext(ctx, "Starting DirtyDataService")
			return nil
		},
		OnStop: func(context.Context) error {
			tlog.DebugContext(ctx, "Stopping DirtyDataService")
			for _, unsub := range unsubscribe {
				unsub()
			}
			return nil
		},
	})

	return p
}

// start or reset timer for 15 seconds
func (p *DirtyDataService) startTimer() {
	if p.timer != nil {
		p.timer.Stop()
	}
	p.eventBus.EmitDirtyData(events.DirtyDataEvent{
		Event:            events.Event{Type: events.EventTypes.ADD},
		DataDirtyTracker: p.dataDirtyTracker,
	})

	p.timer = time.AfterFunc(5*time.Second, func() {
		p.eventBus.EmitDirtyData(events.DirtyDataEvent{
			Event:            events.Event{Type: events.EventTypes.RESTART},
			DataDirtyTracker: p.dataDirtyTracker,
		})
		p.timer = nil
	})
}

// stop the timer
func (p *DirtyDataService) stopTimer() {
	if p.timer != nil {
		p.timer.Stop()
		p.timer = nil
	}
}

func (p *DirtyDataService) setDirtyShares() {
	p.dataDirtyTracker.Shares = true
	p.startTimer()
}

func (p *DirtyDataService) setDirtyUsers() {
	p.dataDirtyTracker.Users = true
	p.startTimer()
}

func (p *DirtyDataService) setDirtySettings() {
	p.dataDirtyTracker.Settings = true
	p.startTimer()
}

// GetDirtyDataTracker returns the dirty data tracker
func (p *DirtyDataService) GetDirtyDataTracker() dto.DataDirtyTracker {
	return p.dataDirtyTracker
}

// Reset all dirty status to false
func (p *DirtyDataService) resetDirtyStatus() {
	p.dataDirtyTracker = dto.DataDirtyTracker{}
	p.stopTimer()
}

// check if timer is running
func (p *DirtyDataService) IsTimerRunning() bool {
	return p.timer != nil
}
