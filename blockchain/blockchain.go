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

// Define paths where blockchain data will be stored
const (
	dbPath      = "./tmp/blocks"
	dbFile      = "./tmp/blocks/MANIFEST"
	genesisData = "First Transaction from Genesis"
)

// Structure of the blockchain
type Blockchain struct {
	LastHash []byte
	Database *badger.DB
}

// Iterator to iterate through the blockchain
type BlockchainIterator struct {
	CurrentHash []byte
	Database    *badger.DB // This database stores blockdata and metadata as key-value pairs
}

// Check if the Database containing information about the blockchain exists
func DBexists() bool {
	if _, err := os.Stat(dbFile); os.IsNotExist(err) {
		return false
	}

	return true
}

// Initialize a blockchain
func InitBlockchain(address string) *Blockchain {
	var lastHash []byte

	if DBexists() {
		fmt.Println("Blockchain already exists!")
		runtime.Goexit()
	}

	// Set required options for the Badger Database
	opts := badger.DefaultOptions
	opts.Dir = dbPath
	opts.ValueDir = dbPath

	db, err := badger.Open(opts)
	Handle(err)

	// Update the database with a Coinbase Txn
	err = db.Update(func(txn *badger.Txn) error {

		cbtx := CoinbaseTx(address, genesisData)
		genesis := Genesis(cbtx)
		fmt.Println("Genesis Created")

		// create a new pair with key as hash of genesis block,
		// and value as the serialized data of the genesis block
		err := txn.Set(genesis.Hash, genesis.Serialize())
		Handle(err)

		// create a new pair with key a "lh" (last hash) and
		// value as the hash of genesis block
		err = txn.Set([]byte("lh"), genesis.Hash)

		lastHash = genesis.Hash
		return err
	})

	Handle(err)

	blockchain := Blockchain{lastHash, db}
	return &blockchain
}

// Continue the already existing blockchain
func ContinueBlockchain(address string) *Blockchain {
	var lastHash []byte

	if DBexists() == false {
		fmt.Println("Blockchain does not exist; Create one!")
		runtime.Goexit()
	}

	// Set required options for the Badger Database
	opts := badger.DefaultOptions
	opts.Dir = dbPath
	opts.ValueDir = dbPath

	db, err := badger.Open(opts)
	Handle(err)

	// Get the details about the latest block in the blockchain from the database
	err = db.Update(func(txn *badger.Txn) error {
		// Use the "lh" (last hash) key to obtain required data
		item, err := txn.Get([]byte("lh"))
		Handle(err)
		lastHash, err = item.Value()
		return err
	})

	Handle(err)

	// Set the current state of the blockchain using data obtained from the database
	blockchain := Blockchain{lastHash, db}
	return &blockchain

}

// Add a block with the given transactions to the blockchain
func (chain *Blockchain) AddBlock(transactions []*Transaction) {
	var lastHash []byte

	// Obtain the last block hash from the database
	err := chain.Database.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("lh"))
		Handle(err)
		lastHash, err = item.Value()
		return err
	})

	Handle(err)

	newBlock := CreateBlock(transactions, lastHash)

	// Add the data of the newly created block to the database
	err = chain.Database.Update(func(txn *badger.Txn) error {
		err := txn.Set(newBlock.Hash, newBlock.Serialize())
		Handle(err)
		err = txn.Set([]byte("lh"), newBlock.Hash)

		chain.LastHash = newBlock.Hash

		return err
	})

	Handle(err)
}

// Create an iterator for a blockchain
func (chain *Blockchain) Iterator() *BlockchainIterator {
	iter := &BlockchainIterator{chain.LastHash, chain.Database}

	return iter
}

// Use the iterator to get data about the next block (actually, previous block in the chain)
func (iter *BlockchainIterator) Next() *Block {
	var block *Block

	err := iter.Database.View(func(txn *badger.Txn) error {
		item, err := txn.Get(iter.CurrentHash)
		Handle(err)
		encodedBlock, err := item.Value()

		block = Deserialize(encodedBlock)

		return err
	})

	Handle(err)

	iter.CurrentHash = block.PrevHash

	return block
}

// Find all unspent transactions for a given user
func (chain *Blockchain) FindUnspentTransactions(pubKeyHash []byte) []Transaction {
	var unspentTx []Transaction

	// Map to store the spent transactions
	spentTXOs := make(map[string][]int)

	iter := chain.Iterator()

	for {
		// Get the next block in the chain
		block := iter.Next()

		// Iterate through all the transactions present in the block
		for _, tx := range block.Transactions {
			txId := hex.EncodeToString(tx.ID)

			// Iterate through all outputs of a transaction
		Outputs:
			for outIdx, out := range tx.Outputs {
				// If some outputs of the transaction are spent, iterate through them
				if spentTXOs[txId] != nil {
					for _, spentOut := range spentTXOs[txId] {
						// If the current output has already been spent,
						// directly start iterating with the next output
						if spentOut == outIdx {
							continue Outputs
						}
					}
				}

				// If the output can be unlocked by the calling user,
				// add it to list of unspent transactions
				if out.IsLockedWithKey(pubKeyHash) {
					unspentTx = append(unspentTx, *tx)
				}
			}

			// If the transaction is not a Coinbase Transaction, iterate through its inputs
			if tx.IsCoinbase() == false {
				for _, in := range tx.Inputs {
					// If the Transaction Input uses the calling user's key,
					// append it to the list of spent transactions
					if in.UsesKey(pubKeyHash) {
						inTxID := hex.EncodeToString(in.ID)
						spentTXOs[inTxID] = append(spentTXOs[inTxID], in.Out)
					}
				}
			}
		}
		// Exit the loop is coinbase transaction is reached
		if len(block.PrevHash) == 0 {
			break
		}
	}

	return unspentTx
}

// Find all Unspent Transaction Outputs for a user
func (chain *Blockchain) FindUTXO(pubKeyHash []byte) []TxOutput {
	var UTXOs []TxOutput
	// First, find all unspent transaction for the user
	unspentTransactions := chain.FindUnspentTransactions(pubKeyHash)

	// Iterate through the above transactions and
	// obtain the unspent outputs for the user
	for _, tx := range unspentTransactions {
		for _, out := range tx.Outputs {
			if out.IsLockedWithKey(pubKeyHash) {
				UTXOs = append(UTXOs, out)
			}
		}
	}

	return UTXOs
}

// Given a user and amount to be spent, find all possible transaction outputs that can be spent
func (chain *Blockchain) FindSpendableOutputs(pubKeyHash []byte, amount int) (int, map[string][]int) {

	// Store list of spendable outputs
	unspentOutputs := make(map[string][]int)

	// Find all unspent transaction for the user
	unspentTransactions := chain.FindUnspentTransactions(pubKeyHash)
	accumulated := 0 // Store total amount accumulated by combining the unspent transactions

Work:
	for _, tx := range unspentTransactions {
		txId := hex.EncodeToString(tx.ID)

		for outIdx, out := range tx.Outputs {
			// If transaction output is locked using user's key and
			// accumulated amount is less than amount to be spent,
			// append this to the spendable outputs and update the accumulated amount
			if out.IsLockedWithKey(pubKeyHash) && accumulated < amount {
				accumulated += out.Value
				unspentOutputs[txId] = append(unspentOutputs[txId], outIdx)

				// check if required amount has already been accumulated
				if accumulated >= amount {
					break Work
				}
			}
		}
	}

	return accumulated, unspentOutputs
}

// Find a transaction using its ID from a blockchain
func (bc *Blockchain) FindTransaction(ID []byte) (Transaction, error) {
	iter := bc.Iterator()
	for {
		block := iter.Next()

		for _, tx := range block.Transactions {
			if bytes.Compare(ID, tx.ID) == 0 {
				return *tx, nil
			}
		}

		if len(block.PrevHash) == 0 {
			break
		}

	}

	return Transaction{}, errors.New("Transaction does not exist")
}

// Sign a transaction using the user's private key
func (bc *Blockchain) SignTransaction(tx *Transaction, privKey ecdsa.PrivateKey) {
	prevTXs := make(map[string]Transaction)

	for _, in := range tx.Inputs {
		prevTx, err := bc.FindTransaction(in.ID)
		Handle(err)
		prevTXs[hex.EncodeToString(prevTx.ID)] = prevTx
	}

	tx.Sign(privKey, prevTXs)
}

// Verify the signature of a transaction
func (bc *Blockchain) VerifyTransaction(tx *Transaction) bool {
	prevTXs := make(map[string]Transaction)

	for _, in := range tx.Inputs {
		prevTx, err := bc.FindTransaction(in.ID)
		Handle(err)
		prevTXs[hex.EncodeToString(prevTx.ID)] = prevTx
	}

	return tx.Verify(prevTXs)
}
