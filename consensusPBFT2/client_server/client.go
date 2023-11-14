package client_server

import (
	"../structs"
	"../tool"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"math/rand"
	"strconv"
	"time"
	"unsafe"
)

type Client struct {
	F                  int
	ClientID           string
	ClientInfo         *ClientInfo
	CandidateNodeTable map[string]*NodeInfo
	ConsensusNodeTable map[string]*NodeInfo
	AccountTable       map[string]int
	ChangeMsgEntrance  chan interface{}
	ReplyMsgEntrance   chan interface{}
}

type Config struct {
	F                  int                    `yaml:"F"`
	ClientTable        map[string]*ClientInfo `yaml:"ClientTable"`
	ConsensusNodeTable map[string]*NodeInfo   `yaml:"ConsensusNodeTable"`
	CandidateNodeTable map[string]*NodeInfo   `yaml:"CandidateNodeTable"`
	AccountTable       map[string]int         `yaml:"AccountTable"`
}

type NodeInfo struct {
	URL        string         `yaml:"URL"`
	PublicKey  rsa.PublicKey  `yaml:"PublicKey"`
	PrivateKey rsa.PrivateKey `yaml:"PrivateKey"`
}

type ClientInfo struct {
	URL        string         `yaml:"URL"`
	PublicKey  rsa.PublicKey  `yaml:"PublicKey"`
	PrivateKey rsa.PrivateKey `yaml:"PrivateKey"`
}

type TimeInfo struct {
	Time   int64   //时间
	Counts int64   //数量
}


//new 一个新的客户端
//参数客户端ID 是string类型，返回值是*client，是个指针
func NewClient(clientID string) *Client {
	//fmt.Println("2")
	yamlFile, _ := ioutil.ReadFile("./resources/clientConfig.yaml")   //读取配置文件，并返回文件内容
	//fmt.Println("clientConfig,yaml:",yamlFile)
	config := Config{}
	//fmt.Println("config",config) //输出为空

	_ = yaml.Unmarshal(yamlFile, &config)    //unmarshal负责解析yaml语言，就是将yamlFile中字符串的数据化为config类型的结构体

	//config.ClientTable["1"].PrivateKey = tool.GetPrivateKey("./resources/client_key/client_private_key.pem")
	//config.ClientTable["1"].PublicKey = tool.GetPublicKey("./resources/client_key/client_public_key.pem")
	//取公钥私钥

	config.ClientTable["1"].PrivateKey = tool.GetPrivateKey("C:\\Users\\DM\\go\\FourNode\\consensusPBFT2\\resources\\client_key\\client_private_key.pem")
	config.ClientTable["1"].PublicKey = tool.GetPublicKey("C:\\Users\\DM\\go\\FourNode\\consensusPBFT2\\resources\\client_key\\client_public_key.pem")

	client := &Client{
		F:                  config.F,
		ClientID:           clientID,
		ClientInfo:         config.ClientTable[clientID],
		CandidateNodeTable: config.CandidateNodeTable,
		ConsensusNodeTable: config.ConsensusNodeTable,
		AccountTable:       config.AccountTable,
		ChangeMsgEntrance:  make(chan interface{}, config.F*3),   //ch := make(chan int)创建通道，创建interface类型的通道
		ReplyMsgEntrance:   make(chan interface{}, 1000000),     //make(chan int, 100) 中100是容量
	}

	go client.sendWRequestMsg()    //发送写请求
	go client.receiveChangeMsg()
	go client.receiveRReplyMsg()
	return client
}
//接受读消息
func (client *Client) receiveRReplyMsg() {
	timeInfo := TimeInfo{
		Time:   0,
		Counts: 0,
	}
	var dTime int64
	for {
		msg := <-client.ReplyMsgEntrance
		replyMsg := msg.(*structs.RReplyMsg)
		dTime = time.Now().UnixNano() - replyMsg.RequestMsg.Timestamp
		if dTime < 100000000 {
			timeInfo.Time += dTime
			timeInfo.Counts++
			fmt.Println(dTime)
			fmt.Println(float64(timeInfo.Counts) / (float64(timeInfo.Time) / 1000000000))
		}
	}
}

func (client *Client) receiveChangeMsg() {
	ChangeMsgs := map[string]*structs.ChangeMsg{}
	flag := true
	for {
		msg := <-client.ChangeMsgEntrance
		if flag {
			changeMsg := msg.(*structs.ChangeMsg)
			ChangeMsgs[changeMsg.NodeID] = changeMsg
			flag = false
			if len(ChangeMsgs) >= client.F*2 {
				delete(client.CandidateNodeTable, changeMsg.OldNodeID)
				ChangeMsgs = make(map[string]*structs.ChangeMsg, 0)
			}
		}

	}
}

// 发送写请求  实现接口
func (client *Client) sendWRequestMsg() {
	//fmt.Println("3")
	//count := 1
	for j := 0; j < 1000; j++ {
		for i := 0; i < 2000; i++ {
			//T1 := time.Now().UnixNano()  //计算产生一条消息的时间
			//fmt.Println(count)
			//time.Sleep(time.Millisecond * 50)
			r := rand.New(rand.NewSource(time.Now().UnixNano()))   //使用时间进行随机生成消息r
			//func Intn(n int) int
			//Intn 以 int 形式返回来自默认 Source 的 [0，n] 中的非负伪随机数
			//r.Intn(client.F*3+1)随即成成一个[0,4]的数字
			//strconv.Itoa(4)，将数字4转为字符4
			n := strconv.Itoa(r.Intn(client.F*3+1) + 1)
			//n为数字，n对应clientConfig中的候选节点配置表 ok表示是否有这个节点的配置，有则为true，无则为false
			//  _表示URL，公钥私钥等Info的结构体
			_,ok := client.CandidateNodeTable[n]    // n为候选节点的配置文件中的数字对应的信息
			//当ok为false时，重新选择随机数
			for !ok {
				r = rand.New(rand.NewSource(time.Now().UnixNano()))
				n = strconv.Itoa(r.Intn(client.F*3+1) + 1)
				_, ok = client.CandidateNodeTable[n]
			}
			e := generateRandomNumber(1, 10, 2)     //在1-10个中产生2个不同的随机数 e为数组
			//fmt.Println("e:",e)
			//a和b分别是字符
			a := strconv.Itoa(e[0])
			b := strconv.Itoa(e[1])
			//fmt.Println(a,b)
			//消息内容：收款人、付款人、金额
			//a和b是字符，代表首付款人
			message := &structs.Message{
				Payee:  a,    //收款人
				Drawee: b,    //付款人
				Amount: rand.Intn(100) + 1,   //金额
			}
			//创建写请求消息
			requestMsg := structs.WRequestMsg{
				Timestamp: 0,
				ClientID:  client.ClientID,
				CandidateNodeID: n,
				Message:   message,
				Signature: tool.GetDigest(*(*[]byte)(unsafe.Pointer(message)), client.ClientInfo.PrivateKey),
			}
			//fmt.Println(requestMsg.Signature)
			//fmt.Println(requestMsg.ClientID)
			//fmt.Println(requestMsg.CandidateNodeID)
			//fmt.Println(requestMsg.Timestamp)
			//fmt.Println(requestMsg.Message)
			jsonMsg, er := json.Marshal(requestMsg)   //jsonMsg是byte类型的数据，将消息转化为byte，方便传输
			if er!=nil{
				fmt.Println("生成json字符串错误")
			}
			//interface作为形参,输出消息类型，以及消息内容到命令行
			tool.LogSendMsg(&requestMsg)
			//发送给候选节点请求，n是候选节点数量中的随机一个
			send(client.CandidateNodeTable[n].URL+"/wRequest", jsonMsg)   //将发送到路由设置部分的对应的URl中
			//T2 := time.Now().UnixNano()
			//fmt.Println(float64(T2 - T1)/1000000000)
			time.Sleep(time.Millisecond * 15)    //休眠15ms再继续发送
		}
	}
	fmt.Println("发送结束")
	time.Sleep(time.Second * 20)
}

//发送读请求
func (client *Client) sendRRequestMsg() {
	for j := 0; j < 1000; j++ {
		for i := 0; i < 2000; i++ {
			//time.Sleep(time.Millisecond * 50)
			r := rand.New(rand.NewSource(time.Now().UnixNano()))
			n := strconv.Itoa(r.Intn(client.F*3+1) + 1)
			_, ok := client.CandidateNodeTable[n]
			for !ok {
				r = rand.New(rand.NewSource(time.Now().UnixNano()))
				n = strconv.Itoa(r.Intn(client.F*3+1) + 1)
				_, ok = client.CandidateNodeTable[n]
			}
			r = rand.New(rand.NewSource(time.Now().UnixNano()))
			a := strconv.Itoa(r.Intn(10) + 1)
			requestMsg := structs.RRequestMsg{
				Timestamp: time.Now().UnixNano(),
				ClientID:  client.ClientID,
				Account:   a,
				Signature: tool.GetDigest([]byte(a), client.ClientInfo.PrivateKey),
			}
			jsonMsg, _ := json.Marshal(requestMsg)
			tool.LogSendMsg(&requestMsg)
			go send(client.CandidateNodeTable["1"].URL+"/rRequest", jsonMsg)
			time.Sleep(time.Millisecond * 12)
		}
	}
	fmt.Println("发送结束")
	time.Sleep(time.Second * 20)
}

//生成count个[start,end)结束的不重复的随机数
func generateRandomNumber(start int, end int, count int) []int {
	//范围检查
	if end < start || (end-start) < count {
		return nil
	}
	//存放结果的slice,创建了一个数组类型的slice，长度为0
	nums := make([]int, 0)

	//随机数生成器，加入时间戳保证每次生成的随机数不一样
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	for len(nums) < count {
		//生成随机数
		num := r.Intn((end - start)) + start
		//查重
		exist := false
		for _, v := range nums {
			if v == num {
				exist = true
				break
			}
		}
		if !exist {
			nums = append(nums, num)
		}
	}
	return nums
}
