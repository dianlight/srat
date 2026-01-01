package updatekey

import _ "embed"

// UpdatePublicKey contains the Ed25519 public key used to verify binary update signatures.
// This key is embedded at build time from docs/update-public-key.pem.
// The corresponding private key is stored securely in GitHub secrets and used during
// the build process to sign release binaries.
//
//go:embed ../../../docs/update-public-key.pem
var UpdatePublicKey string
