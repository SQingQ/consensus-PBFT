package node_server

import (
	"../structs"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/pprof"
)

// 负责网络请求与发送，收到请求后通过节点的通道转发请求体
type NodeServer struct {
	URL  string
	Node *Node
}

// 初始化服务器
func NewNodeServer(nodeID string) *NodeServer {
	node := NewNode(nodeID)
	server := &NodeServer{node.NodeTable[nodeID].URL, node}
	//go server.dispatchBroadCastRequestMsg()
	//go server.getRequestMsg()
	return server
}

// 启动服务，监听请求
func (server *NodeServer) Start() {
	fmt.Printf("NodeServer will be started at %s...\n", server.URL)
	mux := http.NewServeMux()
	server.setRoute(mux)
	s := &http.Server{
		Addr:         server.URL,
		WriteTimeout: 0,
		Handler:      mux,
	}
	_ = s.ListenAndServe()

}

//设置路由
func (server *NodeServer) setRoute(mux *http.ServeMux) {
	mux.HandleFunc("/viewChange", server.getViewChange)
	mux.HandleFunc("/newView", server.getNewView)
	mux.HandleFunc("/request", server.getRequest)
	mux.HandleFunc("/prePrepare", server.getPrePrepare)
	mux.HandleFunc("/prepare", server.getPrepare)
	mux.HandleFunc("/commit", server.getCommit)
	mux.HandleFunc("/broadCast", server.getBroadCastRequest)
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
}

func (server *NodeServer) getViewChange(writer http.ResponseWriter, request *http.Request) {
	var msg structs.ViewChangeMsg
	fmt.Println("接受ViewChange")
	_ = json.NewDecoder(request.Body).Decode(&msg)
	server.Node.ViewChangeMsgEntrance <- &msg
}

func (server *NodeServer) getNewView(writer http.ResponseWriter, request *http.Request) {
	var msg structs.NewViewMsg
	fmt.Println("接受NewView")
	_ = json.NewDecoder(request.Body).Decode(&msg)
	server.Node.NewViewMsgEntrance <- &msg
}

func (server *NodeServer) getBroadCastRequest(writer http.ResponseWriter, request *http.Request) {
	var msg structs.RequestMsg
	//fmt.Println("接受BroadCastRequest")
	_ = json.NewDecoder(request.Body).Decode(&msg)
	server.Node.BroadCastEntrance <- &msg
}

func (server *NodeServer) getRequest(writer http.ResponseWriter, request *http.Request) {
	var msg structs.RequestMsg
	//fmt.Println("接受request")
	_ = json.NewDecoder(request.Body).Decode(&msg)
	server.Node.RequestMsgEntrance <- &msg
}

func (server *NodeServer) getPrePrepare(writer http.ResponseWriter, request *http.Request) {
	var msg structs.PrePrepareMsg
	_ = json.NewDecoder(request.Body).Decode(&msg)
	//fmt.Println("getPrePrepare:", strconv.Itoa(int(msg.SequenceID)))
	server.Node.PrePrepareMsgEntrance <- &msg
}

func (server *NodeServer) getPrepare(writer http.ResponseWriter, request *http.Request) {
	var msg structs.PrepareMsg
	_ = json.NewDecoder(request.Body).Decode(&msg)
	//fmt.Println("getPrepare:" + msg.NodeID)
	server.Node.PrepareMsgEntrance <- &msg
}

func (server *NodeServer) getCommit(writer http.ResponseWriter, request *http.Request) {
	var msg structs.CommitMsg
	_ = json.NewDecoder(request.Body).Decode(&msg)
	//fmt.Println("getCommit:" + msg.NodeID)
	server.Node.CommitMsgEntrance <- &msg
}

func (server *NodeServer) getViewChangeMsg(writer http.ResponseWriter, request *http.Request) {
	var msg structs.CommitMsg
	_ = json.NewDecoder(request.Body).Decode(&msg)
	//fmt.Println("getCommit:" + msg.NodeID)
	server.Node.CommitMsgEntrance <- &msg
}

func (server *NodeServer) getNewViewMsg(writer http.ResponseWriter, request *http.Request) {
	var msg structs.CommitMsg
	_ = json.NewDecoder(request.Body).Decode(&msg)
	//fmt.Println("getCommit:" + msg.NodeID)
	server.Node.CommitMsgEntrance <- &msg
}

func send(url string, msg []byte) {
	buff := bytes.NewBuffer(msg)
	//net := &http.Client{
	//	Timeout:   time.Duration(50 * time.Millisecond),
	//	Transport: &http.Transport{},
	//}
	//request, _ := http.NewRequest("POST", "http://"+url, buff)
	//
	//request.Header.Set("Content-Type", "application/json")
	//resp, _ := net.Do(request)
	_, _ = http.Post("http://"+url, "application/json", buff)
}
