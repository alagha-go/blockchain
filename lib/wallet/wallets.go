package wallet

import (
	"bytes"
	"crypto/elliptic"
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"log"
	"os"
)


const (
	walletFile = "./DB/wallets.data"
)

type Wallets struct {
	Wallets     map[string]*Wallet
}

func (ws *Wallets) LoadFile() error {
	if _, err := os.Stat(walletFile); os.IsNotExist(err) {
		return err
	}

	var wallets Wallets

	fileContent, err := ioutil.ReadFile(walletFile)
	if err != nil {
		return err
	}

	gob.Register(elliptic.P256())
	decodder := gob.NewDecoder(bytes.NewReader(fileContent))
	err = decodder.Decode(&wallets)

	if err != nil {
		return err
	}

	ws.Wallets = wallets.Wallets

	return nil
}

func (wallets *Wallets) SaveFile(){
	var content bytes.Buffer

	gob.Register(elliptic.P256())

	encoder := gob.NewEncoder(&content)

	err := encoder.Encode(wallets)
	HandleError(err)

	// fmt.Println(content.String())

	err = ioutil.WriteFile(walletFile, content.Bytes(),0644)
	if err != nil {
		fmt.Println(err)
	}
}


func CreateWallets() (*Wallets, error) {
	wallets := Wallets{}

	wallets.Wallets = make(map[string]*Wallet)

	err := wallets.LoadFile()

	return &wallets, err
}



func (ws *Wallets) GetWallet(address string) Wallet {
	return *ws.Wallets[address]
}


func (ws *Wallets) GetAllAddresses() []string {
	var addresses []string

	for address := range ws.Wallets{
		addresses = append(addresses, address)
	}

	return addresses
}


func (ws *Wallets) AddWallet() string {
	wallet := MakeWallet()
	address := string(wallet.Address())

	ws.Wallets[address] = wallet

	return address
}







func HandleError(err error) {
	if err != nil {
		log.Panic(err)
	}
}