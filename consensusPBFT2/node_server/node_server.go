package node_server

import (
	"../structs"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/pprof"
	"strconv"
)

// 负责网络请求与发送，收到请求后通过节点的通道转发请求体
type NodeServer struct {
	URL  string
	Node *Node
}

// 初始化服务器
func NewNodeServer(nodeID string, nodeType string) *NodeServer {
	node := NewNode(nodeID, nodeType)   //参数第一位代表节点ID，第二个代表节点类型：0代表共识节点，1代表候选节点
	var server *NodeServer
	if nodeType == "0" {  //共识节点
		server = &NodeServer{node.ConsensusNodeTable[nodeID].URL, node}
	} else {  //候选节点
		server = &NodeServer{node.CandidateNodeTable[nodeID].URL, node}
	}
	return server
}

// 启动服务，监听请求
func (server *NodeServer) Start() {
	fmt.Printf("NodeServer will be started at %s...\n", server.URL)
	mux := http.NewServeMux()  //Handler接口和ServeMux结构
	server.setRoute(mux)       //设置路由
	s := &http.Server{
		Addr:         server.URL,   //服务器的IP地址和端口信息
		WriteTimeout: 0,
		Handler:      mux,          //请求处理函数的路由复制器
	}
	_ = s.ListenAndServe()

}

//设置路由
func (server *NodeServer) setRoute(mux *http.ServeMux) {
	mux.HandleFunc("/request", server.getRequest)     //候选节点将消息发送共识节点的路由
	mux.HandleFunc("/wRequest", server.getWRequest)   //client中写请求的发送中URL尾缀为/wRequest，将走server.getWRequest
	mux.HandleFunc("/rRequest", server.getRRequest)
	mux.HandleFunc("/prePrepare", server.getPrePrepare)  //主节点打包后发送的pre-prepare，传到这里
	mux.HandleFunc("/prepare", server.getPrepare)        //共识节点发送prepare消息
	mux.HandleFunc("/commit", server.getCommit)          //发送commit消息调用getCommit方法
	mux.HandleFunc("/change", server.getChange)          //
	mux.HandleFunc("/debug/pprof/", pprof.Index)         //
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)//
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)//
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)  //
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)    //
}

//接受候选节点发过来的消息，候选节点只发送写请求消息给共识节点
func (server *NodeServer) getRequest(writer http.ResponseWriter, request *http.Request) {
	var msg structs.WRequestMsg        //定义写请求msg
	//fmt.Println("接受request")
	_ = json.NewDecoder(request.Body).Decode(&msg)        //编码
	server.Node.RequestMsgEntrance <- &msg                //将msg传入Node.RequestMsgEntrance
}

//候选节点接收到客户端发送过来的写请求，并且将消息发送至节点写消息缓存区
func (server *NodeServer) getWRequest(writer http.ResponseWriter, request *http.Request) {
	var msg structs.WRequestMsg      //定义写请求msg
	//fmt.Println("接受wRequest")
	_ = json.NewDecoder(request.Body).Decode(&msg)        //编码
	server.Node.WRequestMsgEntrance <- &msg               //将msg传入Node.WRequestMsgEntrance
}

func (server *NodeServer) getRRequest(writer http.ResponseWriter, request *http.Request) {
	var msg structs.RRequestMsg
	//fmt.Println("接受request")
	_ = json.NewDecoder(request.Body).Decode(&msg)
	server.Node.RRequestMsgEntrance <- &msg
}

//接收主节点发送的Pre-prepare消息
func (server *NodeServer) getPrePrepare(writer http.ResponseWriter, request *http.Request) {
	var msg structs.PrePrepareMsg        //定义一个preprepare类型的msg
	_ = json.NewDecoder(request.Body).Decode(&msg)    //编码
	//fmt.Println("getPrePrepare:", strconv.Itoa(int(msg.SequenceID)))
	if msg.Url == server.Node.ConsensusNodeTable[strconv.Itoa(server.Node.ViewID+1)].URL {       //判断是不是主节点发送的pre-prepare消息
		server.Node.PrePrepareMsgEntrance <- &msg                                                //如果是，消息存放在PrePrepareMsgEntrance
	}

}

//获取prepare消息
func (server *NodeServer) getPrepare(writer http.ResponseWriter, request *http.Request) {
	var msg structs.PrepareMsg
	_ = json.NewDecoder(request.Body).Decode(&msg)
	//fmt.Println("getPrepare:" + msg.NodeID)
	if msg.Url == server.Node.ConsensusNodeTable[msg.NodeID].URL {   //确定msg的Url和共识节点表中的对应的ID的Url
		server.Node.PrepareMsgEntrance <- &msg    //消息发送到PrepareMsgEntranc
	}
}

//发送commit消息
func (server *NodeServer) getCommit(writer http.ResponseWriter, request *http.Request) {
	var msg structs.CommitMsg     //定义commitMsg
	_ = json.NewDecoder(request.Body).Decode(&msg)   //编码
	//fmt.Println("getCommit:" + msg.NodeID)
	if msg.Url == server.Node.ConsensusNodeTable[msg.NodeID].URL {
		server.Node.CommitMsgEntrance <- &msg     //将msg存储到CommitMsgEntrance
	}
}

func (server *NodeServer) getChange(writer http.ResponseWriter, request *http.Request) {
	var msg structs.ChangeMsg
	_ = json.NewDecoder(request.Body).Decode(&msg)
	//fmt.Println("getCommit:" + msg.NodeID)
	server.Node.ChangeMsgEntrance <- &msg
}
//发送msg消息，将josn的byte类型的消息发送到指定url
func send(url string, msg []byte) {
	buff := bytes.NewBuffer(msg)   //定义一个缓冲区，缓冲区大小就是msg的大小
	_, _ = http.Post("http://"+url, "application/json", buff)
}
