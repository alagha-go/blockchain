package blockchain

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"tensor/lib/wallet"
)

type TxOutput struct {
	Value 							int
	PubKeyHash						[]byte
}

type TxInput struct {
	ID								[]byte
	Out								int
	Signature						[]byte
	PubKey							[]byte
}


type TxOutputs struct {
	Outputs				[]TxOutput
}



func NewTxOutput(value int, address string) *TxOutput {
	txo := &TxOutput{value, nil}
	txo.Lock([]byte(address))
	fmt.Printf("key: %x\n", txo.PubKeyHash)
	return txo
}


func (in *TxInput) UsesKey(pubKeyHash []byte) bool {
	lockingHash := wallet.PubKeyHash(in.PubKey)

	return bytes.Equal(lockingHash, pubKeyHash)
}

func (out *TxOutput) Lock(address []byte) {
	pubKeyHash := wallet.Base58Decode(address)
	out.PubKeyHash = pubKeyHash[1: len(pubKeyHash)-4]
}


func (out *TxOutput) IsLockedWithKey(pubKeyHash []byte) bool {
	return bytes.Equal(out.PubKeyHash, pubKeyHash)
}


func (outs TxOutputs) Serialize() []byte {
	var buffer bytes.Buffer
	encode := gob.NewEncoder(&buffer)
	err := encode.Encode(outs)
	HandleError(err)
	return buffer.Bytes()
}

func DeserializeOutputs(data []byte) TxOutputs {
	var outputs TxOutputs
	decode := gob.NewDecoder(bytes.NewReader(data))
	err := decode.Decode(&outputs)
	HandleError(err)
	return outputs
}

