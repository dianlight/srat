package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/dianlight/srat/dto"
	"github.com/google/uuid"
)

type Broker struct {
	// Events are pushed to this channel by the main events-gathering routine
	Notifier chan dto.EventMessageEnvelope

	// New client connections are pushed to this channel
	newClients chan chan dto.EventMessageEnvelope

	// Closed client connections are pushed to this channel
	closingClients chan chan dto.EventMessageEnvelope

	// Client connections registry
	clients map[chan dto.EventMessageEnvelope]bool
}

type BrokerInterface interface {
	Stream(w http.ResponseWriter, r *http.Request)
	BroadcastMessage(msg *dto.EventMessageEnvelope) (*dto.EventMessageEnvelope, error)
}

func NewSSEBroker() (broker *Broker) {
	// Instantiate a broker
	broker = &Broker{
		Notifier:       make(chan dto.EventMessageEnvelope, 1),
		newClients:     make(chan chan dto.EventMessageEnvelope),
		closingClients: make(chan chan dto.EventMessageEnvelope),
		clients:        make(map[chan dto.EventMessageEnvelope]bool),
	}

	// Set it running - listening and broadcasting events
	go broker.listen()

	return
}

func (broker *Broker) listen() {
	for {
		select {
		case s := <-broker.newClients:

			// A new client has connected.
			// Register their message channel
			broker.clients[s] = true
			log.Printf("Client added. %d registered clients", len(broker.clients))
			broker.BroadcastMessage(&dto.EventMessageEnvelope{Event: dto.EventHello})
		case s := <-broker.closingClients:

			// A client has dettached and we want to
			// stop sending them messages.
			delete(broker.clients, s)
			log.Printf("Removed client. %d registered clients", len(broker.clients))
		case event := <-broker.Notifier:

			// We got a new event from the outside!
			// Send event to all connected clients
			for clientMessageChan := range broker.clients {
				clientMessageChan <- event
			}
		}
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
// @Failure		500	{object}	ErrorResponse
// @Router			/sse [get]
func (broker *Broker) Stream(w http.ResponseWriter, r *http.Request) {
	// Check if the ResponseWriter supports flushing.
	flusher, ok := w.(http.Flusher)
	if !ok {
		HttpJSONReponse(w, "Streaming unsupported!", &Options{
			Code: http.StatusInternalServerError,
		})
		return
	}

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
}

func (broker *Broker) BroadcastMessage(msg *dto.EventMessageEnvelope) (*dto.EventMessageEnvelope, error) {
	if msg.Id == "" {
		msg.Id = uuid.New().String()
	}
	broker.Notifier <- *msg
	log.Printf("Broadcasted message: %+v\n", msg)
	return msg, nil
}

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
