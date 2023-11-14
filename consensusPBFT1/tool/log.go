package tool

import (
	"../structs"
	"fmt"
	"strconv"
)

func LogSendMsg(msg interface{}) {
	switch msg.(type) {
	case *structs.RequestMsg:
		reqMsg := msg.(*structs.RequestMsg)
		fmt.Printf("[SEND_REQUEST] ClientID: %s, Timestamp: %d, Message: %s\n", reqMsg.ClientID, reqMsg.Timestamp, PrintMessage(reqMsg.Message))
	case *structs.PrePrepareMsg:
		prePrepareMsg := msg.(*structs.PrePrepareMsg)
		fmt.Printf("[SEND_PREPREPARE] SequenceID: %d\n", prePrepareMsg.SequenceID)
		//for _, i := range prePrepareMsg.Block.Data {
		//	fmt.Printf("--[PREPREPARE_Message] ClientID: %s, Timestamp: %d, Message: %s\n", i.ClientID, i.Timestamp, PrintMessage(i.Message))
		//}
	case *structs.PrepareMsg:
		prepareMsg := msg.(*structs.PrepareMsg)
		fmt.Printf("[SEND_PREPARE] NodeID: %s\n", prepareMsg.NodeID)
	case *structs.CommitMsg:
		commitMsg := msg.(*structs.CommitMsg)
		fmt.Printf("[SEND_COMMIT]  NodeID: %s\n", commitMsg.NodeID)
	case *structs.ReplyMsg:
		replyMsg := msg.(*structs.ReplyMsg)
		fmt.Printf("[SEND_REPLY] ClientID: %s, Message: %s, Result: %s\n", replyMsg.RequestMsg.ClientID, PrintMessage(replyMsg.RequestMsg.Message), replyMsg.Result)

	}
}

func PrintMessage(message *structs.Message) string {
	return message.Drawee + " -> " + message.Payee + " - " + strconv.Itoa(message.Amount)
}

func LogBroadCastMsg(msg interface{}) {
	switch msg.(type) {
	case *structs.RequestMsg:
		reqMsg := msg.(*structs.RequestMsg)
		fmt.Printf("[RECEIVE_BROADCAST_REQUEST] ClientID: %s, Timestamp: %d, Message: %s\n", reqMsg.ClientID, reqMsg.Timestamp, PrintMessage(reqMsg.Message))
	}
}

func LogReceiveMsg(msg interface{}) {
	switch msg.(type) {
	case *structs.RequestMsg:
		reqMsg := msg.(*structs.RequestMsg)
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
