package oauthkeys

import (
	"crypto/ed25519"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateKeyPair_UniqueAndDeterministicClientID(t *testing.T) {
	kp1, err := GenerateKeyPair()
	require.NoError(t, err)
	require.NotEmpty(t, kp1.ClientID)
	require.Len(t, kp1.PublicKey, ed25519.PublicKeySize)
	require.Len(t, kp1.PrivateKey, ed25519.PrivateKeySize)
	kp2, err := GenerateKeyPair()
	require.NoError(t, err)
	assert.NotEqual(t, kp1.ClientID, kp2.ClientID)
	assert.NotEqual(t, kp1.PublicKeyB64, kp2.PublicKeyB64)
}

func TestComputeClientID(t *testing.T) {
	kp, err := generateKeyPair()
	require.NoError(t, err)
	cid, err := ComputeClientID(kp.PublicKeyB64)
	require.NoError(t, err)
	assert.Equal(t, kp.ClientID, cid)
	assert.Regexp(t, `^[A-Za-z0-9_-]{43}$`, cid)
}

func TestSignAndVerify(t *testing.T) {
	kp, err := generateKeyPair()
	require.NoError(t, err)
	method := "POST"
	path := "/v1/instances/register"
	body := `{"instance_id":"ha-1","redirect_url":"https://srat.example/cb"}`
	tStr := "1700000000"
	nonce := "test-nonce-1234567890123456"
	bh := BodyHash(body)
	msg := StringToSign(kp.ClientID, method, path, tStr, nonce, bh)
	sigB64 := kp.Sign(method, path, tStr, nonce, body)
	// verify via ed25519
	sigBytes, err := base64URLDecode(sigB64)
	require.NoError(t, err)
	msgBytes := []byte(msg)
	assert.True(t, ed25519.Verify(kp.PublicKey, msgBytes, sigBytes))
	// tamper
	assert.False(t, ed25519.Verify(kp.PublicKey, []byte(msg+"x"), sigBytes))
}

func TestAuthHeaderFormat(t *testing.T) {
	kp, err := generateKeyPair()
	require.NoError(t, err)
	h := kp.AuthHeader("POST", "/v1/start", `{"provider":"dropbox"}`, "1700000000", "nonce-abc-1234567890")
	assert.Contains(t, h, `client_id="`+kp.ClientID+`"`)
	assert.Contains(t, h, `t="1700000000"`)
	assert.Contains(t, h, `nonce="nonce-abc-1234567890"`)
	assert.Contains(t, h, `sig="`)
}

// TestInterop_FixedVectorsMatchTS pins the cross-language contract with the TS broker
// (oauth_broker/tests/interop.test.ts). The seed is SHA256("srat-interop-test-seed");
// both implementations derive the same key and, because Ed25519 is deterministic,
// must produce the exact same signature for the same StringToSign. If the StringToSign
// canonicalization, base64url encoding, or key derivation diverges on either side,
// this test fails.
func TestInterop_FixedVectorsMatchTS(t *testing.T) {
	const seedB64 = "WFke_MEL51VDHYP4vsoahH-OVFitV_1F2R5gMXUAm2U"
	const pubB64 = "eY33ep1dYLmq54IAFGk8jTiCT0Exsi0EzUyNPot4ddU"
	const clientID = "l7wo8EkfW8XQhpuez21Gt3aVd0Esp9f_tgN0OH7JEXA"
	const body = `{"instance_id":"ha-interop","redirect_url":"https://srat.example/cb"}`
	const bodyHash = "3-LZCHH-aCv07EN2EhpnZgbsNAAVBPz6R3KTFxU0FEI"
	const tStr = "1767225600"
	const nonce = "interop-nonce-20260101"
	const expectSig = "NS-toNCT7P3y3n_Fs9XlK3R8iJUeBo38vq6ZBhd5x4TzU5KP6S3RRUlShTCg6cQX7rt8BCD4NQP6eAJzQq9TAg"

	seed, err := base64URLDecode(seedB64)
	require.NoError(t, err)
	priv := PrivateKeyFromSeed(seed)
	pub := priv.Public().(ed25519.PublicKey)

	// Same public key derivation as TS (raw 32B -> b64url)
	assert.Equal(t, pubB64, base64URLEncode([]byte(pub)))

	// Same client_id derivation as TS computeClientId
	cid, err := ComputeClientID(pubB64)
	require.NoError(t, err)
	assert.Equal(t, clientID, cid)

	// Same bodyHash as TS bodyHashBase64Url (raw wire bytes)
	assert.Equal(t, bodyHash, BodyHash(body))

	// Reconstruct the exact StringToSign both sides sign
	sts := StringToSign(clientID, "POST", "/v1/instances/register", tStr, nonce, bodyHash)
	expectedSTS := "l7wo8EkfW8XQhpuez21Gt3aVd0Esp9f_tgN0OH7JEXA\nPOST\n/v1/instances/register\n1767225600\ninterop-nonce-20260101\n" + bodyHash
	assert.Equal(t, expectedSTS, sts)

	// Deterministic signature must match the TS-generated vector exactly
	kp := &KeyPair{
		PrivateKeyB64: seedB64,
		PublicKeyB64:  pubB64,
		ClientID:      clientID,
		PrivateKey:    priv,
		PublicKey:     pub,
	}
	assert.Equal(t, expectSig, kp.Sign("POST", "/v1/instances/register", tStr, nonce, body))

	// And it verifies with the raw public key
	sigBytes, err := base64URLDecode(expectSig)
	require.NoError(t, err)
	assert.True(t, ed25519.Verify(pub, []byte(sts), sigBytes))
	assert.False(t, ed25519.Verify(pub, []byte(sts+"x"), sigBytes))
}
