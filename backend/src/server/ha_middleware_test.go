package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dianlight/srat/internal/ctxkeys"
	"github.com/stretchr/testify/suite"
)

type HAMiddlewareSuite struct {
	suite.Suite
	middleware func(http.Handler) http.Handler
}

func (suite *HAMiddlewareSuite) SetupTest() {
	suite.middleware = NewHAMiddleware()
}

func (suite *HAMiddlewareSuite) TestMissingUserIdHeader() {
	handlerCalled := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	rr := httptest.NewRecorder()

	suite.middleware(handler).ServeHTTP(rr, req)
	suite.Equal(http.StatusUnauthorized, rr.Code)
	suite.False(handlerCalled)
}

func (suite *HAMiddlewareSuite) TestUnauthorizedIP() {
	handlerCalled := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Remote-User-Id", "user123")
	req.RemoteAddr = "192.168.1.100:54321"
	rr := httptest.NewRecorder()

	suite.middleware(handler).ServeHTTP(rr, req)
	suite.Equal(http.StatusUnauthorized, rr.Code)
	suite.False(handlerCalled)
}

func (suite *HAMiddlewareSuite) TestAuthorizedIPAndHeader() {
	handlerCalled := false
	var receivedUserID string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		val := r.Context().Value(ctxkeys.UserID)
		if v, ok := val.(string); ok {
			receivedUserID = v
		}
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Remote-User-Id", "user456")
	req.RemoteAddr = "127.0.0.1:9999"
	rr := httptest.NewRecorder()

	suite.middleware(handler).ServeHTTP(rr, req)
	suite.Equal(http.StatusOK, rr.Code)
	suite.True(handlerCalled)
	suite.Equal("user456", receivedUserID)
}

func (suite *HAMiddlewareSuite) TestAuthorizedIPAndHeader_HAIP() {
	handlerCalled := false
	var receivedUserID string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		val := r.Context().Value(ctxkeys.UserID)
		if v, ok := val.(string); ok {
			receivedUserID = v
		}
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Remote-User-Id", "user789")
	req.RemoteAddr = "172.30.32.2:8080"
	rr := httptest.NewRecorder()

	suite.middleware(handler).ServeHTTP(rr, req)
	suite.Equal(http.StatusOK, rr.Code)
	suite.True(handlerCalled)
	suite.Equal("user789", receivedUserID)
}

func TestHAMiddlewareSuite(t *testing.T) {
	suite.Run(t, new(HAMiddlewareSuite))
}
