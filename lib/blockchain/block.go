package blockchain

import (
	"bytes"
	"encoding/gob"
	"log"
	"time"
)

type Block struct {
	TimeStamp						int64
	Hash							[]byte
	Transactions					[]*Transaction
	PrevHash						[]byte
	Nonce							int
	Height							int
}

func CreateBlock(txs []*Transaction, prevHash []byte, height int) *Block {
	block := &Block{time.Now().Unix(), []byte{}, txs, prevHash, 0, height}
	pow := NewProof(block)
	nonce, hash := pow.Run()

	block.Hash = hash

	block.Nonce = nonce

	return block
}

func Genesis(coinbase *Transaction) *Block{
	return CreateBlock([]*Transaction{coinbase}, []byte{}, 0)
}


func (block *Block) HashTransactions() []byte{
	var txHashes [][]byte

	for _, tx := range block.Transactions {
		txHashes = append(txHashes, tx.Serialize())
	}

	tree := NewMarkleTree(txHashes)

	return tree.RootNode.Data
}


func (block *Block) Serialize() []byte{
	var res bytes.Buffer
	encoder := gob.NewEncoder(&res)

	err := encoder.Encode(block)

	HandleError(err)

	return res.Bytes()
}

func Deserialize(data []byte) *Block{
	var block Block

	decoder := gob.NewDecoder(bytes.NewReader(data))

	err := decoder.Decode(&block)

	HandleError(err)

	return &block
}

func HandleError(err error) {
	if err != nil {
		log.Panic(err)
	}
}