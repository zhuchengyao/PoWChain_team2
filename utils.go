package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

// 计算交易的 Merkle 树根
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

// 解析节点地址
func ParsePeers(peers string) []string {
	if peers == "" {
		return []string{}
	}
	return strings.Split(peers, ",")
}
