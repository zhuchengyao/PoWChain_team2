package main

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"math/big"
)

// 处理新交易
func (node *Node) handleTransaction(request map[string]interface{}) {
	var tx Transaction
	if err := mapToStruct(request["transaction"], &tx); err != nil {
		fmt.Printf("交易解析失败: %v\n", err)
		return
	}
	node.HandleNewTransaction(tx)
}

// 添加新交易到交易池
func (node *Node) HandleNewTransaction(tx Transaction) {
	filePath := fmt.Sprintf("%s_transaction_pool.json", node.Address)
	if node.Blockchain.AddTransactionToPool(tx, node.PublicKeys, filePath) {
		fmt.Printf("交易已添加到交易池: %+v\n", tx)
	} else {
		fmt.Printf("交易验证失败: %+v\n", tx)
	}
}

type Transaction struct {
	Sender    string
	Receiver  string
	Amount    float64
	Signature string
}

// 创建新交易
func NewTransaction(sender, receiver string, amount float64, privateKey *ecdsa.PrivateKey) Transaction {
	tx := Transaction{
		Sender:   sender,
		Receiver: receiver,
		Amount:   amount,
	}
	if privateKey != nil {
		SignTransaction(&tx, privateKey)
	}
	return tx
}

// 签名交易
func SignTransaction(tx *Transaction, privateKey *ecdsa.PrivateKey) {
	txData := fmt.Sprintf("%s%s%f", tx.Sender, tx.Receiver, tx.Amount)
	hash := sha256.Sum256([]byte(txData))
	r, s, _ := ecdsa.Sign(rand.Reader, privateKey, hash[:])
	tx.Signature = fmt.Sprintf("%s:%s", r.String(), s.String())
}

// 验证交易
func VerifyTransaction(tx *Transaction, publicKey *ecdsa.PublicKey) bool {
	txData := fmt.Sprintf("%s%s%f", tx.Sender, tx.Receiver, tx.Amount)
	hash := sha256.Sum256([]byte(txData))
	var r, s big.Int
	n, err := fmt.Sscanf(tx.Signature, "%s:%s", &r, &s)
	if err != nil || n != 2 {
		return false
	}
	return ecdsa.Verify(publicKey, hash[:], &r, &s)
}
