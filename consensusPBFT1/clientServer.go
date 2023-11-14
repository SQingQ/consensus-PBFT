package main

import (
	"./client_server"
	"os"
)

func main() {
	clientID := os.Args[1]
	server := client_server.NewClientServer(clientID)
	server.Start()
}
