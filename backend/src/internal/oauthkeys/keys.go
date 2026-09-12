package oauthkeys

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strings"
)

// KeyPair holds raw Ed25519 keys plus derived client_id
type KeyPair struct {
	PrivateKeyB64 string // base64url raw 32B private seed
	PublicKeyB64  string // base64url raw 32B public
	ClientID      string // base64url SHA256(public)
	PrivateKey    ed25519.PrivateKey
	PublicKey     ed25519.PublicKey
}

func base64URLEncode(b []byte) string {
	return base64.RawURLEncoding.EncodeToString(b)
}

func base64URLDecode(s string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(s)
}

// ComputeClientID returns base64url(SHA256(rawPubkey))
func ComputeClientID(publicKeyB64 string) (string, error) {
	raw, err := base64URLDecode(publicKeyB64)
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(raw)
	return base64URLEncode(h[:]), nil
}

func generateKeyPair() (*KeyPair, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	seed := priv.Seed()
	pubB64 := base64URLEncode([]byte(pub))
	privB64 := base64URLEncode(seed)
	h := sha256.Sum256([]byte(pub))
	cid := base64URLEncode(h[:])
	return &KeyPair{PrivateKeyB64: privB64, PublicKeyB64: pubB64, ClientID: cid, PrivateKey: priv, PublicKey: pub}, nil
}

// GenerateKeyPair is exported for service/DB initialization
func GenerateKeyPair() (*KeyPair, error) { return generateKeyPair() }

func PrivateKeyFromSeed(seed []byte) ed25519.PrivateKey { return ed25519.NewKeyFromSeed(seed) }

func PublicKeyFromBytes(pubRaw []byte) ed25519.PublicKey { return ed25519.PublicKey(pubRaw) }

// BodyHash returns base64url(SHA256(body))
func BodyHash(body string) string {
	h := sha256.Sum256([]byte(body))
	return base64URLEncode(h[:])
}

// StringToSign builds the broker stringToSign (raw wire bytes bodyHash, method uppercased, path without query)
func StringToSign(clientID, method, path, t, nonce, bodyHash string) string {
	return fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%s", clientID, strings.ToUpper(method), path, t, nonce, bodyHash)
}

// Sign returns base64url Ed25519 signature of StringToSign
func (k *KeyPair) Sign(method, path, t, nonce, body string) string {
	bh := BodyHash(body)
	msg := StringToSign(k.ClientID, method, path, t, nonce, bh)
	sig := ed25519.Sign(k.PrivateKey, []byte(msg))
	return base64URLEncode(sig)
}

// AuthHeader returns the Authorization header value
func (k *KeyPair) AuthHeader(method, path, body string, t, nonce string) string {
	sig := k.Sign(method, path, t, nonce, body)
	return fmt.Sprintf(`SRAT-Signature client_id="%s", t="%s", nonce="%s", sig="%s"`, k.ClientID, t, nonce, sig)
}
