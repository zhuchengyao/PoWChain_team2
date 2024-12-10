package main

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"gamechain/account"
	"os"
	"strings"
)

func SaveBlockchain(filePath string, blockchain *Blockchain) {
	data, _ := json.MarshalIndent(blockchain, "", "  ")
	err := os.WriteFile(filePath, data, 0644)
	if err != nil {
		fmt.Printf("保存区块链失败: %v\n", err)
	}
}

// 初始化区块链（包括加载和创世区块的创建）
func initializeBlockchain(filePath string, difficulty int) *Blockchain {
	blockchain := LoadBlockchain(filePath)
	if len(blockchain.Blocks) == 0 {
		// 创建创世区块
		genesisBlock := NewBlock(
			0,               // 区块编号
			"0",             // 前一区块的哈希（创世区块无前区块）
			[]Transaction{}, // 创世区块无交易
			"System",        // 矿工账户（系统账户）
			0.0,             // 奖励（创世区块无奖励）
			difficulty,      // 挖矿难度
		)
		blockchain.Blocks = append(blockchain.Blocks, genesisBlock)
		SaveBlockchain(filePath, blockchain)
		fmt.Println("创世区块已生成并保存")
	}
	return blockchain
}

// CalculateHash 计算区块的哈希
func (b *Block) CalculateHash() string {
	data := fmt.Sprintf("%d%d%s%d%s",
		b.Header.Index,
		b.Header.Timestamp,
		b.Header.PreviousHash,
		b.Header.Nonce,
		b.Header.MerkleRoot,
	)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// ProofOfWork 执行工作量证明
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

func (bc *Blockchain) GetBalance(account string, accounts []account.Account) (float64, bool) {
	exists := false
	balance := 0.0

	// 遍历区块链获取交易记录
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

	// 检查账户是否在 accounts.json 中存在
	if !exists {
		for _, acc := range accounts {
			if acc.Name == account {
				exists = true
				balance = 100.0 // 返回初始余额
				break
			}
		}
	}

	return balance, exists
}
