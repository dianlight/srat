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

// RegisterSse registers a Server-Sent Events (SSE) endpoint with the given API.
// It sets up the SSE endpoint at the path "/sse" with the HTTP GET method and
// provides a summary "Server sent events". The function maps various event types
// to their corresponding data structures and processes the HTTP channel using
// the broadcaster.
//
// Parameters:
//   - api: The huma.API instance to register the SSE endpoint with.
//
// Event Types:
//   - EVENTHELLO:     dto.Welcome
//   - EVENTUPDATE:    dto.ReleaseAsset
//   - EVENTUPDATING:  dto.UpdateProgress
//   - EVENTVOLUMES:   dto.BlockInfo
//   - EVENTHEARTBEAT: dto.HealthPing
//   - EVENTSHARE:     []dto.SharedResource
//
// The function processes the HTTP channel by calling self.broadcaster.ProcessHttpChannel
// with the provided SSE sender.
func (self *BrokerHandler) RegisterSse(api huma.API) {
	sse.Register(api, huma.Operation{
		OperationID: "sse",
		Method:      http.MethodGet,
		Path:        "/sse",
		Summary:     "Server sent events",
	}, map[string]any{
		dto.EventTypes.EVENTHELLO.Name:     dto.Welcome{},
		dto.EventTypes.EVENTUPDATE.Name:    dto.ReleaseAsset{},
		dto.EventTypes.EVENTUPDATING.Name:  dto.UpdateProgress{},
		dto.EventTypes.EVENTVOLUMES.Name:   dto.BlockInfo{},
		dto.EventTypes.EVENTHEARTBEAT.Name: dto.HealthPing{},
		dto.EventTypes.EVENTSHARE.Name:     []dto.SharedResource{},
	}, func(ctx context.Context, input *struct{}, send sse.Sender) {
		self.broadcaster.ProcessHttpChannel(send)
	})
}
