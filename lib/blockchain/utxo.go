package blockchain

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"log"

	"github.com/dgraph-io/badger"
)

var (
	utxoprefix = []byte("utxo-")
	prefixlength = len(utxoprefix)
)

type UTXOSet struct {
	BlockChain 			*BlockChain
}


func (u *UTXOSet) DeleteByPrefix(prefix []byte) {
	deleteKeys := func(KeysForDelete [][]byte) error {
		if err := u.BlockChain.Database.Update(func(txn *badger.Txn) error {
			for _, key := range KeysForDelete {
				if err := txn.Delete(key); err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			return err
		}
		return nil
	}

	collectionSize := 100000

	u.BlockChain.Database.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()

		keysForDelete := make([][]byte, 0, collectionSize)
		keysCollected := 0

		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			key := it.Item().KeyCopy(nil)
			keysForDelete = append(keysForDelete, key)
			keysCollected++
			if keysCollected == collectionSize {
				if err := deleteKeys(keysForDelete); err != nil {
					log.Panic(err)
				}
				keysForDelete = make([][]byte, 0, collectionSize)
				keysCollected = 0
			}
		}
		if keysCollected > 0 {
			if err := deleteKeys(keysForDelete); err != nil {
				log.Panic(err)
			}
		}
		return nil
	})
}


func (u UTXOSet) Reindex() {
	db := u.BlockChain.Database

	u.DeleteByPrefix(utxoprefix)

	UTXO := u.BlockChain.FindUTXO()

	err := db.Update(func(txn *badger.Txn) error {
		for txId, outs := range UTXO {
			key, err := hex.DecodeString(txId)
			HandleError(err)

			key = append(utxoprefix, key...)

			err = txn.Set(key, outs.Serialize())
			HandleError(err)
		}
		return nil
	})
	HandleError(err)
}

func (u UTXOSet) Update(block *Block) {
	db := u.BlockChain.Database

	err := db.Update(func(txn *badger.Txn) error {
		for _, tx := range block.Transactions {
			if !tx.IsCoinbase() {
				for _, input := range tx.Inputs {
					updatedOuts := TxOutputs{}
					inputID := append(utxoprefix, input.ID...)
					item, err := txn.Get(inputID)
					HandleError(err)
					value, err := item.ValueCopy(nil)
					HandleError(err)
					
					outs := DeserializeOutputs(value)

					for Outindex, out := range outs.Outputs {
						if Outindex != input.Out {
							updatedOuts.Outputs = append(updatedOuts.Outputs, out)
						}
					}

					if len(updatedOuts.Outputs) == 0 {
						err := txn.Delete(inputID)
						HandleError(err)
					}else{
						err := txn.Set(inputID, updatedOuts.Serialize())
						HandleError(err)
					}


				}
			}

			newOutputs := TxOutputs{}
			for _, out := range tx.Outputs {
				newOutputs.Outputs = append(newOutputs.Outputs, out)
			}

			txID := append(utxoprefix, tx.ID...)
			err := txn.Set(txID, newOutputs.Serialize())
			HandleError(err)

		}
		return nil
	})
	HandleError(err)
}



func (u UTXOSet) CountTransactions() int {
	db := u.BlockChain.Database
	counter := 0

	err := db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions

		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Seek(utxoprefix); it.ValidForPrefix(utxoprefix); it.Next() {
			counter++
		}
		return nil
	})
	HandleError(err)
	return counter
}



func (u *UTXOSet) FindUnspentTransactions(pubKeyHash []byte) []TxOutput {
	var UTXOs []TxOutput
	
	db := u.BlockChain.Database

	err := db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions

		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Seek(utxoprefix); it.ValidForPrefix(utxoprefix); it.Next() {
			item := it.Item()
			value, err := item.ValueCopy(nil)
			HandleError(err)
			outs := DeserializeOutputs(value)

			for _, out := range outs.Outputs {
				if out.IsLockedWithKey(pubKeyHash) {
					UTXOs = append(UTXOs, out)
				}
			}
		}
		return nil
	})
	HandleError(err)
	
	fmt.Println("DONE")
	
	return UTXOs
}



func (u UTXOSet) FindSpendableOutputs(pubKeyHash []byte, amont int) (int, map[string][]int) {
	unspentOutputs := make(map[string][]int)
	accumulated := 0
	db := u.BlockChain.Database


	err := db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions

		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Seek(utxoprefix); it.ValidForPrefix(utxoprefix); it.Next() {
			item := it.Item()
			k := item.Key()
			value, err := item.ValueCopy(nil)
			HandleError(err)
			k = bytes.TrimPrefix(k, utxoprefix)
			txID := hex.EncodeToString(k)
			outs := DeserializeOutputs(value)

			for outIndex, out :=  range outs.Outputs {
				if out.IsLockedWithKey(pubKeyHash) && accumulated < amont{
				accumulated += out.Value
				unspentOutputs[txID] = append(unspentOutputs[txID], outIndex)

				if accumulated >= amont {
					break 
				}
			}
			}
		}

		return nil
	})
	HandleError(err)

	return accumulated, unspentOutputs
}