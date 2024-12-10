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

// Account 定义账户结构
type Account struct {
	Name       string `json:"name"`
	PrivateKey string `json:"private_key"`
	PublicKey  string `json:"public_key"`
}

// LoadAccounts 加载账户列表并解密密钥
func LoadAccounts(filePath, encryptionKey string) ([]Account, map[string]*ecdsa.PrivateKey, map[string]*ecdsa.PublicKey, error) {
	var accounts []Account
	privateKeys := make(map[string]*ecdsa.PrivateKey)
	publicKeys := make(map[string]*ecdsa.PublicKey)

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("读取账户文件失败: %w", err)
	}

	if err := json.Unmarshal(data, &accounts); err != nil {
		return nil, nil, nil, fmt.Errorf("解析账户文件失败: %w", err)
	}

	for _, acc := range accounts {
		privBytes, err := decrypt(acc.PrivateKey, encryptionKey)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("解密账户私钥失败: %w", err)
		}

		privateKey, err := x509.ParseECPrivateKey(privBytes)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("解析私钥失败: %w", err)
		}

		pubKeyBytes, err := hex.DecodeString(acc.PublicKey)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("解析公钥失败: %w", err)
		}

		publicKey, err := x509.ParsePKIXPublicKey(pubKeyBytes)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("解析公钥失败: %w", err)
		}

		privateKeys[acc.Name] = privateKey
		publicKeys[acc.Name] = publicKey.(*ecdsa.PublicKey)
	}

	return accounts, privateKeys, publicKeys, nil
}

// SaveAccounts 保存账户列表到文件
func SaveAccounts(accounts []Account, filePath string) error {
	data, err := json.MarshalIndent(accounts, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化账户失败: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("保存账户文件失败: %w", err)
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
	if _, err := rand.Read(nonce); err != nil {
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

	nonce, encryptedData := encrypted[:12], encrypted[12:]
	return aesGCM.Open(nil, nonce, encryptedData, nil)
}

// 哈希密钥生成AES密钥
func hashKey(key string) []byte {
	hash := sha256.Sum256([]byte(key))
	return hash[:16]
}

// CreateNewAccount 创建新账户并保存
func CreateNewAccount(
	name string,
	accounts *[]Account,
	privateKeys map[string]*ecdsa.PrivateKey,
	publicKeys map[string]*ecdsa.PublicKey,
	filePath, encryptionKey string,
) error {
	// 检查账户是否已存在
	for _, acc := range *accounts {
		if acc.Name == name {
			return fmt.Errorf("账户 %s 已存在", name)
		}
	}

	// 生成新的密钥对
	privateKey, publicKey := GenerateKeyPair()
	privateKeyBytes, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return fmt.Errorf("私钥序列化失败: %w", err)
	}

	encryptedPrivateKey, err := encrypt(privateKeyBytes, encryptionKey)
	if err != nil {
		return fmt.Errorf("私钥加密失败: %w", err)
	}

	publicKeyBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return fmt.Errorf("公钥序列化失败: %w", err)
	}

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
	if err := SaveAccounts(*accounts, filePath); err != nil {
		return fmt.Errorf("保存账户失败: %w", err)
	}
	fmt.Printf("账户 %s 已成功创建并保存\n", name)
	return nil
}

// GenerateKeyPair 生成新的ECDSA密钥对
func GenerateKeyPair() (*ecdsa.PrivateKey, *ecdsa.PublicKey) {
	privateKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	return privateKey, &privateKey.PublicKey
}

func (bm *BalanceManager) GetAllAccounts() []string {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	var accounts []string
	for account := range bm.balances {
		accounts = append(accounts, account)
	}
	return accounts
}
