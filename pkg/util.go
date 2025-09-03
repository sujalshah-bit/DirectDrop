package pkg

import (
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"io"
	"log"
	"math/rand"
	"net"
	"os"
	"strings"
	"time"
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
