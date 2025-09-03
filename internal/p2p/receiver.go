package p2p

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type TCPClient struct {
	ServerAddr string
	Timeout    time.Duration
	conn       net.Conn
}

// NewTCPClient creates a new TCP client instance
func NewTCPClient(serverAddr string, timeout time.Duration) *TCPClient {
	return &TCPClient{
		ServerAddr: serverAddr,
		Timeout:    timeout,
	}
}

// Connect establishes a connection to the server
func (c *TCPClient) Connect() error {
	conn, err := net.DialTimeout("tcp", c.ServerAddr, c.Timeout)
	if err != nil {
		return err
	}
	c.conn = conn
	return nil
}

// Close closes the connection
func (c *TCPClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// IsConnected checks if the client is connected
func (c *TCPClient) IsConnected() bool {
	return c.conn != nil
}

// Reconnect closes existing connection and establishes a new one
func (c *TCPClient) Reconnect() error {
	c.Close()
	return c.Connect()
}

// ReceiveCode sends a LOOK command and returns the server response
func (c *TCPClient) ReceiveCode(code string) (string, error) {
	if !c.IsConnected() {
		if err := c.Connect(); err != nil {
			return "", err
		}
	}

	_, err := fmt.Fprintf(c.conn, "LOOK %s\n", code)
	if err != nil {
		c.conn = nil
		return "", fmt.Errorf("failed to send lookup request: %w", err)
	}

	// Set read timeout
	if err := c.conn.SetReadDeadline(time.Now().Add(c.Timeout)); err != nil {
		return "", fmt.Errorf("failed to set read timeout: %w", err)
	}

	// Read response from server
	response, err := bufio.NewReader(c.conn).ReadString('\n')
	if err != nil {
		c.conn = nil
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	return strings.TrimSpace(response), nil
}

func (c *TCPClient) RequestData(IP string) {
	fmt.Print("wowowowo\n\n", IP)
	conn, err := net.DialTimeout("tcp", "192.168.29.239:8081", c.Timeout)
	if err != nil {
		log.Fatalf("failed to connect to server %s: %v", IP, err)
	}
	defer conn.Close()

	err = c.RecieveFile(conn)
	if err != nil {
		log.Fatalf("%s: %v", IP, err)
	}

}

func (c *TCPClient) RecieveFile(conn net.Conn) error {
	// read metadata line
	metaBuf := make([]byte, 4096)
	n, err := conn.Read(metaBuf)
	if err != nil {
		return err
	}
	log.Printf("Received metadata (%d bytes)", n)

	var meta map[string]interface{}
	if err := json.Unmarshal(metaBuf[:n], &meta); err != nil {
		return err
	}
	expectedChecksum := meta["checksum"].(string)
	size := int(meta["size"].(float64))
	filename := meta["filename"].(string)

	// make dir
	if err := os.MkdirAll("./download", 0755); err != nil {
		return err
	}
	outputPath := filepath.Join("./download", filename)

	// create empty file
	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	f.Close()
	log.Printf("Created empty file: %s", outputPath)

	// send ack
	if _, err := conn.Write([]byte("OK\n")); err != nil {
		return err
	}
	log.Printf("Sent ack to sender")

	// read compressed data
	compressed := make([]byte, size)
	_, err = io.ReadFull(conn, compressed)
	if err != nil {
		return err
	}
	log.Printf("Received compressed data (%d bytes)", len(compressed))

	// verify checksum
	sum := sha256.Sum256(compressed)
	gotChecksum := hex.EncodeToString(sum[:])
	if gotChecksum != expectedChecksum {
		return fmt.Errorf("checksum mismatch: got %s, expected %s", gotChecksum, expectedChecksum)
	}
	log.Printf("Checksum verified successfully")

	// decompress
	r, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return err
	}
	defer r.Close()
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	log.Printf("Decompressed data size: %d bytes", len(data))

	// write to file
	err = os.WriteFile(outputPath, data, 0644)
	if err == nil {
		log.Printf("Wrote decompressed file content to: %s", outputPath)
	}
	return err
}
