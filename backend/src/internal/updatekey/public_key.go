package updatekey

import _ "embed"

// UpdatePublicKey contains the Minisign public key used to verify binary update signatures.
// This key is embedded at build time from the local update-public-key.pub file.
// The corresponding private key is stored securely in GitHub secrets and used during
// the build process to sign release binaries.
// Format: Minisign public key (compatible with github.com/minio/selfupdate)
//
//go:embed update-public-key.pub
var UpdatePublicKey string
