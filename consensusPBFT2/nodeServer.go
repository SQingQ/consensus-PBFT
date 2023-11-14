package main

import (
	"./node_server"
	"os"
)

func main() {
	nodeID := os.Args[1]    //os.Args[0]第二个参数是用户输入
	nodeType := os.Args[2]
	server := node_server.NewNodeServer(nodeID,nodeType)
	//fmt.Println(server)
	server.Start()
}