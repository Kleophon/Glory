package crypto

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"
)

func TestKeyGenerationAndSigning(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	if len(pub) != ed25519.PublicKeySize {
		t.Errorf("expected public key size %d, got %d", ed25519.PublicKeySize, len(pub))
	}
	if len(priv) != ed25519.PrivateKeySize {
		t.Errorf("expected private key size %d, got %d", ed25519.PrivateKeySize, len(priv))
	}

	msg := []byte("test-message")
	sig := ed25519.Sign(priv, msg)

	if !ed25519.Verify(pub, msg, sig) {
		t.Errorf("signature verification failed")
	}
}
