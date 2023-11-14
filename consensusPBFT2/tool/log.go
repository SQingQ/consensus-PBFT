package tool

import (
	"../structs"
	"fmt"
	"strconv"
)
//
func LogSendMsg(msg interface{}) {
	switch msg.(type) {
	case *structs.WRequestMsg:    //客户端写请求的消息
		reqMsg := msg.(*structs.WRequestMsg)
		fmt.Printf("[SEND_WriteREQUEST] ClientID: %s, Timestamp: %d, Message: %s\n", reqMsg.ClientID, reqMsg.Timestamp, PrintMessage(reqMsg.Message))

	case *structs.RRequestMsg:    //客户端读请求消息
		reqMsg := msg.(*structs.RRequestMsg)
		fmt.Printf("[SEND_ReadREQUEST] ClientID: %s, Timestamp: %d, Message: %s\n", reqMsg.ClientID, reqMsg.Timestamp, reqMsg.Account)

	case *structs.PrePrepareMsg:  //pre-prepare消息
		prePrepareMsg := msg.(*structs.PrePrepareMsg)
		fmt.Printf("[SEND_Pre-Prepare] SequenceID(序列号): %d\n", prePrepareMsg.SequenceID)
		//for _, i := range prePrepareMsg.Block.Data {
		//	fmt.Printf("--[PREPREPARE_Message] ClientID: %s, Timestamp: %d, Message: %s\n", i.ClientID, i.Timestamp, PrintMessage(i.Message))
		//}

	case *structs.PrepareMsg:   //准备消息
		prepareMsg := msg.(*structs.PrepareMsg)
		fmt.Printf("[SEND_Prepare] 的NodeID: %s\n", prepareMsg.NodeID)

	case *structs.CommitMsg:   //提交消息
		commitMsg := msg.(*structs.CommitMsg)
		fmt.Printf("[SEND_Commit]  NodeID: %s\n", commitMsg.NodeID)

	case *structs.WReplyMsg:   //写回复消息
		replyMsg := msg.(*structs.WReplyMsg)
		fmt.Printf("[SEND_Reply] ClientID: %s, Message: %s, Result: %s\n", replyMsg.RequestMsg.ClientID, PrintMessage(replyMsg.RequestMsg.Message), replyMsg.Result)

	}
}
//输出请求消息的格式
func PrintMessage(message *structs.Message) string {
	return message.Drawee + " -> " + message.Payee + " - " + strconv.Itoa(message.Amount)
}

func LogBroadCastMsg(msg interface{}) {
	switch msg.(type) {
	case *structs.WRequestMsg:
		reqMsg := msg.(*structs.WRequestMsg)
		fmt.Printf("[RECEIVE_BROADCAST_REQUEST] ClientID: %s, Timestamp: %d, Message: %s\n", reqMsg.ClientID, reqMsg.Timestamp, PrintMessage(reqMsg.Message))
	}
}

func LogReceiveMsg(msg interface{}) {
	switch msg.(type) {
	case *structs.WRequestMsg:
		reqMsg := msg.(*structs.WRequestMsg)
		fmt.Printf("[RECEIVE_REQUEST] ClientID: %s, Timestamp: %d, Message: %s\n", reqMsg.ClientID, reqMsg.Timestamp, PrintMessage(reqMsg.Message))
	case *structs.PrePrepareMsg:
		prePrepareMsg := msg.(*structs.PrePrepareMsg)
		fmt.Printf("[RECEIVE_PREPREPARE] SequenceID: %d\n", prePrepareMsg.SequenceID)
		//for _, i := range prePrepareMsg.Block.Data {
		//	fmt.Printf("--[PREPREPARE_Message] ClientID: %s, Timestamp: %d, Message: %s\n", i.ClientID, i.Timestamp, PrintMessage(i.Message))
		//}
	case *structs.PrepareMsg:
		prepareMsg := msg.(*structs.PrepareMsg)
		fmt.Printf("[RECEIVE_PREPARE] NodeID: %s\n", prepareMsg.NodeID)
	case *structs.CommitMsg:
		commitMsg := msg.(*structs.CommitMsg)
		fmt.Printf("[RECEIVE_COMMIT] NodeID: %s\n", commitMsg.NodeID)
	}
}
