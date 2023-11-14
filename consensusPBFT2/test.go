package main

import (
	"./tool"
	"strconv"
)

func main() {
	//生成密钥对
	for i := 11; i <= 13; i++ {
		tool.GenerateKey("./resources/consensus_node_key/", "consensus_node"+strconv.Itoa(i))
		//tool.GenerateKey("C:/Users/DM/go/FourNode/consensusPBFT2/resources/candidate_node_key/", "consensus_node"+strconv.Itoa(i))
	}

}
