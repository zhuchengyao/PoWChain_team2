package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

type BlockHeader struct {
	Index        int
	Timestamp    int64
	PreviousHash string
	Nonce        uint64
	MerkleRoot   string
}

type Block struct {
	Header       BlockHeader
	Transactions []Transaction
	Hash         string
}

// 创建新区块
func NewBlock(index int, previousHash string, transactions []Transaction, miner string, reward float64, difficulty int) Block {
	rewardTx := Transaction{
		Sender:   "System",
		Receiver: miner,
		Amount:   reward,
	}
	transactions = append(transactions, rewardTx)
	block := Block{
		Header: BlockHeader{
			Index:        index,
			Timestamp:    time.Now().Unix(),
			PreviousHash: previousHash,
			MerkleRoot:   CalculateMerkleRoot(transactions),
		},
		Transactions: transactions,
	}
	block.ProofOfWork(difficulty)
	return block
}

// 工作量证明
func (b *Block) ProofOfWork(difficulty int) {
	prefix := strings.Repeat("0", difficulty)
	for {
		b.Hash = b.CalculateHash()
		if strings.HasPrefix(b.Hash, prefix) {
			break
		}
		b.Header.Nonce++
	}
}

// 计算区块哈希
func (b *Block) CalculateHash() string {
	headerData := fmt.Sprintf("%d%d%s%d%s", b.Header.Index, b.Header.Timestamp, b.Header.PreviousHash, b.Header.Nonce, b.Header.MerkleRoot)
	hash := sha256.Sum256([]byte(headerData))
	return hex.EncodeToString(hash[:])
}
