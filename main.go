package main

import "fmt"

func main() {
	bc := NewBlockchain()
	defer bc.db.Close() // 程序结束前关闭数据库

	bc.AddBlock("Send 1 BTC to Alice")
	bc.AddBlock("Send 2 BTC to Bob")
	bc.AddBlock("Send 3 BTC to Charlie")

	it := bc.Iterator()

	for {
		block := it.Next()

		fmt.Println("======================")
		fmt.Printf("Prev. hash: %x\n", block.PrevBlockHash)
		for i, tx := range block.Transactions {
			fmt.Printf(" Transaction %d: %s\n", i, tx.Data)
		}
		fmt.Printf("Hash: %x\n", block.Hash)
		fmt.Printf("Nonce: %d\n", block.Nonce)

		pow := NewProofOfWork(block)
		fmt.Printf("PoW Valid: %v\n", pow.Validate())

		if len(block.PrevBlockHash) == 0 {
			// 遍历到创世区块
			break
		}
	}
}
