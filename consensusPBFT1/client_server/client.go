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
	F            int
	ClientID     string
	ClientInfo   *ClientInfo
	NodeTable    map[string]*NodeInfo
	AccountTable map[string]int
}

type Config struct {
	F            int                    `yaml:"F"`
	ClientTable  map[string]*ClientInfo `yaml:"ClientTable"`
	NodeTable    map[string]*NodeInfo   `yaml:"NodeTable"`
	AccountTable map[string]int         `yaml:"AccountTable"`
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

func NewClient(clientID string) *Client {
	yamlFile, _ := ioutil.ReadFile("./resources/clientConfig.yaml")
	config := Config{}
	_ = yaml.Unmarshal(yamlFile, &config)
	config.ClientTable["1"].PrivateKey = tool.GetPrivateKey("./resources/client_key/client_private_key.pem")
	config.ClientTable["1"].PublicKey = tool.GetPublicKey("./resources/client_key/client_public_key.pem")
	client := &Client{F: config.F, ClientID: clientID, ClientInfo: config.ClientTable[clientID], NodeTable: config.NodeTable, AccountTable: config.AccountTable}
	go client.sendRequestMsg()
	return client
}

func (client *Client) sendRequestMsg() {
	for j := 0; j < 1000; j++ {
		for i := 0; i < 200; i++ {
			//time.Sleep(time.Millisecond * 50)
			r := rand.New(rand.NewSource(time.Now().UnixNano()))
			n := strconv.Itoa(r.Intn(client.F*3+1) + 1)
			e := generateRandomNumber(1, 10, 2)
			a := strconv.Itoa(e[0])
			b := strconv.Itoa(e[1])
			message := &structs.Message{
				Payee:  a,
				Drawee: b,
				Amount: rand.Intn(100) + 1,
			}
			requestMsg := structs.RequestMsg{
				Timestamp: 0,
				ClientID:  client.ClientID,
				Message:   message,
				Signature: tool.GetDigest(*(*[]byte)(unsafe.Pointer(message)), client.ClientInfo.PrivateKey),
			}
			jsonMsg, _ := json.Marshal(requestMsg)
			tool.LogSendMsg(&requestMsg)
			send(client.NodeTable[n].URL+"/request", jsonMsg)
			time.Sleep(time.Millisecond * 15)
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
	//存放结果的slice
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
