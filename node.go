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

type Node struct {
	Address         string
	Blockchain      *Blockchain
	TransactionPool []Transaction
	PeerNodes       []string
	PublicKeys      map[string]*ecdsa.PublicKey
	BalanceManager  *account.BalanceManager
}

const (
	RequestTypeSync           = "sync"
	RequestTypeNewBlock       = "new_block"
	RequestTypeNewTransaction = "new_transaction"
	RequestTypeUpdateBalance  = "update_balance"
)

func (node *Node) BroadcastTransaction(tx Transaction) {
	node.broadcast(RequestTypeNewTransaction, map[string]interface{}{"transaction": tx})
}

func (node *Node) BroadcastBlock(block Block) {
	node.broadcast(RequestTypeNewBlock, map[string]interface{}{"block": block})
}

func (node *Node) broadcast(requestType string, data map[string]interface{}) {
	data["type"] = requestType
	for _, peer := range node.PeerNodes {
		if err := sendRequestToPeer(peer, data); err != nil {
			fmt.Printf("广播到节点 %s 失败: %v\n", peer, err)
		} else {
			fmt.Printf("广播成功到节点 %s\n", peer)
		}
	}
}

func (node *Node) HandleConnection(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	message, err := reader.ReadString('\n')
	if err != nil {
		fmt.Printf("读取消息错误: %v\n", err)
		return
	}

	var request map[string]interface{}
	if err := json.Unmarshal([]byte(message), &request); err != nil {
		fmt.Printf("解析消息错误: %v\n", err)
		return
	}

	switch request["type"] {
	case RequestTypeNewTransaction:
		node.handleTransaction(request)
	case RequestTypeNewBlock:
		node.handleBlock(request)
	case RequestTypeSync:
		node.SendBlockchain(conn)
	case RequestTypeUpdateBalance:
		node.updateBalance(request)
	default:
		fmt.Printf("未知请求类型: %v\n", request["type"])
	}
}

func (node *Node) handleTransaction(request map[string]interface{}) {
	var tx Transaction
	if err := mapToStruct(request["transaction"], &tx); err != nil {
		fmt.Printf("交易解析失败: %v\n", err)
		return
	}
	node.HandleNewTransaction(tx)
}

func (node *Node) handleBlock(request map[string]interface{}) {
	var block Block
	if err := mapToStruct(request["block"], &block); err != nil {
		fmt.Printf("区块解析失败: %v\n", err)
		return
	}
	node.HandleNewBlock(block)
}

func (node *Node) updateBalance(request map[string]interface{}) {
	accountName := request["account"].(string)
	newBalance := request["newBalance"].(float64)
	node.BalanceManager.SetBalance(accountName, newBalance)
	if err := node.BalanceManager.SaveBalances(balancesFile); err != nil {
		fmt.Printf("保存余额失败: %v\n", err)
	}
	fmt.Printf("账户 %s 的余额已更新为 %.2f\n", accountName, newBalance)
}

func (node *Node) HandleNewTransaction(tx Transaction) {
	filePath := fmt.Sprintf("%s_transaction_pool.json", node.Address)
	if node.Blockchain.AddTransactionToPool(tx, node.PublicKeys, filePath) {
		fmt.Printf("交易已添加到交易池: %+v\n", tx)
	} else {
		fmt.Printf("交易验证失败: %+v\n", tx)
	}
}

func (node *Node) HandleNewBlock(block Block) {
	lastBlock := node.Blockchain.Blocks[len(node.Blockchain.Blocks)-1]
	if block.Header.PreviousHash == lastBlock.Hash && block.Hash == block.CalculateHash() {
		node.Blockchain.Blocks = append(node.Blockchain.Blocks, block)
		node.Blockchain.ClearTransactionPool(block.Transactions)
		SaveBlockchain(blockchainFile, node.Blockchain)
		fmt.Printf("新块已接受: #%d\n", block.Header.Index)
	} else if block.Header.Index > lastBlock.Header.Index {
		fmt.Println("检测到更长的链，尝试同步...")
		node.SyncBlockchain()
	} else {
		fmt.Println("无效块，已忽略")
	}
}

func (node *Node) Start() {
	listener, err := net.Listen("tcp", node.Address)
	if err != nil {
		fmt.Printf("无法启动节点 %s: %v\n", node.Address, err)
		os.Exit(1)
	}
	defer listener.Close()
	fmt.Printf("节点启动，监听地址: %s\n", node.Address)

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Printf("连接错误: %v\n", err)
			continue
		}
		go node.HandleConnection(conn)
	}
}

func (node *Node) SyncBlockchain() {
	for _, peer := range node.PeerNodes {
		request := map[string]interface{}{"type": RequestTypeSync}
		if err := sendRequestToPeer(peer, request); err != nil {
			fmt.Printf("同步失败: %v\n", err)
			continue
		}

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
		if err := json.Unmarshal([]byte(response), &receivedChain); err != nil {
			fmt.Printf("解析区块链失败: %v\n", err)
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

func (node *Node) SendBlockchain(conn net.Conn) {
	data, _ := json.Marshal(node.Blockchain)
	conn.Write(data)
}

func sendRequestToPeer(peer string, request map[string]interface{}) error {
	conn, err := net.Dial("tcp", peer)
	if err != nil {
		return err
	}
	defer conn.Close()

	data, _ := json.Marshal(request)
	_, err = conn.Write(append(data, '\n'))
	return err
}

func mapToStruct(data interface{}, target interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return json.Unmarshal(jsonData, target)
}

func (node *Node) RunInteractive(
	privateKeys map[string]*ecdsa.PrivateKey,
	accounts *[]account.Account,
	accountsFile, transactionPoolFile, blockchainFile, encryptionKey string,
	balanceManager *account.BalanceManager,
) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("节点 %s 已启动，输入 'help' 查看可用指令。\n", node.Address)

	// 命令映射
	commands := map[string]func([]string){
		"help": node.showHelp,
		"mine": func(args []string) { node.handleMine(args, blockchainFile) },
		"tx": func(args []string) {
			node.handleTransactionCommand(args, privateKeys, transactionPoolFile, balanceManager)
		},
		"sync":    func(args []string) { node.SyncBlockchain() },
		"balance": func(args []string) { node.handleBalanceCommand(args, balanceManager) },
		"create_account": func(args []string) {
			node.handleCreateAccountCommand(args, accounts, privateKeys, accountsFile, encryptionKey, balanceManager)
		},
		"list_accounts":  func(args []string) { node.listAccounts(accounts) },
		"print":          func(args []string) { PrintBlockchain(node.Blockchain) },
		"verify_balance": func(args []string) { node.handleVerifyBalanceCommand(args, balanceManager) },
		"exit":           func(args []string) { node.exitNode(balanceManager) },
	}

	for {
		fmt.Print("> ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input) // 去掉用户输入的空格和换行符

		// 解析命令和参数
		parts := strings.Fields(input)
		if len(parts) == 0 {
			continue
		}
		command, args := parts[0], parts[1:]

		// 执行命令
		if cmdFunc, exists := commands[command]; exists {
			cmdFunc(args)
		} else {
			fmt.Println("未知指令，输入 'help' 查看可用指令。")
		}
	}
}

func (node *Node) showHelp(args []string) {
	fmt.Println("可用指令：")
	fmt.Println("  mine - 挖矿并生成新区块")
	fmt.Println("  tx [sender] [receiver] [amount] - 创建并广播交易")
	fmt.Println("  sync - 从其他节点同步区块链")
	fmt.Println("  balance [account] - 查询账户余额")
	fmt.Println("  create_account [name] - 创建新账户")
	fmt.Println("  list_accounts - 列出所有账户")
	fmt.Println("  print - 打印区块链状态")
	fmt.Println("  verify_balance [account] - 验证账户余额是否与区块链记录一致")
	fmt.Println("  exit - 退出程序")
}

func (node *Node) handleMine(args []string, blockchainFile string) {
	if len(args) != 1 {
		fmt.Println("用法: mine [miner_account]")
		return
	}
	miner := args[0]
	transactions := node.Blockchain.GetTransactionsForBlock()
	if len(transactions) == 0 {
		fmt.Println("没有交易可供打包，跳过挖矿")
		return
	}

	// 打包交易并生成新区块
	node.Blockchain.AddBlock(transactions, miner, node.PublicKeys, blockchainFile)

	// 从交易池中移除已打包的交易
	node.Blockchain.ClearTransactionPool(transactions)

	// 广播新区块
	node.BroadcastBlock(node.Blockchain.Blocks[len(node.Blockchain.Blocks)-1])
	fmt.Printf("新区块已生成并广播，矿工 %s 获得奖励 50.0\n", miner)
}

// parseAmount 将字符串解析为浮点数，如果解析失败则返回 0，并打印错误信息
func parseAmount(amountStr string) float64 {
	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
		fmt.Printf("无效金额: %s\n", amountStr)
		return 0
	}
	return amount
}

func (node *Node) handleTransactionCommand(args []string, privateKeys map[string]*ecdsa.PrivateKey, transactionPoolFile string, balanceManager *account.BalanceManager) {
	if len(args) != 3 {
		fmt.Println("用法: tx [sender] [receiver] [amount]")
		return
	}
	sender, receiver, amountStr := args[0], args[1], args[2]
	amount := parseAmount(amountStr)
	if amount <= 0 {
		return
	}

	if !balanceManager.DeductBalance(sender, amount, balancesFile) {
		fmt.Printf("[TX] 账户 %s 余额不足\n", sender)
		return
	}

	tx := NewTransaction(sender, receiver, amount, privateKeys[sender])
	if node.Blockchain.AddTransactionToPool(tx, node.PublicKeys, transactionPoolFile) {
		balanceManager.AddBalance(receiver, amount, balancesFile)
		node.BroadcastTransaction(tx)
		fmt.Printf("[TX] 交易已广播: %s -> %s (金额: %.2f)\n", sender, receiver, amount)
	} else {
		fmt.Println("[TX] 交易未能加入交易池")
	}
}

func (node *Node) handleBalanceCommand(args []string, balanceManager *account.BalanceManager) {
	if len(args) != 1 {
		fmt.Println("用法: balance [account]")
		return
	}
	account := args[0]
	balance, exists := balanceManager.GetBalance(account)
	if !exists {
		fmt.Printf("账户 %s 不存在\n", account)
	} else {
		fmt.Printf("账户 %s 的余额: %.2f\n", account, balance)
	}
}

func (node *Node) handleCreateAccountCommand(args []string, accounts *[]account.Account, privateKeys map[string]*ecdsa.PrivateKey, accountsFile, encryptionKey string, balanceManager *account.BalanceManager) {
	if len(args) != 1 {
		fmt.Println("用法: create_account [name]")
		return
	}
	name := args[0]
	account.CreateNewAccount(name, accounts, privateKeys, node.PublicKeys, accountsFile, encryptionKey)
	balanceManager.SetBalance(name, 100.0) // 初始化账户余额
	fmt.Printf("账户 %s 已创建\n", name)
}

func (node *Node) listAccounts(accounts *[]account.Account) {
	fmt.Println("现有账户:")
	for _, acc := range *accounts {
		fmt.Printf("- %s\n", acc.Name)
	}
}

func (node *Node) handleVerifyBalanceCommand(args []string, balanceManager *account.BalanceManager) {
	// 参数检查，确保不需要输入账户名
	if len(args) != 0 {
		fmt.Println("用法: verify_balance (无需参数)")
		return
	}

	fmt.Println("开始验证所有账户的余额...")

	// 获取所有账户的名称
	allAccounts := balanceManager.GetAllAccounts()
	if len(allAccounts) == 0 {
		fmt.Println("没有找到任何账户")
		return
	}

	// 遍历每个账户进行验证
	for _, accountName := range allAccounts {
		// 根据区块链计算余额
		calculatedBalance := node.Blockchain.ValidateBalance(accountName)

		// 从余额管理器中获取当前余额
		currentBalance, exists := balanceManager.GetBalance(accountName)
		if !exists {
			fmt.Printf("账户 %s 不存在于余额管理器中，跳过\n", accountName)
			continue
		}

		// 检查余额是否一致
		if calculatedBalance == currentBalance {
			fmt.Printf("账户 %s 的余额验证通过: %.2f\n", accountName, currentBalance)
		} else {
			// 更新余额
			fmt.Printf("账户 %s 的余额不一致: 当前余额=%.2f, 计算余额=%.2f\n", accountName, currentBalance, calculatedBalance)
			balanceManager.SetBalance(accountName, calculatedBalance)

			// 异步保存余额
			go func(account string) {
				if err := balanceManager.SaveBalances(balancesFile); err != nil {
					fmt.Printf("保存账户 %s 的余额失败: %v\n", account, err)
				} else {
					fmt.Printf("账户 %s 的余额已成功保存到文件\n", account)
				}
			}(accountName)

			// 广播更新余额
			node.broadcast(RequestTypeUpdateBalance, map[string]interface{}{
				"account":    accountName,
				"newBalance": calculatedBalance,
			})
			fmt.Printf("账户 %s 的余额已更新为 %.2f，并尝试广播\n", accountName, calculatedBalance)
		}
	}

	fmt.Println("所有账户余额验证完成")
}

func (node *Node) exitNode(balanceManager *account.BalanceManager) {
	fmt.Println("保存余额并退出节点...")
	if err := balanceManager.SaveBalances(balancesFile); err != nil {
		fmt.Printf("保存余额失败: %v\n", err)
	}
	os.Exit(0)
}
