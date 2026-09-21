//go:build pprof

package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
)

func TestPprofRoutePresentInPprofBuild(t *testing.T) {
	router := mux.NewRouter()
	RegisterPprof(router)

	req := httptest.NewRequest(http.MethodGet, "/debug/pprof/", nil)
	match := &mux.RouteMatch{}
	require.True(t, router.Match(req, match), "pprof route must exist in pprof builds")
}
