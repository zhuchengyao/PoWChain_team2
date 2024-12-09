package main

import (
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"os"
)

func SaveBlockchain(filePath string, blockchain *Blockchain) {
	data, _ := json.MarshalIndent(blockchain, "", "  ")
	err := os.WriteFile(filePath, data, 0644)
	if err != nil {
		fmt.Printf("保存区块链失败: %v\n", err)
	}
}

// AddTransactionToPool 添加交易到交易池
func (bc *Blockchain) AddTransactionToPool(tx Transaction, publicKeys map[string]*ecdsa.PublicKey, filePath string) bool {
	publicKey, exists := publicKeys[tx.Sender]
	if !exists {
		fmt.Printf("交易发送方公钥不存在: %s\n", tx.Sender)
		return false
	}
	if VerifyTransaction(&tx, publicKey) {
		bc.TransactionPool = append(bc.TransactionPool, tx)
		SaveBlockchain(filePath, bc)
		fmt.Printf("交易已添加到交易池: %+v\n", tx)
		return true
	}
	fmt.Printf("交易验证失败: %+v\n", tx)
	return false
}

// ClearTransactionPool 清除已打包的交易
func (bc *Blockchain) ClearTransactionPool(transactions []Transaction) {
	remaining := []Transaction{}
	for _, poolTx := range bc.TransactionPool {
		included := false
		for _, blockTx := range transactions {
			// 如果交易池中的交易已包含在区块中，则跳过
			if poolTx == blockTx {
				included = true
				break
			}
		}
		if !included {
			remaining = append(remaining, poolTx)
		}
	}
	bc.TransactionPool = remaining
	fmt.Printf("交易池已更新，移除已打包的交易\n")
}

// PrintBlockchain 打印区块链的状态
func PrintBlockchain(bc *Blockchain) {
	fmt.Println("当前区块链状态:")
	for _, block := range bc.Blocks {
		fmt.Printf("区块 #%d\n", block.Header.Index)
		fmt.Printf("时间戳: %d\n", block.Header.Timestamp)
		fmt.Printf("前一个区块哈希: %s\n", block.Header.PreviousHash)
		fmt.Printf("区块哈希: %s\n", block.Hash)
		fmt.Printf("Merkle 根: %s\n", block.Header.MerkleRoot)
		fmt.Printf("交易列表:\n")
		for _, tx := range block.Transactions {
			fmt.Printf("  %s -> %s: %.2f\n", tx.Sender, tx.Receiver, tx.Amount)
		}
		fmt.Println("------------------------------")
	}
}

func (bc *Blockchain) GetTransactionsForBlock() []Transaction {
	return bc.TransactionPool
}

func (bc *Blockchain) AddBlock(transactions []Transaction, miner string, publicKeys map[string]*ecdsa.PublicKey, filePath string) {
	validTransactions := []Transaction{}
	for _, tx := range transactions {
		if publicKey, exists := publicKeys[tx.Sender]; exists && VerifyTransaction(&tx, publicKey) {
			validTransactions = append(validTransactions, tx)
		} else {
			fmt.Printf("交易验证失败: %+v\n", tx)
		}
	}

	lastBlock := bc.Blocks[len(bc.Blocks)-1]
	newBlock := NewBlock(
		lastBlock.Header.Index+1, // 区块索引
		lastBlock.Hash,           // 前一区块哈希
		validTransactions,        // 验证后的交易
		miner,                    // 矿工账户
		50.0,                     // 挖矿奖励
		bc.Difficulty,            // 挖矿难度
	)
	bc.Blocks = append(bc.Blocks, newBlock)
	SaveBlockchain(filePath, bc)
	fmt.Println("新区块已生成")
}

func (bc *Blockchain) GetBalance(account string) (float64, bool) {
	exists := false
	balance := 0.0

	for _, block := range bc.Blocks {
		for _, tx := range block.Transactions {
			if tx.Sender == account || tx.Receiver == account {
				exists = true
			}
			if tx.Sender == account {
				balance -= tx.Amount
			}
			if tx.Receiver == account {
				balance += tx.Amount
			}
		}
	}

	return balance, exists
}
