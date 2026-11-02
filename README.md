# Glory

**Glory** is a decentralized, cryptographically secured peer-to-peer (P2P) communication and network orchestration utility engineered in Go by **@kleophonx**. It implements mutual Ed25519 identity verification, challenge-response handshakes, local subnet discovery, centralized directory coordination, and strict access control lists (public/private node modes with friends-list authorization) to facilitate zero-trust node connectivity.

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
|   |              Central Hub Server                  |          |
|   |        (Directory Registration & Discovery)      |          |
|   +--------------------------------------------------+          |
+-----------------------------------------------------------------+
```

### Core Subsystems

1. **Cryptographic Identity (`pkg/crypto`)**:
   - Automatically provisions and maintains persistent Ed25519 asymmetric keypairs stored with strict POSIX permissions (`0600`) within the operating system's configuration directory (`~/.config/glory/key`).
2. **Configuration & Access Control (`pkg/config`)**:
   - Manages JSON-formatted configuration (`~/.config/glory/config.json`) defining node visibility (`public` or `private`) and pre-approved friends lists for zero-trust authorization.
3. **Secure Handshake Protocol (`pkg/p2p`)**:
   - Enforces mutual authentication on every TCP socket connection. Peers exchange public keys and random 32-byte nonces, signing challenges under a domain-separated context (`glory-handshake-v1`) to mitigate Man-in-the-Middle (MitM) vectors.
4. **Subnet Discovery (`pkg/p2p/discovery.go`)**:
   - Utilizes UDP broadcast mechanisms to dynamically discover active peer nodes inhabiting the same local broadcast domain. **Private nodes operate in stealth mode**: they ignore discovery probes unless the querying peer is explicitly present in their authorized friends list.
5. **Directory Orchestration Hub (`pkg/p2p/hub.go`)**:
   - Provides a centralized coordination daemon designed for cloud deployment (e.g., any VPS or cloud server), allowing distributed nodes to register their presence and query global peer topologies. **Private nodes refuse to register with central hubs**, ensuring complete anonymity in global listings.

---

## Installation & Testing

Ensure you have **Go 1.22+** installed on your system.

```bash
git clone https://github.com/Kleophon/Glory.git
cd Glory
go build -o glory main.go
```

### Running Test Suite
Glory includes comprehensive unit tests covering cryptographic identity generation, TCP handshake protocols, and directory hub registration:
```bash
go test -v ./...
```

---

## Configuration & Access Control

Glory supports a JSON configuration file located at `~/.config/glory/config.json`. You can manage visibility and authorized friends using the `glory config` command:

### 1. Set Node Visibility (`public` vs `private`)
- **Public Mode** (default): Any peer can discover your node on the local network, register with central hubs, and connect.
  ```bash
  glory config -visibility public
  ```
- **Private Mode (Stealth Mode)**: Your node becomes completely invisible to local subnet scans and refuses to register with central hubs. Incoming connections are strictly rejected unless the connecting peer's Ed25519 public key is explicitly registered in your friends list.
  ```bash
  glory config -visibility private
  ```

### 2. Manage Friends List
Add a trusted peer's 64-character Ed25519 public key hex to your authorized friends list:
```bash
glory config -add-friend <peer-public-key-hex>
```

View current configuration settings:
```bash
glory config
```

---

## Operational Guide

### 1. Inspecting Identity Key
Display your node's Ed25519 public key identifier:
```bash
./glory key
```

### 2. Connecting Local Peers
#### Step A: Discover Local Peers
Scan the local broadcast domain for active peers:
```bash
./glory scan
```
#### Step B: Start Listener & Connect
On Peer 1 (Receiver):
```bash
./glory start -port 4242
```
On Peer 2 (Initiator):
```bash
./glory connect <peer1-ip>:4242 [peer1-public-key-hex]
```

### 3. Connecting Peers Across the World via Central Hub (VPS / Cloud Server)
#### Step A: Deploy Hub on a Remote Server
On your cloud VPS instance:
```bash
./glory server -port 4244
```
#### Step B: Start Public Nodes and Register with Hub
On Public Peer nodes worldwide:
```bash
./glory start -port 4242 -hub <server-ip>:4244
```
#### Step C: List All Active Global Peers
Query the central hub to view online peers worldwide:
```bash
./glory list <server-ip>:4244
```
#### Step D: Connect to a Global Peer
```bash
./glory connect <global-peer-ip>:<port> <global-peer-public-key-hex>
```

---

## Structured Logging

Glory provides timestamped, color-coded, structured operational logs across all subsystems:
- `[INFO]`: General operational state updates.
- `[SUCCESS]`: Successful cryptographic verification, connection establishment, or config updates.
- `[WARN]`: Access control rejections, network warnings, or retry backoffs.
- `[ERROR]`: Fatal failures, bind errors, or handshake rejections.

---

## Author

Developed by **@kleophonx**.
