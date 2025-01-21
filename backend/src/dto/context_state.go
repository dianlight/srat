package dto

import (
	"context"
)

type ContextState struct {
	//	BlockInfo        BlockInfo        `json:"devices"`
	DataDirtyTracker DataDirtyTracker `json:"data_dirty_tracker"`
	// MountPointData   MountPointData   `json:"mount_point_data"`
	// Settings         Settings         `json:"settings"`
	// Users            Users            `json:"users"`
	// AdminUser        User             `json:"admin_users"`
	// SharedResources  SharedResources  `json:"shared_resources"`
}

func (self *ContextState) FromContext(ctx context.Context) *ContextState {
	self = ctx.Value("context_state").(*ContextState)
	return self
}

func (self *ContextState) ToContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, "context_state", self)
}
