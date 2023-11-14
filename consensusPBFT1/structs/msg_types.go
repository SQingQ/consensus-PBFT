package structs

type RequestMsg struct {
	Timestamp int64    `json:"timestamp"` // 时间戳
	ClientID  string   `json:"clientID"`  // 客户端ID
	Message   *Message `json:"message"`   // 消息
	Signature []byte   `json:"signature"` // 消息的签名
}

type Message struct {
	Payee  string `json:"operation"` //收款人
	Drawee string `json:"drawee"`    //付款人
	Amount int    `json:"amount"`    //金额
}

type ViewChangeMsg struct {
	NewViewID  int    `json:"NewViewID"`  // 新视图ID
	SequenceID int64  `json:"sequenceID"` // 分配的ID
	NodeID     string `json:"nodeID"`     // 节点的ID
	Signature  []byte `json:"signature"`  // 签名
}

type NewViewMsg struct {
	NewViewID      int              `json:"NewViewID"`      // 新视图ID
	ViewChangeMsgs []*ViewChangeMsg `json:"ViewChangeMsgs"` // 2f个视图转换消息
	Signature      []byte           `json:"signature"`      // 签名
}

type PrePrepareMsg struct {
	Block      *Block `json:"block"`      // 区块
	ViewID     int    `json:"viewID"`     // 当前视图ID
	SequenceID int64  `json:"sequenceID"` // 分配的ID
	Digest     []byte `json:"digest"`     // block的摘要
	Signature  []byte `json:"signature"`  // block的签名
}

type PrepareMsg struct {
	ViewID     int    `json:"viewID"`     // 当前视图ID
	SequenceID int64  `json:"sequenceID"` // 分配的ID
	NodeID     string `json:"nodeID"`     // 节点的ID
	Digest     []byte `json:"digest"`     // block的摘要
	Signature  []byte `json:"signature"`  // block的签名
}

type CommitMsg struct {
	ViewID     int    `json:"viewID"`     // 当前视图ID
	SequenceID int64  `json:"sequenceID"` // 分配的ID
	NodeID     string `json:"nodeID"`     // 节点的ID
	Digest     []byte `json:"digest"`     // block的摘要
	Signature  []byte `json:"signature"`  // block的签名
}

type ReplyMsg struct {
	RequestMsg RequestMsg `json:"requestMsg"`
	Result     string     `json:"result"`
}
