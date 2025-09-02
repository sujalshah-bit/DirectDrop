package main

import (
	"log"
	"os"
	"time"

	"github.com/sujalshah-bit/DirectDrop/internal/p2p"
	"github.com/sujalshah-bit/DirectDrop/pkg"
)

const (
	SERVERADDR = 0
	CODE       = 1
	ACTION     = 2
	PATH       = 3
)

func main() {
	flags := pkg.HandleFlags()

	if ok := pkg.Validate(flags); !ok {
		os.Exit(1)
	}

	action := *flags[ACTION]
	if action == "share" {
		server := p2p.NewTCPServer("192.168.29.239:8081", *flags[SERVERADDR], 5*time.Second)
		err := server.SendCodeToServer()
		if err != nil {
			log.Printf("Failed to send code: %v", err)
		}
		defer server.Stop()
	} else {
		receiever := p2p.NewTCPClient(*flags[SERVERADDR], 5*time.Second)
		err := receiever.Connect()
		if err != nil {
			log.Printf("Receiver failed to connect to server: %v", err)
			os.Exit(1)
		}
		defer receiever.Close()
		str, err := receiever.ReceiveCode(*flags[CODE])
		if err != nil {
			log.Printf("Receiver failed to receiver info from server: %v", err)
			os.Exit(1)

		}

		log.Printf("Value received: %v", str)
	}

}
