package client_server

import (
	"bytes"
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
	client := NewClient(clientID)
	server := &ClientServer{client.ClientInfo.URL, client}
	return server
}

// 启动服务，监听请求
func (server *ClientServer) Start() {
	fmt.Printf("ClientServer will be started at %s...\n", server.URL)
	mux := http.NewServeMux()
	server.setRoute(mux)
	s := &http.Server{
		Addr:         server.URL,
		WriteTimeout: 0,
		ReadTimeout:  0,
		Handler:      mux,
	}
	_ = s.ListenAndServe()
}

//设置路由
func (server *ClientServer) setRoute(mux *http.ServeMux) {
	mux.HandleFunc("/reply", server.getReply)
}

func (server *ClientServer) getReply(writer http.ResponseWriter, request *http.Request) {
	//var replyMsg structs.ReplyMsg
	//_ = json.NewDecoder(request.Body).Decode(&replyMsg)
	//fmt.Printf("[SEND_REPLY] ClientID: %s, Message: %s, Result: %s\n", replyMsg.RequestMsg.ClientID, tool.PrintMessage(replyMsg.RequestMsg.Message), replyMsg.Result)
}

func send(url string, msg []byte) {
	buff := bytes.NewBuffer(msg)
	//net := &http.Client{
	//	Transport: &http.Transport{
	//		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	//		IdleConnTimeout: time.Duration(2 * time.Millisecond),
	//	},
	//}
	//request, _ := http.NewRequest("POST", "http://"+url, buff)
	//request.Header.Set("Content-Type", "application/json")
	//response, _ := net.Do(request)
	//defer response.Body.Close()
	//_, _ = ioutil.ReadAll(response.Body)
	_, _ = http.Post("http://"+url, "application/json", buff)
}
