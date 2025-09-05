package p2p

import (
	"bufio"
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

	if err := s.shareObject(conn); err != nil {
		log.Printf("Error handling client %s: %v", clientAddr, err)
	}
}

func (s *SharerTCPServer) shareObject(conn net.Conn) error {
	path := *s.flags[config.PATH]
	ok, err := pkg.IsDir(path)
	if err != nil {
		return fmt.Errorf("failed to check path type: %w", err)
	}

	if ok {
		meta := config.Meta{Type: "dir"}
		if err := pkg.SendMetadata(conn, meta); err != nil {
			return err
		}
		if err := pkg.WaitAck(conn); err != nil {
			return fmt.Errorf("receiver did not ack root dir: %w", err)
		}
		return s.shareFolder(conn)
	}

	meta := config.Meta{Type: "file"}
	if err := pkg.SendMetadata(conn, meta); err != nil {
		return err
	}
	if err := pkg.WaitAck(conn); err != nil {
		return fmt.Errorf("receiver did not ack root file: %w", err)
	}
	return s.shareFile(conn)
}

func (s *SharerTCPServer) shareFile(conn net.Conn) error {
	path := *s.flags[config.PATH]
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", path, err)
	}

	compressed, checksum, err := pkg.CompressData(data)
	if err != nil {
		return fmt.Errorf("failed to compress file %s: %w", path, err)
	}

	meta := config.Meta{
		Filename: filepath.Base(path),
		Type:     "file",
		Size:     len(compressed),
		Checksum: checksum,
	}
	if err := pkg.SendMetadata(conn, meta); err != nil {
		return err
	}
	if err := pkg.WaitAck(conn); err != nil {
		return fmt.Errorf("receiver rejected file %s: %w", path, err)
	}

	if _, err := conn.Write(compressed); err != nil {
		return fmt.Errorf("failed to send file %s: %w", path, err)
	}
	log.Printf("Sent file %s (%d bytes)", path, len(compressed))
	return nil
}

func (s *SharerTCPServer) shareFolder(conn net.Conn) error {
	basePath := *s.flags[config.PATH]

	return filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath, _ := filepath.Rel(basePath, path)

		if info.IsDir() {
			meta := config.Meta{Path: relPath, Type: "dir"}
			if err := pkg.SendMetadata(conn, meta); err != nil {
				return err
			}
			if err := pkg.WaitAck(conn); err != nil {
				return fmt.Errorf("receiver rejected dir %s: %w", relPath, err)
			}
			log.Printf("Sent dir: %s", relPath)
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", relPath, err)
		}

		compressed, checksum, err := pkg.CompressData(data)
		if err != nil {
			return fmt.Errorf("failed to compress %s: %w", relPath, err)
		}

		meta := config.Meta{
			Path:     relPath,
			Type:     "file",
			Size:     len(compressed),
			Checksum: checksum,
		}
		if err := pkg.SendMetadata(conn, meta); err != nil {
			return err
		}
		if err := pkg.WaitAck(conn); err != nil {
			return fmt.Errorf("receiver rejected file %s: %w", relPath, err)
		}

		if _, err := conn.Write(compressed); err != nil {
			return fmt.Errorf("failed to send file %s: %w", relPath, err)
		}
		log.Printf("Sent file: %s (%d bytes)", relPath, len(compressed))
		return nil
	})
}

// ==================== CLIENT FUNCTIONALITY TO OTHER SERVERS ====================

// SendCodeToServer sends an ADD command to another server (acts as client)
func (s *SharerTCPServer) SendCodeToServer() error {
	conn, err := net.DialTimeout("tcp", s.Addr, s.Timeout)
	if err != nil {
		return fmt.Errorf("failed to connect to server %s: %w", s.Addr, err)
	}
	defer conn.Close()

	addr := pkg.GetDeviceIPWithPort("8081")

	code := pkg.GenerateRandomString(10)

	_, err = fmt.Fprintf(conn, "ADD %s %s\n", code, addr)
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
