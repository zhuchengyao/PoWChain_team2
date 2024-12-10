package account

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

// AccountBalance 代表单个账户的余额和锁
type AccountBalance struct {
	Balance float64
	Mu      sync.RWMutex
}

// BalanceManager 管理账户余额
type BalanceManager struct {
	balances map[string]*AccountBalance
	mu       sync.RWMutex
	writeMu  sync.Mutex
}

// NewBalanceManager 初始化余额管理器
func NewBalanceManager() *BalanceManager {
	return &BalanceManager{
		balances: make(map[string]*AccountBalance),
	}
}

// SetBalance 设置账户余额
func (bm *BalanceManager) SetBalance(account string, balance float64) {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	if bm.balances[account] == nil {
		bm.balances[account] = &AccountBalance{}
	}

	ab := bm.balances[account]
	ab.Mu.Lock()
	defer ab.Mu.Unlock()
	ab.Balance = balance
}

// GetBalance 获取账户余额
func (bm *BalanceManager) GetBalance(account string) (float64, bool) {
	bm.mu.RLock()
	ab, exists := bm.balances[account]
	bm.mu.RUnlock()

	if !exists {
		return 0, false
	}

	ab.Mu.RLock()
	defer ab.Mu.RUnlock()
	return ab.Balance, true
}

// AddBalance 增加账户余额
func (bm *BalanceManager) AddBalance(account string, amount float64, filePath string) {
	bm.mu.Lock()
	if bm.balances[account] == nil {
		bm.balances[account] = &AccountBalance{}
	}
	ab := bm.balances[account]
	bm.mu.Unlock()

	ab.Mu.Lock()
	ab.Balance += amount
	ab.Mu.Unlock()

	go bm.saveBalancesAsync(filePath)
}

// DeductBalance 扣减账户余额
func (bm *BalanceManager) DeductBalance(account string, amount float64, filePath string) bool {
	bm.mu.RLock()
	ab, exists := bm.balances[account]
	bm.mu.RUnlock()

	if !exists {
		fmt.Printf("账户 %s 不存在\n", account)
		return false
	}

	ab.Mu.Lock()
	defer ab.Mu.Unlock()

	if ab.Balance >= amount {
		ab.Balance -= amount
		go bm.saveBalancesAsync(filePath)
		return true
	}

	fmt.Println("余额不足")
	return false
}

// saveBalancesAsync 异步保存余额到文件
func (bm *BalanceManager) saveBalancesAsync(filePath string) {
	err := bm.SaveBalances(filePath)
	if err != nil {
		fmt.Printf("保存余额失败: %v\n", err)
	} else {
		fmt.Println("余额已成功保存")
	}
}

// SaveBalances 保存所有账户余额到文件
func (bm *BalanceManager) SaveBalances(filePath string) error {
	bm.writeMu.Lock()
	defer bm.writeMu.Unlock()

	bm.mu.RLock()
	defer bm.mu.RUnlock()

	data := make(map[string]float64)
	for account, ab := range bm.balances {
		ab.Mu.RLock()
		data[account] = ab.Balance
		ab.Mu.RUnlock()
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化失败: %w", err)
	}

	if err = os.WriteFile(filePath, jsonData, 0644); err != nil {
		return fmt.Errorf("写入文件失败: %w", err)
	}

	return nil
}

// LoadBalances 从文件加载账户余额
func (bm *BalanceManager) LoadBalances(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("余额文件不存在，初始化为空")
			return nil
		}
		return fmt.Errorf("读取文件失败: %w", err)
	}

	var balances map[string]float64
	if err = json.Unmarshal(data, &balances); err != nil {
		return fmt.Errorf("解析文件失败: %w", err)
	}

	bm.mu.Lock()
	defer bm.mu.Unlock()

	for account, balance := range balances {
		bm.balances[account] = &AccountBalance{
			Balance: balance,
		}
	}

	return nil
}
