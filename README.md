# Glory 🛡️

**Glory** is a decentralized, cryptographically secured peer-to-peer (P2P) communication and network orchestration utility engineered in Go. It implements mutual Ed25519 identity verification, challenge-response handshakes, local subnet discovery, and centralized directory coordination to facilitate robust, zero-trust node connectivity across disparate networks.

---

## Architectural Overview

```
+-----------------------------------------------------------------+
|                     GLORY NETWORK TOPOLOGY                      |
|                                                                 |
|   +------------------+                   +------------------+   |
|   |  Node A (Client) |                   |  Node B (Peer)   |   |
|   +--------+---------+                   +--------+---------+   |
|            |                                      ^             |
|            +---[ Ed25519 Challenge-Response ]-----+             |
|            |       (Mutual Authentication)        |             |
|            v                                      v             |
|   +--------------------------------------------------+          |
|   |               Oracle Cloud Hub Server            |          |
|   |        (Directory Registration & Discovery)      |          |
|   +--------------------------------------------------+          |
+-----------------------------------------------------------------+
```

### Core Subsystems

1. **Cryptographic Identity (`pkg/crypto`)**:
   - Automatically provisions and maintains persistent Ed25519 asymmetric keypairs stored with strict POSIX permissions (`0600`) within the operating system's configuration directory.
2. **Secure Handshake Protocol (`pkg/p2p`)**:
   - Enforces mutual authentication on every TCP socket connection. Peers exchange public keys and random 32-byte nonces, signing challenges under a domain-separated context (`glory-handshake-v1`) to mitigate Man-in-the-Middle (MitM) vectors.
3. **Subnet Discovery (`pkg/p2p/discovery.go`)**:
   - Utilizes UDP broadcast mechanisms to dynamically discover active peer nodes inhabiting the same local broadcast domain.
4. **Directory Orchestration Hub (`pkg/p2p/hub.go`)**:
   - Provides a centralized coordination daemon designed for cloud deployment (e.g., Oracle Cloud infrastructure), allowing distributed nodes to register their presence and query global peer topologies.

---

## Installation

Ensure you have **Go 1.22+** installed on your system.

```bash
git clone https://github.com/Kleophon/Glory.git
cd Glory
go build -o glory main.go
```

---

## Operational Guide

### 1. Inspecting Identity Key
Display your node's Ed25519 public key identifier:
```bash
./glory key
```

### 2. Local Subnet Peer Discovery
Scan the local broadcast domain for active peers:
```bash
./glory scan
```

### 3. Deploying the Central Directory Hub (Oracle Cloud / VPS)
On your cloud server (e.g., Oracle Cloud instance), initiate the directory coordinator daemon:
```bash
./glory server -port 4244
```

### 4. Initiating a Node Registered with the Hub
Start a peer node and register its availability with the central directory hub:
```bash
./glory start -port 4242 -hub <oracle-server-ip>:4244
```

### 5. Querying Global Active Peers
Retrieve a JSON-formatted manifest of all active nodes registered across the global topology:
```bash
./glory list <oracle-server-ip>:4244
```

### 6. Establishing Direct Peer-to-Peer Connection
Connect securely to a remote peer (with optional public key pinning):
```bash
./glory connect <remote-ip>:4242 [expected-peer-public-key-hex]
```

---

## Security Specifications

- **Asymmetric Signatures**: Ed25519 elliptic curve signatures guarantee non-repudiation and socket authenticity.
- **Key Pinning**: Optional argument verification (`[peer-key]`) ensures zero-trust validation against rogue endpoints.
- **Timed Timeouts**: Strict deadline enforcement on TCP handshakes prevents socket exhaustion and slow-loris vectors.

---

## Author

Developed by **@kleophonx**.
