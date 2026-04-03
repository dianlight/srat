package dto

//go:generate go tool goenums -l webevent_type.go
type webEventType int

const (
	eventHello             webEventType = iota // "hello"
	eventUpdating                              // "updating"
	eventVolumes                               // "volumes"
	eventHeartbeat                             // "heartbeat"
	eventShares                                // "shares"
	eventDirtyTracker                          // "dirty_data_tracker"
	eventSmartTestStatus                       // "smart_test_status"
	eventFilesystemTask                        // "filesystem_task"
	eventError                                 // "error"
	eventRepairCommand                         // "repair_command"
	eventAppConfigChanged                      // "app_config_changed"
	eventCommandStarted                        // "command_started"
	eventCommandOutput                         // "command_output"
	eventCommandTerminated                     // "command_terminated"
)
