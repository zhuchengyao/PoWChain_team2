package account

import (
	"os"
	"testing"
)

func TestSaveBalances(t *testing.T) {
	// 临时文件路径
	tempFilePath := "test_balances.json"
	defer func() {
		// 测试完成后删除临时文件
		os.Remove(tempFilePath)
	}()

	// 初始化 BalanceManager 并设置初始数据
	bm := NewBalanceManager()
	bm.SetBalance("account1", 100.0)
	bm.SetBalance("account2", 200.0)

	// 调用 SaveBalances 方法
	err := bm.SaveBalances(tempFilePath)
	if err != nil {
		t.Fatalf("SaveBalances 失败: %v", err)
	}

	// 验证文件是否已创建
	_, err = os.Stat(tempFilePath)
	if os.IsNotExist(err) {
		t.Fatalf("保存的文件不存在: %v", err)
	}

	// 验证文件内容是否正确
	data, err := os.ReadFile(tempFilePath)
	if err != nil {
		t.Fatalf("读取文件失败: %v", err)
	}

	expected := `{
  "account1": 100,
  "account2": 200
}`
	if string(data) != expected {
		t.Errorf("文件内容与预期不符:\n实际: %s\n期望: %s", string(data), expected)
	}
}
