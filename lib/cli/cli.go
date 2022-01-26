package cli

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"strconv"
	"tensor/lib/blockchain"
	"tensor/lib/wallet"
	// "time"
)



type CommandLine struct {}


func (cli *CommandLine) PrintUsage(){
	fmt.Println("Usage:")
	fmt.Println("getbalance -address ADDRESS - get the balance from your account")
	fmt.Println("createblockchain -address ADDRESS - creates a blockchain")
	fmt.Println("printchain  - Prints the blocks in the chain")
	fmt.Println("send -from FROM -to To -amount AMOUNT  - send amount of money to a user")
	fmt.Println("createwallet -Creates a new wallet")
	fmt.Println("listaddresses -lists all the wallets addresses")
	fmt.Println("reindexutxo -Rebuilds the UTXO set")
}

func (cli *CommandLine) ValidateArgs(){
	if len(os.Args) < 2 {
		cli.PrintUsage()
		runtime.Goexit()
	}
}


func (cli *CommandLine) reIndexUtxo() {
	chain := blockchain.ContinueBlockChain("")
	defer chain.Database.Close()
	UTXOSet := blockchain.UTXOSet{chain}
	UTXOSet.Reindex()

	count := UTXOSet.CountTransactions()
	fmt.Printf("Done! There are %d Transactions in the UTXO set.\n", count)
}

func (cli *CommandLine) PrintChain(){

	chain := blockchain.ContinueBlockChain("")
	defer chain.Database.Close()

	iter := chain.Iterator()

	for {
		block := iter.Next()

		fmt.Printf("Previous Hash: %x\n", block.PrevHash)
		fmt.Printf("Hash: %x\n", block.Hash)

		pow := blockchain.NewProof(block)
		fmt.Printf("pow: %s\n", strconv.FormatBool(pow.Validate()))

		for _, tx := range block.Transactions{
			fmt.Println(tx)
		}

		fmt.Println()

		if len(block.PrevHash) == 0 {
			break
		}
	}
}

func (cli *CommandLine) createBlockChain(address string) {
	if !wallet.ValidateAddress(address) {
		log.Panic("Invalid address")
	}
	chain := blockchain.InitBlockChain(address)
	defer chain.Database.Close()
	UTXOSet := blockchain.UTXOSet{chain}
	UTXOSet.Reindex()
	fmt.Println("Finished!")
}


func (cli *CommandLine) GetBalance(address string) {
	if !wallet.ValidateAddress(address) {
		log.Panic("Invalid address")
	}
	chain :=  blockchain.ContinueBlockChain(address)
	UTXOSet := blockchain.UTXOSet{chain}
	defer chain.Database.Close()

	balance := 0

	pubKeyHash := wallet.Base58Decode([]byte(address))
	pubKeyHash = pubKeyHash[1:len(pubKeyHash) - 4]
	
	UTXOs := UTXOSet.FindUnspentTransactions(pubKeyHash)
	fmt.Println(balance)

	for _, out := range UTXOs{
		balance += out.Value
	}


	fmt.Printf("Balance of %s:  %d\n", address, balance)

}



func (cli *CommandLine) Send(from, to string, amount int) {
	if !wallet.ValidateAddress(from) {
		log.Panic("senders address is invalid")
	}
	if !wallet.ValidateAddress(to) {
		log.Panic("receivers address is invalid")
	}
	chain := blockchain.ContinueBlockChain(from)
	UTXOSet := blockchain.UTXOSet{chain}
	defer chain.Database.Close()

	tx := blockchain.NewTransaction(from, to, amount, &UTXOSet)
	cbTx := blockchain.CoinbaseTx(from, "")
	block := chain.AddBlock([]*blockchain.Transaction{cbTx, tx})
	UTXOSet.Update(block)
	fmt.Println("Success!")
}


func (cli *CommandLine) ListAddresses() {
	wallets, err := wallet.CreateWallets()
	HandleError(err, false)
	addresses := wallets.GetAllAddresses()

	for _, address := range addresses {
		fmt.Println(address)
	}
}

func (cli *CommandLine) CreateWallet() {
	wallets, _ := wallet.CreateWallets()
	address := wallets.AddWallet()

	fmt.Printf("New Address is: %s\n", address)
	wallets.SaveFile()
}




func (cli *CommandLine) Run() {
	cli.ValidateArgs()

	getBalanceCmd := flag.NewFlagSet("getbalance", flag.ExitOnError)
	createBlockchainCmd := flag.NewFlagSet("createblockchain", flag.ExitOnError)
	sendCmd := flag.NewFlagSet("send", flag.ExitOnError)
	printChainCmd := flag.NewFlagSet("printChain", flag.ExitOnError)
	CreateWalletCmd := flag.NewFlagSet("createwallet", flag.ExitOnError)
	ListAddressesCmd := flag.NewFlagSet("listaddresses", flag.ExitOnError)
	ReindexUtxocmd  := flag.NewFlagSet("reindexutxo", flag.ExitOnError)

	getBalanceAddress := getBalanceCmd.String("address", "", "The address")
	createBlockchainAddress := createBlockchainCmd.String("address", "", "The address")
	sendFrom := sendCmd.String("from", "", "source wallet address")
	sendTo := sendCmd.String("to", "", "destination wallet address")
	sendAmount := sendCmd.Int("amount", 0, "Amount to send")

	switch os.Args[1]{
		case "getbalance":
			err := getBalanceCmd.Parse(os.Args[2:])
			blockchain.HandleError(err)
		
		case "createblockchain":
			err := createBlockchainCmd.Parse(os.Args[2:])
			blockchain.HandleError(err)
		case "printchain":
			err := printChainCmd.Parse(os.Args[2:])
			blockchain.HandleError(err)
		case "send":
			err := sendCmd.Parse(os.Args[2:])
			blockchain.HandleError(err)
		case "createwallet":
			err := CreateWalletCmd.Parse(os.Args[2:])
			blockchain.HandleError(err)
		case "listaddresses":
			err := ListAddressesCmd.Parse(os.Args[2:])
			blockchain.HandleError(err)
		case "reindexutxo":
			err := ReindexUtxocmd.Parse(os.Args[2:])
			blockchain.HandleError(err)
		default:
			cli.PrintUsage()
			runtime.Goexit()
	}

	if getBalanceCmd.Parsed() {
		if *getBalanceAddress == "" {
			fmt.Println("provide a wallet address")
			runtime.Goexit()
		}
		cli.GetBalance(*getBalanceAddress)
	}

	if createBlockchainCmd.Parsed() {
		if *createBlockchainAddress == "" {
			fmt.Println("provide an address to your wallet")
			runtime.Goexit()
		}
		cli.createBlockChain(*createBlockchainAddress)
	}

	if sendCmd.Parsed(){
		if *sendFrom == ""{
			fmt.Println("provide sender wallet address")
			runtime.Goexit()
		}
		if *sendTo == "" {
			fmt.Println("Provide wallet address to send to")
			runtime.Goexit()
		}
		if *sendAmount < 0 {
			fmt.Println("You cannot send a negative amount")
			runtime.Goexit()
		}
		if *sendAmount == 0 {
			fmt.Println("You cannot send 0 as amount")
			runtime.Goexit()
		}
		cli.Send(*sendFrom, *sendTo, *sendAmount)
	}

	if printChainCmd.Parsed() {
		cli.PrintChain()
		runtime.Goexit()
	}

	if ReindexUtxocmd.Parsed() {
		cli.reIndexUtxo()
		runtime.Goexit()
	}

	if ListAddressesCmd.Parsed() {
		cli.ListAddresses()
		runtime.Goexit()
	}

	if CreateWalletCmd.Parsed() {
		cli.CreateWallet()
		runtime.Goexit()
	}

}


func HandleError(err error, res bool)error{
	if err != nil && res == true {
		return err
	}else if err != nil{
		log.Panic(err)
	}
	return nil
}