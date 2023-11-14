package main

import (
	"./client_server"
	"os"
)

func main() {
	clientID := os.Args[1]     //os.Args[0]第二个参数是用户输入
	//fmt.Println(clientID)
	server := client_server.NewClientServer(clientID)     //初始化服务器
	//顺序就是先初始化服务器1
	//在初始化服务器的时候创建新的客户端2
	//fmt.Println("4")
	//输出ClientServer will be started at localhost:1110...
	//实现sendRRequestMsg()，进行写请求发送 3
	server.Start()
	//只实现了写请求，没有将读请求放进来
}


//候选节点收到client发送过来的消息--------1
//候选节点将消息发送到所有共识节点，接到一条发送一条
//共识节点接受消息后，如果是主节点则消息条数达到2000就可以通知打包
//