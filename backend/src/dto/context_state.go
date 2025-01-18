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

/*
func (self *ContextState) FromJSONConfig(src config.Config) error {
	err := self.Settings.From(src)
	if err != nil {
		return tracerr.Wrap(err)
	}
	err = self.Users.From(src.OtherUsers)
	if err != nil {
		return tracerr.Wrap(err)
	}
	self.AdminUser.Username = src.Username
	self.AdminUser.Password = src.Password
	err = self.SharedResources.From(src.Shares)
	if err != nil {
		return tracerr.Wrap(err)
	}
	return nil
}

func (self ContextState) ToJSONConfig(dst *config.Config) (*config.Config, error) {
	err := self.Settings.To(&dst)
	if err != nil {
		return nil, err
	}
	err = self.Users.To(&dst.OtherUsers)
	if err != nil {
		return nil, err
	}
	dst.Username = self.AdminUser.Username
	dst.Password = self.AdminUser.Password
	err = self.SharedResources.To(&dst.Shares)
	if err != nil {
		return nil, err
	}
	return dst, nil
}
*/
