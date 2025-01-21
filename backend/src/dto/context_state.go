package dto

import (
	"context"
)

type ContextState struct {
	ReadOnlyMode     bool             `json:"read_only_mode"`
	UpdateFilePath   string           `json:"update_file_path"`
	DataDirtyTracker DataDirtyTracker `json:"data_dirty_tracker"`
	SambaConfigFile  string           `json:"samba_config_file"`
}

func (self *ContextState) FromContext(ctx context.Context) *ContextState {
	self = ctx.Value("context_state").(*ContextState)
	return self
}

func (self *ContextState) ToContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, "context_state", self)
}
