package pkg

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"os"
	"strings"
	"time"
	"unsafe"
)

const CHARSET = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func GenerateRandomString(length int) string {
	seededRand := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, length)
	for i := range b {
		b[i] = CHARSET[seededRand.Intn(len(CHARSET))]
	}
	return string(b)
}

func HandleFlags() []*string {
	serverAddr := flag.String("Address", "127.0.0.1:8080", "Server IP address with port")
	code := flag.String("Code", "", "a unique code which will be send to server.")
	action := flag.String("Action", "", "Whether you want to share or receive a file/folder")
	path := flag.String("Path", "", "Address of file/folder")

	flag.Parse()

	return []*string{serverAddr, code, action, path}
}

func Validate(flags []*string) bool {
	if len(os.Args) < 3 {
		log.Fatal("Number of arguments are not enough. Usage: ./myapp [share|receive] [path|code]")
		return false
	}

	serverAddr, code, action, path := flags[0], flags[1], flags[2], flags[3]

	// Validate action
	if *action != "share" && *action != "receive" {
		log.Fatal("Action must be either 'share' or 'receive'")
		return false
	}

	// Validate server address (basic check)
	if *serverAddr == "" || !strings.Contains(*serverAddr, ":") {
		log.Fatal("Server address must be in format 'host:port'")
		return false
	}

	// Action-specific validation
	switch *action {
	case "share":
		if *path == "" {
			log.Fatal("Path is required for share action")
			return false
		}
		// Check if file/directory exists
		if _, err := os.Stat(*path); os.IsNotExist(err) {
			log.Fatalf("Path '%s' does not exist", *path)
			return false
		}
	case "receive":
		if *code == "" {
			log.Fatal("Code is required for receive action")
			return false
		}
	}

	return true
}

// calculateChecksum calculates SHA256 checksum of a file
func CalculateChecksum(file *os.File) (string, error) {
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// GetDeviceIP returns the device's IP address for external communication
func GetDeviceIP() string {
	// Try to get outbound IP first (most reliable)
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err == nil {
		defer conn.Close()
		localAddr := conn.LocalAddr().(*net.UDPAddr)
		return localAddr.IP.String()
	}

	// Fallback to local IP detection
	addrs, err := net.InterfaceAddrs()
	if err == nil {
		for _, addr := range addrs {
			if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
				if ipNet.IP.To4() != nil {
					return ipNet.IP.String()
				}
			}
		}
	}
	return "127.0.0.1"
}

// GetDeviceIPWithPort returns IP:Port string
func GetDeviceIPWithPort(port string) string {
	return GetDeviceIP() + ":" + port
}

func IsDir(path string) (bool, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return false, err
	}

	return info.IsDir(), nil
}

// This func changes the original string
func UnsafeModifyStr(s *string) {
	b := unsafe.Slice(unsafe.StringData(*s), len(*s))

	if len(b) > 0 && b[len(b)-1] == '\n' {
		*s = unsafe.String(unsafe.SliceData(b), len(b)-1)
	}
}

func CompressData(data []byte) ([]byte, string, error) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write(data); err != nil {
		return nil, "", err
	}
	if err := gz.Close(); err != nil {
		return nil, "", err
	}
	compressed := buf.Bytes()
	sum := sha256.Sum256(compressed)
	return compressed, hex.EncodeToString(sum[:]), nil
}

func SendMetadata(conn net.Conn, meta interface{}) error {
	metaBytes, err := json.Marshal(meta)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}
	if _, err := conn.Write(append(metaBytes, '\n')); err != nil {
		return fmt.Errorf("failed to send metadata: %w", err)
	}
	return nil
}

func WaitAck(conn net.Conn) error {
	ack := make([]byte, 3)
	if _, err := conn.Read(ack); err != nil {
		return fmt.Errorf("failed to read ack: %w", err)
	}
	if string(ack) != "OK\n" {
		return fmt.Errorf("unexpected ack: %q", string(ack))
	}
	return nil
}

func VerifyChecksum(data []byte, expected string) error {
	sum := sha256.Sum256(data)
	got := hex.EncodeToString(sum[:])
	if got != expected {
		return fmt.Errorf("expected %s, got %s", expected, got)
	}
	return nil
}

func DecompressData(compressed []byte) ([]byte, error) {
	gr, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return nil, err
	}
	defer gr.Close()
	return io.ReadAll(gr)
}
