package p2p

import (
	"testing"
)

func TestPeerHub(t *testing.T) {
	hub := NewPeerHub()
	hub.Register("192.168.1.50", "aabbccddeeff")

	peers := hub.GetPeers()
	if len(peers) != 1 {
		t.Fatalf("expected 1 peer, got %d", len(peers))
	}

	if peers[0].IP != "192.168.1.50" || peers[0].PublicKey != "aabbccddeeff" {
		t.Errorf("peer info mismatch: %+v", peers[0])
	}
}
