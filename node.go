package main

import (
	"bufio"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"net"
	"os"
)

// 定义节点结构
type Node struct {
	Address    string                      // 节点地址
	Blockchain *Blockchain                 // 节点的区块链实例
	PeerNodes  []string                    // 已连接的其他节点地址
	PublicKeys map[string]*ecdsa.PublicKey // 存储用户的公钥
}

const (
	RequestTypeSync           = "sync"
	RequestTypeNewBlock       = "new_block"
	RequestTypeNewTransaction = "new_transaction"
)

// 广播交易给其他节点
func (node *Node) BroadcastTransaction(tx Transaction) {
	for _, peer := range node.PeerNodes {
		conn, err := net.Dial("tcp", peer)
		if err != nil {
			fmt.Printf("无法连接到节点 %s: %v\n", peer, err)
			continue
		}
		defer conn.Close()

		request := map[string]interface{}{
			"type":        "new_transaction",
			"transaction": tx,
		}
		data, _ := json.Marshal(request)
		data = append(data, '\n') // 添加换行符，确保接收方能正确解析
		conn.Write(data)
		fmt.Printf("向节点 %s 广播交易: %+v\n", peer, tx)
	}
}

// 启动节点服务器，监听连接
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

// 处理节点的连接请求
func (node *Node) HandleConnection(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)

	// 读取消息
	message, err := reader.ReadString('\n')
	if err != nil {
		if err.Error() == "EOF" {
			fmt.Println("连接关闭: 对端已发送完数据")
		} else {
			fmt.Printf("读取消息错误: %v\n", err)
		}
		return
	}

	var request map[string]interface{}
	err = json.Unmarshal([]byte(message), &request)
	if err != nil {
		fmt.Printf("解析消息错误: %v\n", err)
		return
	}

	// 处理请求类型
	switch request["type"] {
	case "new_transaction":
		// 接收并验证新交易
		var tx Transaction
		txData, _ := json.Marshal(request["transaction"])
		json.Unmarshal(txData, &tx)
		node.HandleNewTransaction(tx)
	case "new_block":
		// 接收并验证新块
		node.ReceiveBlock(request)
	case "sync":
		// 返回本地区块链
		node.SendBlockchain(conn)
	default:
		fmt.Printf("未知请求类型: %s\n", request["type"])
	}
}

func (node *Node) HandleNewTransaction(tx Transaction) {
	filePath := fmt.Sprintf("%s_blockchain.json", node.Address) // 动态生成文件路径
	if node.Blockchain.AddTransactionToPool(tx, node.PublicKeys, filePath) {
		fmt.Printf("新交易已添加到交易池: %+v\n", tx)
	} else {
		fmt.Printf("新交易验证失败: %+v\n", tx)
	}
}

// 发送本地区块链给其他节点
func (node *Node) SendBlockchain(conn net.Conn) {
	data, _ := json.Marshal(node.Blockchain)
	conn.Write(data)
}

// 接收其他节点的新块
func (node *Node) ReceiveBlock(request map[string]interface{}) {
	var block Block
	blockData, _ := json.Marshal(request["block"])
	json.Unmarshal(blockData, &block)

	// 验证区块并添加到链
	lastBlock := node.Blockchain.Blocks[len(node.Blockchain.Blocks)-1]
	if block.Header.PreviousHash == lastBlock.Hash && block.Hash == block.CalculateHash() {
		node.Blockchain.Blocks = append(node.Blockchain.Blocks, block)
		fmt.Printf("接收到新块: #%d\n", block.Header.Index)
	} else {
		fmt.Printf("新块无效，拒绝添加: %v\n", block)
	}
}

// 广播新块给所有连接的节点
func (node *Node) BroadcastBlock(block Block) {
	for _, peer := range node.PeerNodes {
		fmt.Printf("向节点 %s 广播新区块 #%d\n", peer, block.Header.Index)
		conn, err := net.Dial("tcp", peer)
		if err != nil {
			fmt.Printf("无法连接到节点 %s: %v\n", peer, err)
			continue
		}
		defer conn.Close()

		request := map[string]interface{}{
			"type":  "new_block",
			"block": block,
		}
		data, _ := json.Marshal(request)
		data = append(data, '\n') // 添加换行符，确保接收方能正确解析
		conn.Write(data)
	}
}
