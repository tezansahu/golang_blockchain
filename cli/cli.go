package cli

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"strconv"

	"github.com/tezansahu/golang_blockchain/blockchain"
	"github.com/tezansahu/golang_blockchain/wallet"
)

type CommandLine struct {
	// blockchain *blockchain.Blockchain
}

func (cli *CommandLine) printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  getbalance -address ADDRESS : Get the balance for an address")
	fmt.Println("  createblockchain -address ADDRESS : Creates a blockchain whose genesis block is mined by the address")
	fmt.Println("  print : Print the blocks in the chain")
	fmt.Println("  send -from FROM -to TO -amount AMOUNT : Send amount from an address to another")
	fmt.Println("  createwallet : Creates a new Wallet")
	fmt.Println("  listaddresses : Lists the addresses in our Wallets file")
}

func (cli *CommandLine) validateArgs() {
	if len(os.Args) < 2 {
		cli.printUsage()
		runtime.Goexit()
	}
}

// func (cli *CommandLine) addBlock(data string) {
// 	cli.blockchain.AddBlock(data)
// 	fmt.Println("Block Added!")
// }

func (cli *CommandLine) printChain() {
	chain := blockchain.ContinueBlockchain("")
	defer chain.Database.Close()
	iter := chain.Iterator()

	for {
		block := iter.Next()
		fmt.Printf("Previous Hash: %x\n", block.PrevHash)

		fmt.Printf("Hash: %x\n", block.Hash)
		fmt.Printf("Nonce: %d\n", block.Nonce)
		pow := blockchain.NewProof(block)
		fmt.Printf("PoW: %s\n", strconv.FormatBool(pow.Validate()))
		for _, tx := range block.Transactions {
			fmt.Println(tx)
		}
		fmt.Println()

		if len(block.PrevHash) == 0 {
			break
		}

	}
}

func (cli *CommandLine) createBlockchain(address string) {
	if !wallet.ValidateAddress(address) {
		log.Panic("Address not valid")
	}
	chain := blockchain.InitBlockchain(address)
	chain.Database.Close()
	fmt.Println("Finished!")
}

func (cli *CommandLine) getBalance(address string) {
	if !wallet.ValidateAddress(address) {
		log.Panic("Address not valid")
	}
	chain := blockchain.ContinueBlockchain(address)
	defer chain.Database.Close()

	balance := 0
	pubKeyHash := wallet.Base58Decode([]byte(address))
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-wallet.ChecksumLength]
	UTXOs := chain.FindUTXO(pubKeyHash)
	for _, UTXO := range UTXOs {
		balance += UTXO.Value
	}

	fmt.Printf("Balance of %s: %d\n", address, balance)
}

func (cli *CommandLine) send(from, to string, amount int) {
	if !wallet.ValidateAddress(from) {
		log.Panic("Address not valid")
	}
	if !wallet.ValidateAddress(to) {
		log.Panic("Address not valid")
	}
	chain := blockchain.ContinueBlockchain(from)
	defer chain.Database.Close()

	tx := blockchain.NewTransaction(from, to, amount, chain)

	chain.AddBlock([]*blockchain.Transaction{tx})
	fmt.Println("Successful!")
}

func (cli *CommandLine) listAddresses() {
	wallets, _ := wallet.CreateWallets()

	addresses := wallets.GetAllAddresses()

	for _, address := range addresses {
		fmt.Println(address)
	}
}

func (cli *CommandLine) createWallet() {
	wallets, _ := wallet.CreateWallets()

	address := wallets.AddWallet()
	wallets.SaveFile()

	fmt.Printf("New Address is: %s\n", address)
}

func (cli *CommandLine) Run() {
	cli.validateArgs()

	getBalanceCmd := flag.NewFlagSet("getbalance", flag.ExitOnError)
	createBlockchainCmd := flag.NewFlagSet("createblockchain", flag.ExitOnError)
	sendCmd := flag.NewFlagSet("send", flag.ExitOnError)
	printChainCmd := flag.NewFlagSet("print", flag.ExitOnError)
	createWalletCmd := flag.NewFlagSet("createwallet", flag.ExitOnError)
	listAddressesCmd := flag.NewFlagSet("listaddresses", flag.ExitOnError)

	getBalanceAddress := getBalanceCmd.String("address", "", "Address whose balance is to be found")
	createBlockchainAddress := createBlockchainCmd.String("address", "", "Address that mines the genesis block of the blockchain")
	sendFromAddress := sendCmd.String("from", "", "Source Wallet address")
	sendToAddress := sendCmd.String("to", "", "Destination Wallet address")
	sendAmount := sendCmd.Int("amount", 0, "Amount to send")

	switch os.Args[1] {

	case "print":
		err := printChainCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "getbalance":
		err := getBalanceCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "send":
		err := sendCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "createblockchain":
		err := createBlockchainCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "createwallet":
		err := createWalletCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "listaddresses":
		err := listAddressesCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	default:
		cli.printUsage()
		runtime.Goexit()
	}

	if getBalanceCmd.Parsed() {
		if *getBalanceAddress == "" {
			getBalanceCmd.Usage()
			runtime.Goexit()
		}
		cli.getBalance(*getBalanceAddress)
	}

	if printChainCmd.Parsed() {
		cli.printChain()
	}

	if createBlockchainCmd.Parsed() {
		if *createBlockchainAddress == "" {
			createBlockchainCmd.Usage()
			runtime.Goexit()
		}
		cli.createBlockchain(*createBlockchainAddress)
	}

	if sendCmd.Parsed() {
		if *sendFromAddress == "" || *sendToAddress == "" || *sendAmount <= 0 {
			sendCmd.Usage()
			runtime.Goexit()
		}
		cli.send(*sendFromAddress, *sendToAddress, *sendAmount)
	}

	if createWalletCmd.Parsed() {
		cli.createWallet()
	}

	if listAddressesCmd.Parsed() {
		cli.listAddresses()
	}
}
