package network

import (
	"bytes"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"runtime"
	"syscall"
	"tensor/lib/blockchain"

	"github.com/vrecan/death/v3"
)

const (
	protocol = "tcp"
	version = 1
	commandLength = 12
)

var (
	nodeAddress string
	minerAddress string
	KnownNodes  = []string{"localhost:3000"}
	blocksInTransit [][]byte
	memoryPool = make(map[string]blockchain.Transaction)
)

type Address struct {
	AddressList						[]string
}

type Block struct {
	AddressFrom						string
	Block							[]byte
}

type GetBlocks struct {
	AddressFrom						string
}

type GetData struct {
	AddressFrom 					string
	Type 							string
	ID								[]byte
}

type Inv struct {
	AddressFrom 					string
	Type 							string
	Items							[][]byte
}

type Tx struct {
	AddressFrom 					string
	Transaction 					[]byte
}

type Version struct {
	Version							int
	BestHeight						int
	AddressFrom						string
}


func CmdToBytes(cmd string) []byte {
	var bytes [commandLength]byte

	for index, command := range cmd {
		bytes[index] = byte(command)
	}

	return bytes[:]
}


func BytesToCmd(bytes []byte) string {
	var cmd []byte

	for _, b := range bytes {
		if b != 0x0 {
			cmd = append(cmd, b)
		}
	}

	return string(cmd)
}


func RequestBlocks() {
	for _, node := range KnownNodes{
		SendGetBlocks(node)
	}
}



func ExtractCmd(request []byte) []byte {
	return request[:commandLength]
}


func SendAddress(address string) {
	nodes := Address{KnownNodes}

	nodes.AddressList = append(nodes.AddressList, nodeAddress)
	payload := GobEncode(nodes)
	request := append(CmdToBytes("address"), payload...)

	SendData(address, request)
}


func SendBlock(address string, block *blockchain.Block) {
	data := Block{nodeAddress, block.Serialize()}
	payload := GobEncode(data)
	request := append(CmdToBytes("block"), payload...)

	SendData(address, request)
}


func SendInv(address, kind string, items [][]byte) {
	inventory := Inv{nodeAddress, kind, items}
	payload := GobEncode(inventory)
	request := append(CmdToBytes("inv"), payload...)

	SendData(address, request)
}


func SendTx(address string, txn *blockchain.Transaction) {
	data := Tx{nodeAddress, txn.Serialize()}
	payload := GobEncode(data)
	request := append(CmdToBytes("tx"), payload...)

	SendData(address, request)
}


func SendVersion(address string, chain *blockchain.BlockChain) {
	bestHeight := chain.GetBestHeight()
	payload := GobEncode(Version{version, bestHeight, nodeAddress})
	request := append(CmdToBytes("version"), payload...)

	SendData(address, request)
}


func SendGetBlocks(address string) {
	payload := GobEncode(GetBlocks{nodeAddress})
	request := append(CmdToBytes("getblocks"), payload...)

	SendData(address, request)
}


func SendGetData(address, kind string, id []byte) {
	payload := GobEncode(GetData{nodeAddress, kind, id})
	request := append(CmdToBytes("getdata"), payload...)

	SendData(address, request)
}



func HandleAddress(request []byte){
	var buff bytes.Buffer
	var payload Address

	buff .Write(request[commandLength:])
	err := gob.NewDecoder(&buff).Decode(&payload)
	HandleError(err)

	KnownNodes = append(KnownNodes, payload.AddressList...)
	fmt.Printf("there are %d known nodes\n", len(KnownNodes))
	RequestBlocks()
}




func HandleInv(request []byte, chain *blockchain.BlockChain) {
	var buff bytes.Buffer
	var payload Inv

	buff .Write(request[commandLength:])
	err := gob.NewDecoder(&buff).Decode(&payload)
	HandleError(err)

	fmt.Printf("Received inventory with %d %s\n", len(payload.Items), payload.Type)

	if payload.Type == "block" {
		blocksInTransit = payload.Items

		blockHash := payload.Items[0]
		SendGetData(payload.AddressFrom, "block", blockHash)

		newInTransit := [][]byte{}

		for _, block := range blocksInTransit {
			if bytes.Compare(block, blockHash) != 0 {
				newInTransit = append(newInTransit, block)
			}
		}
		blocksInTransit = newInTransit
	}

	if payload.Type == "tx" {
		txID := payload.Items[0]

		if memoryPool[hex.EncodeToString(txID)].ID == nil {
			SendGetData(payload.AddressFrom, "tx", txID)
		}
	}
}





func HandleBlock(request []byte, chain *blockchain.BlockChain) {
	var buff bytes.Buffer
	var payload Block

	buff .Write(request[commandLength:])
	err := gob.NewDecoder(&buff).Decode(&payload)
	HandleError(err)

	blockdata := payload.Block
	block := blockchain.Deserialize(blockdata)

	fmt.Println("Received a new block!")
	chain.AddBlock(block) 

	fmt.Printf("Added block %x\n", block.Hash)

	if len(blocksInTransit) > 0 {
		blockHash := blocksInTransit[0]
		SendGetData(payload.AddressFrom, "block", blockHash)

		blocksInTransit = blocksInTransit[1:]
	}else {
		UTXOSet := blockchain.UTXOSet{chain}
		UTXOSet.Reindex()
	}
}


func HandleGetBlocks(request []byte, chain *blockchain.BlockChain) {
	var buff bytes.Buffer
	var payload GetBlocks

	buff .Write(request[commandLength:])
	err := gob.NewDecoder(&buff).Decode(&payload)
	HandleError(err)

	blocks := chain.GetBlockHashes()
	SendInv(payload.AddressFrom, "block", blocks)
}

func HandleGetData(request []byte, chain *blockchain.BlockChain) {
	var buff bytes.Buffer
	var payload GetData

	buff .Write(request[commandLength:])
	err := gob.NewDecoder(&buff).Decode(&payload)
	HandleError(err)

	if payload.Type == "block" {
		block, err := chain.GetBlock([]byte(payload.ID))
		if err != nil {
			return
		}

		SendBlock(payload.AddressFrom, &block)
	}

	if payload.Type == "tx" {
		txID := hex.EncodeToString(payload.ID)
		tx := memoryPool[txID]

		SendTx(payload.AddressFrom, &tx)
	}
}


func HandleVersion(request []byte, chain *blockchain.BlockChain) {
	var buff bytes.Buffer
	var payload Version

	buff .Write(request[commandLength:])
	err := gob.NewDecoder(&buff).Decode(&payload)
	HandleError(err)

	bestHeight := chain.GetBestHeight()
	otherHeight := payload.BestHeight

	if bestHeight < otherHeight {
		SendGetBlocks(payload.AddressFrom)
	}else if bestHeight > otherHeight {
		SendVersion(payload.AddressFrom, chain)
	}

	if !NodeIsKnown(payload.AddressFrom) {
		KnownNodes = append(KnownNodes, payload.AddressFrom)
	}
}


func HandleTx(request []byte, chain *blockchain.BlockChain) {
	var buff bytes.Buffer
	var payload Tx

	buff .Write(request[commandLength:])
	err := gob.NewDecoder(&buff).Decode(&payload)
	HandleError(err)

	txData := payload.Transaction
	tx := blockchain.DeserializeTransaction(txData)
	memoryPool[hex.EncodeToString(tx.ID)] = tx

	fmt.Printf("%s, %d", nodeAddress, len(memoryPool))

	if nodeAddress == KnownNodes[0] {
		for _, node := range KnownNodes {
			if node != nodeAddress && node != payload.AddressFrom {
				SendInv(node, "tx", [][]byte{tx.ID})
			}
		}
	}else {
		if len(memoryPool) >= 2 && len(minerAddress) > 0 {
			MineTx(chain)
		}
	}
}


func MineTx(chain *blockchain.BlockChain) {
	var txs []*blockchain.Transaction

	for id := range memoryPool {
		fmt.Printf("tx: %s\n", memoryPool[id].ID)
		tx := memoryPool[id]
		if chain.VerifyTransaction(&tx){
			txs = append(txs, &tx)
		}
	}

	if len(txs) == 0 {
		fmt.Println("All Transactions are invalid")
		return
	}

	cbtx := blockchain.CoinbaseTx(minerAddress, "")
	txs = append(txs, cbtx)

	newBlock := chain.MineBlock(txs)
	UTXOSet := blockchain.UTXOSet{chain}
	UTXOSet.Reindex()

	fmt.Println("New Block mined")

	for _, tx := range txs {
		txID := hex.EncodeToString(tx.ID)
		delete(memoryPool, txID)
	}

	for _, node := range KnownNodes {
		if node != nodeAddress {
			SendInv(node, "block", [][]byte{newBlock.Hash})
		}
	}

	if len(memoryPool) > 0 {
		MineTx(chain)
	}
}



func SendData(address string, data []byte) {
	var updatedNodes []string
	conn, err := net.Dial(protocol, address)
	
	if err != nil {
		fmt.Printf("%s is not available\n", address)

		for _, node := range KnownNodes {
			if node != address {
				updatedNodes = append(updatedNodes, node)
			}
		}
		KnownNodes = updatedNodes

		return
	}

	defer conn.Close()

	_, err = io.Copy(conn, bytes.NewReader(data))
	HandleError(err)
}



func HandleConnection(conn net.Conn, chain *blockchain.BlockChain) {
	req, err := ioutil.ReadAll(conn)
	HandleError(err)
	defer conn.Close()

	command := BytesToCmd(req[:commandLength])
	fmt.Printf("Received %s command\n", command)

	switch command {
		case "address":
			HandleAddress(req)
		case "block":
			HandleBlock(req, chain)
		case "inv":
			HandleInv(req, chain)
		case "getblocks":
			HandleGetBlocks(req, chain)
		case "getdata":
			HandleGetData(req, chain)
		case "tx":
			HandleTx(req, chain)
		case "version":
			HandleVersion(req, chain)
		default:
			fmt.Println("Unknown command")
	}
}



func StartServer(nodeID, mineraddress string) {
	nodeAddress = fmt.Sprintf("localhost:%s", nodeID)
	minerAddress = mineraddress

	ln, err := net.Listen(protocol, nodeAddress)
	HandleError(err)

	defer ln.Close()

	chain := blockchain.ContinueBlockChain(nodeID)
	defer chain.Database.Close()
	go CloseDB(chain)

	if nodeAddress != KnownNodes[0] {
		SendVersion(KnownNodes[0], chain)
	}

	for {
		conn, err := ln.Accept()
		HandleError(err)
		go HandleConnection(conn, chain)
	}
}




func GobEncode(data interface{}) []byte {
	var buff bytes.Buffer

	err := gob.NewEncoder(&buff).Encode(data)
	HandleError(err)

	return buff.Bytes()
}



func CloseDB(chain *blockchain.BlockChain) {
	d := death.NewDeath(syscall.SIGINT, syscall.SIGTERM, os.Interrupt)

	d.WaitForDeathWithFunc(func(){
		defer os.Exit(1)
		defer runtime.Goexit()
		chain.Database.Close()
	})
}


func NodeIsKnown(address string) bool {
	for _, node := range KnownNodes {
		if node == address {
			return true
		}
	}

	return false
}


func HandleError(err error) {
	if err != nil {
		log.Panic(err)
	}
}