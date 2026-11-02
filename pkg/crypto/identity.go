package crypto

import (
	"crypto/ed25519"
	"crypto/rand"
	"os"
	"path/filepath"
)

const HandshakeCtx = "glory-handshake-v1"

// LoadKey loads the identity key from the config dir, or creates one.
func LoadKey() (ed25519.PrivateKey, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(dir, "glory", "key")

	if seed, err := os.ReadFile(path); err == nil && len(seed) == ed25519.SeedSize {
		return ed25519.NewKeyFromSeed(seed), nil
	}

	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	if err := os.WriteFile(path, priv.Seed(), 0o600); err != nil {
		return nil, err
	}
	return priv, nil
}
