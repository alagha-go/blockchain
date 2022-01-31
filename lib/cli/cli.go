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
	"tensor/lib/network"
)



type CommandLine struct {}


func (cli *CommandLine) PrintUsage(){
	fmt.Println("Usage:")
	fmt.Println("getbalance -address ADDRESS - get the balance from your account")
	fmt.Println("createblockchain -address ADDRESS - creates a blockchain")
	fmt.Println("printchain  - Prints the blocks in the chain")
	fmt.Println("send -from FROM -to To -amount AMOUNT -mine - send amount of money to a user then -mine flag")
	fmt.Println("createwallet -Creates a new wallet")
	fmt.Println("listaddresses -lists all the wallets addresses")
	fmt.Println("reindexutxo -Rebuilds the UTXO set")
	fmt.Println("startnode -miner ADDRESS - Start a node with ID specified in NODE_ID env. var. -miner enables mining")
}

func (cli *CommandLine) ValidateArgs(){
	if len(os.Args) < 2 {
		cli.PrintUsage()
		runtime.Goexit()
	}
}


func (cli *CommandLine) StartNode(nodeID, minerAddress string) {
	fmt.Printf("Starting Node %s\n", nodeID)

	if len(minerAddress) > 0 {
		if wallet.ValidateAddress(minerAddress) {
			fmt.Printf("Mining is on. Address to receive rewards: %s\n", minerAddress)
		}else {
			log.Panic("Wrong miner Address.")
		}
	}

	network.StartServer(nodeID, minerAddress)
}


func (cli *CommandLine) reIndexUtxo(nodeID string) {
	chain := blockchain.ContinueBlockChain(nodeID)
	defer chain.Database.Close()
	UTXOSet := blockchain.UTXOSet{chain}
	UTXOSet.Reindex()

	count := UTXOSet.CountTransactions()
	fmt.Printf("Done! There are %d Transactions in the UTXO set.\n", count)
}

func (cli *CommandLine) PrintChain(nodeID string){

	chain := blockchain.ContinueBlockChain(nodeID)
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

func (cli *CommandLine) createBlockChain(address, nodeID string) {
	if !wallet.ValidateAddress(address) {
		log.Panic("Invalid address")
	}
	chain := blockchain.InitBlockChain(address, nodeID)
	defer chain.Database.Close()
	UTXOSet := blockchain.UTXOSet{chain}
	UTXOSet.Reindex()
	fmt.Println("Finished!")
}


func (cli *CommandLine) GetBalance(address, nodeID string) {
	if !wallet.ValidateAddress(address) {
		log.Panic("Invalid address")
	}
	chain :=  blockchain.ContinueBlockChain(nodeID)
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



func (cli *CommandLine) Send(from, to string, amount int, nodeID string, mineNow bool) {
	if !wallet.ValidateAddress(from) {
		log.Panic("senders address is invalid")
	}
	if !wallet.ValidateAddress(to) {
		log.Panic("receivers address is invalid")
	}
	chain := blockchain.ContinueBlockChain(nodeID)
	UTXOSet := blockchain.UTXOSet{chain}
	defer chain.Database.Close()

	wallets, err := wallet.CreateWallets(nodeID)
	HandleError(err, false)
	wallet := wallets.GetWallet(from)

	tx := blockchain.NewTransaction(&wallet, to, nodeID, amount, &UTXOSet)
	if mineNow {
		cbTx := blockchain.CoinbaseTx(from, "")
		txs := []*blockchain.Transaction{cbTx, tx}
		block := chain.MineBlock(txs)
		UTXOSet.Update(block)
	}else{
		network.SendTx(network.KnownNodes[0], tx)
		fmt.Println("send tx")
	}
	
	fmt.Println("Success!")
}


func (cli *CommandLine) ListAddresses(nodeID string) {
	wallets, err := wallet.CreateWallets(nodeID)
	HandleError(err, false)
	addresses := wallets.GetAllAddresses()

	for _, address := range addresses {
		fmt.Println(address)
	}
}

func (cli *CommandLine) CreateWallet(nodeID string) {
	wallets, _ := wallet.CreateWallets(nodeID)
	address := wallets.AddWallet()

	fmt.Printf("New Address is: %s\n", address)
	wallets.SaveFile(nodeID)
}




func (cli *CommandLine) Run() {
	cli.ValidateArgs()

	nodeID := os.Getenv("NODE_ID")
	if nodeID == "" {
		fmt.Printf("NODE_ID env is not set\n")
		runtime.Goexit()
	}

	getBalanceCmd := flag.NewFlagSet("getbalance", flag.ExitOnError)
	createBlockchainCmd := flag.NewFlagSet("createblockchain", flag.ExitOnError)
	sendCmd := flag.NewFlagSet("send", flag.ExitOnError)
	printChainCmd := flag.NewFlagSet("printChain", flag.ExitOnError)
	CreateWalletCmd := flag.NewFlagSet("createwallet", flag.ExitOnError)
	ListAddressesCmd := flag.NewFlagSet("listaddresses", flag.ExitOnError)
	ReindexUtxocmd  := flag.NewFlagSet("reindexutxo", flag.ExitOnError)
	StartNodecmd  := flag.NewFlagSet("startnode", flag.ExitOnError)

	getBalanceAddress := getBalanceCmd.String("address", "", "The address")
	createBlockchainAddress := createBlockchainCmd.String("address", "", "The address")
	sendFrom := sendCmd.String("from", "", "source wallet address")
	sendTo := sendCmd.String("to", "", "destination wallet address")
	sendAmount := sendCmd.Int("amount", 0, "Amount to send")
	sendMine := sendCmd.Bool("mine", false, "mine emmediately on the same node")
	startNodeMiner := StartNodecmd.String("miner", "", "enable mining mode and send reward this address")

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
		case "startnode":
			err := StartNodecmd.Parse(os.Args[2:])
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
		cli.GetBalance(*getBalanceAddress,  nodeID)
	}

	if createBlockchainCmd.Parsed() {
		if *createBlockchainAddress == "" {
			fmt.Println("provide an address to your wallet")
			runtime.Goexit()
		}
		cli.createBlockChain(*createBlockchainAddress, nodeID)
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
		cli.Send(*sendFrom, *sendTo, *sendAmount, nodeID, *sendMine)
	}

	if StartNodecmd.Parsed() {
		nodeID := os.Getenv("NODE_ID")
		if nodeID == "" {
			StartNodecmd.Usage()
			runtime.Goexit()
		}
		cli.StartNode(nodeID, *startNodeMiner)
	}

	if printChainCmd.Parsed() {
		cli.PrintChain(nodeID)
		runtime.Goexit()
	}

	if ReindexUtxocmd.Parsed() {
		cli.reIndexUtxo(nodeID)
		runtime.Goexit()
	}

	if ListAddressesCmd.Parsed() {
		cli.ListAddresses(nodeID)
		runtime.Goexit()
	}

	if CreateWalletCmd.Parsed() {
		cli.CreateWallet(nodeID)
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