package p2p

import (
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"glory/pkg/crypto"
	"io"
	"net"
	"time"
)

// Handshake swaps public keys and proves each side owns its key
// by signing a random challenge chosen by the other side.
func Handshake(conn net.Conn, priv ed25519.PrivateKey) (ed25519.PublicKey, error) {
	// Ignore deadline error if connection type doesn't support deadlines (e.g. net.Pipe)
	_ = conn.SetDeadline(time.Now().Add(15 * time.Second))
	defer conn.SetDeadline(time.Time{})

	pub := priv.Public().(ed25519.PublicKey)
	nonce := make([]byte, 32)
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	// send: pubkey (32) + nonce (32)
	if _, err := conn.Write(append(append([]byte{}, pub...), nonce...)); err != nil {
		return nil, err
	}

	hello := make([]byte, 64)
	if _, err := io.ReadFull(conn, hello); err != nil {
		return nil, err
	}
	peerPub := ed25519.PublicKey(hello[:32])
	peerNonce := hello[32:]

	// sign the peer's challenge, verify the peer signed ours
	sig := ed25519.Sign(priv, append([]byte(crypto.HandshakeCtx), peerNonce...))
	if _, err := conn.Write(sig); err != nil {
		return nil, err
	}

	peerSig := make([]byte, ed25519.SignatureSize)
	if _, err := io.ReadFull(conn, peerSig); err != nil {
		return nil, err
	}

	if !ed25519.Verify(peerPub, append([]byte(crypto.HandshakeCtx), nonce...), peerSig) {
		return nil, errors.New("peer failed key verification")
	}

	return peerPub, nil
}
