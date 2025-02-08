package api

import (
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
	Heartbeat        int
	//SSEBroker        BrokerInterface
}

/*
func StateFromContext(ctx context.Context) *ContextState {
	//var self *ContextState
	//log.Printf("----> %+v", ctx)
	self, ok := ctx.Value("context_state").(*ContextState)
	if !ok {
		slog.Error("Cannot get 'context_state' from context", "ctx", ctx)
		//self = &ContextState{}
	}
	return self
}

func StateToContext(self *ContextState, ctx context.Context) context.Context {
	//log.Printf("<---- %+v", self)
	return context.WithValue(ctx, "context_state", self)
}
*/
