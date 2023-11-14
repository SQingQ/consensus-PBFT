package structs

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"strconv"
	"time"
)

type Block struct {
	Timestamp     int64
	Data          []RequestMsg
	PrevBlockHash []byte
	Hash          []byte
}

type BlockChain struct {
	blocks []*Block
}

// 设置hash值
func (b *Block) setHash() {
	buf := &bytes.Buffer{}
	_ = binary.Write(buf, binary.BigEndian, b.Data)
	timestamp := []byte(strconv.FormatInt(b.Timestamp, 10))
	headers := bytes.Join([][]byte{b.PrevBlockHash, buf.Bytes(), timestamp}, []byte{})
	hash := sha256.Sum256(headers)
	b.Hash = hash[:]
}

// 向区块链中添加区块
func (bc *BlockChain) AddBlock(block *Block) {
	bc.blocks = append(bc.blocks, block)
}

func (bc *BlockChain) GetBlockChainLength() int {
	return len(bc.blocks)
}

// 生成一个区块
func NewBlock(data []RequestMsg, prevBlockHash []byte) *Block {
	block := &Block{time.Now().Unix(), data, prevBlockHash, []byte{}}
	block.setHash()
	return block
}

// 生成创世区块
func NewGenesisBlock() *Block {
	block := &Block{1552200225, nil, []byte{}, []byte{}}
	block.setHash()
	return block
}

// 生成区块链
func NewBlockChain(block *Block) *BlockChain {
	return &BlockChain{[]*Block{block}}
}
