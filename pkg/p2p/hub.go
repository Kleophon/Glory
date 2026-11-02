package p2p

import (
	"encoding/json"
	"sync"
	"time"
)

type PeerInfo struct {
	IP        string    `json:"ip"`
	PublicKey string    `json:"public_key"`
	LastSeen  time.Time `json:"last_seen"`
}

type PeerHub struct {
	mu    sync.Mutex
	peers map[string]PeerInfo
}

func NewPeerHub() *PeerHub {
	return &PeerHub{
		peers: make(map[string]PeerInfo),
	}
}

func (h *PeerHub) Register(ip, pubKey string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.peers[pubKey] = PeerInfo{
		IP:        ip,
		PublicKey: pubKey,
		LastSeen:  time.Now(),
	}
}

func (h *PeerHub) GetPeers() []PeerInfo {
	h.mu.Lock()
	defer h.mu.Unlock()

	var list []PeerInfo
	now := time.Now()
	for k, p := range h.peers {
		// prune inactive after 5 mins
		if now.Sub(p.LastSeen) < 5*time.Minute {
			list = append(list, p)
		} else {
			delete(h.peers, k)
		}
	}
	return list
}

func (h *PeerHub) ToJSON() []byte {
	peers := h.GetPeers()
	data, _ := json.MarshalIndent(peers, "", "  ")
	return data
}
