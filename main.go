package main

import (
	"flag"
	"fmt"
	"gamechain/account"
	"os"
	"strings"
)

// 入口函数
func main() {
	// 加载账户
	accounts, privateKeys, publicKeys, err := account.LoadAccounts(accountsFile, encryptionKey)
	if err != nil {
		fmt.Printf("加载账户失败: %v\n", err)
		os.Exit(1)
	}

	// 初始化余额管理器
	// 初始化余额管理器
	balanceManager := account.NewBalanceManager()

	// 从文件加载余额数据
	err = balanceManager.LoadBalances(balancesFile)
	if err != nil {
		fmt.Printf("加载余额失败: %v\n", err)
		os.Exit(1)
	}

	// 为新账户设置默认余额
	for _, acc := range accounts {
		if _, exists := balanceManager.GetBalance(acc.Name); !exists {
			balanceManager.SetBalance(acc.Name, 100.0) // 设置默认余额
		}
	}

	// 加载区块链并创建创世区块（如果尚未存在）
	blockchain = initializeBlockchain(blockchainFile, 2)

	// 解析命令行参数
	address := flag.String("address", "localhost:8080", "节点地址")
	peers := flag.String("peers", "", "逗号分隔的其他节点地址")
	flag.Parse()

	peerNodes := []string{}
	if *peers != "" {
		peerNodes = strings.Split(*peers, ",")
	}

	// 初始化节点
	node := Node{
		Address:         *address,
		Blockchain:      blockchain,
		TransactionPool: blockchain.TransactionPool,
		PeerNodes:       peerNodes,
		PublicKeys:      publicKeys,
		BalanceManager:  balanceManager, // 传递 BalanceManager
	}

	// 启动节点
	go node.Start()

	// 交互式命令行
	node.RunInteractive(privateKeys, &accounts, accountsFile, transactionPoolFile, blockchainFile, encryptionKey, balanceManager)
}
