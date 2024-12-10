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
	"time"
)

type Node struct {
	Address         string
	Blockchain      *Blockchain
	TransactionPool []Transaction
	PeerNodes       []string
	PublicKeys      map[string]*ecdsa.PublicKey
	BalanceManager  *account.BalanceManager
}

func (node *Node) BroadcastTransaction(tx Transaction) {
	node.broadcast(RequestTypeNewTransaction, map[string]interface{}{"transaction": tx})
}

func (node *Node) BroadcastBlock(block Block) {
	node.broadcast(RequestTypeNewBlock, map[string]interface{}{"block": block})
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
		fmt.Println("收到同步请求，返回区块链数据")
		node.SendBlockchain(conn)
	case RequestTypeUpdateBalance:
		node.updateBalance(request)
	default:
		fmt.Printf("未知请求类型: %v\n", request["type"])
	}
}

func (node *Node) updateBalance(request map[string]interface{}) {
	accountName, ok := request["account"].(string)
	if !ok {
		fmt.Println("账户名称解析失败")
		return
	}
	newBalance := request["newBalance"].(float64)
	node.BalanceManager.SetBalance(accountName, newBalance)
	if err := node.BalanceManager.SaveBalances(balancesFile); err != nil {
		fmt.Printf("保存余额失败: %v\n", err)
	}
	fmt.Printf("账户 %s 的余额已更新为 %.2f\n", accountName, newBalance)
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

func connectWithTimeout(address string, timeout time.Duration) (net.Conn, error) {
	d := net.Dialer{Timeout: timeout}
	return d.Dial("tcp", address)
}

func (node *Node) SendBlockchain(conn net.Conn) {
	data, _ := json.Marshal(node.Blockchain)
	conn.Write(data)
}

func mapToStruct(data interface{}, target interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return json.Unmarshal(jsonData, target)
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
