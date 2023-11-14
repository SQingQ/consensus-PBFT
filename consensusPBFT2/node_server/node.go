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
	NodeID                string                        // 节点ID
	NodeType              string                        // 节点类型 "0":共识 "1":候补
	ConsensusNodeTable    map[string]*NodeInfo          // 共识节点表
	CandidateNodeTable    map[string]*NodeInfo          // 候补节点表
	ClientTable           map[string]*ClientInfo        // 客户端信息
	AccountTable          map[string]int                // 账户信息
	CreditTable           CreditTable                   // 信誉值阈值
	AdjustmentRange       AdjustmentRange               // 奖惩值
	ViewID                int                           // 当前视图ID
	Flag                  bool                          // 是否为主节点
	Lock                  sync.RWMutex                  // 锁
	MsgNumber             int                           // 打包交易数
	F                     int                           // 最大容错故障节点数
	SequenceID            int64                         // 当前分配ID
	CurrentStage          Stage                         // 当前状态 就是int类型

	RequestMsgBuffer      chan interface{}              // 候选节点请求转发到的缓冲区
	PrePrepareMsgBuffer   *list.List                    // PrePrepare消息缓冲
	PrepareMsgBuffer      *list.List                    // Prepare消息缓冲
	CommitMsgBuffer       *list.List                    // Commit消息缓冲

	ConsensusMsg          *ConsensusMsg                 // 共识消息区

	ChangeMsgs            map[string]*structs.ChangeMsg // 转换消息
	PrepareState          map[string]int                // 状态表  0:正常 1:缺席 2:错误
	BlockHash             []byte                        // 共识区块的hash
	BlockChain            *structs.BlockChain           // 区块链
	PrevBlockHash         []byte                        // 上一个区块的hash

	RequestMsgEntrance    chan interface{}              // 主节点请求入口
	WRequestMsgEntrance   chan interface{}              // 客户端写请求入口
	RRequestMsgEntrance   chan interface{}              // 客户端读请求入口

	PrePrepareMsgEntrance chan interface{}              // 接受消息入口，服务器通过此通道交付请求，先将请求加入缓冲里
	PrepareMsgEntrance    chan interface{}              // 接受消息入口，服务器通过此通道交付请求，先将请求加入缓冲里
	CommitMsgEntrance     chan interface{}              // 接受消息入口，服务器通过此通道交付请求，先将请求加入缓冲里

	ChangeMsgEntrance     chan interface{}              // 通知候补节点转为共识节点

	DispatchConsensus     chan bool                     // 通知主节点开始打包交易，并广播
	DispatchPrePrepare    chan bool                     // 通知副节点收到主节点的PrePrepare，并处理
	DispatchPrepare       chan bool                     // 到达下一阶段后，使用此通道开始从缓冲里取出上一阶段提前收到的消息，并开始等待累计收到2f个请求，广播消息，进入下一阶段
	DispatchCommit        chan bool                     // 到达下一阶段后，使用此通道开始从缓冲里取出上一阶段提前收到的消息，并开始等待累计收到2f个请求，广播消息，进入下一阶段

	Consensus             chan bool                     // 有一定数量交易后通知打包

	//计算tps和延迟需要
	Timeout               chan bool                     // 超时处理
	TimeInfo              *TimeInfo                     // 计算tps
}

//共识消息结构体
type ConsensusMsg struct {
	PrePrepareMsg *structs.PrePrepareMsg             //预准备消息，
	PrepareMsgs   map[string]*structs.PrepareMsg     //准备消息。map类型的，key值是节点ID，value是该节点发送的准备消息
	CommitMsgs    map[string]*structs.CommitMsg      //提交消息，map类型的，key值是节点ID，value是该节点发送的提交消息
}

type Stage int   //将int定义成Stage类型

const (
	Idle        Stage = iota  //iota刚开始是0
	WaitPrepare       // 1
	WaitCommit        // 2
)

type Config struct {
	ViewID             int                    `yaml:"ViewID"`
	F                  int                    `yaml:"F"`
	MsgNumber          int                    `yaml:"MsgNumber"`
	SequenceID         int64                  `yaml:"SequenceID"`
	CreditTable        CreditTable            `yaml:"CreditTable"`
	ClientTable        map[string]*ClientInfo `yaml:"ClientTable"`
	ConsensusNodeTable map[string]*NodeInfo   `yaml:"ConsensusNodeTable"`
	CandidateNodeTable map[string]*NodeInfo   `yaml:"CandidateNodeTable"`
	AccountTable       map[string]int         `yaml:"AccountTable"`
	AdjustmentRange    AdjustmentRange        `yaml:"AdjustmentRange"`
}

type NodeInfo struct {
	URL           string         `yaml:"URL"`
	Credit        int            `yaml:"Credit"`
	PenaltyPeriod int            `yaml:"PenaltyPeriod"`
	PublicKey     rsa.PublicKey  `yaml:"PublicKey"`
	PrivateKey    rsa.PrivateKey `yaml:"PrivateKey"`
}

type CreditTable struct {
	Min  int
	Bad  int
	Init int
	Good int
	Max  int
}

type AdjustmentRange struct {  //奖惩值结构体
	MasterReward    int    `yaml:"MasterReward"`//主节点奖励值
	MasterPunish    int    `yaml:"MasterPunish"`//主节点惩罚值
	SecondaryReward int    `yaml:"SecondaryReward"`//普通共识节点奖励值
	SecondaryPunish int    `yaml:"SecondaryPunish"`//普通共识节点惩罚值
	candidateReward int    `yaml:"candidateReward"`//候选节点的奖励值
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
//新建一个节点
func NewNode(nodeID string, nodeType string) *Node {
	yamlFile, _ := ioutil.ReadFile("./resources/nodeConfig.yaml")  ////读取配置文件，并返回文件内容,yamlFile中存放着配置文件
	config := Config{}

	//执行之后config中的数值就是配置文件中的数值
	_ = yaml.Unmarshal(yamlFile, &config)//unmarshal负责解析yaml语言，就是将yamlFile中字符串的数据化为config类型的结构体

	config.ClientTable["1"].PrivateKey = tool.GetPrivateKey("./resources/client_key/client_private_key.pem")
	config.ClientTable["1"].PublicKey = tool.GetPublicKey("C:\\Users\\DM\\go\\FourNode\\consensusPBFT2\\resources\\client_key\\client_public_key.pem")

	//配置共识节点的公私钥，config.F 表示出错节点数目
	for i := 1; i <= config.F*3+1; i++ {
		id := strconv.Itoa(i)  //将节点的ID的数字转为字符串
		//config.ConsensusNodeTable[id].PrivateKey = tool.GetPrivateKey("./resources/consensus_node_key/consensus_Node" + id + "_private_key.pem")
		//config.ConsensusNodeTable[id].PublicKey = tool.GetPublicKey("./resources/consensus_node_key/consensus_Node" + id + "_public_key.pem")
		config.ConsensusNodeTable[id].PrivateKey = tool.GetPrivateKey("C:\\Users\\DM\\go\\FourNode\\consensusPBFT2\\resources\\consensus_node_key\\consensus_Node" + id + "_private_key.pem")
		config.ConsensusNodeTable[id].PublicKey = tool.GetPublicKey("C:\\Users\\DM\\go\\FourNode\\consensusPBFT2\\resources\\consensus_node_key\\consensus_Node" + id + "_public_key.pem")
	}
	//配置候选节点的公私钥
	for i := 1; i < 5; i++ {
		id := strconv.Itoa(i) //将节点的ID的数字转为字符串
		config.CandidateNodeTable[id].PrivateKey = tool.GetPrivateKey("./resources/candidate_node_key/candidate_Node" + id + "_private_key.pem")
		config.CandidateNodeTable[id].PublicKey = tool.GetPublicKey("./resources/candidate_node_key/candidate_Node" + id + "_public_key.pem")
		//config.CandidateNodeTable[id].PrivateKey = tool.GetPrivateKey("C:\\Users\\DM\\go\\FourNode\\consensusPBFT2\\resources\\candidate_node_key\\candidate_Node" + id + "_private_key.pem")
		//config.CandidateNodeTable[id].PublicKey = tool.GetPublicKey("C:\\Users\\DM\\go\\FourNode\\consensusPBFT2\\resources\\candidate_node_key\\candidate_Node" + id + "_public_key.pem")
	}

	//structs是一个包，包括blockchain.go和msg_types.go中所有方法
	//genesisBlock是创世区块
	//为啥候选节点也要产生区块？？
	genesisBlock := structs.NewGenesisBlock()
	//fmt.Println(genesisBlock)

	//产生区块链
	blockChain := structs.NewBlockChain(genesisBlock)
	//节点赋值
	node := &Node{
		NodeID:              nodeID,  //节点ID
		NodeType:            nodeType,   //节点类型
		ConsensusNodeTable:  config.ConsensusNodeTable,   //共识节点列表，config结构体中存放的配置文件中内容
		CandidateNodeTable:  config.CandidateNodeTable,   //候选节点列表
		ClientTable:         config.ClientTable,          //客户端节点列表
		AccountTable:        config.AccountTable,         //账户信息，有1-10个账户，每个账户有1000资产
		CreditTable:         config.CreditTable,          //信任值分层的信息
		AdjustmentRange:     config.AdjustmentRange,      //奖惩值：主节点奖励(MasterReward)，主节点惩罚(MasterPunish)，候选节点奖励(SecondaryReward)，候选节点惩罚(SecondaryPunish)
		ViewID:              config.ViewID,               //试图号

		//计算方式是：如果节点类型是0并且节点ID为视图号+1
		Flag:                nodeID == strconv.Itoa(config.ViewID+1) && nodeType == "0",    //是否是主节点

		//RWMutex 是单写多读锁，该锁可以加多个读锁或者一个写锁
		//读锁占用的情况下会阻止写，不会阻止读，多个 goroutine 可以同时获取读锁
		//写锁会阻止其他 goroutine（无论读和写）进来，整个锁由该 goroutine 独占
		//适用于读多写少的场景
		//Lock() 加写锁，Unlock() 解写锁
		Lock:                sync.RWMutex{},              //RWMutex是读写互斥锁

		MsgNumber:           config.MsgNumber,            //包中的消息数量：这里是2000
		F:                   config.F,                    //可容错节点数量
		SequenceID:          config.SequenceID,           //序列号，初始为0
		RequestMsgBuffer:    make(chan interface{}, 1000000),// 创建一个空接口类型的通道, 可以存放任意格式.容量一百万。通过候选节点转发请求消息
		PrePrepareMsgBuffer: list.New(),                  //预准备消息的存放，链表存放
		PrepareMsgBuffer:    list.New(),                  //准备消息存放，链表
		CommitMsgBuffer:     list.New(),                  //提交消息存放，链表
		CurrentStage:        Idle,                        //节点状态 Idle（0）

		ConsensusMsg: &ConsensusMsg{                      //共识过程中各阶段的消息存放
			PrePrepareMsg: nil,                           //预准备消息是空
			PrepareMsgs:   make(map[string]*structs.PrepareMsg),//预准备消息
			CommitMsgs:    make(map[string]*structs.CommitMsg),//提交消息
		},

		ChangeMsgs:            make(map[string]*structs.ChangeMsg, 0), //初始长度为0,转换消息
		PrepareState:          make(map[string]int,1),                //状态表  0:正常 1:缺席 2:错误
		BlockHash:             []byte{},                               //共识区块的hash
		BlockChain:            blockChain,                             //区块链
		PrevBlockHash:         genesisBlock.Hash,                      //前一个区块的hash

		RequestMsgEntrance:    make(chan interface{}, 1000000),        //请求消息的入口
		WRequestMsgEntrance:   make(chan interface{}, 1000000),        //写消息的入口
		RRequestMsgEntrance:   make(chan interface{}, 1000000),        //读请求的入口

		PrePrepareMsgEntrance: make(chan interface{}, config.F*3),     //预准备的消息的入口
		PrepareMsgEntrance:    make(chan interface{}, config.F*3),     //准备消息的入口
		CommitMsgEntrance:     make(chan interface{}, config.F*3),     //提交消息的入口
		ChangeMsgEntrance:     make(chan interface{}, config.F*3),     //转换消息的入口

		DispatchConsensus:     make(chan bool),                        //通知主节点开始打包交易，并广播,当值为true的时候，进行打包
		DispatchPrePrepare:    make(chan bool, 1),                     //通知副节点收到主节点的PrePrepare，并处理
		DispatchPrepare:       make(chan bool, 1),                     //到达下一阶段后，使用此通道开始从缓冲里取出上一阶段提前收到的消息，并开始等待累计收到2f个请求，广播消息，进入下一阶段
		DispatchCommit:        make(chan bool, 1),                     //到达下一阶段后，使用此通道开始从缓冲里取出上一阶段提前收到的消息，并开始等待累计收到2f个请求，广播消息，进入下一阶段

		Consensus:             make(chan bool, 1000),                  //有一定数量交易后通知打包
		Timeout:               make(chan bool),                        //超时处理，用于处理主节点不发消息的情况，进行主节点降低到候选节点

		TimeInfo: &TimeInfo{                                           //计算tps
			StartTime: 0,                                              //开始时间
			EndTime:   0,                                              //结束时间
			Time:      0,                                              //时间
			Count:     0,                                              //单次共识计数
			Counts:    0,                                              //总次数
		},
	}

	// 超时
	go node.timeOut()
	//开始共识
	//接受prepare准备消息                               7
	go node.receivePrepareMsg()
	//接受commit消息                                    9
	go node.receiveCommitMsg()
	//打包PrepareMsg消息                                8
	go node.dispatchPrepareMsg()
	//接受CommitMsg消息                                10
	go node.dispatchCommitMsg()
	//共识节点接收Pre-Prepare消息                        5
	go node.receivePrePrepareMsg()
	//通知副节点收到主节点的PrePrepare，并处理              6
	go node.dispatchPrePrepareMsg()
	//打包交易   并且生成pre-prepare消息发送               4
	go node.dispatchConsensus()
	//判断是否是主节点，主节点打包交易                       3
	if node.Flag {
		// 主节点打包交易并进行共识
		node.DispatchConsensus <- true
	}
	//接受节点变更消息
	go node.receiveChangeMsg()
	// 候补节点开始接受读请求
	go node.getRRequestMsg()
	// 候补节点开始接受写请求                             1
	go node.getWRequestMsg()   //候选节点启动后，该函数开始工作
	// 主节点开始接受并且处理候选节点转发的写请求             2
	go node.getRequestMsg()
	return node
}

// 主节点接受候补节点广播的写请求
func (node *Node) getRequestMsg() {
	count := 0              //计数器
	for {
		candidate := [4]int{0,0,0,0}
		msg := <-node.RequestMsgEntrance    //RequestMsgEntrance存储候选节点发过来的消息,传给msg
		if node.NodeType == "0" {           //如果是共识群节点
			requestMsg := msg.(*structs.WRequestMsg)     //requestMsg中存储从msg中取出的WRequestMsg消息
			candidateID := requestMsg.CandidateNodeID    //转发给共识节点的候选节点的ID
			switch candidateID {
			case "1":candidate[0] = candidate[0]+1
			case "2":candidate[1] = candidate[1]+1
			case "3":candidate[2] = candidate[2]+1
			case "4":candidate[3] = candidate[3]+1
			}
			if node.validateWRequestMsg(requestMsg) {    //验证写请求成功

				if node.Flag {                           //验证是否是主节点,如果是主节点
					node.RequestMsgBuffer <- *requestMsg //RequestMsgBuffer是缓冲区,存储候选节点发送的消息,将消息缓存在缓冲区
					count++                              //同时计数加1
					//println("count:", count)
					if count == node.MsgNumber {         //如果count等于打包数量,本次接收结束
						count = 0                        //重置count为0
						node.Consensus <- true           //有一定数量交易后通知打包,打包开关置true,可以开始打包
					}

				}
			}
		}
		//计算候选节点信任值
		for k:= 0 ; k<4 ; k++ {
			if candidate[k] == 1{
				node.CandidateNodeTable["k+1"].Credit += node.AdjustmentRange.candidateReward
			}
		}
		//Node.calculateCreditCandidate(candidate)
	}
}

// 候补节点接受客户端消读请求
func (node *Node) getRRequestMsg() {
	Time := int64(0)
	Counts := int64(0)
	var currentTime int64
	for {
		msg := <-node.RRequestMsgEntrance
		requestMsg := msg.(*structs.RRequestMsg)
		//requestMsg.Timestamp = time.Now().UnixNano()
		if node.NodeType == "1" {
			if node.validateRRequestMsg(requestMsg) {
				rReplyMsg := structs.RReplyMsg{
					RequestMsg: *requestMsg,
					Amount:     node.AccountTable[requestMsg.Account],
				}
				currentTime = time.Now().UnixNano()
				node.sendRReply(&rReplyMsg)
			}
		}
		Counts++
		Time += currentTime - requestMsg.Timestamp
		fmt.Println(currentTime - requestMsg.Timestamp)
		fmt.Println(float64(Counts) / (float64(Time) / 1000000000))
		//fmt.Println("计算TPS")
	}
}

// 候补节点接受客户端写请求
func (node *Node) getWRequestMsg() {
	for {
		msg := <-node.WRequestMsgEntrance  //消息传进来存入msg
		fmt.Println("消息msg传进来到候选节点")
		if node.NodeType == "1" {          //如果是候选节点，处理写请求
			requestMsg := msg.(*structs.WRequestMsg)    //
			//fmt.Println(&requestMsg)
			requestMsg.Timestamp = time.Now().UnixNano() //接受消息的时间戳
			if node.validateWRequestMsg(requestMsg) {  //验证写请求
				//fmt.Println("验证写请求正确")
				//requestMsg就是消息本身
				//在broadcastRequestToConsensusNode中输出终端中的转发至哪一个共识节点的localhost：url
				node.broadcastRequestToConsensusNode(*requestMsg)   //广播请求到共识节点，传值传指针
				//fmt.Println(*requestMsg)      //输出*requestMsg，输出为地址
			}
		}
	}
}

// 验证写请求
func (node *Node) validateWRequestMsg(requestMsg *structs.WRequestMsg) bool {
	return tool.VerifyDigest(*(*[]byte)(unsafe.Pointer(requestMsg.Message)), requestMsg.Signature, node.ClientTable[requestMsg.ClientID].PublicKey) &&
		node.AccountTable[requestMsg.Message.Drawee] >= requestMsg.Message.Amount
}

// 验证读请求
func (node *Node) validateRRequestMsg(requestMsg *structs.RRequestMsg) bool {
	return tool.VerifyDigest([]byte(requestMsg.Account), requestMsg.Signature, node.ClientTable[requestMsg.ClientID].PublicKey)
}

func (node *Node) receiveChangeMsg() {
	for {
		msg := <-node.ChangeMsgEntrance
		if node.NodeType == "1" {
			changeMsg := msg.(*structs.ChangeMsg)
			if node.validateChangeMsg(changeMsg) {
				node.ChangeMsgs[changeMsg.NodeID] = changeMsg
				if len(node.ChangeMsgs) >= node.F*2 {
					node.NodeType = "0"
					node.SequenceID = changeMsg.SequenceID
					node.ConsensusNodeTable[changeMsg.NewNodeID] = node.CandidateNodeTable[node.NodeID]
					delete(node.CandidateNodeTable, node.NodeID)
					node.NodeID = changeMsg.NewNodeID
					if changeMsg.NodeType == "0" { // 主
						node.Flag = true
						node.DispatchConsensus <- true
					} else if changeMsg.NodeType == "1" { // 从
						node.Flag = false
					}
					node.ChangeMsgs = make(map[string]*structs.ChangeMsg, 0)
				}
			}
		}
	}
}

func (node *Node) validateChangeMsg(changeMsg *structs.ChangeMsg) bool {
	return tool.VerifyDigest([]byte(changeMsg.NewNodeID), changeMsg.Signature, node.ConsensusNodeTable[changeMsg.NodeID].PublicKey)
}

// 超时处理，
func (node *Node) timeOut() {
	for {
		<-node.Timeout
		max := 0
		id := "0"
		for i, v := range node.CandidateNodeTable {   //range遍历所有候选节点
			if max < v.Credit {                       //如果max小于候选节点的信任值
				max = v.Credit                        //max就是候选节点中最大的信任值
				id = i                                //将候选节点中最大信任值的节点取出
			}
		}
		node.ConsensusNodeTable[strconv.Itoa(node.ViewID+1)] = node.CandidateNodeTable[id]  //信任值最高的候选节点第view+1个共识节点
		delete(node.CandidateNodeTable, id)                                                 //从候选节点列表中删除信任值最高的(将候选节点中最高的变成共识节点)
		node.sendChangeMsg(node.NodeID, id, "0")                                   //发送转换消息(转化消息就是候选节点转换为共识节点的消息)
	}
}


func (node *Node) clearData() {
	node.PrepareMsgBuffer.Init()
	node.CommitMsgBuffer.Init()
	node.PrePrepareMsgBuffer.Init()
	node.ConsensusMsg.CommitMsgs = make(map[string]*structs.CommitMsg)
	node.ConsensusMsg.PrepareMsgs = make(map[string]*structs.PrepareMsg)
	node.ConsensusMsg.PrePrepareMsg = nil
}

// 如果是主节点，打包消息，并且生成pre-prepare消息发送
func (node *Node) dispatchConsensus() {
	for {
		 <-node.DispatchConsensus           //node.DispatchConsensus 值true
		//time.Sleep(time.Millisecond * 50)
		<-node.Consensus                    //消息条数足够，进行打包
		fmt.Println("打包消息")
		if node.Flag {                      //主节点flag为true
			var requestMsgs []structs.WRequestMsg    //定义一个WRequestMsg类型的写消息类型
			//打包n个交易
			for i := 0; i < node.MsgNumber; i++ {    //打包MsgNumber个交易
				msg := <-node.RequestMsgBuffer
				//fmt.Println(msg)
				requestMsg := msg.(structs.WRequestMsg)
				requestMsgs = append(requestMsgs, requestMsg)    //将requestMsg合并在一起,MsgNumber多少就合并多少
			}

			block := structs.NewBlock(requestMsgs, node.PrevBlockHash)   //生成一个新快

			t := &structs.Block{       //创建一个区块t
				Timestamp:     block.Timestamp,      //时间戳
				Data:          nil,                  //数据
				PrevBlockHash: block.PrevBlockHash,  //前一个区块hash
				Hash:          block.Hash,           //区块的hash
			}

			node.BlockHash = *(*[]byte)(unsafe.Pointer(t))

			prePrepareMsg := structs.PrePrepareMsg{    //定义pre-prepare准备消息
				Block:      block,                     //生成的新区块
				Url:        node.ConsensusNodeTable[node.NodeID].URL,   //发送者的URl
				ViewID:     node.ViewID,               //视图号
				SequenceID: node.SequenceID + 1,       //序列号+1
				Digest:     tool.GenerateHash(node.BlockHash),   //生成摘要
				Signature:  tool.GetDigest(tool.GenerateHash(node.BlockHash), node.ConsensusNodeTable[strconv.Itoa(node.ViewID+1)].PrivateKey),       //生成签名
			}

			tool.LogSendMsg(&prePrepareMsg)   //发送预准备消息到终端:[SEND_Pre-Prepare] SequenceID:

			node.CurrentStage = WaitPrepare   //节点的当前状态,是int型,WaitPrepare = 1，发送pre-prepare消息后，等待prepare消息的状态
			node.ConsensusMsg.PrePrepareMsg = &prePrepareMsg  //ConsensusMsg里边包括预准备消息,准备消息,提交消息

			//重置共识节点状态表,本地存储
			for k := range node.ConsensusNodeTable {
				node.PrepareState[k] = 0        //PrepareState状态表  0:正常 1:缺席 2:错误
			}
			node.PrepareState[node.NodeID] = 0  //主节点状态正常

			node.TimeInfo.StartTime = time.Now().UnixNano()   //计算tps的开始时间。开始时间在打包结束，结束时间在commit消息收集够2F个了
			node.TimeInfo.Count = int64(len(block.Data))      //消息数量

			//广播pre-prepare消息
			node.broadcast(prePrepareMsg, "/prePrepare")

			//到达下一阶段后，使用此通道开始从缓冲里取出上一阶段提前收到的消息，并开始等待累计收到2f个请求，广播消息，进入下一阶段
			node.DispatchPrepare <- true    //打包prepare消息的标记开关
			fmt.Println(node.NodeID + "-已发送Pre-Prepare消息")   //输出到终端已发送pre-prepare消息
		}
	}
}

//共识节点接受主节点发送的pre-prepare消息
func (node *Node) receivePrePrepareMsg() {
	for {
		select {       //select 是 Go 中的一个控制结构，类似于用于通信的 switch 语句。每个 case 必须是一个通信操作
		case msg := <-node.PrePrepareMsgEntrance:  //msg中存放主节点传过来的消息
			prePrepareMsg := msg.(*structs.PrePrepareMsg)    //定义一个PrePrepare类型的消息
			fmt.Println(node.NodeID + "—收到prePrepareMsg-" + strconv.Itoa(int(prePrepareMsg.SequenceID)))
			if node.CurrentStage == Idle && node.NodeType == "0" {   //是共识节点并且状态为节点状态为0
				if node.validateBlock(prePrepareMsg) {               //验证区块，成功并输出接受Pre-Prepare消息
					//tool.LogReceiveMsg(prePrepareMsg)
					fmt.Println(node.NodeID + "-节点已经接受prePrepareMsg消息，消息序列号为-" + strconv.Itoa(int(prePrepareMsg.SequenceID)))
					node.ConsensusMsg.PrePrepareMsg = prePrepareMsg  //将prePrepare消息存储到该节点的数据结构中
					node.sendPrepare()                               //验证无误后将发送Prepare消息
					for k := range node.ConsensusNodeTable {
						node.PrepareState[k] = 0                     //循环更新节点状态
					}
					node.PrepareState[node.NodeID] = 0               //该共识节点状态也为0
					node.CurrentStage = WaitPrepare                  //当前状态为等待prepare状态
					node.DispatchPrepare <- true                     //打包共识节点的标识置为true
				}
			} else if node.NodeType == "0" {                        //节点为共识节点
				// 发送到缓冲区
				if node.ViewID == prePrepareMsg.ViewID &&
					tool.VerifyDigest(prePrepareMsg.Digest, prePrepareMsg.Signature, node.ConsensusNodeTable[strconv.Itoa(node.ViewID+1)].PublicKey) {
					fmt.Println(node.NodeID + "-将prePrepareMsg消息放入缓冲，消息序列号为-" + strconv.Itoa(int(prePrepareMsg.SequenceID)))
					node.PrePrepareMsgBuffer.PushBack(prePrepareMsg) //消息将放入pre-prepare缓冲区
					node.DispatchPrePrepare <- true                  //打包预准备消息标识置为true
				}
			}
		case <-time.After(2000000 * time.Second):                    //主节点错误时，不发送消息
			if !node.Flag && node.NodeType == "0" {                  //
				node.Timeout <- true                                 //Timeout开关置为true
			}
		}

	}
}

//共识节点接受prepare消息
func (node *Node) receivePrepareMsg() {
	for {
		msg := <-node.PrepareMsgEntrance   //标识为true
		prepareMsg := msg.(*structs.PrepareMsg)   //类型的prepareMsg
		//输出
		fmt.Println(node.NodeID + "—收到的prepareMsg是来自于-" + prepareMsg.NodeID + "   消息的id(序列号为):" + strconv.Itoa(int(prepareMsg.SequenceID)))
		//节点当前状态是等待Prepare状态并且节点状态是共识节点
		if node.CurrentStage == WaitPrepare && node.NodeType == "0" {
			if node.validatePrepareMsg(prepareMsg) {      //验证prepare消息正确
				fmt.Println(node.NodeID + "-已经验证并接受prepareMsg消息，PrepareMsg的发送节点ID为:-" + prepareMsg.NodeID)    //输出到终端
				node.ConsensusMsg.PrepareMsgs[prepareMsg.NodeID] = prepareMsg              //将prepareMsg存储到相应的位置
				node.PrepareState[prepareMsg.NodeID] = 0                                   //节点状态为正常
			} else {                                                                       //如果验证不通过
				node.PrepareState[prepareMsg.NodeID] = 2                                   //该节点中存储记录的发送过来prepare消息位置为错误
			}
		} else if node.CurrentStage == Idle && node.NodeType == "0" {                      //如果节点状态是未收到pre-prepare消息的状态
			// 发送到缓冲区
			if node.validatePrepareMsg(prepareMsg) {                                       //
				fmt.Println(node.NodeID + "-将收到的prepareMsg消息放入prepareMsg缓冲中，prepareMsg来自于-" + prepareMsg.NodeID)       //
				node.PrepareMsgBuffer.PushBack(prepareMsg)                                 //将prepareMsg放入缓存区
			}
		}
	}
}

//接收commit消息
func (node *Node) receiveCommitMsg() {
	for {
		msg := <-node.CommitMsgEntrance   //msg消息存储commitMsg消息
		commitMsg := msg.(*structs.CommitMsg)   //commitMsg消息定义一个
		fmt.Println(node.NodeID + "-接收commitMsg，来自于-" + commitMsg.NodeID + " 序列号id:" + strconv.Itoa(int(commitMsg.SequenceID)))
		if node.CurrentStage == WaitCommit && node.NodeType == "0" {   //如果节点状态为等待提交的状态并且节点类型为共识节点
			if node.validateCommitMsg(commitMsg) {      //验证commitMsg消息通过
				fmt.Println(node.NodeID + "-节点接受commitMsg消息，消息来自于-" + commitMsg.NodeID)
				node.ConsensusMsg.CommitMsgs[commitMsg.NodeID] = commitMsg    //commit消息存储到该节点的commit的Map中
			}
		} else if node.NodeType == "0" && (node.CurrentStage == WaitPrepare || node.CurrentStage == Idle) {    //节点为共识节点且节点状态还未达到等待提交的状态，或者还未收到pre-prepare消息
			if node.validateCommitMsg(commitMsg) {     //验证通过
				// 发送到缓冲区储存起来
				fmt.Println(node.NodeID + "-将收到的commitMsg放入commitMsg缓冲中，消息来自于-" + commitMsg.NodeID)   //输出放入缓存
				node.CommitMsgBuffer.PushBack(commitMsg)   //在列表l的后面插入一个值为v的新元素e，并返回e。
				//fmt.Print("commit缓冲数：")
				//fmt.Println(node.CommitMsgBuffer.Len())
			}

		}
	}
}

// 打包Pre-PrepareMsg消息
func (node *Node) dispatchPrePrepareMsg() {       //打包与准备消息
	for {
		<-node.DispatchPrePrepare                 //打包预准备消息的标志为true
 		for {
			if node.CurrentStage == Idle {
				e := node.PrePrepareMsgBuffer.Front()  //Front返回列表l的第一个元素，如果列表为空，则返回nil。
				prePrepareMsg := e.Value.(*structs.PrePrepareMsg)    //取出prePrepare消息
				node.PrePrepareMsgBuffer.Init()                      //Init初始化或清空列表l。
				node.ConsensusMsg.PrePrepareMsg = prePrepareMsg      //存好prePrepare消息
				node.sendPrepare()                                   //发送Prepare消息
				for k := range node.ConsensusNodeTable {             //
					node.PrepareState[k] = 0                         //节点状态为正常
				}
				node.PrepareState[node.NodeID] = 0                   //该节点状态也为正常
				node.CurrentStage = WaitPrepare                      //转化为等待准备状态
				node.DispatchPrepare <- true                         //打包准备消息标志为true
				break
			}
		}
	}
}

// 共识节点发送Prepare——准备消息
func (node *Node) sendPrepare() {
	t := &structs.Block{
		Timestamp:     node.ConsensusMsg.PrePrepareMsg.Block.Timestamp,
		Data:          nil,
		PrevBlockHash: node.ConsensusMsg.PrePrepareMsg.Block.PrevBlockHash,
		Hash:          node.ConsensusMsg.PrePrepareMsg.Block.Hash,
	}
	node.BlockHash = *(*[]byte)(unsafe.Pointer(t))
	prepareMsg := structs.PrepareMsg{    //定义一个prepare消息
		ViewID:     node.ViewID,
		Url:        node.ConsensusNodeTable[node.NodeID].URL,
		SequenceID: node.SequenceID + 1,
		NodeID:     node.NodeID,
		Digest:     node.ConsensusMsg.PrePrepareMsg.Digest,
		Signature:  tool.GetDigest(node.ConsensusMsg.PrePrepareMsg.Digest, node.ConsensusNodeTable[node.NodeID].PrivateKey),
	}
	tool.LogSendMsg(&prepareMsg)        //输出到终端[SEND_Prepare] 的NodeID：
	node.broadcast(prepareMsg, "/prepare")      //广播prepare消息到其他节点
}

// 验证Block
func (node *Node) validateBlock(prePrepareMsg *structs.PrePrepareMsg) bool {
	return node.ViewID == prePrepareMsg.ViewID &&
		node.SequenceID+1 == prePrepareMsg.SequenceID &&
		tool.VerifyDigest(prePrepareMsg.Digest, prePrepareMsg.Signature, node.ConsensusNodeTable[strconv.Itoa(node.ViewID+1)].PublicKey)
}

// 打包PrepareMsg消息
func (node *Node) dispatchPrepareMsg() {
	for {
		<-node.DispatchPrepare           //打包prepare消息标识为true，在接受pre-prepare的时候就将打包prepare和pre-prepare这两个置为true
		fmt.Println("处理prepare缓存中......")
		prepareMsg := &structs.PrepareMsg{}  //定义一个pre-prepare类型的消息
		e := node.PrepareMsgBuffer.Front()   //Front返回列表l的第一个元素，如果列表为空，则返回nil。
		for e != nil {   //如果e不是空的，代表着pre-prepare消息是已经收到了的
			prepareMsg = e.Value.(*structs.PrepareMsg)    //取出prepare消息存起来
			if node.validatePrepareMsg(prepareMsg) {      //验证prepare消息，如果通过验证
				//tool.LogReceiveMsg(prepareMsg)
				node.ConsensusMsg.PrepareMsgs[prepareMsg.NodeID] = prepareMsg  //将验证过的prepare消息存储起来
				node.PrepareState[prepareMsg.NodeID] = 0                       //prepare状态为正常0
			} else {                                                           //
				node.PrepareState[prepareMsg.NodeID] = 1                       //否则的话节点处存储的prepare状态表的状态为缺席
			}
			e = e.Next()                                                       //取下一个列表元素，是空就会跳出循环
		}

		node.PrepareMsgBuffer.Init()                                           //初始化list表，清空该表
		fmt.Println("等待接收足够的2f-1个prepare消息")                          //输出正在等待prepare消息
		for {                                                                  //循环收集prepare消息
			if len(node.ConsensusMsg.PrepareMsgs) >= node.F*2-1 {              //存储prepare消息长度的map大于等于2F+1的时候
				node.CurrentStage = WaitCommit                                 //节点当前状态为等待确认状态
				node.sendCommit()                                              //发送commit消息
				node.DispatchCommit <- true                                    //commit消息的标识置为true
				break      //跳出循环
			}
		}
	}
}

// 在收集到2F+1个prepare消息后发送Commit消息
func (node *Node) sendCommit() {
	commitMsg := structs.CommitMsg{          //定义一个commitMsg
		ViewID:       node.ViewID,           //视图号
		Url:          node.ConsensusNodeTable[node.NodeID].URL, //URL
		SequenceID:   node.SequenceID + 1,                      //序列号
		NodeID:       node.NodeID,                              //发送commitMsg消息的节点ID
		Digest:       node.ConsensusMsg.PrePrepareMsg.Digest,   //摘要
		Signature:    tool.GetDigest(node.ConsensusMsg.PrePrepareMsg.Digest, node.ConsensusNodeTable[node.NodeID].PrivateKey),
		PrepareState: node.PrepareState,     //节点状态是否正常
	}
	tool.LogSendMsg(&commitMsg)              //发送commit消息到窗口中[SEND_Commit]  NodeID:
	node.broadcast(commitMsg, "/commit") //广播commitMsg到共识节点群组
}

// 验证PrepareMsg
func (node *Node) validatePrepareMsg(prepareMsg *structs.PrepareMsg) bool {
	return node.ViewID == prepareMsg.ViewID &&
		node.SequenceID+1 == prepareMsg.SequenceID &&
		tool.VerifyDigest(prepareMsg.Digest, prepareMsg.Signature, node.ConsensusNodeTable[prepareMsg.NodeID].PublicKey)
}

// 打包CommitMsg消息
func (node *Node) dispatchCommitMsg() {
	for {
		<-node.DispatchCommit       //打包结束prepare消息之后，就可以打包commit消息
		fmt.Println("等待2f个commit")   //输出到终端在等待commitMsg消息
		commitMsg := &structs.CommitMsg{} //定义一个commitMsg消息
		e := node.CommitMsgBuffer.Front() //取出list中的第一个元素
		//for循环取出list中的commit消息并存储到不同的
		for e != nil {                    //如果list中不为空
			commitMsg = e.Value.(*structs.CommitMsg)    //则将队列中的提交消息传给commitMsg
			if node.validateCommitMsg(commitMsg) {      //验证commit消息
				//tool.LogReceiveMsg(commitMsg)
				node.ConsensusMsg.CommitMsgs[commitMsg.NodeID] = commitMsg   //验证通过commit存起来
			}
			e = e.Next()         //取出commitList队列中的下一个
		}
		node.CommitMsgBuffer.Init() //重置commitList列表
		//
		for {
			//fmt.Println(len(node.ConsensusMsg.CommitMsgs))
			if len(node.ConsensusMsg.CommitMsgs) >= node.F*2 {    //如果收到了超过2F个commit消息
				if node.Flag {                                    //如果是主节点计算延迟和平均TPS
					node.TimeInfo.EndTime = time.Now().UnixNano() //取共识结束时间，开始时间在打包结束的时候
					if (node.TimeInfo.EndTime - node.TimeInfo.StartTime) < 5000000000 {
						node.TimeInfo.Time += node.TimeInfo.EndTime - node.TimeInfo.StartTime //node.TimeInfo.Time是一轮共识时间
						node.TimeInfo.Counts += node.TimeInfo.Count                           //node.TimeInfo.Count 是总数量

						println("共识延迟：", float64(node.TimeInfo.Time) /1000000000)
						//println("总数目：", float64(node.TimeInfo.Counts))
						//println("单轮数目：", float64(node.TimeInfo.Count))
						//fmt.Println("时间：",float64(node.TimeInfo.EndTime - node.TimeInfo.StartTime))
						fmt.Println("平均tps：",float64(node.TimeInfo.Counts) / (float64(node.TimeInfo.Time) / 1000000000))
						//fmt.Println("单轮tps",float64(node.TimeInfo.Count) / (float64(node.TimeInfo.EndTime - node.TimeInfo.StartTime) / 1000000000))
						//fmt.Println("单轮tps",float64(2000)/ (float64(node.TimeInfo.EndTime - node.TimeInfo.StartTime) / 1000000000))
					}
				}
				//在区块链中添加新区块
				node.SequenceID += 1         //序列号+1
				node.BlockHash = []byte{}    //
				block := node.ConsensusMsg.PrePrepareMsg.Block
				node.PrevBlockHash = block.Hash
				node.BlockChain.AddBlock(block)  //添加区块

				node.calculateCredit()           //计算信任值

				fmt.Println("共识节点1信任值：",node.CandidateNodeTable["1"].Credit)
				fmt.Println("共识节点2信任值：",node.CandidateNodeTable["2"].Credit)
				fmt.Println("共识节点3信任值：",node.CandidateNodeTable["3"].Credit)
				fmt.Println("共识节点4信任值：",node.CandidateNodeTable["4"].Credit)

				fmt.Println("结束一轮共识" + strconv.Itoa(int(node.ConsensusMsg.PrePrepareMsg.SequenceID)))
				//初始化prepare，pre-prepare，commit三个队列和结构体
				node.ConsensusMsg.CommitMsgs = make(map[string]*structs.CommitMsg)   //初始化
				node.ConsensusMsg.PrepareMsgs = make(map[string]*structs.PrepareMsg)
				node.ConsensusMsg.PrePrepareMsg = nil

				//node.sendWReply(block)
				node.PrepareState = make(map[string]int, 0)
				node.clearBuffer()   //清空缓存区
				node.CurrentStage = Idle   //重置节点状态
				if node.Flag {
					node.DispatchConsensus <- true  //对于共识节点，打包共识消息
				}
				break
			}
		}
	}
}

// 验证CommitMsg
func (node *Node) validateCommitMsg(commitMsg *structs.CommitMsg) bool {
	return node.ViewID == commitMsg.ViewID &&
		node.SequenceID+1 == commitMsg.SequenceID &&
		tool.VerifyDigest(commitMsg.Digest, commitMsg.Signature, node.ConsensusNodeTable[commitMsg.NodeID].PublicKey)
}


func (node *Node) sendWReply(block *structs.Block) {
	for _, requestMsg := range block.Data {
		replyMsg := structs.WReplyMsg{
			RequestMsg: requestMsg,
			Result:     "success",
		}
		jsonMsg, _ := json.Marshal(replyMsg)
		send(node.ClientTable[requestMsg.ClientID].URL+"/wReply", jsonMsg)
	}
}


func (node *Node) sendRReply(rReplyMsg *structs.RReplyMsg) {
	jsonMsg, _ := json.Marshal(rReplyMsg)
	send(node.ClientTable[rReplyMsg.RequestMsg.ClientID].URL+"/rReply", jsonMsg)
}

// 广播消息:第一个参数是msg,第二个参数路径 广播msg到路径中的地址
func (node *Node) broadcast(msg interface{}, path string) {
	for nodeID, nodeInfo := range node.ConsensusNodeTable {
		if nodeID == node.NodeID {    //判断节点身份，如果是主节点则不发送消息
			continue
		}
		//fmt.Println(nodeInfo.URL + path)
		jsonMsg, _ := json.Marshal(msg)    //转换为json类型
		send(nodeInfo.URL+path, jsonMsg)   //发送到这个路径
		time.Sleep(time.Millisecond * 15)  //休眠15ms
	}
}

// 候选节点在验证无误后，广播消息到共识节点
func (node *Node) broadcastRequestToConsensusNode(msg interface{}) {
	for _, nodeInfo := range node.ConsensusNodeTable {   //遍历共识节点表中的节点ID，
		jsonMsg, _ := json.Marshal(msg)                  //将msg转化为json数据格式
		fmt.Println("/request to "+nodeInfo.URL)           //输出共识节点的URl,说明发送至哪个共识节点
		send(nodeInfo.URL+"/request", jsonMsg)            //发送请求到共识节点的路由
		//time.Sleep(time.Millisecond * 12)
	}
}

//清空缓存区
func (node *Node) clearBuffer() {
	node.PrepareMsgBuffer.Init()  //初始化prepareMsg缓存区
	node.CommitMsgBuffer.Init()   //清空CommitMsg缓存区
}

//共识节点计算信任值方法
func (node *Node) calculateCredit() {
	count := map[string]map[int]int{}      //
	for k := range node.ConsensusNodeTable {  //循环取共识节点中
		count[k] = map[int]int{0: 0, 1: 0, 2: 0}    //给每个共识节点建立一个map表
	}

	for k, v := range node.ConsensusMsg.CommitMsgs {   //v中存放commitMsg消息
		count[k][v.PrepareState[k]]++    //计数count[k(是节点ID)][v是commitMsg消息，共识消息中的准备状态]
		fmt.Println("count的k：",k)
		fmt.Println("count的每一次循环值：",count[k][v.PrepareState[k]])
	}
	fmt.Println("总体输出count：",count)


	for k, v := range node.PrepareState {  //对上一步的count的map中的key为0的值+1
		count[k][v]++
		fmt.Println("循环PrepareState的结果：",count[k][v],"   k:",k,"v:",v)
	}

	for k := range node.ConsensusNodeTable {   //共识节点群组
		state := 0   //状态正常
		t := count[k][0]   //候选节点中的正常节点数量
		fmt.Println("节点",k," 的状态是正常的数量：",t)
		if count[k][1] > t {//如果缺席节点数量大于正常节点
			state = 1   //状态为1；异常
			t = count[k][1]   //t为异常节点数量
			fmt.Println("t的数量是异常节点数量：",t)
		}
		if count[k][2] > t {   //如果错误节点数量大于正常节点数量
			state = 2
			t = count[k][2]   //t为错误节点数量
			fmt.Println("t的数量是错误节点的数量：",t)
		}
		node.PrepareState[k] = state   // map[1:0 2:0 3:0 4:0]存储的1234是节点编号，0代表该节点状态为正常，
		fmt.Println("节点的prepareState数值：",node.PrepareState[k])
		switch state {
		case 0:     //状态正常的节点多
			if node.ConsensusNodeTable[k].PenaltyPeriod == 0 {    //如果共识节点列表惩罚期为0
				if k == strconv.Itoa(node.ViewID+1) {   //如果该节点为主节点
					node.ConsensusNodeTable[k].Credit += node.AdjustmentRange.MasterReward    //k节点的信任值为信任值+主节点奖励值
					if node.ConsensusNodeTable[k].Credit > 100 {  //如果该节点信任值已经大于100，则取100
						node.ConsensusNodeTable[k].Credit = 100
					}
					fmt.Println("节点",node.NodeID,"中存储的节点",k,"的信任值为",node.ConsensusNodeTable[k].Credit)
				} else {   //如果不是主节点
					node.ConsensusNodeTable[k].Credit += node.AdjustmentRange.SecondaryReward  //信任值+普通共识节点奖励值
					if node.ConsensusNodeTable[k].Credit > 100 {  //如果该节点信任值已经大于100，则取100
						node.ConsensusNodeTable[k].Credit = 100
					}
					fmt.Println("节点",node.NodeID,"中存储的节点",k,"的信任值为",node.ConsensusNodeTable[k].Credit)
				}

/*				if node.ConsensusNodeTable[k].Credit > 100 {  //如果该节点信任值已经大于100，则取100
					node.ConsensusNodeTable[k].Credit = 100
				}*/


			} else {   //如果处在惩罚期，惩罚期减去一轮为正常节点，则节点的信任值为初始值50
				node.ConsensusNodeTable[k].PenaltyPeriod--
				if node.ConsensusNodeTable[k].PenaltyPeriod == 0 {
					node.ConsensusNodeTable[k].Credit = node.CreditTable.Init
				}
			}
		case 1:
			if node.ConsensusNodeTable[k].PenaltyPeriod == 0 {
				if k == strconv.Itoa(node.ViewID+1) {
					node.ConsensusNodeTable[k].Credit -= node.AdjustmentRange.MasterPunish
					if node.ConsensusNodeTable[k].Credit < 0 {
						node.ConsensusNodeTable[k].Credit = 0
						max := 0
						id := "0"
						for i, v := range node.CandidateNodeTable {
							if max < v.Credit {
								max = v.Credit
								id = i
							}
						}
						node.ConsensusNodeTable[k] = node.CandidateNodeTable[id]
						delete(node.CandidateNodeTable, id)
						node.sendChangeMsg(node.NodeID, id, "0")
					} else {
						node.ConsensusNodeTable[k].PenaltyPeriod = 10
					}
				} else {
					node.ConsensusNodeTable[k].Credit -= node.AdjustmentRange.SecondaryPunish
					if node.ConsensusNodeTable[k].Credit < 0 {
						node.ConsensusNodeTable[k].Credit = 0
						max := 0
						id := "0"
						for i, v := range node.CandidateNodeTable {
							if max < v.Credit {
								max = v.Credit
								id = i
							}
						}
						node.ConsensusNodeTable[k] = node.CandidateNodeTable[id]
						delete(node.CandidateNodeTable, id)
						node.sendChangeMsg(node.NodeID, id, "1")
					} else {
						node.ConsensusNodeTable[k].PenaltyPeriod = 5
					}
				}

			} else {
				if k == strconv.Itoa(node.ViewID+1) {
					max := 0
					id := "0"
					for i, v := range node.CandidateNodeTable {
						if max < v.Credit {
							max = v.Credit
							id = i
						}
					}
					node.ConsensusNodeTable[k] = node.CandidateNodeTable[id]
					delete(node.CandidateNodeTable, id)
					node.sendChangeMsg(node.NodeID, id, "0")
				} else {
					max := 0
					id := "0"
					for i, v := range node.CandidateNodeTable {
						if max < v.Credit {
							max = v.Credit
							id = i
						}
					}
					node.ConsensusNodeTable[k] = node.CandidateNodeTable[id]
					delete(node.CandidateNodeTable, id)
					node.sendChangeMsg(node.NodeID, id, "1")
				}
			}
		case 2:
			if k == strconv.Itoa(node.ViewID+1) {
				max := 0
				id := "0"
				for i, v := range node.CandidateNodeTable {
					if max < v.Credit {
						max = v.Credit
						id = i
					}
				}
				node.ConsensusNodeTable[k] = node.CandidateNodeTable[id]
				delete(node.CandidateNodeTable, id)
				node.sendChangeMsg(node.NodeID, id, "0")
			} else {
				max := 0
				id := "0"
				for i, v := range node.CandidateNodeTable {
					if max < v.Credit {
						max = v.Credit
						id = i
					}
				}
				node.ConsensusNodeTable[k] = node.CandidateNodeTable[id]
				delete(node.CandidateNodeTable, id)
				node.sendChangeMsg(k, id, "1")
			}
		}
	}
}

//计算候选节点的信任值
func (node *Node) calculateCreditCandidate() {
	count := map[string]map[int]int{}      //
	for k := range node.ConsensusNodeTable {  //循环取共识节点中
		count[k] = map[int]int{0: 0, 1: 0, 2: 0}    //给每个共识节点建立一个map表
	}

	for k, v := range node.ConsensusMsg.CommitMsgs {   //v中存放commitMsg消息
		count[k][v.PrepareState[k]]++    //计数count[k(是节点ID)][v是commitMsg消息，共识消息中的准备状态]
		fmt.Println("count的k：",k)
		fmt.Println("count的每一次循环值：",count[k][v.PrepareState[k]])
	}
	fmt.Println("总体输出count：",count)


	for k, v := range node.PrepareState {  //对上一步的count的map中的key为0的值+1
		count[k][v]++
		fmt.Println("循环PrepareState的结果：",count[k][v],"   k:",k,"v:",v)
	}

	for k := range node.ConsensusNodeTable {   //共识节点群组
		state := 0   //状态正常
		t := count[k][0]   //候选节点中的正常节点数量
		fmt.Println("节点",k," 的状态是正常的数量：",t)
		if count[k][1] > t {//如果缺席节点数量大于正常节点
			state = 1   //状态为1；异常
			t = count[k][1]   //t为异常节点数量
			fmt.Println("t的数量是异常节点数量：",t)
		}
		if count[k][2] > t {   //如果错误节点数量大于正常节点数量
			state = 2
			t = count[k][2]   //t为错误节点数量
			fmt.Println("t的数量是错误节点的数量：",t)
		}
		node.PrepareState[k] = state   // map[1:0 2:0 3:0 4:0]存储的1234是节点编号，0代表该节点状态为正常，
		fmt.Println("节点的prepareState数值：",node.PrepareState[k])
		switch state {
		case 0:     //状态正常的节点多
			if node.ConsensusNodeTable[k].PenaltyPeriod == 0 {    //如果共识节点列表惩罚期为0
				if k == strconv.Itoa(node.ViewID+1) {   //如果该节点为主节点
					node.ConsensusNodeTable[k].Credit += node.AdjustmentRange.MasterReward    //k节点的信任值为信任值+主节点奖励值
					if node.ConsensusNodeTable[k].Credit > 100 {  //如果该节点信任值已经大于100，则取100
						node.ConsensusNodeTable[k].Credit = 100
					}
					fmt.Println("节点",node.NodeID,"中存储的节点",k,"的信任值为",node.ConsensusNodeTable[k].Credit)
				} else {   //如果不是主节点
					node.ConsensusNodeTable[k].Credit += node.AdjustmentRange.SecondaryReward  //信任值+普通共识节点奖励值
					if node.ConsensusNodeTable[k].Credit > 100 {  //如果该节点信任值已经大于100，则取100
						node.ConsensusNodeTable[k].Credit = 100
					}
					fmt.Println("节点",node.NodeID,"中存储的节点",k,"的信任值为",node.ConsensusNodeTable[k].Credit)
				}

				/*				if node.ConsensusNodeTable[k].Credit > 100 {  //如果该节点信任值已经大于100，则取100
								node.ConsensusNodeTable[k].Credit = 100
							}*/


			} else {   //如果处在惩罚期，惩罚期减去一轮为正常节点，则节点的信任值为初始值50
				node.ConsensusNodeTable[k].PenaltyPeriod--
				if node.ConsensusNodeTable[k].PenaltyPeriod == 0 {
					node.ConsensusNodeTable[k].Credit = node.CreditTable.Init
				}
			}
		case 1:
			if node.ConsensusNodeTable[k].PenaltyPeriod == 0 {
				if k == strconv.Itoa(node.ViewID+1) {
					node.ConsensusNodeTable[k].Credit -= node.AdjustmentRange.MasterPunish
					if node.ConsensusNodeTable[k].Credit < 0 {
						node.ConsensusNodeTable[k].Credit = 0
						max := 0
						id := "0"
						for i, v := range node.CandidateNodeTable {
							if max < v.Credit {
								max = v.Credit
								id = i
							}
						}
						node.ConsensusNodeTable[k] = node.CandidateNodeTable[id]
						delete(node.CandidateNodeTable, id)
						node.sendChangeMsg(node.NodeID, id, "0")
					} else {
						node.ConsensusNodeTable[k].PenaltyPeriod = 10
					}
				} else {
					node.ConsensusNodeTable[k].Credit -= node.AdjustmentRange.SecondaryPunish
					if node.ConsensusNodeTable[k].Credit < 0 {
						node.ConsensusNodeTable[k].Credit = 0
						max := 0
						id := "0"
						for i, v := range node.CandidateNodeTable {
							if max < v.Credit {
								max = v.Credit
								id = i
							}
						}
						node.ConsensusNodeTable[k] = node.CandidateNodeTable[id]
						delete(node.CandidateNodeTable, id)
						node.sendChangeMsg(node.NodeID, id, "1")
					} else {
						node.ConsensusNodeTable[k].PenaltyPeriod = 5
					}
				}

			} else {
				if k == strconv.Itoa(node.ViewID+1) {
					max := 0
					id := "0"
					for i, v := range node.CandidateNodeTable {
						if max < v.Credit {
							max = v.Credit
							id = i
						}
					}
					node.ConsensusNodeTable[k] = node.CandidateNodeTable[id]
					delete(node.CandidateNodeTable, id)
					node.sendChangeMsg(node.NodeID, id, "0")
				} else {
					max := 0
					id := "0"
					for i, v := range node.CandidateNodeTable {
						if max < v.Credit {
							max = v.Credit
							id = i
						}
					}
					node.ConsensusNodeTable[k] = node.CandidateNodeTable[id]
					delete(node.CandidateNodeTable, id)
					node.sendChangeMsg(node.NodeID, id, "1")
				}
			}
		case 2:
			if k == strconv.Itoa(node.ViewID+1) {
				max := 0
				id := "0"
				for i, v := range node.CandidateNodeTable {
					if max < v.Credit {
						max = v.Credit
						id = i
					}
				}
				node.ConsensusNodeTable[k] = node.CandidateNodeTable[id]
				delete(node.CandidateNodeTable, id)
				node.sendChangeMsg(node.NodeID, id, "0")
			} else {
				max := 0
				id := "0"
				for i, v := range node.CandidateNodeTable {
					if max < v.Credit {
						max = v.Credit
						id = i
					}
				}
				node.ConsensusNodeTable[k] = node.CandidateNodeTable[id]
				delete(node.CandidateNodeTable, id)
				node.sendChangeMsg(k, id, "1")
			}
		}
	}
}


func (node *Node) sendChangeMsg(newNodeID string, oldNodeId string, nodeType string) {
	changeMsg := structs.ChangeMsg{
		SequenceID: node.SequenceID,
		NodeID:     node.NodeID,
		NewNodeID:  newNodeID,
		OldNodeID:  oldNodeId,
		NodeType:   nodeType,
		Signature:  tool.GetDigest([]byte(newNodeID), node.ConsensusNodeTable[node.NodeID].PrivateKey),
	}
	jsonMsg, _ := json.Marshal(changeMsg)
	send(node.ConsensusNodeTable[newNodeID].URL+"/change", jsonMsg)
	send(node.ClientTable["1"].URL+"/change", jsonMsg)
}
