package main

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
)

// 交易输入
type TXInput struct {
	Txid      []byte
	Vout      int
	Signature []byte
	PubKey    []byte
}

// 交易输出
type TXOutput struct {
	Value      int
	PubKeyHash []byte
}

type Transaction struct {
	ID   []byte
	Vin  []TXInput
	Vout []TXOutput
}

// 判断是否是coinbase交易
func (tx *Transaction) IsCoinbase() bool {
	return len(tx.Vin) == 1 && len(tx.Vin[0].Txid) == 0 && tx.Vin[0].Vout == -1
}

func (tx *Transaction) SetID() {
	var encoded bytes.Buffer
	enc := gob.NewEncoder(&encoded)
	err := enc.Encode(tx)
	if err != nil {
		panic(err)
	}
	hash := sha256.Sum256(encoded.Bytes())
	tx.ID = hash[:]
}

// 创建coinbase交易
func NewCoinbaseTx(to, data string) *Transaction {
	if data == "" {
		data = fmt.Sprintf("Reward to '%s'", to)
	}
	txin := TXInput{Txid: []byte{}, Vout: -1, Signature: nil, PubKey: []byte(data)}
	txout := TXOutput{Value: 50, PubKeyHash: []byte(to)}
	tx := Transaction{Vin: []TXInput{txin}, Vout: []TXOutput{txout}}
	tx.SetID()
	return &tx
}

// 创建UTXO交易
func NewUTXOTransaction(from, to string, amount int, bc *Blockchain) *Transaction {
	acc, validOutputs := bc.FindSpendableOutputs([]byte(from), amount)
	if acc < amount {
		fmt.Println("ERROR: Not enough funds")
		return nil
	}

	var inputs []TXInput
	var outputs []TXOutput

	for txid, outs := range validOutputs {
		// txid 是 hex.EncodeToString(tx.ID) 得到的十六进制字符串
		txIDBytes, err := hex.DecodeString(txid)
		if err != nil {
			log.Panic(err)
		}

		for _, out := range outs {
			input := TXInput{Txid: txIDBytes, Vout: out, Signature: nil, PubKey: []byte(from)}
			inputs = append(inputs, input)
		}
	}

	outputs = append(outputs, TXOutput{Value: amount, PubKeyHash: []byte(to)})
	if acc > amount {
		outputs = append(outputs, TXOutput{Value: acc - amount, PubKeyHash: []byte(from)})
	}

	tx := Transaction{Vin: inputs, Vout: outputs}
	tx.SetID()
	return &tx
}

// 对交易签名
func (tx *Transaction) Sign(privKey ecdsa.PrivateKey, prevTXs map[string]Transaction) {
	if tx.IsCoinbase() {
		return
	}

	for _, vin := range tx.Vin {
		if prevTXs[string(vin.Txid)].ID == nil {
			log.Panic("ERROR: Previous transaction is not correct")
		}
	}

	txCopy := tx.TrimmedCopy()

	for inId, vin := range txCopy.Vin {
		prevTx := prevTXs[string(vin.Txid)]
		txCopy.Vin[inId].Signature = nil
		txCopy.Vin[inId].PubKey = prevTx.Vout[vin.Vout].PubKeyHash
		txCopy.ID = txCopy.Hash()
		txCopy.Vin[inId].PubKey = nil

		r, s, err := ecdsa.Sign(rand.Reader, &privKey, txCopy.ID)
		if err != nil {
			log.Panic(err)
		}
		signature := append(r.Bytes(), s.Bytes()...)
		tx.Vin[inId].Signature = signature
	}
}

// 验证交易签名
func (tx *Transaction) Verify(prevTXs map[string]Transaction) bool {
	if tx.IsCoinbase() {
		return true
	}

	for _, vin := range tx.Vin {
		if prevTXs[string(vin.Txid)].ID == nil {
			log.Panic("Previous transaction is not correct")
		}
	}

	txCopy := tx.TrimmedCopy()
	curve := elliptic.P256()

	for inId, vin := range tx.Vin {
		prevTx := prevTXs[string(vin.Txid)]
		txCopy.Vin[inId].Signature = nil
		txCopy.Vin[inId].PubKey = prevTx.Vout[vin.Vout].PubKeyHash
		txCopy.ID = txCopy.Hash()
		txCopy.Vin[inId].PubKey = nil

		r := big.Int{}
		s := big.Int{}
		sigLen := len(vin.Signature)
		r.SetBytes(vin.Signature[:(sigLen / 2)])
		s.SetBytes(vin.Signature[(sigLen / 2):])

		x := big.Int{}
		y := big.Int{}
		keyLen := len(vin.PubKey)
		x.SetBytes(vin.PubKey[:(keyLen / 2)])
		y.SetBytes(vin.PubKey[(keyLen / 2):])

		rawPubKey := ecdsa.PublicKey{Curve: curve, X: &x, Y: &y}

		if !ecdsa.Verify(&rawPubKey, txCopy.ID, &r, &s) {
			return false
		}
	}
	return true
}

func (tx *Transaction) TrimmedCopy() Transaction {
	var inputs []TXInput
	var outputs []TXOutput

	for _, vin := range tx.Vin {
		inputs = append(inputs, TXInput{Txid: vin.Txid, Vout: vin.Vout})
	}

	for _, vout := range tx.Vout {
		outputs = append(outputs, TXOutput{Value: vout.Value, PubKeyHash: vout.PubKeyHash})
	}

	txCopy := Transaction{ID: tx.ID, Vin: inputs, Vout: outputs}
	return txCopy
}

func (tx *Transaction) Hash() []byte {
	var hash [32]byte

	var encoded bytes.Buffer
	enc := gob.NewEncoder(&encoded)
	err := enc.Encode(tx)
	if err != nil {
		panic(err)
	}
	hash = sha256.Sum256(encoded.Bytes())
	return hash[:]
}

func (tx *Transaction) Serialize() []byte {
	var encoded bytes.Buffer
	enc := gob.NewEncoder(&encoded)
	err := enc.Encode(tx)
	if err != nil {
		panic(err)
	}
	return encoded.Bytes()
}

func DeserializeTransaction(data []byte) *Transaction {
	var tx Transaction
	dec := gob.NewDecoder(bytes.NewReader(data))
	err := dec.Decode(&tx)
	if err != nil {
		panic(err)
	}
	return &tx
}
