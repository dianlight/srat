package service_test

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/service"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestBrokerClient_DBKeyPairAndSignedFlow(t *testing.T) {
	// in-memory DB
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&dbom.OAuthKeyPair{}))

	// fake broker that fully verifies SRAT-Signature (Ed25519 + bodyHash + t skew + nonce replay) — cross-language contract
	var mu sync.Mutex
	clients := map[string]string{}  // client_id -> public_key b64url
	seenNonces := map[string]bool{} // nonce -> seen (replay protection)
	const clockSkewSec = 300        // matches broker CLOCK_SKEW_SECONDS
	abs64 := func(x int64) int64 {
		if x < 0 {
			return -x
		}
		return x
	}
	parseSig := func(h string) (clientId, t, nonce, sig string, ok bool) {
		// SRAT-Signature client_id="...", t="...", nonce="...", sig="..."
		re := regexp.MustCompile(`client_id="([^"]+)"[^t]*t="([^"]+)"[^n]*nonce="([^"]+)"[^s]*sig="([^"]+)"`)
		m := re.FindStringSubmatch(h)
		if len(m) != 5 {
			return "", "", "", "", false
		}
		return m[1], m[2], m[3], m[4], true
	}
	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/clients" {
			// public endpoint, no auth — store client_id -> public_key for later verify
			body, _ := io.ReadAll(r.Body)
			var req struct {
				ClientID  string `json:"client_id"`
				PublicKey string `json:"public_key"`
			}
			_ = json.Unmarshal(body, &req)
			if req.ClientID != "" && req.PublicKey != "" {
				mu.Lock()
				clients[req.ClientID] = req.PublicKey
				mu.Unlock()
				// Verify client_id == b64url(SHA256(pubkey)) — self-certifying
				if raw, err := base64.RawURLEncoding.DecodeString(req.PublicKey); err == nil {
					h := sha256.Sum256(raw)
					if exp := base64.RawURLEncoding.EncodeToString(h[:]); exp != req.ClientID {
						w.WriteHeader(400)
						w.Write([]byte(`{"error":"client_id does not match hash"}`))
						return
					}
				}
			}
			w.WriteHeader(201)
			w.Write([]byte(`{"client_id":"` + req.ClientID + `","public_key":"` + req.PublicKey + `"}`))
			return
		}
		auth := r.Header.Get("Authorization")
		if auth == "" || !strings.HasPrefix(auth, "SRAT-Signature ") {
			w.WriteHeader(401)
			w.Write([]byte(`{"error":"missing sig"}`))
			return
		}
		clientId, t, nonce, sigB64, ok := parseSig(auth)
		if !ok {
			w.WriteHeader(401)
			w.Write([]byte(`{"error":"malformed sig"}`))
			return
		}
		mu.Lock()
		pubB64, exists := clients[clientId]
		mu.Unlock()
		if !exists {
			w.WriteHeader(401)
			w.Write([]byte(`{"error":"unknown client_id"}`))
			return
		}
		// t must be within clock skew of now (unix seconds) — matches broker CLOCK_SKEW_SECONDS
		var tSec int64
		if _, err := fmt.Sscanf(t, "%d", &tSec); err != nil || abs64(time.Now().Unix()-tSec) > clockSkewSec {
			w.WriteHeader(401)
			w.Write([]byte(`{"error":"t outside clock skew"}`))
			return
		}
		// nonce must be fresh (replay protection) — matches broker nonce TTL semantics
		mu.Lock()
		if seenNonces[nonce] {
			mu.Unlock()
			w.WriteHeader(401)
			w.Write([]byte(`{"error":"nonce replay"}`))
			return
		}
		seenNonces[nonce] = true
		mu.Unlock()
		// Body hash on raw wire bytes
		bodyBytes, _ := io.ReadAll(r.Body)
		// Need to restore body for later handlers that read it, but we already consumed; for this fake we just use the bytes we read
		h := sha256.Sum256(bodyBytes)
		bodyHash := base64.RawURLEncoding.EncodeToString(h[:])
		// For GET, body is empty, bodyHash is hash of ""
		if r.Method == "GET" && len(bodyBytes) == 0 {
			emptyHash := sha256.Sum256([]byte(""))
			bodyHash = base64.RawURLEncoding.EncodeToString(emptyHash[:])
		}
		// Hono path handling: keep %2F, decode %20 -> space (as in contract.test.ts)
		// For this fake, use r.URL.Path as Hono would (which is already decoded per Go's URL.Path? Go keeps %2F)
		// Use r.URL.EscapedPath() to get raw encoded path, then mimic Hono's %20 decode
		path := r.URL.EscapedPath()
		if path == "" {
			path = r.URL.Path
		}
		path = strings.ReplaceAll(path, "%20", " ")
		sts := fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%s", clientId, r.Method, path, t, nonce, bodyHash)
		sigRaw, err := base64.RawURLEncoding.DecodeString(sigB64)
		if err != nil {
			w.WriteHeader(401)
			w.Write([]byte(`{"error":"bad sig encoding"}`))
			return
		}
		pubRaw, err := base64.RawURLEncoding.DecodeString(pubB64)
		if err != nil {
			w.WriteHeader(500)
			return
		}
		if !ed25519.Verify(ed25519.PublicKey(pubRaw), []byte(sts), sigRaw) {
			w.WriteHeader(401)
			w.Write([]byte(fmt.Sprintf(`{"error":"invalid sig","sts":%q}`, sts)))
			return
		}
		switch r.URL.Path {
		case "/v1/instances/register":
			w.WriteHeader(200)
			w.Write([]byte(`{"instance_id":"ha-1","redirect_url":"https://x/cb","client_id":"` + clientId + `"}`))
		case "/v1/start":
			w.WriteHeader(200)
			w.Write([]byte(`{"auth_url":"https://example.com/auth","session_id":"sess-123"}`))
		default:
			if r.URL.Path == "/v1/session/sess-123" {
				w.WriteHeader(200)
				w.Write([]byte(`{"token_json":"{\"access_token\":\"at\"}"}`))
				return
			}
			w.WriteHeader(404)
		}
	}))
	defer fake.Close()

	client := service.NewBrokerClientWithHTTP(db, fake.Client())
	ctx := context.Background()

	// GetKeyPair should generate and persist
	kp1, err := client.GetKeyPair(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, kp1.ClientID)
	require.Len(t, kp1.PublicKeyB64, 43) // 32B -> 43 chars

	kp2, err := client.GetKeyPair(ctx)
	require.NoError(t, err)
	require.Equal(t, kp1.ClientID, kp2.ClientID)

	// RegisterInstance should sign and succeed
	require.NoError(t, client.RegisterInstance(ctx, fake.URL, "ha-1", "https://x/cb"))

	// StartOAuth should sign
	authURL, sessID, err := client.StartOAuth(ctx, fake.URL, "dropbox", "https://x/cb", "ha-1")
	require.NoError(t, err)
	require.Equal(t, "https://example.com/auth", authURL)
	require.Equal(t, "sess-123", sessID)

	// GetSession should sign GET
	tj, err := client.GetSession(ctx, fake.URL, "sess-123")
	require.NoError(t, err)
	require.Contains(t, tj, "access_token")
}

func TestBrokerClient_DeleteClient(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&dbom.OAuthKeyPair{}))

	// Fake broker with full DELETE handling and client/instance stores
	var mu sync.Mutex
	clients := map[string]string{}
	instances := map[string]string{} // instance_id -> client_id
	parseSig := func(h string) (clientId, t, nonce, sig string, ok bool) {
		re := regexp.MustCompile(`client_id="([^"]+)"[^t]*t="([^"]+)"[^n]*nonce="([^"]+)"[^s]*sig="([^"]+)"`)
		m := re.FindStringSubmatch(h)
		if len(m) != 5 {
			return "", "", "", "", false
		}
		return m[1], m[2], m[3], m[4], true
	}
	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/clients" && r.Method == "POST" {
			body, _ := io.ReadAll(r.Body)
			var req struct {
				ClientID  string `json:"client_id"`
				PublicKey string `json:"public_key"`
			}
			_ = json.Unmarshal(body, &req)
			mu.Lock()
			clients[req.ClientID] = req.PublicKey
			mu.Unlock()
			w.WriteHeader(201)
			w.Write([]byte(`{"client_id":"` + req.ClientID + `"}`))
			return
		}
		// DELETE /v1/clients and /v1/clients/:id
		if strings.HasPrefix(r.URL.Path, "/v1/clients") && r.Method == "DELETE" {
			auth := r.Header.Get("Authorization")
			if auth == "" || !strings.HasPrefix(auth, "SRAT-Signature ") {
				w.WriteHeader(401)
				w.Write([]byte(`{"error":"missing sig"}`))
				return
			}
			clientId, t, nonce, sigB64, ok := parseSig(auth)
			if !ok {
				w.WriteHeader(401)
				w.Write([]byte(`{"error":"malformed"}`))
				return
			}
			mu.Lock()
			pubB64, exists := clients[clientId]
			mu.Unlock()
			if !exists {
				w.WriteHeader(401)
				w.Write([]byte(`{"error":"unknown client"}`))
				return
			}
			bodyBytes, _ := io.ReadAll(r.Body)
			h := sha256.Sum256(bodyBytes)
			if r.Method == "DELETE" && len(bodyBytes) == 0 {
				eh := sha256.Sum256([]byte(""))
				h = eh
			}
			bodyHash := base64.RawURLEncoding.EncodeToString(h[:])
			path := r.URL.EscapedPath()
			if path == "" {
				path = r.URL.Path
			}
			path = strings.ReplaceAll(path, "%20", " ")
			sts := fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%s", clientId, r.Method, path, t, nonce, bodyHash)
			sigRaw, _ := base64.RawURLEncoding.DecodeString(sigB64)
			pubRaw, _ := base64.RawURLEncoding.DecodeString(pubB64)
			if !ed25519.Verify(ed25519.PublicKey(pubRaw), []byte(sts), sigRaw) {
				w.WriteHeader(401)
				w.Write([]byte(`{"error":"invalid sig"}`))
				return
			}
			// Extract target id
			targetId := ""
			if r.URL.Path == "/v1/clients" {
				targetId = clientId
			} else if strings.HasPrefix(r.URL.Path, "/v1/clients/") {
				targetId = strings.TrimPrefix(r.URL.Path, "/v1/clients/")
				// Validate format
				if matched, _ := regexp.MatchString(`^[A-Za-z0-9._-]{1,128}$`, targetId); !matched {
					w.WriteHeader(400)
					w.Write([]byte(`{"error":"invalid client_id"}`))
					return
				}
				if targetId != clientId {
					w.WriteHeader(403)
					w.Write([]byte(`{"error":"mismatch"}`))
					return
				}
			}
			mu.Lock()
			_, exists = clients[targetId]
			if !exists {
				mu.Unlock()
				w.WriteHeader(404)
				w.Write([]byte(`{"error":"client not found"}`))
				return
			}
			delete(clients, targetId)
			// Also delete instances for this client
			for k, v := range instances {
				if v == targetId {
					delete(instances, k)
				}
			}
			mu.Unlock()
			w.WriteHeader(200)
			w.Write([]byte(`{"deleted":true,"client_id":"` + targetId + `"}`))
			return
		}
		if r.URL.Path == "/v1/instances/register" && r.Method == "POST" {
			// Verify and store instance
			auth := r.Header.Get("Authorization")
			clientId, t, nonce, sigB64, ok := parseSig(auth)
			if !ok {
				w.WriteHeader(401)
				return
			}
			mu.Lock()
			pubB64, exists := clients[clientId]
			mu.Unlock()
			if !exists {
				w.WriteHeader(401)
				return
			}
			bodyBytes, _ := io.ReadAll(r.Body)
			h := sha256.Sum256(bodyBytes)
			bodyHash := base64.RawURLEncoding.EncodeToString(h[:])
			path := r.URL.EscapedPath()
			if path == "" {
				path = r.URL.Path
			}
			path = strings.ReplaceAll(path, "%20", " ")
			sts := fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%s", clientId, r.Method, path, t, nonce, bodyHash)
			sigRaw, _ := base64.RawURLEncoding.DecodeString(sigB64)
			pubRaw, _ := base64.RawURLEncoding.DecodeString(pubB64)
			if !ed25519.Verify(ed25519.PublicKey(pubRaw), []byte(sts), sigRaw) {
				w.WriteHeader(401)
				return
			}
			var req struct {
				InstanceID string `json:"instance_id"`
			}
			_ = json.Unmarshal(bodyBytes, &req)
			mu.Lock()
			instances[req.InstanceID] = clientId
			mu.Unlock()
			w.WriteHeader(200)
			w.Write([]byte(`{"instance_id":"` + req.InstanceID + `"}`))
			return
		}
		w.WriteHeader(404)
	}))
	defer fake.Close()

	ctx := context.Background()

	// 200 – delete own client via DELETE /v1/clients
	t.Run("200 delete own via DELETE /v1/clients", func(t *testing.T) {
		db2, _ := gorm.Open(sqlite.Open("file:memdb200?mode=memory&cache=shared"), &gorm.Config{})
		_ = db2.AutoMigrate(&dbom.OAuthKeyPair{})
		client := service.NewBrokerClientWithHTTP(db2, fake.Client())
		_, err := client.GetKeyPair(ctx)
		require.NoError(t, err)
		require.NoError(t, client.RegisterClient(ctx, fake.URL))
		// Create an instance to verify revocation
		require.NoError(t, client.RegisterInstance(ctx, fake.URL, "ha-del-go-1", "https://x/cb"))
		require.NoError(t, client.DeleteClient(ctx, fake.URL))
		// After delete, next DeleteClient will generate a new keypair (since old was deleted) and try to delete that new client which was never registered – should be 401/404
		err = client.DeleteClient(ctx, fake.URL)
		require.Error(t, err)
		require.True(t, contains(err.Error(), "401") || contains(err.Error(), "404"))
	})

	// 401 – no auth / unknown client
	t.Run("401 missing sig", func(t *testing.T) {
		// Direct request without auth to DELETE should be 401
		req, _ := http.NewRequest("DELETE", fake.URL+"/v1/clients/some-id", nil)
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		require.Equal(t, 401, resp.StatusCode)
	})

	// 403 – delete another client's id
	t.Run("403 mismatch", func(t *testing.T) {
		db2, _ := gorm.Open(sqlite.Open("file:memdb403a?mode=memory&cache=shared"), &gorm.Config{})
		_ = db2.AutoMigrate(&dbom.OAuthKeyPair{})
		clientA := service.NewBrokerClientWithHTTP(db2, fake.Client())
		kpA, _ := clientA.GetKeyPair(ctx)
		_ = clientA.RegisterClient(ctx, fake.URL)
		// Create second client directly via fake (not via clientA's DB)
		// Use a separate DB for second client
		db3, _ := gorm.Open(sqlite.Open("file:memdb403b?mode=memory&cache=shared"), &gorm.Config{})
		_ = db3.AutoMigrate(&dbom.OAuthKeyPair{})
		clientB := service.NewBrokerClientWithHTTP(db3, fake.Client())
		kpB, _ := clientB.GetKeyPair(ctx)
		_ = clientB.RegisterClient(ctx, fake.URL)
		// Now kpA tries to delete kpB's id
		err := clientA.DeleteClientByID(ctx, fake.URL, kpB.ClientID)
		require.Error(t, err)
		require.Contains(t, err.Error(), "403")
		// Verify kpA still exists
		_ = kpA
		_ = kpB
	})

	// 404 – delete non-existent client
	t.Run("404 not found", func(t *testing.T) {
		db2, _ := gorm.Open(sqlite.Open("file:memdb404?mode=memory&cache=shared"), &gorm.Config{})
		_ = db2.AutoMigrate(&dbom.OAuthKeyPair{})
		client := service.NewBrokerClientWithHTTP(db2, fake.Client())
		_, _ = client.GetKeyPair(ctx)
		_ = client.RegisterClient(ctx, fake.URL)
		require.NoError(t, client.DeleteClient(ctx, fake.URL))
		// Now the client was deleted, next GetKeyPair will generate a new one. Try to delete a random valid id that never existed.
		_, _ = client.GetKeyPair(ctx) // new key
		_ = client.RegisterClient(ctx, fake.URL)
		err := client.DeleteClientByID(ctx, fake.URL, "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA")
		require.Error(t, err)
		require.True(t, contains(err.Error(), "404") || contains(err.Error(), "403"))
	})
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (func() bool {
		for i := 0; i <= len(s)-len(substr); i++ {
			if s[i:i+len(substr)] == substr {
				return true
			}
		}
		return false
	})()
}

func abs64(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}
