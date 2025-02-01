package api

import (
	"net/http"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/server"
	"github.com/dianlight/srat/service"
)

type BrokerHandler struct {
	// Events are pushed to this channel by the main events-gathering routine
	//Notifier chan dto.EventMessageEnvelope

	// New client connections are pushed to this channel
	//newClients chan chan dto.EventMessageEnvelope

	// Closed client connections are pushed to this channel
	//closingClients chan chan dto.EventMessageEnvelope

	// Client connections registry
	//clients map[chan dto.EventMessageEnvelope]bool

	// Listeners for new and closed connections
	//openConnectionListeners  []func(broker BrokerInterface) error
	//closeConnectionListeners []func(broker BrokerInterface) error
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
		//Notifier:       make(chan dto.EventMessageEnvelope, 1),
		//newClients:     make(chan chan dto.EventMessageEnvelope),
		//closingClients: make(chan chan dto.EventMessageEnvelope),
		//clients:        make(map[chan dto.EventMessageEnvelope]bool),
	}

	// Set it running - listening and broadcasting events
	//go broadcaster.listen()

	return
}

func (broker *BrokerHandler) Patterns() []server.RouteDetail {
	return []server.RouteDetail{
		{Pattern: "/sse", Method: "GET", Handler: broker.Stream},
	}
}

/*
func (broker *BrokerHandler) listen() {
	for {
		select {
		case s := <-broker.newClients:

			// A new client has connected.
			// Register their message channel
			broker.clients[s] = true
			log.Printf("Client added. %d registered clients", len(broker.clients))
			broker.BroadcastMessage(&dto.EventMessageEnvelope{Event: dto.EventHello})
			go func() {
				for _, openListener := range broker.openConnectionListeners {
					err := openListener(broker)
					if err != nil {
						slog.Warn("Error in open connection listener", "err", tracerr.SprintSourceColor(err))
					}
				}
			}()

		case s := <-broker.closingClients:

			// A client has dettached and we want to
			// stop sending them messages.
			delete(broker.clients, s)
			log.Printf("Removed client. %d registered clients", len(broker.clients))
			go func() {
				for _, closeListener := range broker.closeConnectionListeners {
					err := closeListener(broker)
					if err != nil {
						slog.Warn("Error in close connection listener", "err", tracerr.SprintSourceColor(err))
					}
				}
			}()
		case event := <-broker.Notifier:
			// We got a new event from the outside!
			// Send event to all connected clients
			for clientMessageChan := range broker.clients {
				clientMessageChan <- event
			}
		}
	}
}
*/

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
// @Failure		500	{object}	ErrorResponse
// @Router			/sse [get]
func (broker *BrokerHandler) Stream(w http.ResponseWriter, r *http.Request) {
	err := broker.broadcaster.ProcessHttpChannel(w, r)
	if err != nil {
		HttpJSONReponse(w, err, &Options{
			Code: http.StatusInternalServerError,
		})
		return
	}
	/*
	   // Each connection registers its own message channel with the Broker's connections registry
	   messageChan := make(chan dto.EventMessageEnvelope)

	   // Signal the broker that we have a new connection
	   broker.newClients <- messageChan

	   // Remove this client from the map of connected clients
	   // when this handler exits.

	   	defer func() {
	   		broker.closingClients <- messageChan
	   	}()

	   w.Header().Set("Content-Type", "text/event-stream")
	   w.Header().Set("Cache-Control", "no-cache")
	   w.Header().Set("Connection", "keep-alive")
	   //w.Header().Set("Access-Control-Allow-Origin", "*")

	   	for {
	   		select {
	   		// Listen to connection close and un-register messageChan
	   		case <-r.Context().Done():
	   			// remove this client from the map of connected clients
	   			broker.closingClients <- messageChan
	   			return

	   		// Listen for incoming messages from messageChan
	   		case msg := <-messageChan:
	   			// Write to the ResponseWriter
	   			j, err := json.Marshal(msg.Data)
	   			if err != nil {
	   				http.Error(w, err.Error(), http.StatusInternalServerError)
	   				return
	   			}
	   			// Server Sent Events compatible
	   			fmt.Fprintf(w, "event: %s\nid: %s\nretry: 30\ndata: %s\n\n", msg.Event, msg.Id, []byte(j))
	   			// Flush the data immediatly instead of buffering it for later.
	   			flusher.Flush()
	   		}
	   	}
	*/
}

/*
func (broker *BrokerHandler) BroadcastMessage(msg *dto.EventMessageEnvelope) (*dto.EventMessageEnvelope, error) {
	if msg.Id == "" {
		msg.Id = uuid.New().String()
	}
	broker.Notifier <- *msg
	slog.Debug("Broadcasted message:", "msg", msg)
	return msg, nil
}

func (broker *BrokerHandler) AddOpenConnectionListener(ws func(broker BrokerInterface) error) error {
	broker.openConnectionListeners = append(broker.openConnectionListeners, ws)
	return nil
}

func (broker *BrokerHandler) AddCloseConnectionListener(ws func(broker BrokerInterface) error) error {
	broker.closeConnectionListeners = append(broker.closeConnectionListeners, ws)
	return nil
}
*/

/*
func main() {
	broker := NewSSEServer()
	router := mux.NewRouter()

	router.HandleFunc("/messages", broker.BroadcastMessage).Methods("POST")

	router.HandleFunc("/stream", broker.Stream).Methods("GET")

	log.Println("Starting server on :8000")
	log.Fatal(http.ListenAndServe(":8000", router))
}
*/

// To test the server, run the following commands in separate terminals:
// Start listening to the stream
//     $ curl -N http://localhost:8000/stream
// Send a message
//     $ curl -X POST -H "Content-Type: application/json" -d '{"name": "Alice", "msg": "Hello"}' http://localhost:8000/messages
