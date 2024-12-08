package main

import (
	"crypto/sha256"
)

// Transaction 是一个简单的交易结构，这里简化为只有一个 Data 字段
type Transaction struct {
	ID   []byte
	Data string
}

// NewSimpleTransaction 创建一笔简单的交易
func NewSimpleTransaction(data string) *Transaction {
	tx := &Transaction{
		Data: data,
	}
	tx.SetID()
	return tx
}

// SetID 根据交易的 Data 生成一个简单的哈希作为交易ID
func (tx *Transaction) SetID() {
	hash := sha256.Sum256([]byte(tx.Data))
	tx.ID = hash[:]
}
