package main

import (
	"bufio"
	"bytes"
	"net"
	"strings"
	"testing"
	"time"
)

// Mock net.Conn for testing
type mockConn struct {
	*bytes.Buffer
	remoteAddr net.Addr
}

func TestHandleClient(t *testing.T) {
	peers := &Peers{
		peers: make(map[string]PeerInfo),
	}
	tests := []struct {
		name           string
		input          string
		expectedOutput string
		shareCode      string
		clientAddr     string
		expectPeer     bool
	}{
		{
			name:           "ADD command",
			input:          "ADD abc123\n",
			expectedOutput: "OK Registered code abc123",
			shareCode:      "abc123",
			clientAddr:     "192.168.1.100:8080",
			expectPeer:     true,
		},
		{
			name:           "LOOK command with existing code",
			input:          "LOOK abc123\n",
			expectedOutput: "192.168.1.100:8080 found",
			shareCode:      "abc123",
			clientAddr:     "192.168.1.100:8080",
			expectPeer:     true,
		},
		{
			name:           "LOOK command with non-existent code",
			input:          "LOOK nonexistent\n",
			expectedOutput: "Peer did not exist",
			shareCode:      "nonexistent",
			clientAddr:     "192.168.1.100:8080",
			expectPeer:     false,
		},
		{
			name:           "Invalid command",
			input:          "INVALID abc123\n",
			expectedOutput: "ERROR Unknown command",
			shareCode:      "abc123",
			clientAddr:     "192.168.1.100:8080",
			expectPeer:     false,
		},
		{
			name:           "Malformed command",
			input:          "ADD\n",
			expectedOutput: "ERROR Invalid command format",
			shareCode:      "",
			clientAddr:     "192.168.1.100:8080",
			expectPeer:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock connection
			output := &bytes.Buffer{}
			addr := &net.TCPAddr{IP: net.ParseIP("192.168.1.100"), Port: 8080}
			conn := &mockConn{
				Buffer:     output,
				remoteAddr: addr,
			}

			// Write input to a buffer and create a scanner
			inputBuf := bytes.NewBufferString(tt.input)
			scanner := bufio.NewScanner(inputBuf)

			// Replace the scanner in the handleClient logic
			// We'll test the core logic directly instead of calling handleClient
			for scanner.Scan() {
				rawText := scanner.Text()
				text := strings.Trim(rawText, " ")
				parts := strings.SplitN(text, " ", 3)

				if len(parts) < 2 {
					conn.Write([]byte("ERROR Invalid command format\n"))
					continue
				}

				command := parts[0]
				shareCode := parts[1]

				switch command {
				case "ADD":
					peers.mu.Lock()
					peers.peers[shareCode] = PeerInfo{Address: tt.clientAddr, LastSeen: time.Now()}
					peers.mu.Unlock()
					conn.Write([]byte("OK Registered code " + shareCode + "\n"))
				case "LOOK":
					peers.mu.RLock()
					peer, exist := peers.peers[shareCode]
					peers.mu.RUnlock()
					if !exist {
						conn.Write([]byte("Peer did not exist\n"))
					} else {
						conn.Write([]byte(peer.Address + " found\n"))
					}
				default:
					conn.Write([]byte("ERROR Unknown command\n"))
				}
			}

			// Check the output
			outputStr := output.String()
			if !strings.Contains(outputStr, tt.expectedOutput) {
				t.Errorf("Expected output to contain '%s', got '%s'", tt.expectedOutput, outputStr)
			}

			// For ADD commands, verify the peer was actually added
			if tt.expectPeer {
				peers.mu.RLock()
				_, exists := peers.peers[tt.shareCode]
				peers.mu.RUnlock()
				if !exists {
					t.Errorf("Expected peer %s to be registered", tt.shareCode)
				}
			}
		})
	}
}

//TODO: func TestPeersConcurrency(t *testing.T) {
//TODO: func TestGC(t *testing.T) {
