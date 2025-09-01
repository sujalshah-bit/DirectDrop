package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"
)

type PeerInfo struct {
	Address  string    `json:"address"`
	LastSeen time.Time `json:"lastSeen"`
}

type Peers struct {
	peers map[string]PeerInfo
	mu    sync.RWMutex
}

func main() {
	listener, err := net.Listen("tcp", ":8080")

	if err != nil {
		log.Fatal("Starting the server: ", err)
	}

	defer listener.Close()

	log.Println("server started on :8080")

	peers := &Peers{
		peers: make(map[string]PeerInfo),
	}
	go gc(peers)
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Fatal("Connecting client: ", err)
		}

		go handleClient(conn, peers)
	}
}

func handleClient(conn net.Conn, peers *Peers) {
	defer conn.Close()
	scanner := bufio.NewScanner(conn)
	ipAddress := conn.RemoteAddr().(*net.TCPAddr)
	clientIP := ipAddress.String()

	log.Printf("Client connected from: %s", clientIP)

	for scanner.Scan() {
		rawText := scanner.Text()
		text := strings.Trim(rawText, " ")
		parts := strings.SplitN(text, " ", 3)

		if len(parts) < 2 {
			fmt.Fprintln(conn, "ERROR Invalid command format")
			continue
		}

		command := parts[0]
		shareCode := parts[1]

		switch command {
		case "ADD":
			peers.mu.Lock()
			peers.peers[shareCode] = PeerInfo{Address: clientIP, LastSeen: time.Now()}
			peers.mu.Unlock()
			fmt.Fprintf(conn, "OK Registered code %s\n", shareCode)
			log.Printf("Registered code: %s for %s", shareCode, clientIP)
		case "LOOK":
			peers.mu.RLock()
			peer, exist := peers.peers[shareCode]
			if !exist {
				fmt.Fprint(conn, "Peer did not exist\n")
			} else {
				fmt.Fprintf(conn, "%s found\n", peer.Address)
				log.Printf("Lookup for code %s: found %s", shareCode, peer.Address)

			}
			peers.mu.RUnlock()
		default:
			fmt.Fprintln(conn, "ERROR Unknown command")

		}
	}
}

func gc(peers *Peers) {
	ticker := time.NewTicker(1 * time.Minute)

	for range ticker.C {
		peers.mu.Lock()
		for code, info := range peers.peers {
			if time.Since(info.LastSeen) > 7*time.Minute {
				log.Printf("Removing stale peer: %s", code)
				delete(peers.peers, code)
			}
		}
		peers.mu.Unlock()
	}
}
