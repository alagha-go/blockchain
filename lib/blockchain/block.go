package blockchain

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"log"
)

type Block struct {
	Hash							[]byte
	Transactions					[]*Transaction
	PrevHash						[]byte
	Nonce							int
}

// func (b *Block) DeriveHash(){
// 	info := bytes.Join([][]byte{b.Data, b.PrevHash}, []byte{})
// 	hash := sha256.Sum256(info)
// 	b.Hash = hash[:]
// }

func CreateBlock(txs []*Transaction, prevHash []byte) *Block {
	block := &Block{[]byte{}, txs, prevHash, 0}
	pow := NewProof(block)
	nonce, hash := pow.Run()

	block.Hash = hash

	block.Nonce = nonce

	return block
}

func Genesis(coinbase *Transaction) *Block{
	return CreateBlock([]*Transaction{coinbase}, []byte{})
}


func (block *Block) HashTransactions() []byte{
	var txHashes [][]byte
	var txHash [32]byte

	for _, tx := range block.Transactions {
		txHashes = append(txHashes, tx.ID)
	}

	txHash = sha256.Sum256(bytes.Join(txHashes, []byte{}))

	return txHash[:]
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