package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"

	"github.com/dianlight/srat/dto"
	"github.com/google/uuid"
	"github.com/ztrue/tracerr"
)

type BroadcasterServiceInterface interface {
	BroadcastMessage(msg *dto.EventMessageEnvelope) (*dto.EventMessageEnvelope, error)
	//listen()
	AddOpenConnectionListener(ws func(broker BroadcasterServiceInterface) error) error
	AddCloseConnectionListener(ws func(broker BroadcasterServiceInterface) error) error
	ProcessHttpChannel(w http.ResponseWriter, r *http.Request) error
}

type BroadcasterService struct {
	ctx            context.Context
	notifier       chan dto.EventMessageEnvelope
	newClients     chan chan dto.EventMessageEnvelope
	closingClients chan chan dto.EventMessageEnvelope
	clients        map[chan dto.EventMessageEnvelope]bool

	// Listeners for new and closed connections
	openConnectionListeners  []func(broker BroadcasterServiceInterface) error
	closeConnectionListeners []func(broker BroadcasterServiceInterface) error
}

func NewBroadcasterService(ctx context.Context) (broker BroadcasterServiceInterface) {
	// Instantiate a broker
	rbroker := &BroadcasterService{
		ctx:            ctx,
		notifier:       make(chan dto.EventMessageEnvelope, 1),
		newClients:     make(chan chan dto.EventMessageEnvelope),
		closingClients: make(chan chan dto.EventMessageEnvelope),
		clients:        make(map[chan dto.EventMessageEnvelope]bool),
	}

	broker = rbroker
	// Set it running - listening and broadcasting events
	go (rbroker).listen()

	return
}

func (broker *BroadcasterService) listen() {
	for {
		select {
		case <-broker.ctx.Done():
			slog.Info("Run process closed", "err", broker.ctx.Err())
			return
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
		case event := <-broker.notifier:
			// We got a new event from the outside!
			// Send event to all connected clients
			for clientMessageChan := range broker.clients {
				clientMessageChan <- event
			}
		}
	}
}

func (broker *BroadcasterService) BroadcastMessage(msg *dto.EventMessageEnvelope) (*dto.EventMessageEnvelope, error) {
	if msg.Id == "" {
		msg.Id = uuid.New().String()
	}
	broker.notifier <- *msg
	//slog.Debug("Broadcasted message:", "msg", msg)
	return msg, nil
}

func (broker *BroadcasterService) AddOpenConnectionListener(ws func(broker BroadcasterServiceInterface) error) error {
	broker.openConnectionListeners = append(broker.openConnectionListeners, ws)
	return nil
}

func (broker *BroadcasterService) AddCloseConnectionListener(ws func(broker BroadcasterServiceInterface) error) error {
	broker.closeConnectionListeners = append(broker.closeConnectionListeners, ws)
	return nil
}

func (broker *BroadcasterService) ProcessHttpChannel(w http.ResponseWriter, r *http.Request) error {
	// Check if the ResponseWriter supports flushing.
	flusher, ok := w.(http.Flusher)
	if !ok {
		return tracerr.Errorf("Streaming unsupported!")
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
	w.Header().Set("X-Accel-Buffering", "no")
	//w.Header().Set("Access-Control-Allow-Origin", "*")

	for {
		select {
		// Listen to connection close and un-register messageChan
		case <-r.Context().Done():
			// remove this client from the map of connected clients
			broker.closingClients <- messageChan
			return nil

		// Listen for incoming messages from messageChan
		case msg := <-messageChan:
			// Write to the ResponseWriter
			j, err := json.Marshal(msg.Data)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return tracerr.Wrap(err)
			}
			// Server Sent Events compatible
			fmt.Fprintf(w, "event: %s\nid: %s\nretry: 3000\ndata: %s\n\n", msg.Event, msg.Id, []byte(j))
			// Flush the data immediatly instead of buffering it for later.
			flusher.Flush()
		}
	}
}
