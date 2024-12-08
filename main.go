package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"time"
)

func (bc *Blockchain) SaveToFile(filePath string) error {
	data, err := json.MarshalIndent(bc, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, data, 0644)
}

// 从文件加载区块链状态
func LoadBlockchainFromFile(filePath string) (*Blockchain, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	var blockchain Blockchain
	err = json.Unmarshal(data, &blockchain)
	if err != nil {
		return nil, err
	}
	return &blockchain, nil
}

// 生成公私钥对
func GenerateKeyPair() (*ecdsa.PrivateKey, *ecdsa.PublicKey) {
	privateKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	return privateKey, &privateKey.PublicKey
}

// 验证交易签名
func VerifyTransaction(tx *Transaction, publicKey *ecdsa.PublicKey) bool {
	// 计算交易的哈希值
	txData := fmt.Sprintf("%s%s%f", tx.Sender, tx.Receiver, tx.Amount)
	hash := sha256.Sum256([]byte(txData))

	// 从签名中解析 r 和 s
	var r, s big.Int
	fmt.Sscanf(tx.Signature, "%s:%s", &r, &s)

	// 验证签名
	return ecdsa.Verify(publicKey, hash[:], &r, &s)
}

// 为交易生成签名
func SignTransaction(tx *Transaction, privateKey *ecdsa.PrivateKey) {
	// 计算交易的哈希值
	txData := fmt.Sprintf("%s%s%f", tx.Sender, tx.Receiver, tx.Amount)
	hash := sha256.Sum256([]byte(txData))

	// 使用私钥签名
	r, s, _ := ecdsa.Sign(rand.Reader, privateKey, hash[:])

	// 将签名存储为字符串（便于存储和传输）
	tx.Signature = fmt.Sprintf("%s:%s", r.String(), s.String())
}

// 交易结构
type Transaction struct {
	Sender    string  // 发送方
	Receiver  string  // 接收方
	Amount    float64 // 金额
	Signature string  // 签名
}

// 区块头结构
type BlockHeader struct {
	Index        int    // 区块编号
	Timestamp    int64  // 时间戳
	PreviousHash string // 前一个区块的哈希
	Nonce        uint64 // 随机数
	MerkleRoot   string // 交易的 Merkle 根
}

func CalculateMerkleRoot(transactions []Transaction) string {
	if len(transactions) == 0 {
		return ""
	}

	// 将交易的哈希值作为叶子节点
	var hashes []string
	for _, tx := range transactions {
		txData := fmt.Sprintf("%s%s%f%s", tx.Sender, tx.Receiver, tx.Amount, tx.Signature)
		hash := sha256.Sum256([]byte(txData))
		hashes = append(hashes, hex.EncodeToString(hash[:]))
	}

	// 递归计算 Merkle 树的根
	for len(hashes) > 1 {
		var newLevel []string
		for i := 0; i < len(hashes); i += 2 {
			if i+1 < len(hashes) {
				// 合并相邻两个哈希值
				combinedHash := sha256.Sum256([]byte(hashes[i] + hashes[i+1]))
				newLevel = append(newLevel, hex.EncodeToString(combinedHash[:]))
			} else {
				// 如果是奇数个节点，直接将最后一个节点提升到下一层
				newLevel = append(newLevel, hashes[i])
			}
		}
		hashes = newLevel
	}

	// 最后一个节点就是 Merkle 根
	return hashes[0]
}

// 区块结构
type Block struct {
	Header       BlockHeader   // 区块头
	Transactions []Transaction // 交易列表
	Hash         string        // 当前区块的哈希
}

// 计算区块的哈希值
func (b *Block) CalculateHash() string {
	blockData := fmt.Sprintf("%d%d%s%d%+v",
		b.Header.Index,
		b.Header.Timestamp,
		b.Header.PreviousHash,
		b.Header.Nonce,
		b.Transactions,
	)
	fmt.Printf("计算哈希的输入数据: %s\n", blockData)

	hash := sha256.Sum256([]byte(blockData))
	return hex.EncodeToString(hash[:])
}

// 工作量证明：找到满足难度要求的哈希

func (b *Block) ProofOfWork(difficulty int) {
	prefix := make([]byte, difficulty)
	for i := range prefix {
		prefix[i] = '0'
	}
	target := string(prefix)

	const logInterval = 10000 // 每 10,000 次打印一次
	startTime := time.Now()
	for {
		b.Hash = b.CalculateHash()
		if b.Header.Nonce%logInterval == 0 {
			fmt.Printf("尝试 Nonce: %d, Hash: %s\n", b.Header.Nonce, b.Hash)
		}
		if b.Hash[:difficulty] == target {
			elapsed := time.Since(startTime)
			fmt.Printf("找到有效哈希: %s (Nonce: %d)\n", b.Hash, b.Header.Nonce)
			fmt.Printf("挖矿耗时: %s\n", elapsed)
			break
		}
		b.Header.Nonce++
	}
}

func NewBlock(index int, previousHash string, transactions []Transaction, difficulty int) Block {
	block := Block{
		Header: BlockHeader{
			Index:        index,
			Timestamp:    time.Now().Unix(),
			PreviousHash: previousHash,
			MerkleRoot:   CalculateMerkleRoot(transactions), // 计算 Merkle 根
		},
		Transactions: transactions,
	}

	// 执行工作量证明
	block.ProofOfWork(difficulty)
	return block
}

func PrintBlockchain(blockchain *Blockchain) {
	for _, block := range blockchain.Blocks {
		fmt.Printf("区块 #%d\n", block.Header.Index)
		fmt.Printf("时间戳: %d\n", block.Header.Timestamp)
		fmt.Printf("前一个哈希: %s\n", block.Header.PreviousHash)
		fmt.Printf("Merkle 根: %s\n", block.Header.MerkleRoot)
		fmt.Printf("Nonce: %d\n", block.Header.Nonce)
		fmt.Printf("哈希: %s\n", block.Hash)
		fmt.Printf("交易: %+v\n", block.Transactions)
		fmt.Println("-------------------------")
	}
}

// 向区块链中添加一个区块
func (bc *Blockchain) AddBlock(transactions []Transaction, publicKeys map[string]*ecdsa.PublicKey) {
	validTransactions := []Transaction{}
	for _, tx := range transactions {
		// 验证签名
		if publicKey, exists := publicKeys[tx.Sender]; exists && VerifyTransaction(&tx, publicKey) {
			validTransactions = append(validTransactions, tx)
		} else {
			fmt.Printf("交易验证失败: %+v\n", tx)
		}
	}

	// 创建新区块
	lastBlock := bc.Blocks[len(bc.Blocks)-1]
	newBlock := NewBlock(lastBlock.Header.Index+1, lastBlock.Hash, validTransactions, bc.Difficulty)
	bc.Blocks = append(bc.Blocks, newBlock)
}

// 区块链结构
type Blockchain struct {
	Blocks          []Block       // 区块链
	Difficulty      int           // 挖矿难度
	TransactionPool []Transaction // 未确认交易池
}

func (bc *Blockchain) AddTransactionToPool(tx Transaction, publicKeys map[string]*ecdsa.PublicKey, filePath string) bool {
	if publicKey, exists := publicKeys[tx.Sender]; exists && VerifyTransaction(&tx, publicKey) {
		bc.TransactionPool = append(bc.TransactionPool, tx)
		fmt.Printf("交易已加入交易池: %+v\n", tx)

		// 保存区块链状态到文件
		if err := bc.SaveToFile(filePath); err != nil {
			fmt.Printf("保存区块链失败: %v\n", err)
		} else {
			fmt.Println("交易池已成功保存到文件:", filePath)
		}
		return true
	}
	fmt.Printf("交易验证失败: %+v\n", tx)
	return false
}

func (bc *Blockchain) GetTransactionsForBlock() []Transaction {
	// 简单实现：直接返回交易池中的所有交易
	return bc.TransactionPool
}

func (bc *Blockchain) ClearTransactionPool(transactions []Transaction) {
	remaining := []Transaction{}
	for _, poolTx := range bc.TransactionPool {
		included := false
		for _, blockTx := range transactions {
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
	fmt.Printf("已清理交易池中的已打包交易\n")
}

// func (bc *Blockchain) AddBlock(transactions []Transaction, publicKeys map[string]*ecdsa.PublicKey, filePath string) {
// 	validTransactions := []Transaction{}
// 	for _, tx := range transactions {
// 		// 验证签名
// 		if publicKey, exists := publicKeys[tx.Sender]; exists && VerifyTransaction(&tx, publicKey) {
// 			validTransactions = append(validTransactions, tx)
// 		} else {
// 			fmt.Printf("交易验证失败: %+v\n", tx)
// 		}
// 	}

// 	// 创建新区块
// 	lastBlock := bc.Blocks[len(bc.Blocks)-1]
// 	newBlock := NewBlock(lastBlock.Header.Index+1, lastBlock.Hash, validTransactions, bc.Difficulty)
// 	bc.Blocks = append(bc.Blocks, newBlock)

// 	// 保存区块链状态到文件
// 	if err := bc.SaveToFile(filePath); err != nil {
// 		fmt.Printf("保存区块链失败: %v\n", err)
// 	} else {
// 		fmt.Println("区块链已成功保存到文件:", filePath)
// 	}
// }

// 创建一个新交易
func NewTransaction(sender string, receiver string, amount float64, privateKey *ecdsa.PrivateKey) Transaction {
	tx := Transaction{
		Sender:   sender,
		Receiver: receiver,
		Amount:   amount,
	}

	// 如果 privateKey 为 nil，打印警告并跳过签名
	if privateKey == nil {
		fmt.Printf("警告：交易发送方 %s 的私钥为空，未生成签名\n", sender)
		return tx
	}

	SignTransaction(&tx, privateKey)
	return tx
}

func main() {
	// 文件路径
	node1FilePath := "node1_blockchain.json"
	node2FilePath := "node2_blockchain.json"
	node3FilePath := "node3_blockchain.json"

	// 生成密钥对
	alicePrivateKey, alicePublicKey := GenerateKeyPair()
	bobPrivateKey, bobPublicKey := GenerateKeyPair()
	charliePrivateKey, charliePublicKey := GenerateKeyPair()

	// 公钥映射
	publicKeys := map[string]*ecdsa.PublicKey{
		"Alice":   alicePublicKey,
		"Bob":     bobPublicKey,
		"Charlie": charliePublicKey,
	}

	// 加载节点 1 的区块链
	node1Blockchain, err := LoadBlockchainFromFile(node1FilePath)
	if err != nil {
		fmt.Println("节点 1: 未找到区块链文件，创建新区块链")
		node1Blockchain = &Blockchain{Difficulty: 2, Blocks: []Block{}}
	}

	// 加载节点 2 的区块链
	node2Blockchain, err := LoadBlockchainFromFile(node2FilePath)
	if err != nil {
		fmt.Println("节点 2: 未找到区块链文件，创建新区块链")
		node2Blockchain = &Blockchain{Difficulty: 2, Blocks: []Block{}}
	}

	// 加载节点 3 的区块链
	node3Blockchain, err := LoadBlockchainFromFile(node3FilePath)
	if err != nil {
		fmt.Println("节点 3: 未找到区块链文件，创建新区块链")
		node3Blockchain = &Blockchain{Difficulty: 2, Blocks: []Block{}}
	}

	// 创建节点
	node1 := Node{
		Address:    "localhost:8080",
		Blockchain: node1Blockchain,
		PeerNodes:  []string{"localhost:8081", "localhost:8082"},
		PublicKeys: publicKeys,
	}

	node2 := Node{
		Address:    "localhost:8081",
		Blockchain: node2Blockchain,
		PeerNodes:  []string{"localhost:8080", "localhost:8082"},
		PublicKeys: publicKeys,
	}

	node3 := Node{
		Address:    "localhost:8082",
		Blockchain: node3Blockchain,
		PeerNodes:  []string{"localhost:8080", "localhost:8081"},
		PublicKeys: publicKeys,
	}

	// 创世区块（仅当区块链为空时生成）
	if len(node1.Blockchain.Blocks) == 0 {
		genesisBlock := NewBlock(0, "0", []Transaction{}, node1.Blockchain.Difficulty)
		node1.Blockchain.Blocks = append(node1.Blockchain.Blocks, genesisBlock)
	}

	// 保存区块链状态
	defer node1.Blockchain.SaveToFile(node1FilePath)
	defer node2.Blockchain.SaveToFile(node2FilePath)
	defer node3.Blockchain.SaveToFile(node3FilePath)

	// 启动节点
	go node1.Start()
	go node2.Start()
	go node3.Start()

	// 模拟交易广播
	time.Sleep(1 * time.Second)
	tx := NewTransaction("Alice", "Bob", 10, alicePrivateKey)
	node1.BroadcastTransaction(tx)

	// 模拟 Bob 发起交易
	time.Sleep(1 * time.Second)
	txFromBob := NewTransaction("Bob", "Charlie", 5, bobPrivateKey)
	node2.BroadcastTransaction(txFromBob)

	// 模拟 Charlie 发起交易
	time.Sleep(1 * time.Second)
	txFromCharlie := NewTransaction("Charlie", "Alice", 2, charliePrivateKey)
	node3.BroadcastTransaction(txFromCharlie)

	// 等待同步完成
	time.Sleep(2 * time.Second)

	// 打印每个节点的交易池状态
	fmt.Println("节点1的交易池:")
	for _, tx := range node1.Blockchain.TransactionPool {
		fmt.Printf("交易: %+v\n", tx)
	}

	fmt.Println("节点2的交易池:")
	for _, tx := range node2.Blockchain.TransactionPool {
		fmt.Printf("交易: %+v\n", tx)
	}

	fmt.Println("节点3的交易池:")
	for _, tx := range node3.Blockchain.TransactionPool {
		fmt.Printf("交易: %+v\n", tx)
	}
}
