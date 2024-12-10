# **GameChain**

**GameChain** 是一个简单的区块链系统，支持账户管理、交易处理和分布式网络同步功能。该系统旨在帮助开发者理解区块链的基本原理和实现。

---

## **功能概览**

- **账户管理**：
  - 支持创建新账户。
  - 查询账户余额。
  - 验证账户余额与区块链记录的一致性。

- **交易处理**：
  - 支持创建并广播交易。
  - 验证交易签名。
  - 使用交易池存储未打包交易。

- **区块链**：
  - 创建新区块并完成工作量证明（Proof of Work）。
  - 支持分布式节点间的区块链同步。

- **网络通信**：
  - 节点之间通过 TCP 通信。
  - 支持同步区块链、新交易和新区块的广播。

- **挖矿奖励**：
  - 挖矿成功后，矿工账户获得固定奖励。

---

## **目录结构**

```plaintext
.
├── account
│   ├── account.go           # 账户管理模块
│   └── account_test.go      # 账户管理测试
├── block.go             # 区块相关逻辑
├── blockchain.go        # 区块链主逻辑
├── constants.go         # 项目常量定义
├── main.go              # 入口文件
├── network.go           # 网络通信相关逻辑
├── transaction.go       # 交易处理模块
├── utils.go             # 工具函数
├── README.md            # 项目说明文件
├── accounts.json        # 账户数据文件
├── blockchain.json      # 区块链数据文件
├── transaction_pool.json # 交易池文件
└── balances.json        # 账户余额数据
```

## **运行环境**

### **依赖**

- **Go**: 版本 1.18 及以上
- **操作系统**：支持 Windows、Linux 和 macOS

### **安装**

1. 确保已安装 Go 环境：
   ```bash
   go version

2. 克隆项目代码：
   ```bash
   git clone https://github.com/zhuchengyao/PoWChain_team2.git
   cd PoWChain_team2
   ```

3. 安装依赖（如有第三方库）。

   如果项目中使用了第三方依赖，可以通过以下命令安装：
   ```bash
   go mod tidy
   ```
## **运行指南**

### **启动节点**

运行以下命令启动节点：
```bash
go run . --address <your-node-address> --peers <comma-separated-peer-nodes>
```

示例，开启3个节点，确保对应端口未被占用:
```bash
go run . --address localhost:8080 --peers localhost:8081,localhost:8082

go run . --address localhost:8081 --peers localhost:8080,localhost:8082

go run . --address localhost:8082 --peers localhost:8080,localhost:8081
```

### **交互式命令行**

启动节点后，进入交互模式。以下是可用的命令：

| 命令                | 功能描述                                              |
|---------------------|-----------------------------------------------------|
| `mine <miner>`      | 挖矿并生成新区块，指定矿工账户                      |
| `tx <from> <to> <amount>` | 创建并广播交易                                   |
| `sync`              | 从其他节点同步区块链                                |
| `balance <account>` | 查询账户余额                                        |
| `create_account <name>` | 创建新账户                                       |
| `list_accounts`     | 列出所有账户                                        |
| `print`             | 打印区块链状态                                      |
| `verify_balance`    | 验证所有账户余额是否与区块链记录一致                |
| `exit`              | 退出节点                                            |

### **示例操作**

1. 创建账户：
   ```bash
   create_account Alice
   create_account Bob
   ```

2. 挖矿：
   ```bash
   mine Alice
   ```

3. 创建交易：
   ```bash
   tx Alice Bob 10
   ```

4. 查看账户余额：
   ```bash
   balance Alice
   ```

5. 验证区块一致性：
   ```bash
   verify_balance
   ```

6. 退出节点：
   ```bash
   exit
   ```

## **文件说明**

| 文件名                  | 描述                                   |
|-------------------------|----------------------------------------|
| `accounts.json`         | 存储账户信息                         |
| `blockchain.json`       | 存储区块链数据                       |
| `transaction_pool.json` | 存储未确认交易                       |
| `balances.json`         | 存储账户余额                         |

---

## **扩展与改进**

- **加密**：
  - 使用更强的加密算法保护交易数据。
  - 实现私钥的安全存储和加密。

- **优化同步机制**：
  - 支持更高效的链同步（如轻节点）。
  - 引入分叉链处理机制。

- **图形界面**：
  - 开发简单的 Web 或桌面应用，用于管理账户和交易。

- **智能合约支持**：
  - 添加智能合约执行功能。

---

## **参与贡献**

欢迎提交 Issue 和 Pull Request 以改进项目。请确保您的代码通过所有测试并符合代码风格。

---

## **许可**

本项目使用 [MIT 许可](LICENSE)。

---

## **开发者信息**

- **作者**：ChengyaoZhu from Team 2  
- **联系方式**：zhucy23@m.fudan.edu.cn  
- **GitHub**：[@zhuchengyao](https://github.com/zhuchengyao)