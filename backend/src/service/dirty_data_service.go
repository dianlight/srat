package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	"github.com/dianlight/srat/tlog"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
)

type DirtyDataServiceInterface interface {
	GetDirtyDataTracker() dto.DataDirtyTracker
	AddRestartCallback(callback func() errors.E)
	ResetDirtyStatus()
	IsTimerRunning() bool
}

type DirtyDataService struct {
	ctx              context.Context
	dataDirtyTracker dto.DataDirtyTracker
	timer            *time.Timer
	restartCallbacks *[]func() errors.E
}

func NewDirtyDataService(lc fx.Lifecycle, ctx context.Context, eventBus events.EventBusInterface) DirtyDataServiceInterface {
	p := new(DirtyDataService)
	p.ctx = ctx
	p.dataDirtyTracker = dto.DataDirtyTracker{}
	p.restartCallbacks = &[]func() errors.E{}

	unsubscribe := make([]func(), 3)

	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			tlog.Trace("Starting DirtyDataService")
			// Register event bus listeners
			if eventBus != nil {
				unsubscribe[0] = eventBus.OnShare(func(event events.ShareEvent) {
					slog.Debug("DirtyDataService received Share event", "share", event.Share.Name)
					p.setDirtyShares()
				})
				unsubscribe[1] = eventBus.OnUser(func(event events.UserEvent) {
					slog.Debug("DirtyDataService received User event", "user", event.User.Username)
					p.setDirtyUsers()
				})
				unsubscribe[2] = eventBus.OnSetting(func(event events.SettingEvent) {
					slog.Debug("DirtyDataService received Setting event", "setting", event.Setting)
					p.setDirtySettings()
				})
			}
			return nil
		},
		OnStop: func(context.Context) error {
			tlog.Trace("Stopping DirtyDataService")
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
	p.timer = time.AfterFunc(5*time.Second, func() {
		p.dataDirtyTracker = dto.DataDirtyTracker{}
		for _, callback := range *p.restartCallbacks {
			slog.Debug("Calling callback for Restart", "callback", callback)
			err := callback()
			if err != nil {
				slog.Warn("Error in restart callback", "err", err)
			}
		}
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

// add a callback to be called when the timer is triggered
func (p *DirtyDataService) AddRestartCallback(callback func() errors.E) {
	*p.restartCallbacks = append(*p.restartCallbacks, callback)
}

// Reset all dirty status to false
func (p *DirtyDataService) ResetDirtyStatus() {
	p.dataDirtyTracker = dto.DataDirtyTracker{}
	p.stopTimer()
}

// check if timer is running
func (p *DirtyDataService) IsTimerRunning() bool {
	return p.timer != nil
}
