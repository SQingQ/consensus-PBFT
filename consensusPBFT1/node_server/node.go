package node_server

import (
	"../structs"
	"../tool"
	"container/list"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"strconv"
	"sync"
	"time"
	"unsafe"
)

type Node struct {
	NodeID                string                   // 节点ID
	NodeTable             map[string]*NodeInfo     // 节点表
	ClientTable           map[string]*ClientInfo   // 客户端信息
	AccountTable          map[string]int           // 账户信息
	ViewID                int                      // 当前视图ID
	Flag                  bool                     // 是否为主节点
	Count                 int                      // 计数器
	Lock                  sync.RWMutex             // 锁
	MsgNumber             int                      // 打包交易数
	F                     int                      // 最大容错故障节点数
	TimeDuration          int                      // 区块共识间隔
	SequenceID            int64                    // 当前分配ID
	CurrentStage          Stage                    // 当前状态
	RequestMsgBuffer      chan interface{}         // 客户端请求缓冲区
	PrePrepareMsgBuffer   *list.List               // PrePrepare消息缓冲
	PrepareMsgBuffer      *list.List               // Prepare消息缓冲
	CommitMsgBuffer       *list.List               // Commit消息缓冲
	ConsensusMsg          *ConsensusMsg            // 共识消息区
	BlockHash             []byte                   // 共识区块的hash
	BlockChain            *structs.BlockChain      // 区块链
	PrevBlockHash         []byte                   // 上一个区块的hash
	RequestMsgEntrance    chan interface{}         // 客户端请求消息入口
	BroadCastEntrance     chan interface{}         // 广播消息入口
	ViewChangeMsgEntrance chan interface{}         // 视图转换消息入口
	NewViewMsgEntrance    chan interface{}         // 新视图消息入口
	PrePrepareMsgEntrance chan interface{}         // 接受消息入口，服务器通过此通道交付请求，先将请求加入缓冲里
	PrepareMsgEntrance    chan interface{}         // 接受消息入口，服务器通过此通道交付请求，先将请求加入缓冲里
	CommitMsgEntrance     chan interface{}         // 接受消息入口，服务器通过此通道交付请求，先将请求加入缓冲里
	DispatchConsensus     chan bool                // 通知主节点开始打包交易，并广播
	DispatchPrePrepare    chan bool                // 通知副节点收到主节点的PrePrepare，并处理
	DispatchPrepare       chan bool                // 到达下一阶段后，使用此通道开始从缓冲里取出上一阶段提前收到的消息，并开始等待累计收到2f个请求，广播消息，进入下一阶段
	DispatchCommit        chan bool                // 到达下一阶段后，使用此通道开始从缓冲里取出上一阶段提前收到的消息，并开始等待累计收到2f个请求，广播消息，进入下一阶段
	TimeOut               chan bool                // 触发视图转换
	ViewChange            chan bool                // 等待2F个视图转换
	Consensus             chan bool                // 有一定数量交易后通知打包
	ViewChangeMsgs        []*structs.ViewChangeMsg // 视图转换消息
	TimeInfo              *TimeInfo                // 计算tps
}

type ConsensusMsg struct {
	PrePrepareMsg *structs.PrePrepareMsg
	PrepareMsgs   map[string]*structs.PrepareMsg
	CommitMsgs    map[string]*structs.CommitMsg
}

type Stage int

const (
	Idle        Stage = iota
	WaitPrepare       // 1
	WaitCommit        // 2
	WaitNewView       // 3
)

type Config struct {
	ViewID       int                    `yaml:"ViewID"`
	F            int                    `yaml:"F"`
	MsgNumber    int                    `yaml:"MsgNumber"`
	TimeDuration int                    `yaml:"TimeDuration"`
	SequenceID   int64                  `yaml:"SequenceID"`
	ClientTable  map[string]*ClientInfo `yaml:"ClientTable"`
	NodeTable    map[string]*NodeInfo   `yaml:"NodeTable"`
	AccountTable map[string]int         `yaml:"AccountTable"`
}

type NodeInfo struct {
	URL        string         `yaml:"URL"`
	PublicKey  rsa.PublicKey  `yaml:"PublicKey"`
	PrivateKey rsa.PrivateKey `yaml:"PrivateKey"`
}

type TimeInfo struct {
	StartTime int64
	EndTime   int64
	Time      int64
	Count     int64
	Counts    int64
}

type ClientInfo struct {
	URL        string         `yaml:"URL"`
	PublicKey  rsa.PublicKey  `yaml:"PublicKey"`
	PrivateKey rsa.PrivateKey `yaml:"PrivateKey"`
}

func NewNode(nodeID string) *Node {
	yamlFile, _ := ioutil.ReadFile("./resources/nodeConfig.yaml")
	config := Config{}
	_ = yaml.Unmarshal(yamlFile, &config)

	config.ClientTable["1"].PrivateKey = tool.GetPrivateKey("./resources/client_key/client_private_key.pem")
	config.ClientTable["1"].PublicKey = tool.GetPublicKey("./resources/client_key/client_public_key.pem")

	for i := 1; i <= config.F*3+1; i++ {
		id := strconv.Itoa(i)
		config.NodeTable[id].PrivateKey = tool.GetPrivateKey("./resources/consensus_node_key/consensus_Node" + id + "_private_key.pem")
		config.NodeTable[id].PublicKey = tool.GetPublicKey("./resources/consensus_node_key/consensus_Node" + id + "_public_key.pem")
	}

	genesisBlock := structs.NewGenesisBlock()
	blockChain := structs.NewBlockChain(genesisBlock)
	node := &Node{
		NodeID:              nodeID,
		NodeTable:           config.NodeTable,
		ClientTable:         config.ClientTable,
		AccountTable:        config.AccountTable,
		ViewID:              config.ViewID,
		Flag:                nodeID == strconv.Itoa(config.ViewID+1),
		Count:               0,
		Lock:                sync.RWMutex{},
		MsgNumber:           config.MsgNumber,
		F:                   config.F,
		TimeDuration:        config.TimeDuration,
		SequenceID:          config.SequenceID,
		RequestMsgBuffer:    make(chan interface{}, 100000),
		PrePrepareMsgBuffer: list.New(),
		PrepareMsgBuffer:    list.New(),
		CommitMsgBuffer:     list.New(),
		CurrentStage:        Idle,
		ConsensusMsg: &ConsensusMsg{
			PrePrepareMsg: nil,
			PrepareMsgs:   make(map[string]*structs.PrepareMsg),
			CommitMsgs:    make(map[string]*structs.CommitMsg),
		},
		BlockHash:             []byte{},
		BlockChain:            blockChain,
		PrevBlockHash:         genesisBlock.Hash,
		RequestMsgEntrance:    make(chan interface{}, 100000),
		BroadCastEntrance:     make(chan interface{}, 100000),
		ViewChangeMsgEntrance: make(chan interface{}, config.F*3),
		NewViewMsgEntrance:    make(chan interface{}, config.F*3),
		PrePrepareMsgEntrance: make(chan interface{}, config.F*3),
		PrepareMsgEntrance:    make(chan interface{}, config.F*3),
		CommitMsgEntrance:     make(chan interface{}, config.F*3),
		DispatchConsensus:     make(chan bool),
		DispatchPrePrepare:    make(chan bool, 1),
		DispatchPrepare:       make(chan bool, 1),
		DispatchCommit:        make(chan bool, 1),
		TimeOut:               make(chan bool, 1),
		ViewChange:            make(chan bool, 1),
		Consensus:             make(chan bool, 1000),
		ViewChangeMsgs:        []*structs.ViewChangeMsg{},
		TimeInfo: &TimeInfo{
			StartTime: 0,
			EndTime:   0,
			Count:     0,
			Counts:    0,
		},
	}

	// 超时发送视图转换
	go node.timeOut()

	go node.dispatchViewChange()

	go node.dispatchNewView()

	// 开始接受共识消息

	go node.receivePrepareMsg()

	go node.receiveCommitMsg()

	go node.dispatchPrepareMsg()

	go node.dispatchCommitMsg()

	go node.receivePrePrepareMsg()

	go node.dispatchPrePrepareMsg()

	go node.dispatchConsensus()

	if node.Flag {
		// 主节点打包交易并进行共识
		node.DispatchConsensus <- true
	}

	// 开始接受request消息
	go node.getRequestMsg()

	return node
}

// 接受客户端消息
func (node *Node) getRequestMsg() {
	count := 0
	for {
		select {
		case msg := <-node.BroadCastEntrance:
			requestMsg := msg.(*structs.RequestMsg)
			if node.validateRequestMsg(requestMsg) && requestMsg.Timestamp < time.Now().UnixNano() {
				if node.Flag {
					node.RequestMsgBuffer <- *requestMsg
					count++
					if count == node.MsgNumber {
						count = 0
						node.Consensus <- true
					}
				}
			}
		case msg := <-node.RequestMsgEntrance:
			requestMsg := msg.(*structs.RequestMsg)
			requestMsg.Timestamp = time.Now().UnixNano()
			if node.validateRequestMsg(requestMsg) {
				if node.Flag {
					node.RequestMsgBuffer <- *requestMsg
					node.Lock.Lock()
					count++
					if count == node.MsgNumber {
						count = 0
						node.Consensus <- true
					}
					node.Lock.Unlock()
				}
				node.broadcastRequest(*requestMsg)
			}
		}
	}
}

// 验证RequestMsg
func (node *Node) validateRequestMsg(requestMsg *structs.RequestMsg) bool {
	return tool.VerifyDigest(*(*[]byte)(unsafe.Pointer(requestMsg.Message)), requestMsg.Signature, node.ClientTable[requestMsg.ClientID].PublicKey) &&
		node.AccountTable[requestMsg.Message.Drawee] >= requestMsg.Message.Amount
}

// 打包交易
func (node *Node) timeOut() {
	for {
		<-node.TimeOut
		viewChangeMsg := structs.ViewChangeMsg{
			NewViewID:  (node.ViewID + 1) % (node.F*3 + 1),
			SequenceID: node.SequenceID,
			NodeID:     node.NodeID,
			Signature:  tool.GetDigest([]byte(node.NodeID), node.NodeTable[node.NodeID].PrivateKey),
		}
		fmt.Println("开始视图转换")
		node.TimeInfo.StartTime = time.Now().UnixNano()
		node.broadcast(viewChangeMsg, "/viewChange")
	}
}

func (node *Node) dispatchViewChange() {
	for {
		msg := <-node.ViewChangeMsgEntrance
		viewChangeMsg := msg.(*structs.ViewChangeMsg)
		if node.validateViewChange(viewChangeMsg) {
			node.ViewChangeMsgs = append(node.ViewChangeMsgs, viewChangeMsg)
		}
		if len(node.ViewChangeMsgs) == node.F*2 {
			if node.NodeID == strconv.Itoa((node.ViewID+1)%(node.F*3+1)+1) {
				newViewMsg := structs.NewViewMsg{
					NewViewID:      (node.ViewID + 1) % (node.F*3 + 1),
					ViewChangeMsgs: node.ViewChangeMsgs,
					Signature:      tool.GetDigest([]byte(node.NodeID), node.NodeTable[node.NodeID].PrivateKey),
				}
				node.broadcast(newViewMsg, "/newView")
				node.clearData()
				node.ViewID = (node.ViewID + 1) % (node.F*3 + 1)
				node.Flag = true
				node.CurrentStage = Idle
				fmt.Println(node.ViewID)
				fmt.Println("视图转换结束,成为主节点")
				node.DispatchConsensus <- true
			} else {
				node.ViewChange <- true
			}
		}
	}
}

func (node *Node) dispatchNewView() {
	for {
		msg := <-node.NewViewMsgEntrance
		newView := msg.(*structs.NewViewMsg)
		if node.validateNewView(newView) {
			<-node.ViewChange
			node.clearData()
			node.ViewID = (node.ViewID + 1) % (node.F*3 + 1)
			node.Flag = false
			node.CurrentStage = Idle
			fmt.Println(node.ViewID)
			node.TimeInfo.EndTime = time.Now().UnixNano()
			println("视图转换时间：", node.TimeInfo.EndTime-node.TimeInfo.StartTime)
			fmt.Println("视图转换结束")
		}
	}
}

func (node *Node) clearData() {
	node.PrepareMsgBuffer.Init()
	node.CommitMsgBuffer.Init()
	node.PrePrepareMsgBuffer.Init()
	node.ConsensusMsg.CommitMsgs = make(map[string]*structs.CommitMsg)
	node.ConsensusMsg.PrepareMsgs = make(map[string]*structs.PrepareMsg)
	node.ConsensusMsg.PrePrepareMsg = nil
	node.ViewChangeMsgs = []*structs.ViewChangeMsg{}
}

// 验证viewChange
func (node *Node) validateViewChange(viewChangeMsg *structs.ViewChangeMsg) bool {
	return (node.ViewID+1)%(node.F*3+1) == viewChangeMsg.NewViewID &&
		node.SequenceID == viewChangeMsg.SequenceID &&
		tool.VerifyDigest([]byte(viewChangeMsg.NodeID), viewChangeMsg.Signature, node.NodeTable[viewChangeMsg.NodeID].PublicKey)
}

// 验证newView
func (node *Node) validateNewView(newViewMsg *structs.NewViewMsg) bool {
	return (node.ViewID+1)%(node.F*3+1) == newViewMsg.NewViewID &&
		len(newViewMsg.ViewChangeMsgs) == node.F*2 &&
		tool.VerifyDigest([]byte(strconv.Itoa(newViewMsg.NewViewID+1)), newViewMsg.Signature, node.NodeTable[strconv.Itoa(newViewMsg.NewViewID+1)].PublicKey)
}

// 打包交易
func (node *Node) dispatchConsensus() {
	for {
		<-node.DispatchConsensus
		//time.Sleep(time.Millisecond * 50)
		<-node.Consensus
		println("[长度]", len(node.Consensus))
		var requestMsgs []structs.RequestMsg
		for i := 0; i < node.MsgNumber; i++ {
			msg := <-node.RequestMsgBuffer
			requestMsg := msg.(structs.RequestMsg)
			requestMsgs = append(requestMsgs, requestMsg)
		}
		block := structs.NewBlock(requestMsgs, node.PrevBlockHash)
		t := &structs.Block{
			Timestamp:     block.Timestamp,
			Data:          nil,
			PrevBlockHash: block.PrevBlockHash,
			Hash:          block.Hash,
		}
		node.BlockHash = *(*[]byte)(unsafe.Pointer(t))
		prePrepareMsg := structs.PrePrepareMsg{
			Block:      block,
			ViewID:     node.ViewID,
			SequenceID: node.SequenceID + 1,
			Digest:     tool.GenerateHash(node.BlockHash),
			Signature:  tool.GetDigest(tool.GenerateHash(node.BlockHash), node.NodeTable[strconv.Itoa(node.ViewID+1)].PrivateKey),
		}
		if node.NodeID == "1" {
			prePrepareMsg.ViewID = -1
			node.broadcast(prePrepareMsg, "/prePrepare")
			//fmt.Println("已发送preprepare")
		} else {
			tool.LogSendMsg(&prePrepareMsg)
			node.CurrentStage = WaitPrepare
			node.ConsensusMsg.PrePrepareMsg = &prePrepareMsg

			node.TimeInfo.StartTime = time.Now().UnixNano()
			node.TimeInfo.Count = int64(len(block.Data))

			node.broadcast(prePrepareMsg, "/prePrepare")
			node.DispatchPrepare <- true
			//fmt.Println("已发送preprepare")
		}
	}

}

func (node *Node) receivePrePrepareMsg() {
	for {
		select {
		case msg := <-node.PrePrepareMsgEntrance:
			prePrepareMsg := msg.(*structs.PrePrepareMsg)
			//fmt.Println(node.NodeID + "-prePrepareMsg-" + strconv.Itoa(int(prePrepareMsg.SequenceID)))
			if node.CurrentStage == Idle {
				if node.validateBlock(prePrepareMsg) {
					//tool.LogReceiveMsg(prePrepareMsg)
					//fmt.Println(node.NodeID + "-接受prePrepareMsg消息-" + strconv.Itoa(int(prePrepareMsg.SequenceID)))
					node.ConsensusMsg.PrePrepareMsg = prePrepareMsg
					node.sendPrepare()
					node.CurrentStage = WaitPrepare
					node.DispatchPrepare <- true
				} else {
					node.TimeOut <- true
				}
			} else {
				// 发送到缓冲区
				if node.ViewID == prePrepareMsg.ViewID &&
					tool.VerifyDigest(prePrepareMsg.Digest, prePrepareMsg.Signature, node.NodeTable[strconv.Itoa(node.ViewID+1)].PublicKey) {
					//fmt.Println(node.NodeID + "-放入prePrepareMsg缓冲-" + strconv.Itoa(int(prePrepareMsg.SequenceID)))
					node.PrePrepareMsgBuffer.PushBack(prePrepareMsg)
					node.DispatchPrePrepare <- true
				} else {
					node.TimeOut <- true
				}
			}
		case <-time.After(2000000 * time.Second):
			if !node.Flag {
				node.TimeOut <- true
			}
		}

	}
}

func (node *Node) receivePrepareMsg() {
	for {
		msg := <-node.PrepareMsgEntrance
		prepareMsg := msg.(*structs.PrepareMsg)
	//	fmt.Println(node.NodeID + "-prepareMsg-" + prepareMsg.NodeID + " id:" + strconv.Itoa(int(prepareMsg.SequenceID)))
		if node.CurrentStage == WaitPrepare {
			if node.validatePrepareMsg(prepareMsg) {
				//fmt.Println(node.NodeID + "-接受prepareMsg消息-" + prepareMsg.NodeID)
				node.ConsensusMsg.PrepareMsgs[prepareMsg.NodeID] = prepareMsg
			}
		} else if node.CurrentStage == Idle {
			// 发送到缓冲区
			if node.validatePrepareMsg(prepareMsg) {
			//	fmt.Println(node.NodeID + "-放入prepareMsg缓冲-" + prepareMsg.NodeID)
				node.PrepareMsgBuffer.PushBack(prepareMsg)
			}
		}
	}
}

func (node *Node) receiveCommitMsg() {
	for {
		msg := <-node.CommitMsgEntrance
		commitMsg := msg.(*structs.CommitMsg)
		//fmt.Println(node.NodeID + "-commitMsg-" + commitMsg.NodeID + " id:" + strconv.Itoa(int(commitMsg.SequenceID)))
		if node.CurrentStage == WaitCommit {
			if node.validateCommitMsg(commitMsg) {
				//fmt.Println(node.NodeID + "-接受commitMsg消息-" + commitMsg.NodeID)
				node.ConsensusMsg.CommitMsgs[commitMsg.NodeID] = commitMsg
			}
		} else if node.CurrentStage == WaitPrepare || node.CurrentStage == Idle {
			if node.validateCommitMsg(commitMsg) {
				// 发送到缓冲区
				//fmt.Println(node.NodeID + "-放入commitMsg缓冲-" + commitMsg.NodeID)
				node.CommitMsgBuffer.PushBack(commitMsg)
				//fmt.Print("commit缓冲数：")
				//fmt.Println(node.CommitMsgBuffer.Len())
			}

		}
	}
}

// 接受PrePrepareMsg消息
func (node *Node) dispatchPrePrepareMsg() {
	for {
		<-node.DispatchPrePrepare
		for {
			if node.CurrentStage == Idle {
				e := node.PrePrepareMsgBuffer.Front()
				prePrepareMsg := e.Value.(*structs.PrePrepareMsg)
				node.PrePrepareMsgBuffer.Init()
				node.ConsensusMsg.PrePrepareMsg = prePrepareMsg
				node.sendPrepare()
				node.CurrentStage = WaitPrepare
				node.DispatchPrepare <- true
				break
			}
		}
	}
}

// 副节点接受PrePrepare消息
func (node *Node) sendPrepare() {
	t := &structs.Block{
		Timestamp:     node.ConsensusMsg.PrePrepareMsg.Block.Timestamp,
		Data:          nil,
		PrevBlockHash: node.ConsensusMsg.PrePrepareMsg.Block.PrevBlockHash,
		Hash:          node.ConsensusMsg.PrePrepareMsg.Block.Hash,
	}
	node.BlockHash = *(*[]byte)(unsafe.Pointer(t))
	prepareMsg := structs.PrepareMsg{
		ViewID:     node.ViewID,
		SequenceID: node.SequenceID + 1,
		NodeID:     node.NodeID,
		Digest:     node.ConsensusMsg.PrePrepareMsg.Digest,
		Signature:  tool.GetDigest(node.ConsensusMsg.PrePrepareMsg.Digest, node.NodeTable[node.NodeID].PrivateKey),
	}
	tool.LogSendMsg(&prepareMsg)
	node.broadcast(prepareMsg, "/prepare")
}

// 验证Block
func (node *Node) validateBlock(prePrepareMsg *structs.PrePrepareMsg) bool {
	return node.ViewID == prePrepareMsg.ViewID &&
		node.SequenceID+1 == prePrepareMsg.SequenceID &&
		tool.VerifyDigest(prePrepareMsg.Digest, prePrepareMsg.Signature, node.NodeTable[strconv.Itoa(node.ViewID+1)].PublicKey)
}

// 接受PrepareMsg消息
func (node *Node) dispatchPrepareMsg() {
	for {
		<-node.DispatchPrepare
		//fmt.Println("处理缓存")
		prepareMsg := &structs.PrepareMsg{}
		e := node.PrepareMsgBuffer.Front()
		for e != nil {
			prepareMsg = e.Value.(*structs.PrepareMsg)
			if node.validatePrepareMsg(prepareMsg) {
				//tool.LogReceiveMsg(prepareMsg)
				node.ConsensusMsg.PrepareMsgs[prepareMsg.NodeID] = prepareMsg
			}
			e = e.Next()
		}

		node.PrepareMsgBuffer.Init()
		//fmt.Println("等待2f-1个prepare")
		for {
			if len(node.ConsensusMsg.PrepareMsgs) >= node.F*2-1 {
				node.CurrentStage = WaitCommit
				node.sendCommit()
				node.DispatchCommit <- true
				break
			}
		}
	}
}

// 发送Commit
func (node *Node) sendCommit() {
	commitMsg := structs.CommitMsg{
		ViewID:     node.ViewID,
		SequenceID: node.SequenceID + 1,
		NodeID:     node.NodeID,
		Digest:     node.ConsensusMsg.PrePrepareMsg.Digest,
		Signature:  tool.GetDigest(node.ConsensusMsg.PrePrepareMsg.Digest, node.NodeTable[node.NodeID].PrivateKey),
	}
	tool.LogSendMsg(&commitMsg)
	node.broadcast(commitMsg, "/commit")
}

// 验证PrepareMsg
func (node *Node) validatePrepareMsg(prepareMsg *structs.PrepareMsg) bool {
	return node.ViewID == prepareMsg.ViewID &&
		node.SequenceID+1 == prepareMsg.SequenceID &&
		tool.VerifyDigest(prepareMsg.Digest, prepareMsg.Signature, node.NodeTable[prepareMsg.NodeID].PublicKey)
}

// 接受CommitMsg消息
func (node *Node) dispatchCommitMsg() {
	for {
		<-node.DispatchCommit
		fmt.Println("等待2f个commit")
		commitMsg := &structs.CommitMsg{}
		e := node.CommitMsgBuffer.Front()
		for e != nil {
			commitMsg = e.Value.(*structs.CommitMsg)
			if node.validateCommitMsg(commitMsg) {
				//tool.LogReceiveMsg(commitMsg)
				node.ConsensusMsg.CommitMsgs[commitMsg.NodeID] = commitMsg
			}
			e = e.Next()
		}
		node.CommitMsgBuffer.Init()
		for {
			//fmt.Println(len(node.ConsensusMsg.CommitMsgs))
			if len(node.ConsensusMsg.CommitMsgs) >= node.F*2 {
				if node.Flag {
					node.TimeInfo.EndTime = time.Now().UnixNano()
					node.TimeInfo.Time += node.TimeInfo.EndTime - node.TimeInfo.StartTime
					node.TimeInfo.Counts += node.TimeInfo.Count
					//fmt.Println("消息数目：",float64(node.MsgNumber))
					//println("Time", float64(node.TimeInfo.EndTime-node.TimeInfo.StartTime)/1000000000)
					println("tps", float64(node.MsgNumber)/(float64(node.TimeInfo.EndTime-node.TimeInfo.StartTime)/1000000000))
					println("delay", float64(node.TimeInfo.EndTime-node.TimeInfo.StartTime)/1000000000)
				}
				node.SequenceID += 1
				node.BlockHash = []byte{}
				block := node.ConsensusMsg.PrePrepareMsg.Block
				node.PrevBlockHash = block.Hash
				node.BlockChain.AddBlock(block)
				fmt.Println("结束一轮共识" + strconv.Itoa(int(node.ConsensusMsg.PrePrepareMsg.SequenceID)))
				node.ConsensusMsg.CommitMsgs = make(map[string]*structs.CommitMsg)
				node.ConsensusMsg.PrepareMsgs = make(map[string]*structs.PrepareMsg)
				node.ConsensusMsg.PrePrepareMsg = nil
				//node.sendReply(block)
				node.clearBuffer()
				node.CurrentStage = Idle
				if node.Flag {
					node.DispatchConsensus <- true
				}
				break
			}
		}
	}
}

// 发送ReplyMsg
func (node *Node) sendReply(block *structs.Block) {
	for _, requestMsg := range block.Data {
		replyMsg := structs.ReplyMsg{
			RequestMsg: requestMsg,
			Result:     "success",
		}
		jsonMsg, _ := json.Marshal(replyMsg)
		//tool.LogSendMsg(&replyMsg)
		go send(node.ClientTable[requestMsg.ClientID].URL+"/reply", jsonMsg)
	}
}

// 验证CommitMsg
func (node *Node) validateCommitMsg(commitMsg *structs.CommitMsg) bool {
	return node.ViewID == commitMsg.ViewID &&
		node.SequenceID+1 == commitMsg.SequenceID &&
		tool.VerifyDigest(commitMsg.Digest, commitMsg.Signature, node.NodeTable[commitMsg.NodeID].PublicKey)
}

// 广播消息
func (node *Node) broadcast(msg interface{}, path string) {
	for nodeID, nodeInfo := range node.NodeTable {
		if nodeID == node.NodeID {
			continue
		}
		jsonMsg, _ := json.Marshal(msg)
		fmt.Println(nodeInfo.URL + path)
		time.Sleep(time.Millisecond * 15)
		send(nodeInfo.URL+path, jsonMsg)
	}
}

// 广播消息
func (node *Node) broadcastRequest(msg interface{}) {
	for nodeID, nodeInfo := range node.NodeTable {
		if nodeID == node.NodeID {
			continue
		}
		jsonMsg, _ := json.Marshal(msg)
		send(nodeInfo.URL+"/broadCast", jsonMsg)
		time.Sleep(time.Millisecond * 15)
	}
}

func (node *Node) clearBuffer() {
	node.PrepareMsgBuffer.Init()
	node.CommitMsgBuffer.Init()
}
