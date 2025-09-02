package p2p

import (
	"bufio"
	"fmt"
	"net"
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
