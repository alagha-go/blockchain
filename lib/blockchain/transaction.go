package blockchain

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"strings"
	"tensor/lib/wallet"
)

type Transaction struct {
	ID								[]byte
	Inputs 							[]TxInput
	Outputs 						[]TxOutput
}


func (tx Transaction) Serialize() []byte {
	var encoded bytes.Buffer

	encode := gob.NewEncoder(&encoded)
	err := encode.Encode(tx)
	HandleError(err)
	
	return encoded.Bytes()
}

func (tx *Transaction) Hash() []byte {
	var hash [32]byte

	txCopy := *tx
	txCopy.ID = []byte{}

	hash = sha256.Sum256(txCopy.Serialize())

	return hash[:]
}





func CoinbaseTx(to, data string) *Transaction {
	if data == "" {
		randomData := make([]byte, 24)
		_, err := rand.Read(randomData)
		HandleError(err)
		data = fmt.Sprintf("%x", randomData)
	}

	fmt.Println(data)

	Txin := TxInput{[]byte{}, -1, nil, []byte(data)}
	Txout := NewTxOutput(20, to)

	tx := Transaction{nil, []TxInput{Txin}, []TxOutput{*Txout}}

	tx.ID = tx.Hash()

	return &tx
}


func (tx *Transaction) IsCoinbase() bool {
	return len(tx.Inputs) == 1 && len(tx.Inputs[0].ID) == 0 && tx.Inputs[0].Out == -1
}


func (tx *Transaction) TrimmedCopy() Transaction {
	var inputs []TxInput
	var outputs []TxOutput

	for _, input := range tx.Inputs {
		inputs = append(inputs, TxInput{input.ID, input.Out, nil, nil})
	}

	for _, output := range tx.Outputs {
		outputs = append(outputs, TxOutput{output.Value, output.PubKeyHash})
	}

	txCopy := Transaction{tx.ID, inputs, outputs}

	return txCopy
}


func (tx *Transaction) Sign(privKey ecdsa.PrivateKey, prevTxs map[string]Transaction) {
	if tx.IsCoinbase() {
		return
	}

	for _, in := range tx.Inputs {
		if prevTxs[hex.EncodeToString(in.ID)].ID == nil{
			log.Panic("Error: Previous Transaction does not exist")
		}
	}

	txCopy := tx.TrimmedCopy()

	for inputId, input := range txCopy.Inputs {
		prevTxs := prevTxs[hex.EncodeToString(input.ID)]
		txCopy.Inputs[inputId].Signature = nil
		txCopy.Inputs[inputId].PubKey = prevTxs.Outputs[input.Out].PubKeyHash
		txCopy.ID = txCopy.Hash()
		txCopy.Inputs[inputId].PubKey = nil

		r, s, err := ecdsa.Sign(rand.Reader, &privKey, txCopy.ID)
		HandleError(err)
		Signature := append(r.Bytes(), s.Bytes()...)

		tx.Inputs[inputId].Signature = Signature
	}
}



func NewTransaction(from, to string, amount int, UTXO *UTXOSet) *Transaction {
	var inputs []TxInput
	var outputs []TxOutput

	wallets, err := wallet.CreateWallets()
	w := wallets.GetWallet(from)
	pubKeyHash := wallet.PubKeyHash(w.PublicKey)

	accumulator, validOutputs := UTXO.FindSpendableOutputs(pubKeyHash, amount)

	HandleError(err)

	if accumulator < amount {
		log.Panic("Error: not enough funds")
	}

	for txid, out := range validOutputs {
		txID, err := hex.DecodeString(txid)
		HandleError(err)

		for _, out := range out{
			input := TxInput{txID, out, nil, w.PublicKey}
			inputs = append(inputs, input)
		}
	}

	outputs = append(outputs, *NewTxOutput(amount, to))

	if accumulator > amount {
		outputs = append(outputs, *NewTxOutput(accumulator-amount, from))
	}

	tx := Transaction{nil, inputs, outputs}
	tx.ID = tx.Hash()

	UTXO.BlockChain.SignTransaction(&tx, w.PrivateKey)


	return &tx
}

func (tx *Transaction) Verify(prevTxs map[string]Transaction) bool {
	if tx.IsCoinbase() {
		return true
	}

	for _, in := range tx.Inputs {
		if prevTxs[hex.EncodeToString(in.ID)].ID == nil{
			log.Panic("Error: Previous Transaction does not exist")
		}
	}

	txCopy := tx.TrimmedCopy()
	curve := elliptic.P256()

	for index, input := range tx.Inputs {
		prevTxs := prevTxs[hex.EncodeToString(input.ID)]
		txCopy.Inputs[index].Signature = nil
		txCopy.Inputs[index].PubKey = prevTxs.Outputs[input.Out].PubKeyHash
		txCopy.ID = txCopy.Hash()
		txCopy.Inputs[index].PubKey = nil

		
		r := big.Int{}
		s := big.Int{}
		sigLen := len(input.Signature)
		r.SetBytes(input.Signature[:(sigLen / 2)])
		s.SetBytes(input.Signature[(sigLen / 2):])

		x := big.Int{}
		y := big.Int{}
		keyLen := len(input.PubKey)
		x.SetBytes(input.PubKey[:(keyLen / 2)])
		y.SetBytes(input.PubKey[(keyLen / 2):])

		rawPubKey := ecdsa.PublicKey{curve, &x, &y}

		if ecdsa.Verify(&rawPubKey, txCopy.ID, &r, &s) == false {
			return false
		}
	}

	return true
}




func (tx Transaction) String() string {
	var lines []string

	lines = append(lines, fmt.Sprintf("---  Trasaction %x:", tx.ID))

	for index, input := range tx.Inputs {
		lines = append(lines, fmt.Sprintf("		Input %d:", index))
		lines = append(lines, fmt.Sprintf("			TXID: %x", input.ID))
		lines = append(lines, fmt.Sprintf("			Out: %d", input.Out))
		lines = append(lines, fmt.Sprintf("			Signature: %x", input.Signature))
		lines = append(lines, fmt.Sprintf("			PubKey: %x", input.PubKey))
	}

	for index, output := range tx.Outputs {
		lines = append(lines, fmt.Sprintf("		Output %d:", index))
		lines = append(lines, fmt.Sprintf("			Value: %d", output.Value))
		lines = append(lines, fmt.Sprintf("			Script: %x", output.PubKeyHash))
	}

	return strings.Join(lines, "\n")
}