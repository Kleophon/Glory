package cli

import (
	"crypto/ed25519"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"glory/pkg/config"
	"glory/pkg/crypto"
	"glory/pkg/p2p"
	"net"
	"os"
	"strings"
	"time"
)

const (
	Version     = "0.6.0"
	DefaultPort = "4242"
	Author      = "@kleophonx"
)

func logInfo(format string, args ...any) {
	fmt.Printf("[%s] [INFO] "+format+"\n", append([]any{time.Now().Format("15:04:05")}, args...)...)
}

func logSuccess(format string, args ...any) {
	fmt.Printf("[%s] [SUCCESS] "+format+"\n", append([]any{time.Now().Format("15:04:05")}, args...)...)
}

func logWarn(format string, args ...any) {
	fmt.Printf("[%s] [WARN] "+format+"\n", append([]any{time.Now().Format("15:04:05")}, args...)...)
}

func logError(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "[%s] [ERROR] "+format+"\n", append([]any{time.Now().Format("15:04:05")}, args...)...)
}

func Run() {
	if len(os.Args) < 2 {
		PrintUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	switch cmd {
	case "-v", "--version", "version":
		logInfo("glory v%s (Secure P2P Peer Network)", Version)
		logInfo("Author: %s", Author)
		return
	case "-h", "--help", "help":
		PrintUsage()
		return
	}

	priv, err := crypto.LoadKey()
	if err != nil {
		logError("Failed to load/generate identity key: %v", err)
		os.Exit(1)
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		logWarn("Could not load configuration file, using defaults: %v", err)
		cfg = config.DefaultConfig()
	}

	switch cmd {
	case "start":
		err = cmdStart(priv, cfg, os.Args[2:])
	case "connect":
		err = cmdConnect(priv, os.Args[2:])
	case "scan":
		err = cmdScan(priv)
	case "server":
		err = cmdServer(os.Args[2:])
	case "list":
		err = cmdList(os.Args[2:])
	case "config":
		err = cmdConfig(cfg, os.Args[2:])
	case "key":
		logInfo("Public Key: %s", hex.EncodeToString(priv.Public().(ed25519.PublicKey)))
		return
	default:
		logError("Unknown command: %s", cmd)
		PrintUsage()
		os.Exit(1)
	}

	if err != nil {
		logError("%v", err)
		os.Exit(1)
	}
}

func cmdStart(priv ed25519.PrivateKey, cfg *config.Config, args []string) error {
	fs := flag.NewFlagSet("start", flag.ExitOnError)
	port := fs.String("port", DefaultPort, "Port to listen on for incoming peer connections")
	hubAddr := fs.String("hub", "", "Optional central hub address to register this peer with")

	fs.Usage = func() {
		logInfo("Glory v%s - Start Command (Author: %s)", Version, Author)
		fmt.Println("Usage: glory start [options]")
		fmt.Println("\nOptions:")
		fs.PrintDefaults()
	}
	fs.Parse(args)

	// Start UDP discovery responder in background (private nodes only respond to friends)
	go p2p.StartDiscoveryResponder(priv, cfg)

	listener, err := net.Listen("tcp", ":"+*port)
	if err != nil {
		return fmt.Errorf("failed to bind to port %s: %w", *port, err)
	}
	defer listener.Close()

	pubHex := hex.EncodeToString(priv.Public().(ed25519.PublicKey))

	logInfo("==================================================")
	logInfo("  Glory P2P Node Initialized (v%s)", Version)
	logInfo("  Author: %s", Author)
	logInfo("==================================================")
	logInfo("Local Identity Public Key : %s", pubHex)
	logInfo("Node Visibility Mode      : %s", strings.ToUpper(cfg.Visibility))
	if cfg.Visibility == "private" {
		logInfo("Authorized Friends Count  : %d", len(cfg.Friends))
		logInfo("Stealth Mode Active       : Hidden from public subnet scans and hub directories.")
	}
	logInfo("Listening Address         : 0.0.0.0:%s", *port)

	// Only public nodes register with central hub directories
	if *hubAddr != "" {
		if cfg.Visibility == "private" {
			logWarn("Node visibility is PRIVATE. Refusing to register with central hub to preserve stealth status.")
		} else {
			logInfo("Hub Registry Target       : %s", *hubAddr)
			go registerWithHub(*hubAddr, pubHex, *port)
		}
	}
	logInfo("--------------------------------------------------")
	logInfo("Awaiting incoming peer connections...")
	logInfo("==================================================")

	for {
		conn, err := listener.Accept()
		if err != nil {
			logError("Socket accept error: %v", err)
			continue
		}

		remoteAddr := conn.RemoteAddr().String()
		logInfo("Incoming TCP connection established from %s", remoteAddr)

		peerPub, err := p2p.Handshake(conn, priv)
		if err != nil {
			logError("Handshake failed with %s: %v", remoteAddr, err)
			conn.Close()
			continue
		}

		peerHex := hex.EncodeToString(peerPub)

		if !cfg.IsAllowed(peerHex) {
			logWarn("Access denied for peer %s (Key: %s). Node visibility is PRIVATE and key is not in friends list.", remoteAddr, peerHex)
			_, _ = conn.Write([]byte("ERROR: Access denied. Node is private.\n"))
			conn.Close()
			continue
		}

		logSuccess("Handshake completed successfully with %s", remoteAddr)
		logSuccess("Verified Peer Public Key: %s", peerHex)
		logInfo("--------------------------------------------------")

		p2p.Chat(conn)
		logInfo("Connection closed. Resuming listener...")
	}
}

func registerWithHub(hubAddr, pubKey, port string) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		conn, err := net.DialTimeout("tcp", hubAddr, 5*time.Second)
		if err == nil {
			_, _ = fmt.Fprintf(conn, "REGISTER %s %s\n", pubKey, port)
			conn.Close()
			logInfo("Successfully pinged central hub registry at %s", hubAddr)
		} else {
			logWarn("Failed to contact central hub at %s: %v", hubAddr, err)
		}
	}
}

func cmdConnect(priv ed25519.PrivateKey, args []string) error {
	fs := flag.NewFlagSet("connect", flag.ExitOnError)
	fs.Usage = func() {
		logInfo("Glory v%s - Connect Command (Author: %s)", Version, Author)
		fmt.Println("Usage: glory connect <ip[:port]> [peer-key]")
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

	logInfo("Initiating TCP connection to remote endpoint %s...", addr)
	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", addr, err)
	}

	logInfo("Executing cryptographic Ed25519 handshake...")
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

	logSuccess("Connected successfully to remote endpoint %s", addr)
	logSuccess("Verified Peer Public Key: %s", gotKey)
	if expectedKey != "" {
		logSuccess("Key Pinning Verification Passed [✓]")
	}
	logInfo("--------------------------------------------------")
	logInfo("Interactive session active. Type messages below.")
	logInfo("==================================================")

	p2p.Chat(conn)
	return nil
}

func cmdScan(priv ed25519.PrivateKey) error {
	pubHex := hex.EncodeToString(priv.Public().(ed25519.PublicKey))
	logInfo("Initiating local subnet broadcast scan...")
	peers := p2p.ScanLocalNetwork(pubHex)

	if len(peers) == 0 {
		logWarn("No active Glory peers discovered on local subnet.")
		return nil
	}

	logSuccess("Discovered %d active peer(s) on local subnet:", len(peers))
	for _, p := range peers {
		logInfo("  -> %s", p)
	}
	return nil
}

func cmdServer(args []string) error {
	fs := flag.NewFlagSet("server", flag.ExitOnError)
	port := fs.String("port", "4244", "Port for the central directory hub server")
	fs.Parse(args)

	hub := p2p.NewPeerHub()
	ln, err := net.Listen("tcp", ":"+*port)
	if err != nil {
		return err
	}
	defer ln.Close()

	logInfo("==================================================")
	logInfo("  Glory Central Hub Coordinator Daemon (v%s)", Version)
	logInfo("  Author: %s", Author)
	logInfo("==================================================")
	logSuccess("Hub daemon active and listening on port %s...", *port)

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
				logInfo("Hub registered node %s from IP %s", pubKey[:12], ip)
			} else if strings.HasPrefix(line, "LIST") {
				_, _ = c.Write(hub.ToJSON())
				logInfo("Hub served peer list to client %s", c.RemoteAddr().String())
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

	logInfo("Fetching active peer directory from hub %s...", hubAddr)
	conn, err := net.DialTimeout("tcp", hubAddr, 5*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect to hub: %w", err)
	}
	defer conn.Close()

	_, _ = fmt.Fprintln(conn, "LIST")
	buf := make([]byte, 4096)
	n, _ := conn.Read(buf)
	logSuccess("=== Active Global Peer Registry ===")
	fmt.Println(string(buf[:n]))
	return nil
}

func cmdConfig(cfg *config.Config, args []string) error {
	fs := flag.NewFlagSet("config", flag.ExitOnError)
	visibility := fs.String("visibility", "", "Set node visibility ('public' or 'private')")
	addFriend := fs.String("add-friend", "", "Add a peer public key hex to the friends list")
	fs.Parse(args)

	modified := false
	if *visibility != "" {
		v := strings.ToLower(*visibility)
		if v != "public" && v != "private" {
			return errors.New("visibility must be either 'public' or 'private'")
		}
		cfg.Visibility = v
		modified = true
		logSuccess("Node visibility updated to: %s", v)
	}

	if *addFriend != "" {
		keyHex := strings.TrimSpace(*addFriend)
		if len(keyHex) != 64 {
			logWarn("Warning: public key length is not 64 hex characters (Ed25519 expected length)")
		}
		cfg.Friends = append(cfg.Friends, keyHex)
		modified = true
		logSuccess("Added public key to friends list: %s", keyHex)
	}

	if modified {
		if err := cfg.Save(); err != nil {
			return fmt.Errorf("failed to save configuration: %w", err)
		}
		logSuccess("Configuration successfully persisted to disk.")
	} else {
		logInfo("Current Configuration:")
		logInfo("  Visibility : %s", cfg.Visibility)
		logInfo("  Friends    : %d registered", len(cfg.Friends))
		for i, f := range cfg.Friends {
			logInfo("    [%d] %s", i+1, f)
		}
	}
	return nil
}

func PrintUsage() {
	logInfo("Glory v%s - Secure Peer-to-Peer Network CLI (Author: %s)", Version, Author)
	fmt.Printf(`
Usage:
  glory <command> [arguments]

Commands:
  start           Start a P2P node and wait for connections (supports -hub <server>)
  connect         Connect securely to a remote peer IP
  scan            Search and discover active Glory peers on your local network
  server          Start a central directory hub server (any VPS/cloud server)
  list            List active online peers registered on a central hub server
  config          Manage node configuration (visibility, friends list)
  key             Display your local Ed25519 identity public key
  version         Show current version number and author details
  help            Show this help message

Use "glory <command> --help" for detailed flag options.
`)
}
