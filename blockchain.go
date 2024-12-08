package main

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
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

func NewBlockchain(minerAddress, nodeID string) *Blockchain {
	dbFile := fmt.Sprintf("blockchain_%s.db", nodeID)
	db, err := bbolt.Open(dbFile, 0600, nil)
	if err != nil {
		log.Panic(err)
	}

	var tip []byte

	err = db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		if b == nil {
			genesis := NewGenesisBlock(minerAddress)
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

func NewGenesisBlock(toAddress string) *Block {
	coinbase := NewCoinbaseTx(toAddress, "Genesis Block")
	return NewBlock([]*Transaction{coinbase}, []byte{})
}

func (bc *Blockchain) AddBlock(transactions []*Transaction) *Block {
	var lastHash []byte

	err := bc.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		lastHash = b.Get([]byte(latestKey))
		return nil
	})
	if err != nil {
		log.Panic(err)
	}

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

	return newBlock
}

func (bc *Blockchain) Iterator() *BlockchainIterator {
	return &BlockchainIterator{currentHash: bc.tip, db: bc.db}
}

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

// MineBlock 模拟挖矿新块
func (bc *Blockchain) MineBlock(transactions []*Transaction) *Block {
	return bc.AddBlock(transactions)
}

// 查找未花费输出的交易列表
func (bc *Blockchain) FindUnspentTransactions(pubKeyHash []byte) []Transaction {
	var unspentTXs []Transaction
	spentTXOs := make(map[string][]int)

	it := bc.Iterator()

	for {
		block := it.Next()

		for _, tx := range block.Transactions {
			txID := hex.EncodeToString(tx.ID)
		Outputs:
			for outIdx, out := range tx.Vout {
				if spentTXOs[txID] != nil {
					for _, spentOut := range spentTXOs[txID] {
						if spentOut == outIdx {
							continue Outputs
						}
					}
				}
				if string(out.PubKeyHash) == string(pubKeyHash) {
					unspentTXs = append(unspentTXs, *tx)
				}
			}

			if !tx.IsCoinbase() {
				for _, in := range tx.Vin {
					if string(in.PubKey) == string(pubKeyHash) {
						inTxID := hex.EncodeToString(in.Txid)
						spentTXOs[inTxID] = append(spentTXOs[inTxID], in.Vout)
					}
				}
			}
		}

		if len(block.PrevBlockHash) == 0 {
			break
		}
	}

	return unspentTXs
}

func (bc *Blockchain) FindUTXO(pubKeyHash []byte) []TXOutput {
	var UTXOs []TXOutput
	unspentTransactions := bc.FindUnspentTransactions(pubKeyHash)

	for _, tx := range unspentTransactions {
		for _, out := range tx.Vout {
			if string(out.PubKeyHash) == string(pubKeyHash) {
				UTXOs = append(UTXOs, out)
			}
		}
	}
	return UTXOs
}

func (bc *Blockchain) FindSpendableOutputs(pubKeyHash []byte, amount int) (int, map[string][]int) {
	unspentOutputs := make(map[string][]int)
	unspentTXs := bc.FindUnspentTransactions(pubKeyHash)
	accumulated := 0

Work:
	for _, tx := range unspentTXs {
		txID := hex.EncodeToString(tx.ID)
		for outIdx, out := range tx.Vout {
			if string(out.PubKeyHash) == string(pubKeyHash) {
				accumulated += out.Value
				unspentOutputs[txID] = append(unspentOutputs[txID], outIdx)
				if accumulated >= amount {
					break Work
				}
			}
		}
	}
	return accumulated, unspentOutputs
}

func (bc *Blockchain) FindTransaction(ID []byte) Transaction {
	it := bc.Iterator()

	for {
		block := it.Next()
		for _, tx := range block.Transactions {
			if bytes.Equal(tx.ID, ID) {
				return *tx
			}
		}
		if len(block.PrevBlockHash) == 0 {
			break
		}
	}
	log.Panic("Transaction not found")
	return Transaction{}
}

func (bc *Blockchain) SignTransaction(tx *Transaction, privKey ecdsa.PrivateKey) {
	prevTXs := bc.FindPreviousTransactions(tx)
	tx.Sign(privKey, prevTXs)
}

func (bc *Blockchain) VerifyTransaction(tx *Transaction) bool {
	if tx.IsCoinbase() {
		return true
	}
	prevTXs := bc.FindPreviousTransactions(tx)
	return tx.Verify(prevTXs)
}

func (bc *Blockchain) FindPreviousTransactions(tx *Transaction) map[string]Transaction {
	prevTXs := make(map[string]Transaction)
	for _, vin := range tx.Vin {
		prevTx := bc.FindTransaction(vin.Txid)
		prevTXs[string(prevTx.ID)] = prevTx
	}
	return prevTXs
}

// Send 一笔交易（from给to转账amount）
func (bc *Blockchain) Send(from, to string, amount int, wallets *Wallets) {
	w := wallets.GetWallet(from)
	tx := NewUTXOTransaction(from, to, amount, bc)
	if tx == nil {
		return
	}
	coinbase := NewCoinbaseTx(from, "")

	// 从wallet中获取ecdsa.PrivateKey类型的私钥
	privKey := w.getPrivateKey()

	bc.SignTransaction(tx, privKey)
	newBlock := bc.MineBlock([]*Transaction{coinbase, tx})
	fmt.Printf("New block mined with hash: %x\n", newBlock.Hash)
}

func (bc *Blockchain) GetBlockHashes() [][]byte {
	var blocks [][]byte
	it := bc.Iterator()

	for {
		block := it.Next()
		blocks = append(blocks, block.Hash)

		if len(block.PrevBlockHash) == 0 {
			break
		}
	}

	return blocks
}

func (bc *Blockchain) GetBlock(blockHash []byte) (Block, error) {
	var block *Block

	err := bc.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		data := b.Get(blockHash)
		if data == nil {
			return fmt.Errorf("no block found")
		}

		block = DeserializeBlock(data)
		return nil
	})
	if err != nil {
		return Block{}, err
	}

	return *block, nil
}

func (bc *Blockchain) GetBestHeight() int {
	// 返回区块链最高区块的高度
	// 在区块中添加Height字段或通过遍历计数确定高度
	// 简化处理：遍历整条链计数（不高效，但示例足够）
	height := 0
	it := bc.Iterator()
	for {
		block := it.Next()
		height++
		if len(block.PrevBlockHash) == 0 {
			break
		}
	}
	return height - 1 // 减1是因为计数从1开始
}
