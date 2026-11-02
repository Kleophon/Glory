package p2p

import (
	"fmt"
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

	// Broadcast beacon
	broadcastAddr, err := net.ResolveUDPAddr("udp4", "255.255.255.255:"+DiscoveryPort)
	if err != nil {
		// Fallback to loopback/local if broadcast fails
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
