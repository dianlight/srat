package service_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/service"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestBrokerClient_Coverage_NewBrokerClientAndBrokerURLFromEnv(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:mem_newbroker?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&dbom.OAuthKeyPair{}))
	c := service.NewBrokerClient(db)
	require.NotNil(t, c)
	t.Run("BrokerURLFromEnv fallbacks", func(t *testing.T) {
		orig1, orig2 := os.Getenv("BROKER_PUBLIC_URL"), os.Getenv("SRAT_BROKER_URL")
		defer func() { _ = os.Setenv("BROKER_PUBLIC_URL", orig1); _ = os.Setenv("SRAT_BROKER_URL", orig2) }()
		_ = os.Unsetenv("BROKER_PUBLIC_URL")
		_ = os.Unsetenv("SRAT_BROKER_URL")
		require.Equal(t, "https://srat-oauth-broker-production.lucio-tarantino.workers.dev", service.BrokerURLFromEnv())
		_ = os.Setenv("SRAT_BROKER_URL", "https://fallback.example.com")
		require.Equal(t, "https://fallback.example.com", service.BrokerURLFromEnv())
		_ = os.Setenv("BROKER_PUBLIC_URL", "https://primary.example.com")
		require.Equal(t, "https://primary.example.com", service.BrokerURLFromEnv())
	})
}

func TestBrokerClient_Coverage_GetKeyPair_CorruptDB(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:mem_corrupt2?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&dbom.OAuthKeyPair{}))
	client := service.NewBrokerClientWithHTTP(db, http.DefaultClient)
	ctx := context.Background()
	require.NoError(t, db.Create(&dbom.OAuthKeyPair{ID: "default", PrivateKey: "!!!not-b64!!!", PublicKey: "!!!", ClientID: "bad"}).Error)
	_, errE := client.GetKeyPair(ctx)
	require.Error(t, errE)
	require.NoError(t, db.Delete(&dbom.OAuthKeyPair{}, "id = ?", "default").Error)
	priv := base64.RawURLEncoding.EncodeToString(make([]byte, 32))
	require.NoError(t, db.Create(&dbom.OAuthKeyPair{ID: "default", PrivateKey: priv, PublicKey: "!!!bad!!!", ClientID: "bad2"}).Error)
	_, errE = client.GetKeyPair(ctx)
	require.Error(t, errE)
}

func TestBrokerClient_Coverage_RegisterAndSession_Errors(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:mem_sesserr_cov?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&dbom.OAuthKeyPair{}))
	t.Run("RegisterClient non-201", func(t *testing.T) {
		fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(500); w.Write([]byte(`server error`)) }))
		defer fake.Close()
		client := service.NewBrokerClientWithHTTP(db, fake.Client())
		require.Error(t, client.RegisterClient(context.Background(), fake.URL))
	})
	t.Run("GetSession 404", func(t *testing.T) {
		fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/v1/clients" {
				body, _ := io.ReadAll(r.Body)
				var req map[string]string
				_ = json.Unmarshal(body, &req)
				w.WriteHeader(201)
				_ = json.NewEncoder(w).Encode(req)
				return
			}
			w.WriteHeader(404)
			w.Write([]byte(`{"error":"not found"}`))
		}))
		defer fake.Close()
		db2, _ := gorm.Open(sqlite.Open("file:mem_sess404?mode=memory&cache=shared"), &gorm.Config{})
		_ = db2.AutoMigrate(&dbom.OAuthKeyPair{})
		client := service.NewBrokerClientWithHTTP(db2, fake.Client())
		_, errE := client.GetSession(context.Background(), fake.URL, "missing-sess")
		require.Error(t, errE)
		require.Contains(t, errE.Error(), "404")
	})
	t.Run("GetSession invalid json", func(t *testing.T) {
		fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/v1/clients" {
				body, _ := io.ReadAll(r.Body)
				var req map[string]string
				_ = json.Unmarshal(body, &req)
				w.WriteHeader(201)
				_ = json.NewEncoder(w).Encode(req)
				return
			}
			w.WriteHeader(200)
			w.Write([]byte(`{not-json`))
		}))
		defer fake.Close()
		db2, _ := gorm.Open(sqlite.Open("file:mem_sessbad?mode=memory&cache=shared"), &gorm.Config{})
		_ = db2.AutoMigrate(&dbom.OAuthKeyPair{})
		client := service.NewBrokerClientWithHTTP(db2, fake.Client())
		_, errE := client.GetSession(context.Background(), fake.URL, "sess-badjson")
		require.Error(t, errE)
	})
	t.Run("StartOAuth 400", func(t *testing.T) {
		fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/v1/clients" {
				body, _ := io.ReadAll(r.Body)
				var req map[string]string
				_ = json.Unmarshal(body, &req)
				w.WriteHeader(201)
				_ = json.NewEncoder(w).Encode(req)
				return
			}
			if r.URL.Path == "/v1/start" {
				w.WriteHeader(400)
				w.Write([]byte(`{"error":"bad provider"}`))
				return
			}
			w.WriteHeader(404)
		}))
		defer fake.Close()
		db3, _ := gorm.Open(sqlite.Open("file:mem_start400?mode=memory&cache=shared"), &gorm.Config{})
		_ = db3.AutoMigrate(&dbom.OAuthKeyPair{})
		client := service.NewBrokerClientWithHTTP(db3, fake.Client())
		_, _, errE := client.StartOAuth(context.Background(), fake.URL, "bad-provider", "https://x/cb", "ha-err")
		require.Error(t, errE)
		require.Contains(t, errE.Error(), "400")
	})
	t.Run("RegisterInstance tolerates 409 already exists", func(t *testing.T) {
		db4, _ := gorm.Open(sqlite.Open("file:mem_409cov?mode=memory&cache=shared"), &gorm.Config{})
		_ = db4.AutoMigrate(&dbom.OAuthKeyPair{})
		callCount := 0
		fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/v1/clients" {
				callCount++
				if callCount == 1 {
					w.WriteHeader(409)
					w.Write([]byte(`{"error":"already registered"}`))
					return
				}
				body, _ := io.ReadAll(r.Body)
				var req map[string]string
				_ = json.Unmarshal(body, &req)
				w.WriteHeader(201)
				_ = json.NewEncoder(w).Encode(req)
				return
			}
			if r.URL.Path == "/v1/instances/register" {
				w.WriteHeader(200)
				w.Write([]byte(`{"instance_id":"ha-409"}`))
				return
			}
			w.WriteHeader(404)
		}))
		defer fake.Close()
		client := service.NewBrokerClientWithHTTP(db4, fake.Client())
		require.NoError(t, client.RegisterInstance(context.Background(), fake.URL, "ha-409", "https://x/cb"))
	})
	t.Run("DeleteClientByID 403 mismatch", func(t *testing.T) {
		db5, _ := gorm.Open(sqlite.Open("file:mem_del403?mode=memory&cache=shared"), &gorm.Config{})
		_ = db5.AutoMigrate(&dbom.OAuthKeyPair{})
		fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/v1/clients" && r.Method == "POST" {
				body, _ := io.ReadAll(r.Body)
				var req map[string]string
				_ = json.Unmarshal(body, &req)
				w.WriteHeader(201)
				_ = json.NewEncoder(w).Encode(req)
				return
			}
			if strings.HasPrefix(r.URL.Path, "/v1/clients/") && r.Method == "DELETE" {
				w.WriteHeader(403)
				w.Write([]byte(`{"error":"mismatch"}`))
				return
			}
			w.WriteHeader(404)
		}))
		defer fake.Close()
		client := service.NewBrokerClientWithHTTP(db5, fake.Client())
		_, _ = client.GetKeyPair(context.Background())
		_ = client.RegisterClient(context.Background(), fake.URL)
		require.Error(t, client.DeleteClientByID(context.Background(), fake.URL, "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"))
	})
	t.Run("RegisterInstance non-409 RegisterClient failure", func(t *testing.T) {
		db6, _ := gorm.Open(sqlite.Open("file:mem_reginst500?mode=memory&cache=shared"), &gorm.Config{})
		_ = db6.AutoMigrate(&dbom.OAuthKeyPair{})
		fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/v1/clients" {
				w.WriteHeader(500)
				w.Write([]byte(`server error`))
				return
			}
			w.WriteHeader(404)
		}))
		defer fake.Close()
		client := service.NewBrokerClientWithHTTP(db6, fake.Client())
		errE := client.RegisterInstance(context.Background(), fake.URL, "ha-fail-reg", "https://x/cb")
		require.Error(t, errE)
		require.Contains(t, errE.Error(), "500")
	})
	t.Run("RegisterInstance instance 400", func(t *testing.T) {
		db7, _ := gorm.Open(sqlite.Open("file:mem_reginst400?mode=memory&cache=shared"), &gorm.Config{})
		_ = db7.AutoMigrate(&dbom.OAuthKeyPair{})
		fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/v1/clients" {
				body, _ := io.ReadAll(r.Body)
				var req map[string]string
				_ = json.Unmarshal(body, &req)
				w.WriteHeader(201)
				_ = json.NewEncoder(w).Encode(req)
				return
			}
			if r.URL.Path == "/v1/instances/register" {
				w.WriteHeader(400)
				w.Write([]byte(`{"error":"bad redirect"}`))
				return
			}
			w.WriteHeader(404)
		}))
		defer fake.Close()
		client := service.NewBrokerClientWithHTTP(db7, fake.Client())
		errE := client.RegisterInstance(context.Background(), fake.URL, "ha-bad-redirect", "not-a-url")
		require.Error(t, errE)
		require.Contains(t, errE.Error(), "400")
	})
	t.Run("RegisterInstance GetKeyPair failure", func(t *testing.T) {
		db8, _ := gorm.Open(sqlite.Open("file:mem_reginst_kpfail?mode=memory&cache=shared"), &gorm.Config{})
		_ = db8.AutoMigrate(&dbom.OAuthKeyPair{})
		// corrupt DB to make GetKeyPair fail
		_ = db8.Create(&dbom.OAuthKeyPair{ID: "default", PrivateKey: "!!!", PublicKey: "!!!", ClientID: "bad"}).Error
		client := service.NewBrokerClientWithHTTP(db8, http.DefaultClient)
		errE := client.RegisterInstance(context.Background(), "https://example.com", "ha-kp-fail", "https://x/cb")
		require.Error(t, errE)
	})
}

func TestBrokerClient_Coverage_DeleteClientByID_Success(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open("file:mem_del_success?mode=memory&cache=shared"), &gorm.Config{})
	_ = db.AutoMigrate(&dbom.OAuthKeyPair{})
	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/clients" && r.Method == "POST" {
			body, _ := io.ReadAll(r.Body)
			var req map[string]string
			_ = json.Unmarshal(body, &req)
			w.WriteHeader(201)
			_ = json.NewEncoder(w).Encode(req)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/v1/clients/") && r.Method == "DELETE" {
			w.WriteHeader(200)
			w.Write([]byte(`{"deleted":true}`))
			return
		}
		if r.URL.Path == "/v1/clients" && r.Method == "DELETE" {
			w.WriteHeader(200)
			w.Write([]byte(`{"deleted":true}`))
			return
		}
		w.WriteHeader(404)
	}))
	defer fake.Close()
	client := service.NewBrokerClientWithHTTP(db, fake.Client())
	ctx := context.Background()
	kp, _ := client.GetKeyPair(ctx)
	_ = client.RegisterClient(ctx, fake.URL)
	// Delete by own ID should succeed and clear DB
	require.NoError(t, client.DeleteClientByID(ctx, fake.URL, kp.ClientID))
	// Verify DB cleared - next GetKeyPair should generate new key (different ClientID)
	kp2, _ := client.GetKeyPair(ctx)
	require.NotEqual(t, kp.ClientID, kp2.ClientID)
	// Delete with different ID should still succeed at broker but not clear own DB? Actually code only clears if clientID == kp.ClientID, so deleting other ID with 200 should not clear - test that
	// re-register to have valid kp2
	_ = client.RegisterClient(ctx, fake.URL)
	require.NoError(t, client.DeleteClientByID(ctx, fake.URL, kp2.ClientID))
}

func TestBrokerClient_Coverage_DoSignedAndDelete_Errors(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open("file:mem_doerr?mode=memory&cache=shared"), &gorm.Config{})
	_ = db.AutoMigrate(&dbom.OAuthKeyPair{})
	client := service.NewBrokerClientWithHTTP(db, http.DefaultClient)
	ctx := context.Background()
	_, _ = client.GetKeyPair(ctx)
	// invalid URL should cause http.NewRequest error inside doSigned
	_, _, errE := client.StartOAuth(ctx, "http://[invalid", "dropbox", "https://x/cb", "ha-err-url")
	require.Error(t, errE)
	// context canceled should cause Do error
	canceledCtx, cancel := context.WithCancel(ctx)
	cancel()
	_, _, errE = client.StartOAuth(canceledCtx, "https://example.com", "dropbox", "https://x/cb", "ha-canceled")
	require.Error(t, errE)
	// DeleteClient failure 500
	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/clients" && r.Method == "POST" {
			body, _ := io.ReadAll(r.Body)
			var req map[string]string
			_ = json.Unmarshal(body, &req)
			w.WriteHeader(201)
			_ = json.NewEncoder(w).Encode(req)
			return
		}
		w.WriteHeader(500)
		w.Write([]byte(`server error`))
	}))
	defer fake.Close()
	_ = client.RegisterClient(ctx, fake.URL)
	require.Error(t, client.DeleteClient(ctx, fake.URL))
	// GetSession with canceled context
	_, errE = client.GetSession(canceledCtx, "https://example.com", "sess-123")
	require.Error(t, errE)
}
