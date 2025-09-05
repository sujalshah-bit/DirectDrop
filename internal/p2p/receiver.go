package p2p

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sujalshah-bit/DirectDrop/internal/config"
	"github.com/sujalshah-bit/DirectDrop/pkg"
)

type TCPClient struct {
	ServerAddr string
	Timeout    time.Duration
	TargetDir  string
	conn       net.Conn
}

// NewTCPClient creates a new TCP client instance
func NewTCPClient(serverAddr string, timeout time.Duration) *TCPClient {
	return &TCPClient{
		ServerAddr: serverAddr,
		Timeout:    timeout,
		TargetDir:  "./download",
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

func (c *TCPClient) RequestData(IP string) error {
	pkg.UnsafeModifyStr(&IP) // assuming you really need this
	conn, err := net.DialTimeout("tcp", IP, c.Timeout)
	if err != nil {
		return fmt.Errorf("failed to connect to server %s: %w", IP, err)
	}
	defer conn.Close()

	return c.receiveObject(conn)
}

func (c *TCPClient) receiveObject(conn net.Conn) error {
	reader := bufio.NewReader(conn)

	metaLine, err := reader.ReadBytes('\n')
	if err != nil {
		return fmt.Errorf("failed to read metadata: %w", err)
	}

	var meta config.Meta
	if err := json.Unmarshal(metaLine, &meta); err != nil {
		return fmt.Errorf("invalid metadata: %w", err)
	}

	// Send ack back
	if _, err := conn.Write([]byte("OK\n")); err != nil {
		return fmt.Errorf("failed to send ack: %w", err)
	}

	switch meta.Type {
	case config.DIR:
		return c.receiveFolder(conn)
	case config.FILE:
		return c.receiveFile(conn)
	default:
		return fmt.Errorf("unknown object type: %s", meta.Type)
	}
}

func (c *TCPClient) receiveFile(conn net.Conn) error {
	reader := bufio.NewReader(conn)

	// read metadata
	metaLine, err := reader.ReadBytes('\n')
	if err != nil {
		return fmt.Errorf("failed to read file metadata: %w", err)
	}

	var meta config.Meta
	if err := json.Unmarshal(metaLine, &meta); err != nil {
		return fmt.Errorf("invalid file metadata: %w", err)
	}

	outputDir := "./download"
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create download dir: %w", err)
	}
	outputPath := filepath.Join(outputDir, meta.Filename)

	// create empty file
	if f, err := os.Create(outputPath); err == nil {
		f.Close()
	} else {
		return fmt.Errorf("failed to create file %s: %w", outputPath, err)
	}

	// send ack
	if _, err := conn.Write([]byte("OK\n")); err != nil {
		return fmt.Errorf("failed to send ack: %w", err)
	}

	// read compressed data
	compressed := make([]byte, meta.Size)
	if _, err := io.ReadFull(reader, compressed); err != nil {
		return fmt.Errorf("failed to read file data: %w", err)
	}

	// verify checksum
	if err := pkg.VerifyChecksum(compressed, meta.Checksum); err != nil {
		return fmt.Errorf("file checksum error for %s: %w", meta.Filename, err)
	}

	// decompress
	data, err := pkg.DecompressData(compressed)
	if err != nil {
		return fmt.Errorf("failed to decompress file %s: %w", meta.Filename, err)
	}

	// write to file
	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", outputPath, err)
	}

	log.Printf("File received: %s (%d bytes)", outputPath, len(data))
	return nil
}

func (c *TCPClient) receiveFolder(conn net.Conn) error {
	reader := bufio.NewReader(conn)

	for {
		metaLine, err := reader.ReadBytes('\n')
		if err == io.EOF {
			break // end of folder
		}
		if err != nil {
			return fmt.Errorf("failed to read folder metadata: %w", err)
		}

		var meta config.Meta
		if err := json.Unmarshal(metaLine, &meta); err != nil {
			return fmt.Errorf("invalid folder metadata: %w", err)
		}

		fullPath := filepath.Join(c.TargetDir, meta.Path)

		if meta.Type == "dir" {
			// create directory
			if err := os.MkdirAll(fullPath, 0755); err != nil {
				return fmt.Errorf("failed to create dir %s: %w", fullPath, err)
			}
			if _, err := conn.Write([]byte("OK\n")); err != nil {
				return fmt.Errorf("failed to ack dir %s: %w", fullPath, err)
			}
			log.Printf("Directory created: %s", fullPath)
			continue
		}

		// --- File ---
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			return fmt.Errorf("failed to create parent dirs for %s: %w", fullPath, err)
		}
		if f, err := os.Create(fullPath); err == nil {
			f.Close()
		} else {
			return fmt.Errorf("failed to create file %s: %w", fullPath, err)
		}

		if _, err := conn.Write([]byte("OK\n")); err != nil {
			return fmt.Errorf("failed to ack file %s: %w", fullPath, err)
		}

		compressed := make([]byte, meta.Size)
		if _, err := io.ReadFull(reader, compressed); err != nil {
			return fmt.Errorf("failed to read file %s: %w", meta.Path, err)
		}

		if err := pkg.VerifyChecksum(compressed, meta.Checksum); err != nil {
			return fmt.Errorf("checksum error for %s: %w", meta.Path, err)
		}

		data, err := pkg.DecompressData(compressed)
		if err != nil {
			return fmt.Errorf("decompression error for %s: %w", meta.Path, err)
		}

		if err := os.WriteFile(fullPath, data, 0644); err != nil {
			return fmt.Errorf("failed to write file %s: %w", fullPath, err)
		}

		log.Printf("File written: %s (%d bytes)", fullPath, len(data))
	}

	return nil
}
