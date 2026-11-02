package p2p

import (
	"bufio"
	"fmt"
	"net"
	"os"
)

// Chat pipes stdin to the peer and the peer's messages to the screen.
func Chat(conn net.Conn) {
	done := make(chan struct{})

	go func() {
		sc := bufio.NewScanner(conn)
		for sc.Scan() {
			fmt.Printf("\r\033[Kpeer> %s\nme> ", sc.Text())
		}
		close(done)
	}()

	go func() {
		sc := bufio.NewScanner(os.Stdin)
		fmt.Print("me> ")
		for sc.Scan() {
			text := sc.Text()
			if text != "" {
				if _, err := fmt.Fprintln(conn, text); err != nil {
					break
				}
			}
			fmt.Print("me> ")
		}
		conn.Close()
	}()

	<-done
	conn.Close()
	fmt.Println("\n[Connection closed]")
}
