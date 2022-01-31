package blockchain

import "github.com/dgraph-io/badger"

type BlockChainIterator struct {
	CurrentHash					[]byte
	Database					*badger.DB
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