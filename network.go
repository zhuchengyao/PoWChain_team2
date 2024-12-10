package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"time"
)

const (
	RequestTypeSync           = "sync"
	RequestTypeNewBlock       = "new_block"
	RequestTypeNewTransaction = "new_transaction"
	RequestTypeUpdateBalance  = "update_balance"
)

// 广播消息
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

// 发送请求到节点
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

// 区块链同步逻辑
func (node *Node) SyncBlockchain() {
	fmt.Println("开始同步区块链...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	done := make(chan bool)
	go func() {
		for _, peer := range node.PeerNodes {
			conn, err := connectWithTimeout(peer, 5*time.Second)
			if err != nil {
				fmt.Printf("无法连接到节点 %s: %v\n", peer, err)
				continue
			}
			defer conn.Close()

			request := map[string]interface{}{
				"type": "sync",
			}
			jsonData, _ := json.Marshal(request)
			conn.Write(jsonData)

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
		done <- true
	}()

	select {
	case <-ctx.Done():
		fmt.Println("同步操作超时")
	case <-done:
		fmt.Println("同步完成")
	}
}
