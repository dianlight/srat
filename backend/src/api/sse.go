package api

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/sse"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/service"
)

type BrokerHandler struct {
	broadcaster service.BroadcasterServiceInterface
}

type BrokerInterface interface {
	Stream(w http.ResponseWriter, r *http.Request)
	BroadcastMessage(msg any) (any, error)
}

func NewSSEBroker(broadcaster service.BroadcasterServiceInterface) (broker *BrokerHandler) {
	// Instantiate a broker
	broker = &BrokerHandler{
		broadcaster: broadcaster,
	}
	return
}

func (self *BrokerHandler) RegisterSse(api huma.API) {
	sse.Register(api, huma.Operation{
		OperationID: "sse",
		Method:      http.MethodGet,
		Path:        "/sse",
		Summary:     "Server sent events",
	}, map[string]any{
		dto.EventTypes.EVENTHELLO.Name:     []dto.EventType{},
		dto.EventTypes.EVENTUPDATE.Name:    dto.ReleaseAsset{},
		dto.EventTypes.EVENTUPDATING.Name:  dto.UpdateProgress{},
		dto.EventTypes.EVENTVOLUMES.Name:   dto.BlockInfo{},
		dto.EventTypes.EVENTHEARTBEAT.Name: dto.HealthPing{},
		dto.EventTypes.EVENTSHARE.Name:     []dto.SharedResource{},
	}, func(ctx context.Context, input *struct{}, send sse.Sender) {
		self.broadcaster.ProcessHttpChannel(send)
	})
}
