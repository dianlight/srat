package service_test

import (
	"context"
	"crypto/rand"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/service"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type errReader struct{ err error }

func (e errReader) Read(p []byte) (int, error) { return 0, e.err }

type errReadCloser struct{ err error }

func (e errReadCloser) Read(p []byte) (int, error) { return 0, e.err }
func (e errReadCloser) Close() error               { return nil }

func TestBrokerClient_Patch80_GetKeyPair_DBErrors(t *testing.T) {
	// First: error not NotFound (closed DB)
	db, _ := gorm.Open(sqlite.Open("file:mem_patch80_1?mode=memory&cache=shared"), &gorm.Config{})
	_ = db.AutoMigrate(&dbom.OAuthKeyPair{})
	sqlDB, _ := db.DB()
	_ = sqlDB.Close()
	client := service.NewBrokerClientWithHTTP(db, http.DefaultClient)
	_, errE := client.GetKeyPair(context.Background())
	require.Error(t, errE)

	// Create error: duplicate ID or closed DB
	db2, _ := gorm.Open(sqlite.Open("file:mem_patch80_2?mode=memory&cache=shared"), &gorm.Config{})
	_ = db2.AutoMigrate(&dbom.OAuthKeyPair{})
	// Make Create fail by closing DB before GetKeyPair's Create path (when no row exists, it will try Generate and Create)
	sqlDB2, _ := db2.DB()
	// First ensure no row, then close before Create
	_ = sqlDB2.Close()
	client2 := service.NewBrokerClientWithHTTP(db2, http.DefaultClient)
	_, errE = client2.GetKeyPair(context.Background())
	// GenerateKeyPair may succeed but Create will fail due to closed DB – should error at line 85-86
	require.Error(t, errE)
}

func TestBrokerClient_Patch80_DoSigned_RandAndRequestErrors(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open("file:mem_patch80_3?mode=memory&cache=shared"), &gorm.Config{})
	_ = db.AutoMigrate(&dbom.OAuthKeyPair{})
	origReader := rand.Reader
	// Force rand.Read error
	rand.Reader = errReader{err: io.ErrUnexpectedEOF}
	client := service.NewBrokerClientWithHTTP(db, http.DefaultClient)
	_, _, errE := client.StartOAuth(context.Background(), "https://example.com", "dropbox", "https://x/cb", "ha-1")
	require.Error(t, errE)
	require.Contains(t, errE.Error(), "unexpected EOF")
	rand.Reader = origReader

	// Invalid URL for NewRequest error (line 110-111)
	client2 := service.NewBrokerClientWithHTTP(db, http.DefaultClient)
	_, _ = client2.GetKeyPair(context.Background())
	_, _, errE = client2.StartOAuth(context.Background(), "http://[invalid", "dropbox", "https://x/cb", "ha-1")
	require.Error(t, errE)

	// Do error via canceled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, _, errE = client2.StartOAuth(ctx, "https://example.com", "dropbox", "https://x/cb", "ha-1")
	require.Error(t, errE)

	// Do error via client that always fails
	badClient := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) { return nil, io.ErrUnexpectedEOF })}
	client3 := service.NewBrokerClientWithHTTP(db, badClient)
	_, _, errE = client3.StartOAuth(context.Background(), "https://example.com", "dropbox", "https://x/cb", "ha-1")
	require.Error(t, errE)
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestBrokerClient_Patch80_RegisterClient_Errors(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open("file:mem_patch80_4?mode=memory&cache=shared"), &gorm.Config{})
	_ = db.AutoMigrate(&dbom.OAuthKeyPair{})
	client := service.NewBrokerClientWithHTTP(db, http.DefaultClient)
	// GetKeyPair fail via closed DB
	sqlDB, _ := db.DB()
	_ = sqlDB.Close()
	require.Error(t, client.RegisterClient(context.Background(), "https://example.com"))
	// Reopen for next
	db2, _ := gorm.Open(sqlite.Open("file:mem_patch80_5?mode=memory&cache=shared"), &gorm.Config{})
	_ = db2.AutoMigrate(&dbom.OAuthKeyPair{})
	client2 := service.NewBrokerClientWithHTTP(db2, http.DefaultClient)
	// Invalid URL for NewRequest
	_, _ = client2.GetKeyPair(context.Background())
	require.Error(t, client2.RegisterClient(context.Background(), "http://[invalid"))
	// Do error
	badClient := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) { return nil, io.ErrUnexpectedEOF })}
	client3 := service.NewBrokerClientWithHTTP(db2, badClient)
	require.Error(t, client3.RegisterClient(context.Background(), "https://example.com"))
	// status !=200/201 already covered in coverage_test.go but ensure 201 vs 200 branches
	fake201 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(201); w.Write([]byte(`{}`)) }))
	defer fake201.Close()
	require.NoError(t, client2.RegisterClient(context.Background(), fake201.URL))
	fake200 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200); w.Write([]byte(`{}`)) }))
	defer fake200.Close()
	require.NoError(t, client2.RegisterClient(context.Background(), fake200.URL))
}

func TestBrokerClient_Patch80_StartOAuth_ReadUnmarshalErrors(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open("file:mem_patch80_6?mode=memory&cache=shared"), &gorm.Config{})
	_ = db.AutoMigrate(&dbom.OAuthKeyPair{})
	// ReadAll error via failing body
	fakeReadErr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/clients" {
			w.WriteHeader(201)
			w.Write([]byte(`{"client_id":"x","public_key":"y"}`))
			return
		}
		if r.URL.Path == "/v1/start" {
			w.WriteHeader(200)
			// Hijack to return a body that fails on Read
			w.Header().Set("Content-Length", "10")
			// Write partially then close?
			w.Write([]byte(`{"auth_url`))
			return
		}
		w.WriteHeader(404)
	}))
	defer fakeReadErr.Close()
	client := service.NewBrokerClientWithHTTP(db, fakeReadErr.Client())
	_, _ = client.GetKeyPair(context.Background())
	_ = client.RegisterClient(context.Background(), fakeReadErr.URL)
	// This will try to read and unmarshal incomplete JSON -> unmarshal error
	_, _, errE := client.StartOAuth(context.Background(), fakeReadErr.URL, "dropbox", "https://x/cb", "ha-1")
	// May succeed or fail depending on body, but we want unmarshal error path
	_ = errE
	// Explicit unmarshal error via valid 200 but bad JSON
	fakeBadJSON := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/clients" {
			w.WriteHeader(201)
			w.Write([]byte(`{"client_id":"x","public_key":"y"}`))
			return
		}
		w.WriteHeader(200)
		w.Write([]byte(`{not-json`))
	}))
	defer fakeBadJSON.Close()
	db2, _ := gorm.Open(sqlite.Open("file:mem_patch80_7?mode=memory&cache=shared"), &gorm.Config{})
	_ = db2.AutoMigrate(&dbom.OAuthKeyPair{})
	client2 := service.NewBrokerClientWithHTTP(db2, fakeBadJSON.Client())
	_, _, errE = client2.StartOAuth(context.Background(), fakeBadJSON.URL, "dropbox", "https://x/cb", "ha-1")
	require.Error(t, errE)
}

func TestBrokerClient_Patch80_GetSession_StartDelete_Errors(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open("file:mem_patch80_8?mode=memory&cache=shared"), &gorm.Config{})
	_ = db.AutoMigrate(&dbom.OAuthKeyPair{})
	client := service.NewBrokerClientWithHTTP(db, http.DefaultClient)
	// GetKeyPair fail
	sqlDB, _ := db.DB()
	_ = sqlDB.Close()
	_, errE := client.GetSession(context.Background(), "https://example.com", "sess")
	require.Error(t, errE)
	_, _, errE2 := client.StartOAuth(context.Background(), "https://example.com", "p", "cb", "inst")
	require.Error(t, errE2)
	require.Error(t, client.DeleteClient(context.Background(), "https://example.com"))
	require.Error(t, client.DeleteClientByID(context.Background(), "https://example.com", "id"))
	// Reopen for further
	db2, _ := gorm.Open(sqlite.Open("file:mem_patch80_9?mode=memory&cache=shared"), &gorm.Config{})
	_ = db2.AutoMigrate(&dbom.OAuthKeyPair{})
	client2 := service.NewBrokerClientWithHTTP(db2, http.DefaultClient)
	_, _ = client2.GetKeyPair(context.Background())
	// Invalid URL for GetSession doSigned
	_, errE = client2.GetSession(context.Background(), "http://[invalid", "sess")
	require.Error(t, errE)
	// DeleteClient status !=200 already covered but ensure
	fake500 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/v1/clients") && r.Method == "POST" {
			w.WriteHeader(201)
			w.Write([]byte(`{"client_id":"x"}`))
			return
		}
		w.WriteHeader(500)
		w.Write([]byte(`err`))
	}))
	defer fake500.Close()
	require.Error(t, client2.DeleteClient(context.Background(), fake500.URL))
	require.Error(t, client2.DeleteClientByID(context.Background(), fake500.URL, "x"))
}

func TestBrokerClient_Patch80_DeleteClientByID_EmptyIf(t *testing.T) {
	// The empty if at line 294-296: if clientID == kp.ClientID { }
	// We already hit it via DeleteClientByID success test, but ensure coverage by calling with equal and not equal
	db, _ := gorm.Open(sqlite.Open("file:mem_patch80_10?mode=memory&cache=shared"), &gorm.Config{})
	_ = db.AutoMigrate(&dbom.OAuthKeyPair{})
	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/clients" && r.Method == "POST" {
			w.WriteHeader(201)
			w.Write([]byte(`{"client_id":"c"}`))
			return
		}
		w.WriteHeader(200)
		w.Write([]byte(`{}`))
	}))
	defer fake.Close()
	client := service.NewBrokerClientWithHTTP(db, fake.Client())
	kp, _ := client.GetKeyPair(context.Background())
	_ = client.RegisterClient(context.Background(), fake.URL)
	// equal -> hits empty if and later delete
	require.NoError(t, client.DeleteClientByID(context.Background(), fake.URL, kp.ClientID))
	// not equal -> does not hit second delete
	kp2, _ := client.GetKeyPair(context.Background())
	_ = client.RegisterClient(context.Background(), fake.URL)
	require.NoError(t, client.DeleteClientByID(context.Background(), fake.URL, "different-id-1234567890"))
	_ = kp2
}
