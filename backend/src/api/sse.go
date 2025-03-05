package api

import (
	"net/http"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/service"
	"github.com/go-fuego/fuego"
	"github.com/go-fuego/fuego/option"
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

func (handler *BrokerHandler) Routers(srv *fuego.Server) error {
	fuego.Get(srv, "/sse", handler.Stream, option.Description("Open a SSE stream"), option.Tags("system"))
	//fuego.Get(srv, "/sse/events", handler.EventTypeList,option.Description("Return a list of available WSChannel events"),option.Tags("system"))
	return nil
}

func (handler *BrokerHandler) Stream(c fuego.ContextNoBody) (*dto.EventMessageEnvelope, error) {
	err := handler.broadcaster.ProcessHttpChannel(c.Response(), c.Request())
	if err != nil {
		return nil, err
	} else {
		return nil, nil
	}
}

/*
// EventTypeList godoc
//
//	@Summary		EventTypeList
//	@Description	Return a list of available WSChannel events
//	@Tags			system
//	@Produce		json
//	@Success		200	{object}	[]dto.EventType
//	@Failure		500	{object}	string
//	@Router			/sse/events [get]
func (broker *BrokerHandler) EventTypeList(c fuego.ContextNoBody) ([]dto.EventType, error) {
	HttpJSONReponse(w, dto.EventTypes, nil)
}
*/
