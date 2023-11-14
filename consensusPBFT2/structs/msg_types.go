package structs
//写请求消息
type WRequestMsg struct {
	Timestamp int64    `json:"timestamp"` // 时间戳
	ClientID  string   `json:"clientID"`  // 客户端ID
	CandidateNodeID string     `json:"candidateNodeID"`
	Message   *Message `json:"message"`   // 消息
	Signature []byte   `json:"signature"` // 消息的签名
}
//消息
type Message struct {
	Payee  string `json:"operation"` //收款人
	Drawee string `json:"drawee"`    //付款人
	Amount int    `json:"amount"`    //金额
}
//读请求消息
type RRequestMsg struct {
	Timestamp int64  `json:"timestamp"` // 时间戳
	ClientID  string `json:"clientID"`  // 客户端ID
	Account   string `json:"Account"`   // 账号
	Signature []byte `json:"signature"` // 消息的签名
}
//预准备消息
type PrePrepareMsg struct {
	Block      *Block `json:"block"`      // 区块
	Url        string `json:"Url"`        // 发送者的URl
	ViewID     int    `json:"viewID"`     // 当前视图ID
	SequenceID int64  `json:"sequenceID"` // 分配的ID
	Digest     []byte `json:"digest"`     // block的摘要
	Signature  []byte `json:"signature"`  // block的签名
}
//请求消息
type PrepareMsg struct {
	ViewID     int    `json:"viewID"`     // 当前视图ID
	Url        string `json:"Url"`        // 发送者的URl
	SequenceID int64  `json:"sequenceID"` // 分配的序列号
	NodeID     string `json:"nodeID"`     // 节点的ID
	Digest     []byte `json:"digest"`     // block的摘要
	Signature  []byte `json:"signature"`  // block的签名
}
//提交消息
type CommitMsg struct {
	ViewID       int            `json:"viewID"`       // 当前视图ID
	Url          string         `json:"Url"`          // 发送者的URl
	SequenceID   int64          `json:"sequenceID"`   // 分配的ID
	NodeID       string         `json:"nodeID"`       // 节点的ID
	Digest       []byte         `json:"digest"`       // block的摘要
	Signature    []byte         `json:"signature"`    // block的签名
	PrepareState map[string]int `json:"prepareState"` // prepare阶段的状态
}
//
type ChangeMsg struct {
	SequenceID int64  `json:"sequenceID"` // 分配的ID
	NodeID     string `json:"nodeID"`     // 发送节点的ID
	NewNodeID  string `json:"NewNodeID"`  // 节点的新ID
	OldNodeID  string `json:"OldNodeID"`  // 节点的旧ID
	NodeType   string `json:"NewNodeID"`  // 节点的类型
	Signature  []byte `json:"signature"`  // 消息的签名
}
//写请求回复
type WReplyMsg struct {
	RequestMsg WRequestMsg `json:"requestMsg"`
	Result     string      `json:"result"`
}
//读请求回复
type RReplyMsg struct {
	RequestMsg RRequestMsg `json:"requestMsg"`
	Amount     int         `json:"result"`
}
