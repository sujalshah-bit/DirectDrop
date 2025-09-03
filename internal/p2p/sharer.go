package p2p

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/sujalshah-bit/DirectDrop/internal/config"
	"github.com/sujalshah-bit/DirectDrop/pkg"
)

type SharerTCPServer struct {
	Addr       string // Note that this address is of another server.
	PeerAddr   string
	Timeout    time.Duration
	listener   net.Listener
	clients    map[net.Conn]bool
	clientsMux sync.Mutex
	codes      map[string]string // Store codes: code -> value
	codesMux   sync.Mutex
	flags      []*string
	wg         sync.WaitGroup
}

// NewSharerTCPServer creates a new TCP server instance
func NewSharerTCPServer(peerAddr, addr string, timeout time.Duration, flags []*string) *SharerTCPServer {
	return &SharerTCPServer{
		Addr:     addr,
		PeerAddr: peerAddr,
		Timeout:  timeout,
		clients:  make(map[net.Conn]bool),
		codes:    make(map[string]string),
		flags:    flags,
	}
}

// Start starts the TCP server
func (s *SharerTCPServer) Start() error {
	listener, err := net.Listen("tcp", s.PeerAddr)
	if err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}
	s.listener = listener

	log.Printf("TCP Server started on %s\n", s.PeerAddr)

	// Start accepting connections
	go s.acceptConnections()

	return nil
}

// Stop stops the TCP server
func (s *SharerTCPServer) Stop() error {
	if s.listener != nil {
		log.Fatal("Closing Sharer TCP server")
		return s.listener.Close()
	}
	s.wg.Wait()
	return nil
}

// acceptConnections accepts incoming connections
func (s *SharerTCPServer) acceptConnections() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			if strings.Contains(err.Error(), "use of closed network connection") {
				// Server was stopped
				return
			}
			log.Printf("Error accepting connection: %v\n", err)
			continue
		}

		s.clientsMux.Lock()
		s.clients[conn] = true
		s.clientsMux.Unlock()

		log.Printf("Waiting for connection....\n")

		s.wg.Add(1)
		// Handle each client in a separate goroutine
		go s.handleClient(conn)

	}
}

// handleClient handles communication with a single client
func (s *SharerTCPServer) handleClient(conn net.Conn) {
	defer func() {
		conn.Close()
		s.clientsMux.Lock()
		delete(s.clients, conn)
		s.clientsMux.Unlock()
		s.wg.Done()
	}()

	clientAddr := conn.RemoteAddr().String()
	log.Printf("Client connected: %s\n", clientAddr)

	err := s.shareFile(conn)
	if err != nil {
		log.Printf("Error sharing file %v", err)
	}

}

func (s *SharerTCPServer) shareFile(conn net.Conn) error {
	if conn == nil {
		return fmt.Errorf("not connected to server")
	}

	path := *s.flags[config.PATH]

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	log.Printf("Read file: %s (%d bytes)", path, len(data))

	// compress
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write(data); err != nil {
		return err
	}
	gz.Close()
	compressed := buf.Bytes()
	log.Printf("Compressed data size: %d bytes", len(compressed))

	// checksum on compressed data
	sum := sha256.Sum256(compressed)
	checksum := hex.EncodeToString(sum[:])
	log.Printf("Checksum computed: %s", checksum)

	// send metadata (JSON)
	meta := map[string]interface{}{
		"filename": filepath.Base(path), // only send base name
		"size":     len(compressed),
		"checksum": checksum,
	}
	metaBytes, _ := json.Marshal(meta)
	if _, err := conn.Write(append(metaBytes, '\n')); err != nil {
		return err
	}
	log.Printf("Sent metadata: %s", string(metaBytes))

	// wait for ack from receiver
	ack := make([]byte, 3)
	if _, err := conn.Read(ack); err != nil {
		return err
	}
	log.Printf("Received ack: %s", string(ack))

	if string(ack) != "OK\n" {
		return fmt.Errorf("receiver rejected metadata")
	}

	// send compressed data
	_, err = conn.Write(compressed)
	if err == nil {
		log.Printf("Sent compressed file content (%d bytes)", len(compressed))
	}
	return err
}

// ==================== CLIENT FUNCTIONALITY TO OTHER SERVERS ====================

// SendCodeToServer sends an ADD command to another server (acts as client)
func (s *SharerTCPServer) SendCodeToServer() error {
	conn, err := net.DialTimeout("tcp", s.Addr, s.Timeout)
	if err != nil {
		return fmt.Errorf("failed to connect to server %s: %w", s.Addr, err)
	}
	defer conn.Close()

	code := pkg.GenerateRandomString(10)

	_, err = fmt.Fprintf(conn, "ADD %s\n", code)
	if err != nil {
		return fmt.Errorf("failed to send code to server %s: %w", s.Addr, err)
	}

	// Read response from server
	reader := bufio.NewReader(conn)
	response, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read response from server %s: %w", s.Addr, err)
	}

	response = strings.TrimSpace(response)
	log.Printf("Server %s responded: %s\n", s.Addr, response)

	// Store the code locally as well
	s.codesMux.Lock()
	s.codes[code] = "shared_with_server"
	s.codesMux.Unlock()

	log.Printf("Code sent to server %s: %s\n", s.Addr, code)
	return nil
}

// GetCodes returns all stored codes (for debugging/monitoring)
func (s *SharerTCPServer) GetCodes() map[string]string {
	s.codesMux.Lock()
	defer s.codesMux.Unlock()

	// Return a copy to avoid concurrent access issues
	codesCopy := make(map[string]string)
	for k, v := range s.codes {
		codesCopy[k] = v
	}
	return codesCopy
}
