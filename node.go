package main

import (
	"bufio"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"gamechain/account"
	"net"
	"os"
	"strconv"
	"strings"
)

// 定义节点结构
type Node struct {
	Address         string                      // 节点地址
	Blockchain      *Blockchain                 // 区块链实例
	TransactionPool []Transaction               // 本地交易池
	PeerNodes       []string                    // 已连接的其他节点地址
	PublicKeys      map[string]*ecdsa.PublicKey // 用户公钥存储
}

const (
	RequestTypeSync           = "sync"
	RequestTypeNewBlock       = "new_block"
	RequestTypeNewTransaction = "new_transaction"
)

func (node *Node) BroadcastTransaction(tx Transaction) {
	request := map[string]interface{}{
		"type":        RequestTypeNewTransaction,
		"transaction": tx,
	}
	for _, peer := range node.PeerNodes {
		if err := sendRequestToPeer(peer, request); err != nil {
			continue
		}
	}
}

// 广播新区块给其他节点
func (node *Node) BroadcastBlock(block Block) {
	request := map[string]interface{}{
		"type":  RequestTypeNewBlock,
		"block": block,
	}
	for _, peer := range node.PeerNodes {
		if err := sendRequestToPeer(peer, request); err != nil {
			continue
		}
	}
}

// 处理来自其他节点的连接
func (node *Node) HandleConnection(conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)
	message, err := reader.ReadString('\n')
	if err != nil {
		fmt.Printf("读取消息错误: %v\n", err)
		return
	}

	var request map[string]interface{}
	err = json.Unmarshal([]byte(message), &request)
	if err != nil {
		fmt.Printf("解析消息错误: %v\n", err)
		return
	}

	switch request["type"] {
	case RequestTypeNewTransaction:
		var tx Transaction
		txData, _ := json.Marshal(request["transaction"])
		json.Unmarshal(txData, &tx)
		node.HandleNewTransaction(tx)

	case RequestTypeNewBlock:
		var block Block
		blockData, _ := json.Marshal(request["block"])
		json.Unmarshal(blockData, &block)
		node.HandleNewBlock(block)

	case RequestTypeSync:
		node.SendBlockchain(conn)

	default:
		fmt.Printf("未知请求类型: %v\n", request["type"])
	}
}

// 处理新交易
func (node *Node) HandleNewTransaction(tx Transaction) {
	filePath := fmt.Sprintf("%s_transaction_pool.json", node.Address)
	if node.Blockchain.AddTransactionToPool(tx, node.PublicKeys, filePath) {
		fmt.Printf("新交易已添加到交易池: %+v\n", tx)
	} else {
		fmt.Printf("新交易验证失败: %+v\n", tx)
	}
}

// 处理新块
func (node *Node) HandleNewBlock(block Block) {
	lastBlock := node.Blockchain.Blocks[len(node.Blockchain.Blocks)-1]

	// 验证区块的合法性
	if block.Header.PreviousHash == lastBlock.Hash && block.Hash == block.CalculateHash() {
		node.Blockchain.Blocks = append(node.Blockchain.Blocks, block)
		node.Blockchain.ClearTransactionPool(block.Transactions) // 修复方法未定义问题
		SaveBlockchain(blockchainFile, node.Blockchain)
		fmt.Printf("新块已接受: #%d\n", block.Header.Index)
	} else if block.Header.Index > lastBlock.Header.Index {
		// 长链规则：尝试同步链
		fmt.Println("检测到更长的链，尝试同步...")
		node.SyncBlockchain()
	} else {
		fmt.Println("接收到无效的块，已忽略")
	}
}

// 启动节点服务器
func (node *Node) Start() {
	listener, err := net.Listen("tcp", node.Address)
	if err != nil {
		fmt.Printf("无法启动节点 %s: %v\n", node.Address, err)
		os.Exit(1)
	}
	defer listener.Close()

	fmt.Printf("节点已启动，监听地址: %s\n", node.Address)
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Printf("连接错误: %v\n", err)
			continue
		}
		go node.HandleConnection(conn)
	}
}

// 同步区块链
func (node *Node) SyncBlockchain() {
	for _, peer := range node.PeerNodes {
		request := map[string]interface{}{
			"type": RequestTypeSync,
		}
		if err := sendRequestToPeer(peer, request); err != nil {
			fmt.Printf("同步请求发送失败: %v\n", err)
			continue
		}

		// 接收区块链数据
		conn, err := net.Dial("tcp", peer)
		if err != nil {
			fmt.Printf("无法连接到节点 %s: %v\n", peer, err)
			continue
		}
		defer conn.Close()

		reader := bufio.NewReader(conn)
		response, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("从节点 %s 接收数据失败: %v\n", peer, err)
			continue
		}

		var receivedChain Blockchain
		err = json.Unmarshal([]byte(response), &receivedChain)
		if err != nil {
			fmt.Printf("解析区块链数据失败: %v\n", err)
			continue
		}

		if len(receivedChain.Blocks) > len(node.Blockchain.Blocks) {
			node.Blockchain = &receivedChain
			SaveBlockchain(blockchainFile, node.Blockchain)
			fmt.Printf("已从节点 %s 同步到更长的链\n", peer)
		} else {
			fmt.Printf("节点 %s 的链较短，无需更新\n", peer)
		}
	}
}

// 返回区块链数据
func (node *Node) SendBlockchain(conn net.Conn) {
	data, _ := json.Marshal(node.Blockchain)
	conn.Write(data)
}

// 交互模式
func (node *Node) RunInteractive(
	privateKeys map[string]*ecdsa.PrivateKey,
	accounts *[]account.Account,
	accountsFile,
	transactionPoolFile,
	blockchainFile,
	encryptionKey string,
	balanceManager *account.BalanceManager, // 添加 BalanceManager 参数
) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("节点 %s 已启动，输入 'help' 查看可用指令。\n", node.Address)

	for {
		fmt.Print("> ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input) // 去掉用户输入的空格和换行符

		switch {
		case input == "help":
			fmt.Println("可用指令：")
			fmt.Println("  mine - 挖矿并生成新区块")
			fmt.Println("  tx [sender] [receiver] [amount] - 创建并广播交易")
			fmt.Println("  sync - 从其他节点同步区块链")
			fmt.Println("  balance [account] - 查询账户余额")
			fmt.Println("  create_account [name] - 创建新账户")
			fmt.Println("  list_accounts - 列出所有账户")
			fmt.Println("  print - 打印区块链状态")
			fmt.Println("  exit - 退出程序")

		case input == "mine":
			fmt.Print("请输入矿工账户名称: ")
			minerInput, err := reader.ReadString('\n')
			if err != nil {
				fmt.Println("读取矿工账户名称失败:", err)
				return
			}
			miner := strings.TrimSpace(minerInput)

			transactions := node.Blockchain.GetTransactionsForBlock()
			if len(transactions) == 0 {
				fmt.Println("没有交易可供打包，跳过挖矿")
				continue
			}

			node.Blockchain.AddBlock(transactions, miner, node.PublicKeys, blockchainFile)
			node.BroadcastBlock(node.Blockchain.Blocks[len(node.Blockchain.Blocks)-1])
			fmt.Printf("新区块已生成并广播，矿工 %s 获得奖励 50.0\n", miner)

		case strings.HasPrefix(input, "tx "):
			args := strings.Split(input, " ")
			if len(args) != 4 {
				fmt.Println("用法: tx [sender] [receiver] [amount]")
				continue
			}
			sender := args[1]
			receiver := args[2]
			amount := parseAmount(args[3])
			privateKey, exists := privateKeys[sender]
			if !exists {
				fmt.Printf("发送方账户 %s 不存在或私钥丢失\n", sender)
				continue
			}

			// 检查余额是否充足
			if !balanceManager.DeductBalance(sender, amount) {
				fmt.Printf("账户 %s 余额不足\n", sender)
				continue
			}

			tx := NewTransaction(sender, receiver, amount, privateKey)

			// 调用 AddTransactionToPool，并打印 TransactionPool
			if node.Blockchain.AddTransactionToPool(tx, node.PublicKeys, transactionPoolFile) {
				fmt.Printf("交易已加入本地交易池: %+v\n", tx)
				balanceManager.AddBalance(receiver, amount)
			} else {
				fmt.Println("交易未能加入交易池")
			}

			node.BroadcastTransaction(tx)
			fmt.Printf("交易已广播: %s -> %s (金额: %.2f)\n", sender, receiver, amount)

		case strings.HasPrefix(input, "balance "):
			args := strings.Split(input, " ")
			if len(args) != 2 {
				fmt.Println("用法: balance [account]")
				continue
			}
			account := args[1]
			balance, exists := balanceManager.GetBalance(account)
			if !exists {
				fmt.Printf("账户 %s 不存在\n", account)
			} else {
				fmt.Printf("账户 %s 的余额: %.2f\n", account, balance)
			}

		case strings.HasPrefix(input, "create_account "):
			args := strings.Split(input, " ")
			if len(args) != 2 {
				fmt.Println("用法: create_account [name]")
				continue
			}
			name := args[1]
			account.CreateNewAccount(name, accounts, privateKeys, node.PublicKeys, accountsFile, encryptionKey)
			balanceManager.SetBalance(name, 100.0) // 初始化账户余额
			fmt.Printf("账户 %s 已创建\n", name)

		case input == "list_accounts":
			fmt.Println("现有账户:")
			for _, acc := range *accounts {
				fmt.Printf("- %s\n", acc.Name)
			}

		case input == "print":
			PrintBlockchain(node.Blockchain)

		case input == "sync":
			node.SyncBlockchain()

		case input == "exit":
			fmt.Println("退出节点...")
			return

		default:
			fmt.Println("未知指令，输入 'help' 查看可用指令。")
		}
	}
}

// 辅助函数：解析金额
func parseAmount(amountStr string) float64 {
	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
		fmt.Printf("无效金额: %s\n", amountStr)
		return 0
	}
	return amount
}

// sendRequestToPeer 向指定节点发送请求
func sendRequestToPeer(peer string, request map[string]interface{}) error {
	conn, err := net.Dial("tcp", peer)
	if err != nil {
		fmt.Printf("无法连接到节点 %s: %v\n", peer, err)
		return err
	}
	defer conn.Close()

	data, _ := json.Marshal(request)
	data = append(data, '\n') // 确保接收方能够正确解析
	_, err = conn.Write(data)
	if err != nil {
		fmt.Printf("发送请求到节点 %s 失败: %v\n", peer, err)
	}
	return err
}
