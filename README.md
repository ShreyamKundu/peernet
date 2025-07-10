# **PeerNet: A Decentralized File Sharing Proof-of-Concept**

PeerNet is a robust proof-of-concept for a decentralized peer-to-peer (P2P) file sharing system, drawing inspiration from BitTorrent. It demonstrates how individual "peers" can directly exchange file data, while a central "tracker" orchestrates discovery, manages authentication, and incentivizes network participation through a reputation and token-based economy.


## **✨ Features**



* **Decentralized File Transfer:** File chunks are transferred directly between peers using gRPC, eliminating central bottlenecks for data.
* **Centralized Tracker for Discovery:** A dedicated tracker service helps peers find each other and discover available file chunks.
* **JWT-Based Authentication:** Secure registration and authenticated API interactions between peers and the tracker using JSON Web Tokens (JWTs).
* **Reputation & Token Economy:**
    * Peers receive a reputation_score and token_balance.
    * Successful file chunk uploads increase the uploader's reputation and tokens.
    * Failed uploads decrease reputation and tokens, incentivizing reliable behavior.
    * File lookups prioritize peers with higher reputation scores.
* **File Chunking & Hashing:** Files are split into fixed-size chunks, with SHA256 hashes for both individual chunks and the entire file, ensuring data integrity.
* **Containerized Deployment:** All services (tracker, peers, database) are Dockerized and orchestrated using Docker Compose for easy setup and isolation.
* **Command-Line Interface (CLI):** User-friendly CLI for peer registration, file sharing, and downloading.


## **🏛️ Architecture**

PeerNet is comprised of two primary, independently containerized services:



1. **Tracker Service (Go, Gin, PostgreSQL):**
    * **Role:** Acts as the central directory. It stores metadata about registered peers, available files, and which peers possess which file chunks. It handles peer registration, file announcements, and file lookups.
    * **Authentication:** Secures its API endpoints using JWTs issued upon peer registration.
    * **Reputation Engine:** A background process that periodically updates peer reputation scores and token balances based on feedback received from downloaders.
    * **Database:** Uses PostgreSQL to persist all network state, including peer profiles, file metadata, and reputation events.
2. **Peer Client Service (Go, gRPC, Cobra CLI):**
    * **Role:** The active participants in the P2P network. Peers can share local files and download files from other peers.
    * **CLI:** Provides commands for register (with the tracker), share (a local file), and download (a file by its hash).
    * **File Management:** Chunks files for sharing and reassembles downloaded chunks.
    * **gRPC Server:** When sharing, a peer starts a gRPC server to serve its file chunks directly to other requesting peers.
    * **Tracker Interaction:** Communicates with the tracker via authenticated HTTP requests (using JWTs) for registration, announcing shared chunks, and looking up peers for downloads.
    * **Direct P2P Transfer:** Initiates direct gRPC connections to other peers to request and receive file chunks.
    * **Feedback Mechanism:** Reports success or failure of chunk downloads to the tracker, contributing to the reputation system.


### **Communication Flow:**



* **Peer ↔ Tracker:** Authenticated HTTP (REST API) for registration, file announcements, lookups, and feedback.
* **Peer ↔ Peer:** Direct gRPC for high-performance file chunk transfer.


## **🛠️ Technologies Used**



* **Go (Golang):** Primary programming language for both Tracker and Peer services.
* **Docker & Docker Compose:** For containerization, orchestration, and local development environment setup.
* **Gin Web Framework:** For building the Tracker's RESTful API.
* **PostgreSQL:** Relational database for the Tracker's persistent storage.
* **gRPC:** High-performance RPC framework for peer-to-peer chunk transfers.
* **JWT (JSON Web Tokens):** For secure authentication between peers and the tracker.
* **Cobra CLI:** For building the Peer client's command-line interface.
* **Bcrypt:** For secure password hashing.


## 🚀 Getting Started

To run the **PeerNet demo**, you'll need **Docker** and **Docker Compose** installed on your system.

### 1. Clone the repository
```bash
git clone https://github.com/ShreyamKundu/peernet.git
cd peernet
```
### 2. Run the demo script

The `run_demo.sh` script automates the entire process:

- Cleans up old containers
- Prepares sample files
- Builds and starts all services
- Registers peers
- Shares a file from Peer 1
- Downloads it with Peer 2

Run:

```bash
./run_demo.sh
```
Follow the on-screen narrative in your terminal to see PeerNet in action!


## **📺 Demo Video**

A comprehensive demo video showcasing PeerNet's functionality and architecture will be embedded here soon.


## **📚 Setup Guides**

Detailed setup guides for local development, advanced configurations, and troubleshooting will be provided here.


## **💡 Future Improvements (Potential Areas)**



* **TLS/SSL Encryption:** Implement TLS for all gRPC and HTTP communications for enhanced security.
* **NAT Traversal:** Integrate STUN/TURN/ICE for peers to connect across various network topologies.
* **Concurrent & Resumable Downloads:** Optimize downloader for parallel chunk fetching from multiple peers and support resuming interrupted downloads.
* **Advanced Peer Discovery:** Explore Distributed Hash Tables (DHTs) for a truly decentralized peer discovery mechanism, reducing reliance on a single tracker.
* **UI/Dashboard:** Develop a simple web UI for the tracker to visualize network activity, peer status, and file availability.
