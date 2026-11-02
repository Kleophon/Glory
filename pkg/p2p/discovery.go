package p2p

import (
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"glory/pkg/config"
	"net"
	"time"
)

const DiscoveryPort = "4243"
const DiscoveryMsg = "GLORY_DISCOVER_V1"

// ScanLocalNetwork broadcasts a UDP ping on the local subnet to find other running Glory peers.
func ScanLocalNetwork(myPub string) []string {
	var foundPeers []string

	l, err := net.ListenPacket("udp4", ":0")
	if err != nil {
		fmt.Println("Error starting network scanner:", err)
		return nil
	}
	defer l.Close()

	l.SetDeadline(time.Now().Add(2 * time.Second))

	broadcastAddr, err := net.ResolveUDPAddr("udp4", "255.255.255.255:"+DiscoveryPort)
	if err != nil {
		return nil
	}

	payload := []byte(fmt.Sprintf("%s:%s", DiscoveryMsg, myPub))
	_, _ = l.WriteTo(payload, broadcastAddr)

	buf := make([]byte, 1024)
	for {
		n, remoteAddr, err := l.ReadFrom(buf)
		if err != nil {
			break
		}
		msg := string(buf[:n])
		if len(msg) > len(DiscoveryMsg) {
			peerKey := msg[len(DiscoveryMsg)+1:]
			peerInfo := fmt.Sprintf("%s (Key: %s...)", remoteAddr.(*net.UDPAddr).IP.String(), peerKey[:12])
			foundPeers = append(foundPeers, peerInfo)
		}
	}

	return foundPeers
}

// StartDiscoveryResponder listens for UDP discovery beacons and responds if authorized by config.
func StartDiscoveryResponder(priv ed25519.PrivateKey, cfg *config.Config) {
	addr, err := net.ResolveUDPAddr("udp4", ":"+DiscoveryPort)
	if err != nil {
		return
	}

	conn, err := net.ListenUDP("udp4", addr)
	if err != nil {
		return
	}
	defer conn.Close()

	pubHex := hex.EncodeToString(priv.Public().(ed25519.PublicKey))
	buf := make([]byte, 1024)

	for {
		n, remoteAddr, err := conn.ReadFrom(buf)
		if err != nil {
			continue
		}

		msg := string(buf[:n])
		if len(msg) > len(DiscoveryMsg) {
			parts := msg[len(DiscoveryMsg)+1:]
			// Check if querier is allowed (if private, must be a friend)
			if cfg.IsAllowed(parts) {
				response := []byte(fmt.Sprintf("%s:%s", DiscoveryMsg, pubHex))
				_, _ = conn.WriteTo(response, remoteAddr)
			}
		}
	}
}
