package blockchain

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/dgraph-io/badger"
)

const (
	// DBpath = "./DB/blocks"
	// DBFile = "./DB/blocks/MANIFEST"
	GenesisData = "First Transaction from Genesis"
	dbPath = "./DB/blocks_%s"
)

type BlockChain struct {
	LastHash					[]byte
	Database					*badger.DB
}


func DBexists(path string) bool {
	if _, err := os.Stat(path+"/MANIFEST"); os.IsNotExist(err){
		return false
	}
	return true
}


func InitBlockChain(address, nodeID string) *BlockChain{
	path := fmt.Sprintf(dbPath, nodeID)
	var lastHash []byte

	if DBexists(path) {
		fmt.Println("Blockchain already exists")
		runtime.Goexit()
	}


	opts := badger.DefaultOptions(path)
	opts.Dir = path
	opts.ValueDir = path

	db, err := OpenDB(path, opts)
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


func ContinueBlockChain(nodeID string) *BlockChain{
	path := fmt.Sprintf(dbPath, nodeID)
	var lastHash []byte
	
	if !DBexists(path){
		fmt.Println("No existing blockchain found. Create one!")
		runtime.Goexit()
	}
	
	
	opts := badger.DefaultOptions(path)
	opts.Dir = path
	opts.ValueDir = path
	
	db, err := OpenDB(path, opts)
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



func (chain *BlockChain)AddBlock(block *Block) {
	err := chain.Database.Update(func(txn *badger.Txn) error {
		if _, err := txn.Get(block.Hash); err ==  nil {
			return nil
		}

		blockData := block.Serialize()
		err := txn.Set(block.Hash, blockData)
		HandleError(err)

		item, err := txn.Get([]byte("lh"))
		HandleError(err)
		lastHash, _ := item.ValueCopy(nil)

		item, err = txn.Get(lastHash)
		HandleError(err)
		lastBlockData, _ := item.ValueCopy(nil)

		lastBlock := Deserialize(lastBlockData)

		if block.Height > lastBlock.Height {
			err := txn.Set([]byte("lh"), block.Hash)
			HandleError(err)
			chain.LastHash = block.Hash
		}

		return nil
	})
	HandleError(err)
}


func (chain *BlockChain) GetBlock(blockHash []byte) (Block, error) {
	var block Block

	err := chain.Database.View(func(txn *badger.Txn) error {
		if item, err := txn.Get(blockHash); err != nil {
			return errors.New("Block is not Found")
		}else {
			blockData, _ := item.ValueCopy(nil)

			block = *Deserialize(blockData)
		}
		
		return nil
	})

	if err != nil {
		return block, err
	}

	return block, err
}


func (chain *BlockChain) GetBlockHashes() [][]byte {
	var blocks [][]byte

	iter := chain.Iterator()

	for {
		block := iter.Next()
		blocks = append(blocks, block.Hash)

		if len(block.PrevHash) == 0 {
			break
		}
	}

	return blocks
}


func (chain *BlockChain) GetBestHeight() int {
	var lastBlock Block

	err := chain.Database.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("lh"))
		HandleError(err)

		lastHash, _ := item.ValueCopy(nil)
		item, err = txn.Get(lastHash)
		HandleError(err)
		lastBlockData, _ := item.ValueCopy(nil)
		lastBlock = *Deserialize(lastBlockData)

		return nil
	})
	HandleError(err)

	return lastBlock.Height
}



func (chain *BlockChain) MineBlock(transactions []*Transaction) *Block {
	var lastHash []byte
	var lastHeight int


	for _, tx := range transactions {
		if !chain.VerifyTransaction(tx) {
			log.Panic("invalid Transaction")
		}
	}


	err := chain.Database.View(func(txn *badger.Txn) error{
		item, err := txn.Get([]byte("lh"))
		HandleError(err)
		lastHash, err = item.ValueCopy([]byte("lh"))
		return err

		item, err = txn.Get(lastHash)
		HandleError(err)

		lastBlockData, err := item.ValueCopy(nil)
		lastBlock := Deserialize(lastBlockData)

		lastHeight = lastBlock.Height

		return err
	})
	HandleError(err)

	newBlock := CreateBlock(transactions, lastHash, lastHeight+1)

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


func (bc *BlockChain) VerifyTransaction(tx *Transaction) bool {
	if tx.IsCoinbase() {
		return true
	}
	
	prevTxs := make(map[string]Transaction)


	for _, input := range tx.Inputs {
		prevTX, err := bc.FindTransaction(input.ID)
		HandleError(err)
		prevTxs[hex.EncodeToString(prevTX.ID)] = prevTX
	}

	return tx.Verify(prevTxs)
}




func Retry(dir string, originalOpts badger.Options) (*badger.DB, error) {
	lockPath := filepath.Join(dir, "LOCK")
	if err := os.Remove(lockPath); err != nil {
		return nil, fmt.Errorf(`Removing "Lock" %s`, err)
	}

	retryOpts := originalOpts
	retryOpts.Truncate = true
	db, err := badger.Open(retryOpts)

	return db, err
}


func OpenDB(dir string, opts badger.Options) (*badger.DB, error) {
	if db, err := badger.Open(opts); err != nil {
		if strings.Contains(err.Error(), "LOCK") {
			if db, err := Retry(dir, opts); err != nil {
				log.Println("Database unlocked, value log truncated")
				return db, nil
			}
			log.Println("could not unlock database:", err)
		}
		return nil, err
	}else {
		return db, nil
	}
}