package service

import (
	"context"
	"log/slog"

	"github.com/danielgtaylor/huma/v2/sse"
)

type BroadcasterServiceInterface interface {
	BroadcastMessage(msg any) (any, error)
	ProcessHttpChannel(send sse.Sender)
}

type BroadcasterService struct {
	ctx      context.Context
	notifier chan any
}

func NewBroadcasterService(ctx context.Context) (broker BroadcasterServiceInterface) {
	// Instantiate a broker
	rbroker := &BroadcasterService{
		ctx:      ctx,
		notifier: make(chan any, 1),
	}

	broker = rbroker
	return
}

func (broker *BroadcasterService) BroadcastMessage(msg any) (any, error) {
	broker.notifier <- msg
	return msg, nil
}

func (broker *BroadcasterService) ProcessHttpChannel(send sse.Sender) {
	for {
		select {
		case <-broker.ctx.Done():
			slog.Info("Run process closed", "err", broker.ctx.Err())
			return
		case event := <-broker.notifier:
			send.Data(event)
		}
	}
}
