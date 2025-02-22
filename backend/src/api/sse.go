package api

import (
	"net/http"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/server"
	"github.com/dianlight/srat/service"
)

type BrokerHandler struct {
	broadcaster service.BroadcasterServiceInterface
}

type BrokerInterface interface {
	Stream(w http.ResponseWriter, r *http.Request)
	BroadcastMessage(msg *dto.EventMessageEnvelope) (*dto.EventMessageEnvelope, error)
	AddOpenConnectionListener(ws func(broker BrokerInterface) error) error
	AddCloseConnectionListener(ws func(broker BrokerInterface) error) error
}

func NewSSEBroker(broadcaster service.BroadcasterServiceInterface) (broker *BrokerHandler) {
	// Instantiate a broker
	broker = &BrokerHandler{
		broadcaster: broadcaster,
	}
	return
}

func (handler *BrokerHandler) Patterns() []server.RouteDetail {
	return []server.RouteDetail{
		{Pattern: "/sse", Method: "GET", Handler: handler.Stream},
		{Pattern: "/sse/events", Method: "GET", Handler: handler.EventTypeList},
	}
}

// SSE Stream godoc
//
// @Summary		Open a SSE stream
// @Description	Open a SSE stream
//
//	@Accept			json
//	@Produce		text/event-stream
//
// @Tags			system
// @Success		200	{object} dto.EventMessageEnvelope
// @Failure		500	{object}	dto.ErrorInfo
// @Router			/sse [get]
func (handler *BrokerHandler) Stream(w http.ResponseWriter, r *http.Request) {
	err := handler.broadcaster.ProcessHttpChannel(w, r)
	if err != nil {
		HttpJSONReponse(w, err, &Options{
			Code: http.StatusInternalServerError,
		})
		return
	}
}

// EventTypeList godoc
//
//	@Summary		EventTypeList
//	@Description	Return a list of available WSChannel events
//	@Tags			system
//	@Produce		json
//	@Success		200	{object}	[]dto.EventType
//	@Failure		500	{object}	string
//	@Router			/sse/events [get]
func (broker *BrokerHandler) EventTypeList(w http.ResponseWriter, rq *http.Request) {
	HttpJSONReponse(w, dto.EventTypes, nil)
}
