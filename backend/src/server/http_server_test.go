package server

import (
	"testing"

	"github.com/dianlight/srat/dto"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

func TestNewMuxRouter(t *testing.T) {
	apiCtx := &dto.ContextState{
		SecureMode: false,
	}

	// Create a minimal WebSocketHandler for testing
	// We can pass nil since we're just testing router creation
	router := NewMuxRouter(apiCtx, nil)

	assert.NotNil(t, router)
	assert.IsType(t, &mux.Router{}, router)
}

func TestNewMuxRouterWithSecureMode(t *testing.T) {
	apiCtx := &dto.ContextState{
		SecureMode: true,
	}

	router := NewMuxRouter(apiCtx, nil)

	assert.NotNil(t, router)
	assert.IsType(t, &mux.Router{}, router)
}

func TestNewMuxRouterWithoutSecureMode(t *testing.T) {
	apiCtx := &dto.ContextState{
		SecureMode: false,
	}

	router := NewMuxRouter(apiCtx, nil)

	assert.NotNil(t, router)
	assert.IsType(t, &mux.Router{}, router)
}
