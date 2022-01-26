package blockchain

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"runtime"

	"github.com/dgraph-io/badger"
)

const (
	DBpath = "./DB/blocks"
	DBFile = "./DB/blocks/MANIFEST"
	GenesisData = "First Transaction from Genesis"
)

type BlockChain struct {
	LastHash					[]byte
	Database					*badger.DB
}

type BlockChainIterator struct {
	CurrentHash					[]byte
	Database					*badger.DB
}

func DBexists() bool {
	if _, err := os.Stat(DBFile); os.IsNotExist(err){
		return false
	}
	return true
}


func InitBlockChain(address string) *BlockChain{
	var lastHash []byte

	if DBexists() {
		fmt.Println("Blockchain already exists")
		runtime.Goexit()
	}


	opts := badger.DefaultOptions(DBpath)
	opts.Dir = DBpath
	opts.ValueDir = DBpath

	db, err := badger.Open(opts)
	HandleError(err)

	err = db.Update(func(txn *badger.Txn) error{
			cbtx := CoinbaseTx(address, GenesisData)
			genesis := Genesis(cbtx)
			fmt.Println("genesis Proved")
			err := txn.Set(genesis.Hash, genesis.Serialize())
			HandleError(err)
			err = txn.Set([]byte("lh"), genesis.Hash)
			HandleError(err)
			lastHash = genesis.Hash
			return err
	})

	HandleError(err)

	blockchain := BlockChain{LastHash: lastHash, Database: db}

	return &blockchain
}


func ContinueBlockChain(address string) *BlockChain{
	var lastHash []byte

	if DBexists() == false{
		fmt.Println("Blockchain Does Not Exist")
		runtime.Goexit()
		// return nil
	}


	opts := badger.DefaultOptions(DBpath)
	opts.Dir = DBpath
	opts.ValueDir = DBpath

	db, err := badger.Open(opts)
	HandleError(err)

	err = db.Update(func(txn *badger.Txn) error{
			item, err := txn.Get([]byte("lh"))
			HandleError(err)
			lastHash, err = item.ValueCopy([]byte("lh"))
			return err
	})

	HandleError(err)

	blockchain := BlockChain{LastHash: lastHash, Database: db}

	return &blockchain
}


func (chain *BlockChain) AddBlock(transactions []*Transaction) *Block {
	var lastHash []byte


	err := chain.Database.View(func(txn *badger.Txn) error{
		item, err := txn.Get([]byte("lh"))
		HandleError(err)
		lastHash, err = item.ValueCopy([]byte("lh"))
		return err
	})
	HandleError(err)

	newBlock := CreateBlock(transactions, lastHash)

	err = chain.Database.Update(func(txn *badger.Txn) error{
		err := txn.Set(newBlock.Hash, newBlock.Serialize())
		HandleError(err)
		err = txn.Set([]byte("lh"), newBlock.Hash)
		chain.LastHash = newBlock.Hash
		return err
	})
	HandleError(err)

	return newBlock
}


func (chain *BlockChain) Iterator() *BlockChainIterator {
	iter := &BlockChainIterator{chain.LastHash, chain.Database}

	return iter
}


func (iter *BlockChainIterator) Next() *Block {
	var block *Block

	err := iter.Database.View(func(txn *badger.Txn) error {
		item, err := txn.Get(iter.CurrentHash)
		HandleError(err)

		encodedBlock, err := item.ValueCopy([]byte("lh"))
		
		block = Deserialize(encodedBlock)

		return err
	})
	HandleError(err)

	iter.CurrentHash = block.PrevHash

	return block
}


func (chain *BlockChain) FindUTXO() map[string]TxOutputs {
	UTXO := make(map[string]TxOutputs)
	spentTxos := make(map[string][]int)

	iter := chain.Iterator()

	for {
		block := iter.Next()

		for _, tx := range block.Transactions {
			txID := hex.EncodeToString(tx.ID)

			outputs:
			for index, out := range tx.Outputs{
				if spentTxos[txID] != nil {
					for _, spentOut := range spentTxos[txID] {
						if spentOut == index {
							continue outputs
						}
					}
				}
				outs := UTXO[txID]
				outs.Outputs = append(outs.Outputs, out)
				UTXO[txID] = outs
			}
			if  !tx.IsCoinbase(){
				for _, input := range tx.Inputs{
					inputTxID := hex.EncodeToString(input.ID)
					spentTxos[inputTxID] = append(spentTxos[inputTxID], input.Out)
				}
			}
		}

		if len(block.PrevHash) == 0 {
			break
		}

	}

	return UTXO
}




func (bc *BlockChain) FindTransaction(ID []byte) (Transaction, error) {
	iter := bc.Iterator()

	for {
		block := iter.Next()

		for _, tx := range block.Transactions {
			if bytes.Compare(tx.ID, ID) == 0 {
				return *tx, nil
			}
		}

		if len(block.PrevHash) == 0 {
			break
		}
	}

	return  Transaction{}, errors.New("Transaction does not exist")
}


func (bc *BlockChain) SignTransaction(tx *Transaction, privKey ecdsa.PrivateKey) {
	prevTxs := make(map[string]Transaction)

	for _, input := range tx.Inputs {
		prevTX, err := bc.FindTransaction(input.ID)
		HandleError(err)
		prevTxs[hex.EncodeToString(prevTX.ID)] = prevTX
	}

	tx.Sign(privKey, prevTxs)
}


func (bc *BlockChain) VerifyTransaction(tx Transaction) bool {
	prevTxs := make(map[string]Transaction)

	for _, input := range tx.Inputs {
		prevTX, err := bc.FindTransaction(input.ID)
		HandleError(err)
		prevTxs[hex.EncodeToString(prevTX.ID)] = prevTX
	}

	return tx.Verify(prevTxs)
}