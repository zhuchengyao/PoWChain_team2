package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"time"
)

type Block struct {
	Timestamp     int64
	Transactions  []*Transaction
	PrevBlockHash []byte
	Hash          []byte
	Nonce         int
}

func NewBlock(transactions []*Transaction, prevBlockHash []byte) *Block {
	block := &Block{
		Timestamp:     time.Now().Unix(),
		Transactions:  transactions,
		PrevBlockHash: prevBlockHash,
	}
	pow := NewProofOfWork(block)
	nonce, hash := pow.Run()
	block.Hash = hash
	block.Nonce = nonce
	return block
}

func NewGenesisBlock() *Block {
	genesisTransactions := []*Transaction{NewSimpleTransaction("Genesis Block")}
	return NewBlock(genesisTransactions, []byte{})
}

// Serialize 将区块序列化为字节数组
func (b *Block) Serialize() []byte {
	var result bytes.Buffer
	encoder := gob.NewEncoder(&result)
	err := encoder.Encode(b)
	if err != nil {
		panic(err)
	}
	return result.Bytes()
}

// DeserializeBlock 从字节数组反序列化为Block
func DeserializeBlock(d []byte) *Block {
	var block Block
	decoder := gob.NewDecoder(bytes.NewReader(d))
	err := decoder.Decode(&block)
	if err != nil {
		panic(err)
	}
	return &block
}

// 在 block.go 顶部确保导入
// import "crypto/sha256"

func (b *Block) HashTransactions() []byte {
	var txHashes []byte

	for _, tx := range b.Transactions {
		txHashes = append(txHashes, tx.ID...)
	}
	// 对拼接后的txHashes再进行一次sha256，得到最终交易摘要
	hash := sha256.Sum256(txHashes)
	return hash[:]
}

// 在 block.go 顶部确保导入
// import "crypto/sha256"
