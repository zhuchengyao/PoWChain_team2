package main

import (
	"log"

	"go.etcd.io/bbolt"
)

type BlockchainIterator struct {
	currentHash []byte
	db          *bbolt.DB
}

func (it *BlockchainIterator) Next() *Block {
	var block *Block

	err := it.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		encodedBlock := b.Get(it.currentHash)
		block = DeserializeBlock(encodedBlock)
		return nil
	})
	if err != nil {
		log.Panic(err)
	}

	it.currentHash = block.PrevBlockHash
	return block
}
