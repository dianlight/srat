package service

import (
	"context"
	"time"

	"github.com/dianlight/srat/dto"
)

type DirtyDataServiceInterface interface {
	SetDirtyShares()
	SetDirtyVolumes()
	SetDirtyUsers()
	SetDirtySettings()
	GetDirtyDataTracker() dto.DataDirtyTracker
	AddRestartCallback(callback func())
	ResetDirtyStatus()
	IsTimerRunning() bool
}

type DirtyDataService struct {
	ctx              context.Context
	dataDirtyTracker dto.DataDirtyTracker
	timer            *time.Timer
	restartCallbacks *[]func()
}

func NewDirtyDataService(ctx context.Context) DirtyDataServiceInterface {
	p := new(DirtyDataService)
	p.ctx = ctx
	p.dataDirtyTracker = dto.DataDirtyTracker{}
	p.restartCallbacks = &[]func(){}
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
			callback()
		}
	})
}

// stop the timer
func (p *DirtyDataService) stopTimer() {
	if p.timer != nil {
		p.timer.Stop()
		p.timer = nil
	}
}

func (p *DirtyDataService) SetDirtyShares() {
	p.dataDirtyTracker.Shares = true
	p.startTimer()
}

func (p *DirtyDataService) SetDirtyVolumes() {
	p.dataDirtyTracker.Volumes = true
	p.startTimer()
}

func (p *DirtyDataService) SetDirtyUsers() {
	p.dataDirtyTracker.Users = true
	p.startTimer()
}

func (p *DirtyDataService) SetDirtySettings() {
	p.dataDirtyTracker.Settings = true
	p.startTimer()
}

// GetDirtyDataTracker returns the dirty data tracker
func (p *DirtyDataService) GetDirtyDataTracker() dto.DataDirtyTracker {
	return p.dataDirtyTracker
}

// add a callback to be called when the timer is triggered
func (p *DirtyDataService) AddRestartCallback(callback func()) {
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
