package main

import (
	"encoding/gob"
	"log"
	"os"
)

const walletFile = "wallet.dat"

type Wallets struct {
	Wallets map[string]*Wallet
}

func NewWallets() (*Wallets, error) {
	ws := Wallets{}
	ws.Wallets = make(map[string]*Wallet)

	err := ws.LoadFromFile()
	if err != nil {
		// 文件不存在时不报错，返回空钱包集
		return &ws, nil
	}

	return &ws, nil
}

func (ws *Wallets) CreateWallet() string {
	w := NewWallet()
	address := string(w.GetAddress())
	ws.Wallets[address] = w
	return address
}

func (ws *Wallets) GetAddresses() []string {
	var addresses []string
	for addr := range ws.Wallets {
		addresses = append(addresses, addr)
	}
	return addresses
}

func (ws *Wallets) GetWallet(address string) Wallet {
	return *ws.Wallets[address]
}

func (ws *Wallets) LoadFromFile() error {
	if _, err := os.Stat(walletFile); os.IsNotExist(err) {
		return err
	}

	file, err := os.Open(walletFile)
	if err != nil {
		return err
	}
	defer file.Close()

	var wallets Wallets
	decoder := gob.NewDecoder(file)
	err = decoder.Decode(&wallets)
	if err != nil {
		return err
	}

	ws.Wallets = wallets.Wallets
	return nil
}

func (ws *Wallets) SaveToFile() {
	file, err := os.Create(walletFile)
	if err != nil {
		log.Panic(err)
	}
	defer file.Close()

	encoder := gob.NewEncoder(file)
	err = encoder.Encode(ws)
	if err != nil {
		log.Panic(err)
	}
}
