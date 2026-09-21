package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dianlight/srat/dto"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

func TestPprofRouteAbsentInProduction(t *testing.T) {
	router := mux.NewRouter()
	RegisterPprof(router)

	req := httptest.NewRequest(http.MethodGet, "/debug/pprof/", nil)
	match := &mux.RouteMatch{}
	assert.False(t, router.Match(req, match), "pprof route must not exist in production builds")

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestNewMuxRouterNoPprofInProduction(t *testing.T) {
	router := NewMuxRouter(&dto.ContextState{SecureMode: false}, nil)

	req := httptest.NewRequest(http.MethodGet, "/debug/pprof/", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusNotFound, rr.Code)
}
