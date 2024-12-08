package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"net"
)

const protocol = "tcp"

var nodeAddress string
var KnownNodes = []string{}
var blocksInTransit = [][]byte{}

type Message struct {
	Command string
	Payload []byte
}

type Version struct {
	Version    int
	BestHeight int
	AddrFrom   string
}

type GetBlocks struct {
	AddrFrom string
}

type Inv struct {
	AddrFrom string
	Type     string
	Items    [][]byte
}

type GetData struct {
	AddrFrom string
	Type     string
	ID       []byte
}

type BlockData struct {
	AddrFrom string
	Block    []byte
}

type TxData struct {
	AddrFrom    string
	Transaction []byte
}

func commandToBytes(cmd string) []byte {
	var bytes [12]byte
	for i, c := range cmd {
		bytes[i] = byte(c)
	}
	return bytes[:]
}

func bytesToCommand(bytes []byte) string {
	var cmd []byte
	for _, b := range bytes {
		if b != 0x00 {
			cmd = append(cmd, b)
		}
	}
	return string(cmd)
}

func sendData(addr string, data []byte) {
	fmt.Printf("[sendData] Attempting to connect to %s\n", addr)
	conn, err := net.Dial(protocol, addr)
	if err != nil {
		fmt.Printf("[sendData] Cannot connect to %s: %v\n", addr, err)
		var updatedNodes []string
		for _, node := range KnownNodes {
			if node != addr {
				updatedNodes = append(updatedNodes, node)
			}
		}
		KnownNodes = updatedNodes
		return
	}
	defer conn.Close()
	fmt.Printf("[sendData] Connected to %s, now sending data...\n", addr)

	_, err = io.Copy(conn, bytes.NewReader(data))
	if err != nil {
		fmt.Printf("[sendData] Error writing data to %s: %v\n", addr, err)
		return
	}

	fmt.Printf("[sendData] Successfully sent data to %s\n", addr)
}

func sendVersion(toAddress string, bc *Blockchain) {
	bestHeight := bc.GetBestHeight()
	v := Version{Version: 1, BestHeight: bestHeight, AddrFrom: nodeAddress}
	payload := gobEncode(v)
	request := append(commandToBytes("version"), payload...)

	fmt.Printf("[sendVersion] To: %s, BestHeight: %d\n", toAddress, bestHeight)

	sendData(toAddress, request)
}

func sendGetBlocks(toAddress string) {
	payload := gobEncode(GetBlocks{AddrFrom: nodeAddress})
	request := append(commandToBytes("getblocks"), payload...)
	sendData(toAddress, request)
}

func sendInv(toAddress string, kind string, items [][]byte) {
	inv := Inv{AddrFrom: nodeAddress, Type: kind, Items: items}
	payload := gobEncode(inv)
	request := append(commandToBytes("inv"), payload...)
	sendData(toAddress, request)
}

func sendGetData(toAddress string, kind string, id []byte) {
	payload := gobEncode(GetData{AddrFrom: nodeAddress, Type: kind, ID: id})
	request := append(commandToBytes("getdata"), payload...)
	sendData(toAddress, request)
}

func sendBlock(toAddress string, b *Block) {
	data := BlockData{AddrFrom: nodeAddress, Block: b.Serialize()}
	payload := gobEncode(data)
	request := append(commandToBytes("block"), payload...)
	sendData(toAddress, request)
}

func sendTx(toAddress string, tx *Transaction) {
	data := TxData{AddrFrom: nodeAddress, Transaction: tx.Serialize()}
	payload := gobEncode(data)
	request := append(commandToBytes("tx"), payload...)
	sendData(toAddress, request)
}

func handleConnection(conn net.Conn, bc *Blockchain) {
	request, err := io.ReadAll(conn)
	if err != nil {
		log.Panic(err)
	}
	command := bytesToCommand(request[:12])

	switch command {
	case "version":
		handleVersion(request[12:], bc)
	case "getblocks":
		handleGetBlocks(request[12:], bc)
	case "inv":
		handleInv(request[12:], bc)
	case "getdata":
		handleGetData(request[12:], bc)
	case "block":
		handleBlock(request[12:], bc)
	case "tx":
		handleTx(request[12:], bc)
	default:
		fmt.Println("Unknown command!")
	}

	conn.Close()
}

func handleVersion(payload []byte, bc *Blockchain) {
	var buff bytes.Buffer
	var version Version
	buff.Write(payload)
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&version)
	if err != nil {
		log.Panic(err)
	}

	fmt.Printf("[handleVersion] Received version from %s. Version: %d, BestHeight: %d\n",
		version.AddrFrom, version.Version, version.BestHeight)

	bestHeight := bc.GetBestHeight()
	foreignHeight := version.BestHeight

	if foreignHeight > bestHeight {
		fmt.Printf("[handleVersion] Foreign node (%s) has higher height (%d > %d). Sending getblocks.\n",
			version.AddrFrom, foreignHeight, bestHeight)
		sendGetBlocks(version.AddrFrom)
	} else if foreignHeight < bestHeight {
		fmt.Printf("[handleVersion] Our chain is longer. Sending version to %s.\n", version.AddrFrom)
		sendVersion(version.AddrFrom, bc)
	}

	if !nodeIsKnown(version.AddrFrom) {
		KnownNodes = append(KnownNodes, version.AddrFrom)
		fmt.Printf("[handleVersion] Added %s to known nodes.\n", version.AddrFrom)
	}
}

func handleGetBlocks(payload []byte, bc *Blockchain) {
	var buff bytes.Buffer
	var getblocks GetBlocks
	buff.Write(payload)
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&getblocks)
	if err != nil {
		log.Panic(err)
	}

	fmt.Printf("[handleGetBlocks] %s requested block hashes.\n", getblocks.AddrFrom)
	blocks := bc.GetBlockHashes()
	sendInv(getblocks.AddrFrom, "block", blocks)
}

func handleInv(payload []byte, bc *Blockchain) {
	var buff bytes.Buffer
	var inv Inv
	buff.Write(payload)
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&inv)
	if err != nil {
		log.Panic(err)
	}

	fmt.Printf("[handleInv] Received inv from %s. Type: %s, Items: %d\n", inv.AddrFrom, inv.Type, len(inv.Items))

	if inv.Type == "block" && len(inv.Items) > 0 {
		blocksInTransit = inv.Items
		blockHash := inv.Items[0]
		fmt.Printf("[handleInv] Requesting block data for first hash: %x\n", blockHash)
		sendGetData(inv.AddrFrom, "block", blockHash)
		blocksInTransit = blocksInTransit[1:]
	}
}

func handleGetData(payload []byte, bc *Blockchain) {
	var buff bytes.Buffer
	var getData GetData
	buff.Write(payload)
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&getData)
	if err != nil {
		log.Panic(err)
	}

	fmt.Printf("[handleGetData] %s requested %s with ID %x\n", getData.AddrFrom, getData.Type, getData.ID)

	if getData.Type == "block" {
		block, err := bc.GetBlock(getData.ID)
		if err != nil {
			fmt.Printf("[handleGetData] Block not found: %x\n", getData.ID)
			return
		}
		sendBlock(getData.AddrFrom, &block)
	}
}

func handleBlock(payload []byte, bc *Blockchain) {
	var buff bytes.Buffer
	var blockData BlockData
	buff.Write(payload)
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&blockData)
	if err != nil {
		log.Panic(err)
	}

	block := DeserializeBlock(blockData.Block)
	fmt.Printf("[handleBlock] Received block from %s. Hash: %x\n", blockData.AddrFrom, block.Hash)
	bc.AddBlock(block.Transactions)

	if len(blocksInTransit) > 0 {
		blockHash := blocksInTransit[0]
		fmt.Printf("[handleBlock] Requesting next block data: %x\n", blockHash)
		sendGetData(blockData.AddrFrom, "block", blockHash)
		blocksInTransit = blocksInTransit[1:]
	} else {
		// UTXOSet := UTXOSet{bc}
		// UTXOSet.Update(block)
	}
}

func handleTx(payload []byte, bc *Blockchain) {
	var buff bytes.Buffer
	var txData TxData
	buff.Write(payload)
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&txData)
	if err != nil {
		log.Panic(err)
	}

	tx := DeserializeTransaction(txData.Transaction)
	fmt.Printf("[handleTx] Received tx %x from %s\n", tx.ID, txData.AddrFrom)
	// 在这里可以加入将交易放入交易池的逻辑
}

func StartServer(nodeID string, minerAddress string) {
	nodeAddress = fmt.Sprintf("127.0.0.1:%s", nodeID)
	fmt.Printf("StartServer called with nodeID=%s, minerAddress=%s\n", nodeID, minerAddress)
	fmt.Printf("nodeAddress='%s'\n", nodeAddress)
	fmt.Printf("KnownNodes=%v\n", KnownNodes)

	if len(KnownNodes) > 0 {
		fmt.Printf("KnownNodes[0]='%s'\n", KnownNodes[0])
	} else {
		fmt.Println("KnownNodes is empty at this point.")
	}

	// 在这里监听端口
	ln, err := net.Listen(protocol, nodeAddress)
	if err != nil {
		log.Panic(err)
	}
	defer ln.Close()

	// 使用nodeID创建区块链实例，从而使用独立的db文件
	bc := NewBlockchain(minerAddress, nodeID)
	fmt.Println("Blockchain initialized.")

	if len(KnownNodes) == 0 {
		KnownNodes = append(KnownNodes, nodeAddress)
		fmt.Printf("This is the genesis node. KnownNodes now: %v\n", KnownNodes)
	}

	fmt.Printf("Before if condition: nodeAddress='%s'\n", nodeAddress)
	if len(KnownNodes) > 0 {
		fmt.Printf("KnownNodes[0]='%s'\n", KnownNodes[0])
	}
	fmt.Println("Checking condition: nodeAddress != KnownNodes[0]")

	// 对比条件：如果本节点不是Genesis节点，就向已知节点发送version
	if len(KnownNodes) > 0 && nodeAddress != KnownNodes[0] {
		fmt.Printf("Condition met: nodeAddress (%s) != KnownNodes[0](%s), sending version\n", nodeAddress, KnownNodes[0])
		sendVersion(KnownNodes[0], bc)
	} else {
		if len(KnownNodes) == 0 {
			fmt.Println("KnownNodes is empty, cannot compare, this likely is genesis node.")
		} else {
			if nodeAddress == KnownNodes[0] {
				fmt.Printf("nodeAddress == KnownNodes[0], so this node is the genesis node (no sendVersion). nodeAddress=%s KnownNodes[0]=%s\n", nodeAddress, KnownNodes[0])
			} else {
				fmt.Println("Condition not met for unknown reasons.")
			}
		}
	}

	fmt.Printf("Node %s is ready and listening...\n", nodeID)

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Panic(err)
		}
		fmt.Printf("Accepted connection from %s\n", conn.RemoteAddr())
		go handleConnection(conn, bc)
	}
}

func gobEncode(data interface{}) []byte {
	var buff bytes.Buffer
	enc := gob.NewEncoder(&buff)
	err := enc.Encode(data)
	if err != nil {
		log.Panic(err)
	}
	return buff.Bytes()
}

func nodeIsKnown(addr string) bool {
	for _, node := range KnownNodes {
		if node == addr {
			return true
		}
	}
	return false
}
