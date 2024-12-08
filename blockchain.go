package main

import (
	"log"

	"go.etcd.io/bbolt"
)

const dbFile = "blockchain.db"
const blocksBucket = "blocks"
const latestKey = "l"

type Blockchain struct {
	tip []byte
	db  *bbolt.DB
}

// NewBlockchain 创建一条区块链，如果已有数据库则加载最新区块，否则创建创世区块
func NewBlockchain() *Blockchain {
	var tip []byte

	db, err := bbolt.Open(dbFile, 0600, nil)
	if err != nil {
		log.Panic(err)
	}

	err = db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		if b == nil {
			// 数据库不存在区块信息，需要创建创世区块
			genesis := NewGenesisBlock()
			b, err := tx.CreateBucket([]byte(blocksBucket))
			if err != nil {
				return err
			}
			err = b.Put(genesis.Hash, genesis.Serialize())
			if err != nil {
				return err
			}
			err = b.Put([]byte(latestKey), genesis.Hash)
			if err != nil {
				return err
			}
			tip = genesis.Hash
		} else {
			// 已有区块链数据，读取tip
			tip = b.Get([]byte(latestKey))
		}
		return nil
	})
	if err != nil {
		log.Panic(err)
	}

	bc := Blockchain{tip: tip, db: db}
	return &bc
}

// AddBlock 添加区块并写入数据库
func (bc *Blockchain) AddBlock(data string) {
	var lastHash []byte

	err := bc.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		lastHash = b.Get([]byte(latestKey))
		return nil
	})
	if err != nil {
		log.Panic(err)
	}

	transactions := []*Transaction{NewSimpleTransaction(data)}
	newBlock := NewBlock(transactions, lastHash)

	err = bc.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		err := b.Put(newBlock.Hash, newBlock.Serialize())
		if err != nil {
			return err
		}

		err = b.Put([]byte(latestKey), newBlock.Hash)
		if err != nil {
			return err
		}
		bc.tip = newBlock.Hash
		return nil
	})
	if err != nil {
		log.Panic(err)
	}
}

// Iterator 返回区块链迭代器，用于遍历链
func (bc *Blockchain) Iterator() *BlockchainIterator {
	return &BlockchainIterator{currentHash: bc.tip, db: bc.db}
}
