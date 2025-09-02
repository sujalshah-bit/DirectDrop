package pkg

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
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
	code := flag.String("Code", "127.0.0.1:8080", "a unique code which will be send to server.")
	action := flag.String("Action", "", "Whether you want to share or receive a file/folder")
	path := flag.String("Path", "", "Address of file/folder")

	flag.Parse()

	return []*string{serverAddr, code, action, path}
}

func Validate(flags []*string) bool {
	if len(os.Args) < 4 {
		fmt.Print("sdf")
		log.Fatal("Number of arguments are not enough. Usage: ./myapp [share|receive] [path|code] [serverAddress]")
		return false
	}

	serverAddr, code, action, path := flags[0], flags[1], flags[2], flags[3]

	// Validate action
	if *action != "share" && *action != "receive" {
		fmt.Print("a")
		log.Fatal("Action must be either 'share' or 'receive'")
		return false
	}

	// Validate server address (basic check)
	if *serverAddr == "" || !strings.Contains(*serverAddr, ":") {
		fmt.Print("b")
		log.Fatal("Server address must be in format 'host:port'")
		return false
	}

	// Action-specific validation
	switch *action {
	case "share":
		fmt.Print("c")
		if *path == "" {
			fmt.Print("empty\n")
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
			fmt.Print("d")
			log.Fatal("Code is required for receive action")
			return false
		}
	}

	fmt.Print("e")
	return true
}
