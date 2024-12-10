package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
)

// Account 账户结构
type Account struct {
	Name       string `json:"name"`
	PrivateKey string `json:"private_key"` // 加密存储的私钥
	PublicKey  string `json:"public_key"`
}

// 加密数据
func encrypt(data []byte, key string) (string, error) {
	block, err := aes.NewCipher([]byte(hashKey(key)))
	if err != nil {
		return "", err
	}
	nonce := make([]byte, 12)
	_, err = rand.Read(nonce)
	if err != nil {
		return "", err
	}
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	encrypted := aesGCM.Seal(nil, nonce, data, nil)
	return hex.EncodeToString(append(nonce, encrypted...)), nil
}

// 解密数据
func decrypt(data string, key string) ([]byte, error) {
	encrypted, err := hex.DecodeString(data)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher([]byte(hashKey(key)))
	if err != nil {
		return nil, err
	}
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := encrypted[:12]
	encryptedData := encrypted[12:]
	return aesGCM.Open(nil, nonce, encryptedData, nil)
}

// 哈希密钥
func hashKey(key string) []byte {
	hash := sha256.Sum256([]byte(key))
	return hash[:16] // AES-128 密钥长度
}

// 加载账户
func LoadAccounts(filePath, encryptionKey string) ([]Account, map[string]*ecdsa.PrivateKey, map[string]*ecdsa.PublicKey) {
	var accounts []Account
	privateKeys := make(map[string]*ecdsa.PrivateKey)
	publicKeys := make(map[string]*ecdsa.PublicKey)

	data, err := os.ReadFile(filePath)
	if err == nil {
		json.Unmarshal(data, &accounts)
		for _, acc := range accounts {
			privBytes, _ := decrypt(acc.PrivateKey, encryptionKey)
			privateKey, _ := x509.ParseECPrivateKey(privBytes)
			pubKeyBytes, _ := hex.DecodeString(acc.PublicKey)
			publicKey, _ := x509.ParsePKIXPublicKey(pubKeyBytes)
			privateKeys[acc.Name] = privateKey
			publicKeys[acc.Name] = publicKey.(*ecdsa.PublicKey)

			// 初始化余额映射
			if _, exists := balanceMap[acc.Name]; !exists {
				balanceMap[acc.Name] = 100.0 // 默认初始余额
			}
		}
	}

	return accounts, privateKeys, publicKeys
}

// 保存账户
func SaveAccounts(accounts []Account, filePath string) {
	data, _ := json.MarshalIndent(accounts, "", "  ")
	os.WriteFile(filePath, data, 0644)
}

var balanceMap = make(map[string]float64) // 全局余额映射

func CreateNewAccount(name string, accounts *[]Account, privateKeys map[string]*ecdsa.PrivateKey, publicKeys map[string]*ecdsa.PublicKey, filePath, encryptionKey string) {
	for _, acc := range *accounts {
		if acc.Name == name {
			fmt.Printf("账户 %s 已存在\n", name)
			return
		}
	}

	privateKey, publicKey := GenerateKeyPair()
	privateKeyBytes, _ := x509.MarshalECPrivateKey(privateKey)
	encryptedPrivateKey, _ := encrypt(privateKeyBytes, encryptionKey)
	publicKeyBytes, _ := x509.MarshalPKIXPublicKey(publicKey)

	// 添加新账户
	newAccount := Account{
		Name:       name,
		PrivateKey: encryptedPrivateKey,
		PublicKey:  hex.EncodeToString(publicKeyBytes),
	}
	*accounts = append(*accounts, newAccount)
	privateKeys[name] = privateKey
	publicKeys[name] = publicKey

	// 初始化账户的余额
	balanceMap[name] = 100.0 // 设置初始余额为 100.0

	SaveAccounts(*accounts, filePath)
	fmt.Printf("账户 %s 已创建，初始余额为 100.0\n", name)
}

func InitializeAccountBalance(account string, amount float64) {
	// 创建一个“系统账户”（假设名称为"System"）向新账户转账
	tx := Transaction{
		Sender:   "System", // 系统账户
		Receiver: account,
		Amount:   amount,
	}
	// 直接添加到第一个区块
	if len(blockchain.Blocks) > 0 {
		blockchain.Blocks[0].Transactions = append(blockchain.Blocks[0].Transactions, tx)
		fmt.Printf("系统已向账户 %s 分配 %.2f 初始余额\n", account, amount)
	} else {
		fmt.Println("区块链尚未初始化，无法分配初始余额")
	}
}
