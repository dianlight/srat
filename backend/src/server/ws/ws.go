// Package sse provides utilities for working with Server Sent Events (SSE).
package ws

import "gitlab.com/tozd/go/errors"

type Message struct {
	ID   int
	Data any
}

type Sender func(Message) errors.E
