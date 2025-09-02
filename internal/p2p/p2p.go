package p2p

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/sujalshah-bit/DirectDrop/pkg"
)

type TCPServer struct {
	Addr       string // Note that this address is of another server.
	PeerAddr   string // Note that this address is of another server.
	Timeout    time.Duration
	listener   net.Listener
	clients    map[net.Conn]bool
	clientsMux sync.Mutex
	codes      map[string]string // Store codes: code -> value
	codesMux   sync.Mutex
}

// NewTCPServer creates a new TCP server instance
func NewTCPServer(peerAddr, addr string, timeout time.Duration) *TCPServer {
	return &TCPServer{
		Addr:     addr,
		PeerAddr: peerAddr,
		Timeout:  timeout,
		clients:  make(map[net.Conn]bool),
		codes:    make(map[string]string),
	}
}

// Start starts the TCP server
func (s *TCPServer) Start() error {
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
func (s *TCPServer) Stop() error {
	if s.listener != nil {
		return s.listener.Close()
	}
	return nil
}

// acceptConnections accepts incoming connections
func (s *TCPServer) acceptConnections() {
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
		// Handle each client in a separate goroutine
		go s.handleClient(conn)
	}
}

// handleClient handles communication with a single client
func (s *TCPServer) handleClient(conn net.Conn) {
	defer func() {
		conn.Close()
		s.clientsMux.Lock()
		delete(s.clients, conn)
		s.clientsMux.Unlock()
	}()

	clientAddr := conn.RemoteAddr().String()
	log.Printf("Client connected: %s\n", clientAddr)

	// TODO

}

// ==================== CLIENT FUNCTIONALITY TO OTHER SERVERS ====================

// SendCodeToServer sends an ADD command to another server (acts as client)
func (s *TCPServer) SendCodeToServer() error {
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
func (s *TCPServer) GetCodes() map[string]string {
	s.codesMux.Lock()
	defer s.codesMux.Unlock()

	// Return a copy to avoid concurrent access issues
	codesCopy := make(map[string]string)
	for k, v := range s.codes {
		codesCopy[k] = v
	}
	return codesCopy
}
