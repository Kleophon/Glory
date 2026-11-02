package p2p

import (
	"io"
	"net"
	"sync"
)

// Tunnel proxies raw TCP traffic bidirectionally between the established connection
// and a local service (like a Minecraft server or port).
func Tunnel(conn net.Conn, localTarget string) error {
	localConn, err := net.Dial("tcp", localTarget)
	if err != nil {
		return err
	}
	defer localConn.Close()

	TunnelDirect(conn, localConn)
	return nil
}

// TunnelDirect proxies traffic bidirectionally between two active connections.
func TunnelDirect(c1, c2 net.Conn) {
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		io.Copy(c1, c2)
		if tc, ok := c1.(*net.TCPConn); ok {
			tc.CloseWrite()
		}
	}()

	go func() {
		defer wg.Done()
		io.Copy(c2, c1)
		if tc, ok := c2.(*net.TCPConn); ok {
			tc.CloseWrite()
		}
	}()

	wg.Wait()
}
