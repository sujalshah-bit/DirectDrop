package main

import (
	"log"
	"os"
	"time"

	"github.com/sujalshah-bit/DirectDrop/internal/config"
	"github.com/sujalshah-bit/DirectDrop/internal/p2p"
	"github.com/sujalshah-bit/DirectDrop/pkg"
)

func main() {
	flags := pkg.HandleFlags()

	if ok := pkg.Validate(flags); !ok {
		os.Exit(1)
	}

	action := *flags[config.ACTION]
	if action == "share" {
		serverAddress := pkg.GetDeviceIPWithPort("8081")
		server := p2p.NewSharerTCPServer(serverAddress, *flags[config.SERVER_ADDRESS], config.TIMEOUT*time.Second, flags)
		if err := server.Start(); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
		err := server.SendCodeToServer()
		if err != nil {
			log.Printf("Failed to send code: %v", err)
		}
		defer server.Stop()
		// Block main so server keeps running
		select {}
	} else {
		receiever := p2p.NewTCPClient(*flags[config.SERVER_ADDRESS], config.TIMEOUT*time.Second)
		err := receiever.Connect()
		if err != nil {
			log.Printf("Receiver failed to connect to server: %v", err)
			os.Exit(1)
		}
		defer receiever.Close()
		sharerIP, err := receiever.ReceiveCode(*flags[config.CODE])
		if err != nil {
			log.Printf("Receiver failed to receiver info from server: %v", err)
			os.Exit(1)

		}

		log.Printf("Value received: %v", sharerIP)
		receiever.RequestData(sharerIP)

	}

}
