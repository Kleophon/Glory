package p2p

import (
	"crypto/ed25519"
	"crypto/rand"
	"net"
	"testing"
)

func TestHandshakeIntegration(t *testing.T) {
	_, priv1, _ := ed25519.GenerateKey(rand.Reader)
	_, priv2, _ := ed25519.GenerateKey(rand.Reader)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer ln.Close()

	errChan := make(chan error, 2)
	var pub1, pub2 ed25519.PublicKey

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			errChan <- err
			return
		}
		defer conn.Close()
		var hErr error
		pub2, hErr = Handshake(conn, priv1)
		errChan <- hErr
	}()

	go func() {
		conn, err := net.Dial("tcp", ln.Addr().String())
		if err != nil {
			errChan <- err
			return
		}
		defer conn.Close()
		var hErr error
		pub1, hErr = Handshake(conn, priv2)
		errChan <- hErr
	}()

	for i := 0; i < 2; i++ {
		if err := <-errChan; err != nil {
			t.Fatalf("handshake error: %v", err)
		}
	}

	expectedPub1 := priv1.Public().(ed25519.PublicKey)
	expectedPub2 := priv2.Public().(ed25519.PublicKey)

	if string(pub1) != string(expectedPub1) {
		t.Errorf("peer 2 got incorrect public key for peer 1")
	}
	if string(pub2) != string(expectedPub2) {
		t.Errorf("peer 1 got incorrect public key for peer 2")
	}
}
