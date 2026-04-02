package dto

import "reflect"

type WebEventMapTypes map[string]any

var WebEventMap = WebEventMapTypes{
	WebEventTypes.EVENTHELLO.String():             Welcome{},
	WebEventTypes.EVENTUPDATING.String():          UpdateProgress{},
	WebEventTypes.EVENTVOLUMES.String():           []*Disk{},
	WebEventTypes.EVENTHEARTBEAT.String():         HealthPing{},
	WebEventTypes.EVENTSHARES.String():            []SharedResource{},
	WebEventTypes.EVENTDIRTYTRACKER.String():      DataDirtyTracker{},
	WebEventTypes.EVENTSMARTTESTSTATUS.String():   SmartTestStatus{},
	WebEventTypes.EVENTERROR.String():             &ErrorDataModel{},
	WebEventTypes.EVENTFILESYSTEMTASK.String():    FilesystemTask{},
	WebEventTypes.EVENTREPAIRCOMMAND.String():     RepairCommandMessage{},
	WebEventTypes.EVENTAPPCONFIGCHANGED.String():  AppConfigChangedNotification{},
	WebEventTypes.EVENTCOMMANDSTARTED.String():    CommandStartedNotification{},
	WebEventTypes.EVENTCOMMANDOUTPUT.String():     CommandOutputNotification{},
	WebEventTypes.EVENTCOMMANDTERMINATED.String(): CommandTerminatedNotification{},
}

func (WebEventMapTypes) IsValidEvent(event any) bool {
	for _, atype := range WebEventMap {
		if reflect.TypeOf(event) == reflect.TypeOf(atype) {
			return true
		}
	}
	return false
}
