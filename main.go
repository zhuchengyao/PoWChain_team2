package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"math/big"
	"os"
	"strings"
	"time"
)

// Transaction 定义交易结构
type Transaction struct {
	Sender    string  // 发送方
	Receiver  string  // 接收方
	Amount    float64 // 金额
	Signature string  // 签名
}

// BlockHeader 定义区块头
type BlockHeader struct {
	Index        int    // 区块编号
	Timestamp    int64  // 时间戳
	PreviousHash string // 前一个区块哈希
	Nonce        uint64 // 工作量证明随机数
	MerkleRoot   string // Merkle 树根
}

// Block 定义区块结构
type Block struct {
	Header       BlockHeader   // 区块头
	Transactions []Transaction // 交易列表
	Hash         string        // 区块哈希
}

// Blockchain 定义区块链结构
type Blockchain struct {
	Blocks          []Block       // 区块列表
	Difficulty      int           // 挖矿难度
	TransactionPool []Transaction // 未确认的交易池
}

var blockchain *Blockchain // 全局区块链实例

// GenerateKeyPair 生成公私钥对
func GenerateKeyPair() (*ecdsa.PrivateKey, *ecdsa.PublicKey) {
	privateKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	return privateKey, &privateKey.PublicKey
}

// CalculateMerkleRoot 计算交易的 Merkle 树根
func CalculateMerkleRoot(transactions []Transaction) string {
	if len(transactions) == 0 {
		return ""
	}

	hashes := []string{}
	for _, tx := range transactions {
		txData := fmt.Sprintf("%s%s%f%s", tx.Sender, tx.Receiver, tx.Amount, tx.Signature)
		hash := sha256.Sum256([]byte(txData))
		hashes = append(hashes, hex.EncodeToString(hash[:]))
	}

	for len(hashes) > 1 {
		var nextLevel []string
		for i := 0; i < len(hashes); i += 2 {
			if i+1 < len(hashes) {
				combined := sha256.Sum256([]byte(hashes[i] + hashes[i+1]))
				nextLevel = append(nextLevel, hex.EncodeToString(combined[:]))
			} else {
				nextLevel = append(nextLevel, hashes[i])
			}
		}
		hashes = nextLevel
	}

	return hashes[0]
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

// NewBlock 创建新区块
func NewBlock(index int, previousHash string, transactions []Transaction, miner string, reward float64, difficulty int) Block {
	// 添加矿工奖励交易
	rewardTx := Transaction{
		Sender:   "System", // 系统账户
		Receiver: miner,    // 矿工账户
		Amount:   reward,   // 固定奖励金额
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

	// 工作量证明
	block.ProofOfWork(difficulty)
	return block
}

// LoadBlockchain 从文件加载区块链
func LoadBlockchain(filePath string) *Blockchain {
	data, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Println("未找到区块链文件，创建新区块链")
		return &Blockchain{Difficulty: 2}
	}
	var blockchain Blockchain
	json.Unmarshal(data, &blockchain)
	return &blockchain
}

// NewTransaction 创建交易
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

// VerifyTransaction 验证交易签名
func VerifyTransaction(tx *Transaction, publicKey *ecdsa.PublicKey) bool {
	txData := fmt.Sprintf("%s%s%f", tx.Sender, tx.Receiver, tx.Amount)
	hash := sha256.Sum256([]byte(txData))

	fmt.Printf("验证交易哈希: %x\n", hash) // 打印交易哈希
	var r, s big.Int
	n, err := fmt.Sscanf(tx.Signature, "%s:%s", &r, &s) // 调试解析结果
	if err != nil || n != 2 {
		fmt.Printf("签名解析失败: %v (n=%d)\n", err, n)
		return false
	}
	fmt.Printf("验证签名: r=%s, s=%s\n", r.String(), s.String()) // 打印签名值

	return ecdsa.Verify(publicKey, hash[:], &r, &s)
}

// SignTransaction 使用私钥签名
func SignTransaction(tx *Transaction, privateKey *ecdsa.PrivateKey) {
	txData := fmt.Sprintf("%s%s%f", tx.Sender, tx.Receiver, tx.Amount)
	hash := sha256.Sum256([]byte(txData))
	r, s, _ := ecdsa.Sign(rand.Reader, privateKey, hash[:])
	tx.Signature = fmt.Sprintf("%s:%s", r.String(), s.String())

	// 调试：打印签名内容
	fmt.Printf("生成交易签名: %s\n", tx.Signature)
}

// 定义文件路径常量
const (
	accountsFile        = "accounts.json"
	blockchainFile      = "blockchain.json"
	transactionPoolFile = "transaction_pool.json"
	encryptionKey       = "my_secure_password" // 用于加解密私钥的密钥
)

// 入口函数
func main() {
	// 加载账户
	accounts, privateKeys, publicKeys := LoadAccounts(accountsFile, encryptionKey)
	blockchain = LoadBlockchain(blockchainFile)
	if len(blockchain.Blocks) == 0 {
		// 创建创世区块
		genesisBlock := NewBlock(0, "0", []Transaction{}, "System", 0, blockchain.Difficulty)
		blockchain.Blocks = append(blockchain.Blocks, genesisBlock)
		SaveBlockchain(blockchainFile, blockchain)
	}

	// 解析命令行参数
	address := flag.String("address", "localhost:8080", "节点地址")
	peers := flag.String("peers", "", "逗号分隔的其他节点地址")
	flag.Parse()

	peerNodes := []string{}
	if *peers != "" {
		peerNodes = strings.Split(*peers, ",")
	}

	// 加载区块链
	blockchain := LoadBlockchain(blockchainFile)
	if len(blockchain.Blocks) == 0 {
		// 创建创世区块
		genesisBlock := NewBlock(
			0,                     // 区块编号
			"0",                   // 前一区块的哈希（创世区块无前区块）
			[]Transaction{},       // 创世区块无交易
			"System",              // 矿工账户（系统账户）
			0.0,                   // 奖励（创世区块无奖励）
			blockchain.Difficulty, // 挖矿难度
		)
		blockchain.Blocks = append(blockchain.Blocks, genesisBlock)
		SaveBlockchain(blockchainFile, blockchain)
	}

	// 初始化节点
	node := Node{
		Address:         *address,
		Blockchain:      blockchain,
		TransactionPool: blockchain.TransactionPool,
		PeerNodes:       peerNodes,
		PublicKeys:      publicKeys,
	}

	// 启动节点
	go node.Start()

	// 交互式命令行
	node.RunInteractive(privateKeys, &accounts, accountsFile, transactionPoolFile, blockchainFile, encryptionKey)
}
