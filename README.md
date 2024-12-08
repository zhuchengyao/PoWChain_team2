# PoWChain_team2
##
目前已经完成：
1. 区块数据结构和基础区块链逻辑
2. PoW挖矿机制
3. 交易结构与Coinbase交易
4. 钱包公私钥生成和签名验证
5. 基本的P2P通信与节点间握手（version消息发送与响应）、区块与交易的同步逻辑
6. 数据持久化与多节点独立的数据库存储

尝试运行：
```
go run . --port 3001 --miner minerAddress
go run . --port 3002 --seeds "127.0.0.1:3001" --miner anotherMiner_2
go run . --port 3003 --seeds "localhost:3001" --miner anotherMiner_3
```
注意，必须先执行3001 genesis，然后再执行3002和3003。