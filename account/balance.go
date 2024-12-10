package account

import "sync"

// BalanceManager 管理账户余额
type BalanceManager struct {
	balances map[string]float64
	mu       sync.RWMutex
}

// NewBalanceManager 初始化余额管理器
func NewBalanceManager() *BalanceManager {
	return &BalanceManager{balances: make(map[string]float64)}
}

// SetBalance 设置账户余额
func (bm *BalanceManager) SetBalance(account string, balance float64) {
	bm.mu.Lock()
	defer bm.mu.Unlock()
	bm.balances[account] = balance
}

// GetBalance 获取账户余额
func (bm *BalanceManager) GetBalance(account string) (float64, bool) {
	bm.mu.RLock()
	defer bm.mu.RUnlock()
	balance, exists := bm.balances[account]
	return balance, exists
}

// AddBalance 增加账户余额
func (bm *BalanceManager) AddBalance(account string, amount float64) {
	bm.mu.Lock()
	defer bm.mu.Unlock()
	bm.balances[account] += amount
}

// DeductBalance 扣减账户余额
func (bm *BalanceManager) DeductBalance(account string, amount float64) bool {
	bm.mu.Lock()
	defer bm.mu.Unlock()
	if bm.balances[account] >= amount {
		bm.balances[account] -= amount
		return true
	}
	return false
}
