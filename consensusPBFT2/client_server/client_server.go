package client_server

import (
	"../structs"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// 负责网络请求与发送，收到请求后通过节点的通道转发请求体
type ClientServer struct {
	URL    string
	Client *Client
}

// 初始化服务器
func NewClientServer(clientID string) *ClientServer {
	//fmt.Println("1")
	client := NewClient(clientID)     //创建客户端
	server := &ClientServer{client.ClientInfo.URL, client}
	return server
}

// 启动服务，监听请求   实现接口
func (server *ClientServer) Start() {
	fmt.Printf("ClientServer will be started at %s...\n", server.URL)
	mux := http.NewServeMux()     //Handler接口和ServeMux结构
	//fmt.Println("mux",mux)
	server.setRoute(mux)         //设置路由
	s := &http.Server{
		Addr:         server.URL,   //服务器的IP地址和端口信息
		WriteTimeout: 0,
		ReadTimeout:  0,
		Handler:      mux,           //请求处理函数的路由复制器
	}
	_ = s.ListenAndServe()
}

//设置路由   实现接口
func (server *ClientServer) setRoute(mux *http.ServeMux) {
	mux.HandleFunc("/rReply", server.getRReply)  //read回复
	mux.HandleFunc("/wReply", server.getWReply)  //写回复
	mux.HandleFunc("/change", server.getRReply)  //回复
}

func (server *ClientServer) getRReply(writer http.ResponseWriter, request *http.Request) {
	var replyMsg structs.RReplyMsg
	_ = json.NewDecoder(request.Body).Decode(&replyMsg)
	server.Client.ReplyMsgEntrance <- &replyMsg
	//fmt.Printf("[SEND_REPLY] ClientID: %s, Message: %s, Result: %s\n", replyMsg.RequestMsg.ClientID, tool.PrintMessage(replyMsg.RequestMsg.Message), replyMsg.Result)
}

func (server *ClientServer) getWReply(writer http.ResponseWriter, request *http.Request) {
	//var replyMsg structs.ReplyMsg
	//_ = json.NewDecoder(request.Body).Decode(&replyMsg)
	//fmt.Printf("[SEND_REPLY] ClientID: %s, Message: %s, Result: %s\n", replyMsg.RequestMsg.ClientID, tool.PrintMessage(replyMsg.RequestMsg.Message), replyMsg.Result)
}

func (server *ClientServer) getChange(writer http.ResponseWriter, request *http.Request) {
	var changeMsg structs.ChangeMsg
	_ = json.NewDecoder(request.Body).Decode(&changeMsg)
	server.Client.ChangeMsgEntrance <- &changeMsg
	//fmt.Printf("[SEND_REPLY] ClientID: %s, Message: %s, Result: %s\n", replyMsg.RequestMsg.ClientID, tool.PrintMessage(replyMsg.RequestMsg.Message), replyMsg.Result)
}
//客户端发送消息给候选节点
func send(url string, msg []byte) {
	buff := bytes.NewBuffer(msg)    //定义一个缓冲区，缓冲区大小就是msg的大小
	//fmt.Println(buff)
	_, _ = http.Post("http://"+url, "application/json", buff)
}
