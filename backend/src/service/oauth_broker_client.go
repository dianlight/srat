package service

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/internal/oauthkeys"
	"gitlab.com/tozd/go/errors"
	"gorm.io/gorm"
)

type BrokerClientInterface interface {
	GetKeyPair(ctx context.Context) (*oauthkeys.KeyPair, errors.E)
	RegisterClient(ctx context.Context, brokerURL string) errors.E
	RegisterInstance(ctx context.Context, brokerURL, instanceID, redirectURL string) errors.E
	StartOAuth(ctx context.Context, brokerURL, provider, callbackURL, instanceID string) (authURL string, sessionID string, errE errors.E)
	GetSession(ctx context.Context, brokerURL, sessionID string) (tokenJSON string, errE errors.E)
	DeleteClient(ctx context.Context, brokerURL string) errors.E
	DeleteClientByID(ctx context.Context, brokerURL, clientID string) errors.E
}

type brokerClient struct {
	db         *gorm.DB
	httpClient *http.Client
}

func NewBrokerClient(db *gorm.DB) BrokerClientInterface {
	return &brokerClient{
		db:         db,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// NewBrokerClientWithHTTP is for tests injection
func NewBrokerClientWithHTTP(db *gorm.DB, hc *http.Client) BrokerClientInterface {
	return &brokerClient{db: db, httpClient: hc}
}

func (b *brokerClient) GetKeyPair(ctx context.Context) (*oauthkeys.KeyPair, errors.E) {
	var row dbom.OAuthKeyPair
	err := b.db.WithContext(ctx).First(&row, "id = ?", "default").Error
	if err == nil {
		seed, err2 := base64.RawURLEncoding.DecodeString(row.PrivateKey)
		if err2 != nil {
			return nil, errors.WithStack(err2)
		}
		pubRaw, err2 := base64.RawURLEncoding.DecodeString(row.PublicKey)
		if err2 != nil {
			return nil, errors.WithStack(err2)
		}
		priv := oauthkeys.PrivateKeyFromSeed(seed)
		pub := oauthkeys.PublicKeyFromBytes(pubRaw)
		return &oauthkeys.KeyPair{
			PrivateKeyB64: row.PrivateKey,
			PublicKeyB64:  row.PublicKey,
			ClientID:      row.ClientID,
			PrivateKey:    priv,
			PublicKey:     pub,
		}, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.WithStack(err)
	}
	// generate new
	kp, err := oauthkeys.GenerateKeyPair()
	if err != nil {
		return nil, errors.WithStack(err)
	}
	row = dbom.OAuthKeyPair{
		ID:         "default",
		PrivateKey: kp.PrivateKeyB64,
		PublicKey:  kp.PublicKeyB64,
		ClientID:   kp.ClientID,
	}
	if err := b.db.WithContext(ctx).Create(&row).Error; err != nil {
		return nil, errors.WithStack(err)
	}
	return kp, nil
}

func (b *brokerClient) doSigned(ctx context.Context, method, url, body string, kp *oauthkeys.KeyPair) (*http.Response, errors.E) {
	t := fmt.Sprintf("%d", time.Now().Unix())
	nonceBytes := make([]byte, 16)
	if _, err := rand.Read(nonceBytes); err != nil {
		return nil, errors.WithStack(err)
	}
	nonce := base64.RawURLEncoding.EncodeToString(nonceBytes) + fmt.Sprintf("-%d", time.Now().UnixNano()%1000)
	// ensure nonce >=16 chars and matches regex
	if len(nonce) < 16 {
		nonce = nonce + "0000000000000000"
	}
	// Build path for signing (only path, not host/query)
	path := urlPath(url)
	var bodyStr string = body
	if bodyStr == "" && method == "GET" {
		bodyStr = ""
	}
	authHeader := kp.AuthHeader(method, path, bodyStr, t, nonce)
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewBufferString(body))
	if err != nil {
		return nil, errors.WithStack(err)
	}
	req.Header.Set("Authorization", authHeader)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := b.httpClient.Do(req)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return resp, nil
}

func urlPath(full string) string {
	// extract path from full URL for signing (must match broker's c.req.path)
	if idx := strings.Index(full, "://"); idx >= 0 {
		rest := full[idx+3:]
		if k := strings.Index(rest, "/"); k >= 0 {
			p := rest[k:]
			if q := strings.Index(p, "?"); q >= 0 {
				return p[:q]
			}
			return p
		}
		return "/"
	}
	if len(full) == 0 {
		return "/"
	}
	if full[0] != '/' {
		full = "/" + full
	}
	if q := strings.Index(full, "?"); q >= 0 {
		return full[:q]
	}
	return full
}

func (b *brokerClient) RegisterClient(ctx context.Context, brokerURL string) errors.E {
	kp, errE := b.GetKeyPair(ctx)
	if errE != nil {
		return errE
	}
	url := brokerURL + "/v1/clients"
	bodyMap := map[string]string{"client_id": kp.ClientID, "public_key": kp.PublicKeyB64}
	bodyBytes, _ := json.Marshal(bodyMap)
	body := string(bodyBytes)
	// client registration is public, no signature
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBufferString(body))
	if err != nil {
		return errors.WithStack(err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := b.httpClient.Do(req)
	if err != nil {
		return errors.WithStack(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 201 && resp.StatusCode != 200 {
		bts, _ := io.ReadAll(resp.Body)
		return errors.Errorf("register client failed %d: %s", resp.StatusCode, string(bts))
	}
	return nil
}

func (b *brokerClient) RegisterInstance(ctx context.Context, brokerURL, instanceID, redirectURL string) errors.E {
	kp, errE := b.GetKeyPair(ctx)
	if errE != nil {
		return errE
	}
	// ensure client registered
	if errE := b.RegisterClient(ctx, brokerURL); errE != nil {
		// 409 is ok (already exists) – RegisterClient handles it as success only on 200/201, but we should tolerate 409
		// try to ignore 409 here: if RegisterClient returns error containing 409, continue
		if !isAlreadyExists(errE) {
			return errE
		}
	}
	url := brokerURL + "/v1/instances/register"
	bodyMap := map[string]string{"instance_id": instanceID, "redirect_url": redirectURL}
	bodyBytes, _ := json.Marshal(bodyMap)
	body := string(bodyBytes)
	resp, errE := b.doSigned(ctx, "POST", url, body, kp)
	if errE != nil {
		return errE
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		bts, _ := io.ReadAll(resp.Body)
		return errors.Errorf("register instance failed %d: %s", resp.StatusCode, string(bts))
	}
	return nil
}

func (b *brokerClient) StartOAuth(ctx context.Context, brokerURL, provider, callbackURL, instanceID string) (string, string, errors.E) {
	kp, errE := b.GetKeyPair(ctx)
	if errE != nil {
		return "", "", errE
	}
	url := brokerURL + "/v1/start"
	bodyMap := map[string]string{"provider": provider, "srat_callback_url": callbackURL, "instance_id": instanceID}
	bodyBytes, _ := json.Marshal(bodyMap)
	body := string(bodyBytes)
	resp, errE := b.doSigned(ctx, "POST", url, body, kp)
	if errE != nil {
		return "", "", errE
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		bts, _ := io.ReadAll(resp.Body)
		return "", "", errors.Errorf("start oauth failed %d: %s", resp.StatusCode, string(bts))
	}
	var out struct {
		AuthURL   string `json:"auth_url"`
		SessionID string `json:"session_id"`
	}
	bts, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", errors.WithStack(err)
	}
	if err := json.Unmarshal(bts, &out); err != nil {
		return "", "", errors.WithStack(err)
	}
	return out.AuthURL, out.SessionID, nil
}

func (b *brokerClient) GetSession(ctx context.Context, brokerURL, sessionID string) (string, errors.E) {
	kp, errE := b.GetKeyPair(ctx)
	if errE != nil {
		return "", errE
	}
	url := brokerURL + "/v1/session/" + sessionID
	resp, errE := b.doSigned(ctx, "GET", url, "", kp)
	if errE != nil {
		return "", errE
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		bts, _ := io.ReadAll(resp.Body)
		return "", errors.Errorf("get session failed %d: %s", resp.StatusCode, string(bts))
	}
	bts, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", errors.WithStack(err)
	}
	var out struct {
		TokenJSON string `json:"token_json"`
	}
	if err := json.Unmarshal(bts, &out); err != nil {
		return "", errors.WithStack(err)
	}
	return out.TokenJSON, nil
}

func (b *brokerClient) DeleteClient(ctx context.Context, brokerURL string) errors.E {
	kp, errE := b.GetKeyPair(ctx)
	if errE != nil {
		return errE
	}
	url := brokerURL + "/v1/clients"
	resp, errE := b.doSigned(ctx, "DELETE", url, "", kp)
	if errE != nil {
		return errE
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		bts, _ := io.ReadAll(resp.Body)
		return errors.Errorf("delete client failed %d: %s", resp.StatusCode, string(bts))
	}
	_ = b.db.WithContext(ctx).Delete(&dbom.OAuthKeyPair{}, "id = ?", "default").Error
	_ = b.db.WithContext(ctx).Delete(&dbom.OAuthKeyPair{}, "client_id = ?", kp.ClientID).Error
	return nil
}

func (b *brokerClient) DeleteClientByID(ctx context.Context, brokerURL, clientID string) errors.E {
	kp, errE := b.GetKeyPair(ctx)
	if errE != nil {
		return errE
	}
	// Use the provided clientID for the path, but sign with the authenticated kp's clientId
	// The broker will verify that the path id matches the authenticated clientId (403 otherwise)
	url := brokerURL + "/v1/clients/" + clientID
	// Also support DELETE /v1/clients (without id) when clientID == kp.ClientID
	if clientID == kp.ClientID {
		// Try both? Use the :id variant as primary
	}
	resp, errE := b.doSigned(ctx, "DELETE", url, "", kp)
	if errE != nil {
		return errE
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		bts, _ := io.ReadAll(resp.Body)
		return errors.Errorf("delete client failed %d: %s", resp.StatusCode, string(bts))
	}
	// Also delete local DB entry for this client (so next GetKeyPair generates a new one)
	// Only delete if we deleted our own client
	if clientID == kp.ClientID {
		_ = b.db.WithContext(ctx).Delete(&dbom.OAuthKeyPair{}, "id = ?", "default").Error
		// Also try delete by client_id in case id is client_id (for future)
		_ = b.db.WithContext(ctx).Delete(&dbom.OAuthKeyPair{}, "client_id = ?", clientID).Error
	}
	return nil
}

func isAlreadyExists(errE errors.E) bool {
	if errE == nil {
		return false
	}
	s := errE.Error()
	return strings.Contains(s, "409") || strings.Contains(s, "already registered")
}

// Ensure broker URL from env fallback
func BrokerURLFromEnv() string {
	if v := os.Getenv("BROKER_PUBLIC_URL"); v != "" {
		return v
	}
	if v := os.Getenv("SRAT_BROKER_URL"); v != "" {
		return v
	}
	return "https://srat-oauth-broker-production.lucio-tarantino.workers.dev"
}
