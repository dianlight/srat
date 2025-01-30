package api

import (
	"context"

	"github.com/dianlight/srat/dto"
)

type ContextState struct {
	ReadOnlyMode     bool
	UpdateFilePath   string
	DataDirtyTracker dto.DataDirtyTracker
	SambaConfigFile  string
	Template         []byte
	DockerInterface  string
	DockerNet        string
	SSEBroker        BrokerInterface
}

func StateFromContext(ctx context.Context) *ContextState {
	var self *ContextState
	//log.Printf("----> %+v", ctx)
	self = ctx.Value("context_state").(*ContextState)
	return self
}

func StateToContext(self *ContextState, ctx context.Context) context.Context {
	//log.Printf("<---- %+v", self)
	return context.WithValue(ctx, "context_state", self)
}
