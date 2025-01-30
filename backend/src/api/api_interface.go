package api

import "context"

type Api interface {
}

type ApiEventEmitter[DTO any] interface {
	EventEmitter(ctx context.Context, data DTO) error
	// Background(ctx context.Context) error
}
