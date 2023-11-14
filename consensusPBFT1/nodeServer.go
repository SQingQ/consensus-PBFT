package main

import (
	"./node_server"
	"os"
)

func main() {
	nodeID := os.Args[1]
	server := node_server.NewNodeServer(nodeID)
	server.Start()
}
