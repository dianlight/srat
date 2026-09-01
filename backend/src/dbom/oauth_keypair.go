package dbom

import "time"

// OAuthKeyPair stores the per-install Ed25519 keypair for broker SRAT-Signature auth.
// Single row with ID="default" per installation; client_id is derived as base64url(SHA256(pubkey)).
type OAuthKeyPair struct {
	ID         string `gorm:"primaryKey"` // fixed "default"
	PrivateKey string `gorm:"not null"`   // base64url raw 32B seed
	PublicKey  string `gorm:"not null"`   // base64url raw 32B pub
	ClientID   string `gorm:"not null;uniqueIndex"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
