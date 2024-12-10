package main

import (
	"bufio"
	"crypto/ecdsa"
	"fmt"
	"gamechain/account"
	"os"
	"strings"
)

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
	// fmt.Println("  sync - 从其他节点同步区块链")
	fmt.Println("  balance [account] - 查询账户余额")
	fmt.Println("  create_account [name] - 创建新账户")
	fmt.Println("  list_accounts - 列出所有账户")
	fmt.Println("  print - 打印区块链状态")
	fmt.Println("  verify_balance [account] - 验证账户余额是否与区块链记录一致")
	fmt.Println("  exit - 退出程序")
}
