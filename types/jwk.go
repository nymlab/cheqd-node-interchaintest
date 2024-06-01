package sdjwttypes

type OkpJwk struct {
	// KeyType represents the cryptographic algorithm used with the key.
	KeyType string `json:"kty"`
	// KeyID is the unique identifier for the key.
	KeyID string `json:"kid,omitempty"`
	// PublicKeyUse specifies the intended use of the public key.
	PublicKeyUse string `json:"use,omitempty"`
	// Algorithm specifies the cryptographic algorithm used with the key.
	Algorithm string `json:"alg,omitempty"`
	// KeyOperations specifies the operation(s) for which the key is intended to be used.
	KeyOperations []string `json:"key_ops,omitempty"`
	// Curve represents the curve used by the key.
	Curve string `json:"crv,omitempty"`
	// X represents the x-coordinate for the elliptic curve point.
	X string `json:"x,omitempty"`
}
