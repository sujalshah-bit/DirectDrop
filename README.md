# DirectDrop

`DirectDrop` is a peer-to-peer (P2P) file and folder sharing tool written in Go.
It uses a lightweight coordination server to connect peers and transfer data directly.
The server does not store any files — it only handles coordination.

## Components

* **server**: runs the coordination server
* **direct\_drop (app)**: peer binary used for both sharing and receiving

## Installation

### Option 1: Download from Releases

Prebuilt binaries are available on the [GitHub Releases page](https://github.com/yourusername/direct_drop/releases).
Download the appropriate binary for your OS and architecture.

### Option 2: Build from Source

Clone the repository and use the provided `Makefile`:

```bash
git clone https://github.com/yourusername/direct_drop.git
cd direct_drop
make build
```

This will create binaries in the `bin/` directory:

* `bin/app` (peer binary)
* `bin/server` (coordination server)

You can also run directly without building:

```bash
make dev-app ARGS="-Action share -Path ./file.txt -Address 127.0.0.1:9000"
make dev-server ARGS="-Address 127.0.0.1:9000"
```

## Usage

### 1. Start the server

```bash
./server -Address <IP>:<Port>
```

The server must be reachable by both peers.

### 2. Share a file or folder

On the sending peer:

```bash
./direct_drop -Action share -Path ./path/to/file_or_folder -Address <IP>:<Port>
```

This command prints a **code**.
Share this code with the receiver via any out-of-band method (chat, email, etc.).

### 3. Receive a file or folder

On the receiving peer:

```bash
./direct_drop -Action receive -Code <code> -Address <IP>:<Port>
```

The file or folder will be downloaded to the download folder in current directory .

## Notes

* Works with both **files and folders**.
* The server only coordinates peers and does not store files.
* Ensure that firewalls and network settings allow connections on the chosen port.


## License

[Apache License](./LICENSE)
