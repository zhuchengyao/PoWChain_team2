package account

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/elliptic"
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
	PrivateKey string `json:"private_key"`
	PublicKey  string `json:"public_key"`
}

// 加载账户
func LoadAccounts(filePath, encryptionKey string) ([]Account, map[string]*ecdsa.PrivateKey, map[string]*ecdsa.PublicKey, error) {
	var accounts []Account
	privateKeys := make(map[string]*ecdsa.PrivateKey)
	publicKeys := make(map[string]*ecdsa.PublicKey)

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("读取账户文件失败: %v", err)
	}

	err = json.Unmarshal(data, &accounts)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("解析账户文件失败: %v", err)
	}

	for _, acc := range accounts {
		privBytes, err := decrypt(acc.PrivateKey, encryptionKey)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("解密账户私钥失败: %v", err)
		}
		privateKey, err := x509.ParseECPrivateKey(privBytes)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("解析私钥失败: %v", err)
		}
		pubKeyBytes, _ := hex.DecodeString(acc.PublicKey)
		publicKey, err := x509.ParsePKIXPublicKey(pubKeyBytes)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("解析公钥失败: %v", err)
		}
		privateKeys[acc.Name] = privateKey
		publicKeys[acc.Name] = publicKey.(*ecdsa.PublicKey)
	}

	return accounts, privateKeys, publicKeys, nil
}

// 保存账户
func SaveAccounts(accounts []Account, filePath string) error {
	data, err := json.MarshalIndent(accounts, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化账户失败: %v", err)
	}
	err = os.WriteFile(filePath, data, 0644)
	if err != nil {
		return fmt.Errorf("保存账户文件失败: %v", err)
	}
	return nil
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
	return hash[:16]
}

func CreateNewAccount(
	name string,
	accounts *[]Account,
	privateKeys map[string]*ecdsa.PrivateKey,
	publicKeys map[string]*ecdsa.PublicKey,
	filePath, encryptionKey string,
) {
	// 检查账户是否已存在
	for _, acc := range *accounts {
		if acc.Name == name {
			fmt.Printf("账户 %s 已存在\n", name)
			return
		}
	}

	// 生成新的密钥对
	privateKey, publicKey := GenerateKeyPair()
	privateKeyBytes, _ := x509.MarshalECPrivateKey(privateKey)
	encryptedPrivateKey, _ := encrypt(privateKeyBytes, encryptionKey)
	publicKeyBytes, _ := x509.MarshalPKIXPublicKey(publicKey)

	// 创建新账户并添加到列表
	newAccount := Account{
		Name:       name,
		PrivateKey: encryptedPrivateKey,
		PublicKey:  hex.EncodeToString(publicKeyBytes),
	}
	*accounts = append(*accounts, newAccount)
	privateKeys[name] = privateKey
	publicKeys[name] = publicKey

	// 保存账户到文件
	err := SaveAccounts(*accounts, filePath)
	if err != nil {
		fmt.Printf("保存账户失败: %v\n", err)
		return
	}
	fmt.Printf("账户 %s 已成功创建并保存\n", name)
}

func GenerateKeyPair() (*ecdsa.PrivateKey, *ecdsa.PublicKey) {
	privateKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	return privateKey, &privateKey.PublicKey
}
