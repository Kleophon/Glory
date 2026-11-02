package cli

import (
	"crypto/ed25519"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"glory/pkg/crypto"
	"glory/pkg/p2p"
	"net"
	"os"
	"strings"
	"time"
)

const (
	Version     = "0.4.0"
	DefaultPort = "4242"
	Author      = "@kleophonx"
)

func Run() {
	if len(os.Args) < 2 {
		PrintUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	switch cmd {
	case "-v", "--version", "version":
		fmt.Printf("glory v%s (Secure P2P Peer Network)\n", Version)
		fmt.Printf("Author: %s\n", Author)
		return
	case "-h", "--help", "help":
		PrintUsage()
		return
	}

	priv, err := crypto.LoadKey()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error loading/generating identity key:", err)
		os.Exit(1)
	}

	switch cmd {
	case "start":
		err = cmdStart(priv, os.Args[2:])
	case "connect":
		err = cmdConnect(priv, os.Args[2:])
	case "scan":
		err = cmdScan(priv)
	case "server":
		err = cmdServer(os.Args[2:])
	case "list":
		err = cmdList(os.Args[2:])
	case "key":
		fmt.Printf("Public Key: %s\n", hex.EncodeToString(priv.Public().(ed25519.PublicKey)))
		return
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", cmd)
		PrintUsage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func cmdStart(priv ed25519.PrivateKey, args []string) error {
	fs := flag.NewFlagSet("start", flag.ExitOnError)
	port := fs.String("port", DefaultPort, "Port to listen on for incoming peer connections")
	hubAddr := fs.String("hub", "", "Optional Oracle server hub address to register this peer with")

	fs.Usage = func() {
		fmt.Printf("Glory v%s - Start Command\n", Version)
		fmt.Printf("Author: %s\n\n", Author)
		fmt.Println("Usage:")
		fmt.Println("  glory start [options]")
		fmt.Println("\nDescription:")
		fmt.Println("  Starts a secure P2P node and waits for incoming connections.")
		fmt.Println("  Performs cryptographic Ed25519 mutual authentication on every connection.")
		fmt.Println("  Optionally registers with your Oracle server hub so others can discover your node.")
	}
	fs.Parse(args)

	listener, err := net.Listen("tcp", ":"+*port)
	if err != nil {
		return fmt.Errorf("failed to bind to port %s: %w", *port, err)
	}
	defer listener.Close()

	pubHex := hex.EncodeToString(priv.Public().(ed25519.PublicKey))

	fmt.Println("==================================================")
	fmt.Printf("  Glory P2P Node Started (v%s)\n", Version)
	fmt.Printf("  Author: %s\n", Author)
	fmt.Println("==================================================")
	fmt.Printf("Your Identity Key : %s\n", pubHex)
	fmt.Printf("Listening Address : 0.0.0.0:%s\n", *port)
	if *hubAddr != "" {
		fmt.Printf("Hub Registry      : %s (Registering...)\n", *hubAddr)
		go registerWithHub(*hubAddr, pubHex, *port)
	}
	fmt.Println("--------------------------------------------------")
	fmt.Println("Waiting for incoming peer connections...")
	fmt.Println("==================================================")

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Accept error:", err)
			continue
		}

		remoteAddr := conn.RemoteAddr().String()
		fmt.Printf("\n[+] Incoming connection from %s\n", remoteAddr)

		peerPub, err := p2p.Handshake(conn, priv)
		if err != nil {
			fmt.Printf("[-] Handshake failed with %s: %v\n", remoteAddr, err)
			conn.Close()
			continue
		}

		peerHex := hex.EncodeToString(peerPub)
		fmt.Printf("[✓] Handshake successful!\n")
		fmt.Printf("[✓] Peer Public Key: %s\n", peerHex)
		fmt.Println("--------------------------------------------------")

		p2p.Chat(conn)
		fmt.Println("\n[*] Resuming listener, waiting for new connections...")
	}
}

func registerWithHub(hubAddr, pubKey, port string) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		// Simple HTTP or TCP ping to hub (or tcp dial)
		conn, err := net.DialTimeout("tcp", hubAddr, 5*time.Second)
		if err == nil {
			fmt.Fprintf(conn, "REGISTER %s %s\n", pubKey, port)
			conn.Close()
		}
	}
}

func cmdConnect(priv ed25519.PrivateKey, args []string) error {
	fs := flag.NewFlagSet("connect", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Printf("Glory v%s - Connect Command\n", Version)
		fmt.Printf("Author: %s\n\n", Author)
		fmt.Println("Usage:")
		fmt.Println("  glory connect <ip[:port]> [peer-key]")
		fmt.Println("\nDescription:")
		fmt.Println("  Connects securely to a remote Glory peer over TCP with Ed25519 authentication.")
	}
	fs.Parse(args)

	remaining := fs.Args()
	if len(remaining) < 1 {
		fs.Usage()
		return errors.New("missing target address <ip[:port]>")
	}

	rawAddr := remaining[0]
	var expectedKey string
	if len(remaining) >= 2 {
		expectedKey = remaining[1]
	}

	addr := rawAddr
	if _, _, err := net.SplitHostPort(addr); err != nil {
		addr = net.JoinHostPort(addr, DefaultPort)
	}

	fmt.Printf("[*] Connecting to peer at %s...\n", addr)
	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", addr, err)
	}

	fmt.Println("[*] Performing cryptographic handshake...")
	peerPub, err := p2p.Handshake(conn, priv)
	if err != nil {
		conn.Close()
		return fmt.Errorf("handshake failed: %w", err)
	}

	gotKey := hex.EncodeToString(peerPub)

	if expectedKey != "" && !strings.EqualFold(expectedKey, gotKey) {
		conn.Close()
		return fmt.Errorf("security alert: key mismatch! Expected %s, got %s", expectedKey, gotKey)
	}

	fmt.Println("==================================================")
	fmt.Printf("  Connected Successfully to %s\n", addr)
	fmt.Printf("  Author: %s\n", Author)
	fmt.Println("==================================================")
	fmt.Printf("Peer Public Key : %s\n", gotKey)
	if expectedKey != "" {
		fmt.Println("Key Verification: Verified & Trusted [✓]")
	}
	fmt.Println("--------------------------------------------------")
	fmt.Println("Type your messages below. Press Ctrl+C to exit.")
	fmt.Println("==================================================")

	p2p.Chat(conn)
	return nil
}

func cmdScan(priv ed25519.PrivateKey) error {
	pubHex := hex.EncodeToString(priv.Public().(ed25519.PublicKey))
	fmt.Printf("[*] Scanning local network for Glory peers (Author: %s)...\n", Author)
	peers := p2p.ScanLocalNetwork(pubHex)

	if len(peers) == 0 {
		fmt.Println("[-] No peers found on local network.")
		return nil
	}

	fmt.Println("\n[+] Discovered Peers on Local Network:")
	for _, p := range peers {
		fmt.Printf("  -> %s\n", p)
	}
	return nil
}

func cmdServer(args []string) error {
	fs := flag.NewFlagSet("server", flag.ExitOnError)
	port := fs.String("port", "4244", "Port for the central hub registry server")
	fs.Parse(args)

	hub := p2p.NewPeerHub()
	ln, err := net.Listen("tcp", ":"+*port)
	if err != nil {
		return err
	}
	defer ln.Close()

	fmt.Printf("==================================================\n")
	fmt.Printf("  Glory Central Hub Server (v%s)\n", Version)
	fmt.Printf("  Author: %s\n", Author)
	fmt.Printf("==================================================\n")
	fmt.Printf("Hub Server listening on port %s...\n", *port)

	for {
		conn, err := ln.Accept()
		if err != nil {
			continue
		}
		go func(c net.Conn) {
			defer c.Close()
			buf := make([]byte, 512)
			n, _ := c.Read(buf)
			line := string(buf[:n])
			parts := strings.Fields(line)
			if len(parts) >= 2 && parts[0] == "REGISTER" {
				pubKey := parts[1]
				ip, _, _ := net.SplitHostPort(c.RemoteAddr().String())
				hub.Register(ip, pubKey)
			} else if strings.HasPrefix(line, "LIST") {
				_, _ = c.Write(hub.ToJSON())
			}
		}(conn)
	}
}

func cmdList(args []string) error {
	if len(args) < 1 {
		return errors.New("usage: glory list <hub-ip[:port]>")
	}
	hubAddr := args[0]
	if _, _, err := net.SplitHostPort(hubAddr); err != nil {
		hubAddr = net.JoinHostPort(hubAddr, "4244")
	}

	conn, err := net.DialTimeout("tcp", hubAddr, 5*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect to hub: %w", err)
	}
	defer conn.Close()

	_, _ = fmt.Fprintln(conn, "LIST")
	buf := make([]byte, 4096)
	n, _ := conn.Read(buf)
	fmt.Println("=== Active Global Peers Registered on Hub ===")
	fmt.Println(string(buf[:n]))
	return nil
}

func PrintUsage() {
	fmt.Printf(`Glory v%s - Secure Peer-to-Peer Network CLI
Author: %s

Usage:
  glory <command> [arguments]

Commands:
  start           Start a P2P node and wait for connections (supports -hub <server>)
  connect         Connect securely to a remote peer IP
  scan            Search and discover active Glory peers on your local network
  server          Start a central discovery hub server (ideal for Oracle cloud)
  list            List active online peers registered on a central hub server
  key             Display your local Ed25519 identity public key
  version         Show current version number and author details
  help            Show this help message

Use "glory <command> --help" for detailed flag options.
`, Version, Author)
}
