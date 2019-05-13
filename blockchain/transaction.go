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

	"github.com/tezansahu/golang_blockchain/wallet"
)

// Function to serialize the transaction structure into bytes
func (txn *Transaction) Serialize() []byte {
	var res bytes.Buffer
	encoder := gob.NewEncoder(&res)

	err := encoder.Encode(txn)

	Handle(err)

	return res.Bytes()
}

// Hash the data within a transaction after serializing it
func (txn *Transaction) Hash() []byte {
	var hash [32]byte

	txCopy := *txn
	txCopy.ID = []byte{}

	hash = sha256.Sum256(txCopy.Serialize())

	return hash[:]
}

// func (tx *Transaction) SetID() {
// 	var hash [32]byte
// 	var encoded bytes.Buffer

// 	encoder := gob.NewEncoder(&encoded)
// 	err := encoder.Encode(tx)
// 	Handle(err)

// 	hash = sha256.Sum256(encoded.Bytes())

// 	tx.ID = hash[:]
// }

// Create a Coinbase Transaction for user creating the blockchain
func CoinbaseTx(to, data string) *Transaction {
	if data == "" {
		data = fmt.Sprintf("Coins to %s", to)
	}

	txInput := TxInput{[]byte{}, -1, nil, []byte(data)}
	txOutput := NewTXOutput(100, to)

	txn := Transaction{nil, []TxInput{txInput}, []TxOutput{*txOutput}}
	txn.ID = txn.Hash()

	return &txn
}

// Check is a transaction is a coinbase transaction
func (tx *Transaction) IsCoinbase() bool {
	return len(tx.Inputs) == 1 && len(tx.Inputs[0].ID) == 0 && tx.Inputs[0].Out == -1
}

// Create a new transaction
func NewTransaction(from, to string, amount int, chain *Blockchain) *Transaction {
	var inputs []TxInput
	var outputs []TxOutput

	// Get sending user's data from the wallets
	wallets, err := wallet.CreateWallets()
	Handle(err)
	w := wallets.GetWallet(from)
	pubKeyHash := wallet.PublicKeyHash(w.PublicKey)

	// Get spendable outputs of the sending user
	acc, validOutputs := chain.FindSpendableOutputs(pubKeyHash, amount)

	// Check if enough funds are available for transfer
	if acc < amount {
		log.Panic("Error: Funds not enough")
	}

	// Use the spendable outputs to create Transaction Inputs for the current transaction
	for txid, outs := range validOutputs {
		txID, err := hex.DecodeString(txid)
		Handle(err)

		for _, out := range outs {
			input := TxInput{txID, out, nil, w.PublicKey}
			inputs = append(inputs, input)
		}
	}

	// Create Transaction Output transferring the required amount to receiver
	outputs = append(outputs, *NewTXOutput(amount, to))

	// Create Transaction Output transferring the excess accumulated amount back to sender
	if acc > amount {
		outputs = append(outputs, *NewTXOutput(acc-amount, from))
	}

	tx := Transaction{nil, inputs, outputs}
	tx.ID = tx.Hash()

	// Sign the transaction with sender's Private Key
	chain.SignTransaction(&tx, w.PrivateKey)

	return &tx

}

// Function to create a customised copy of a transaction
func (tx *Transaction) TrimmedCopy() Transaction {
	var inputs []TxInput
	var outputs []TxOutput

	// Use only ID and Out Indices of every input, and set signature and public key as nil
	for _, in := range tx.Inputs {
		inputs = append(inputs, TxInput{in.ID, in.Out, nil, nil})
	}

	for _, out := range tx.Outputs {
		outputs = append(outputs, TxOutput{out.Value, out.PubKeyHash})
	}

	txCopy := Transaction{tx.ID, inputs, outputs}

	return txCopy
}

// Sign a transaction using user's private key
func (tx *Transaction) Sign(privKey ecdsa.PrivateKey, prevTXs map[string]Transaction) {

	// If transaction is a coinbase trnasaction, no need to sign
	if tx.IsCoinbase() {
		return
	}

	// Check if the Transaction Inputs reference valid previous transactions
	for _, in := range tx.Inputs {
		if prevTXs[hex.EncodeToString(in.ID)].ID == nil {
			log.Panic("ERROR: Previous transaction does not exist!")
		}
	}

	// Make a trimmed copy of the transaction
	txCopy := tx.TrimmedCopy()

	// Iterate through the inputs of the cpoied transaction
	for inId, in := range txCopy.Inputs {
		prevTx := prevTXs[hex.EncodeToString(in.ID)]
		txCopy.Inputs[inId].Signature = nil                            // Double check to see that Signature is nil
		txCopy.Inputs[inId].PubKey = prevTx.Outputs[in.Out].PubKeyHash // Set the PubKey of input copy as PubKeyHash of referenced output
		txCopy.ID = txCopy.Hash()                                      // Now hash the transaction copy and set it as ID of copy
		txCopy.Inputs[inId].PubKey = nil                               // Again change the PubKey field of current input to nil

		// Get the signature on the ID of the transaction copy
		r, s, err := ecdsa.Sign(rand.Reader, &privKey, txCopy.ID)
		Handle(err)

		signature := append(r.Bytes(), s.Bytes()...)

		// Set the value of Signatute of the current Transaction Input using the sign obtained
		tx.Inputs[inId].Signature = signature
	}
}

// Verify a transaction
func (tx *Transaction) Verify(prevTXs map[string]Transaction) bool {

	// Return true ifit is a  coinbase trnasaction
	if tx.IsCoinbase() {
		return true
	}

	// Check if the Transaction Inputs reference valid previous transactions
	for _, in := range tx.Inputs {
		if prevTXs[hex.EncodeToString(in.ID)].ID == nil {
			log.Panic("ERROR: Previous transaction does not exist!")
		}
	}

	txCopy := tx.TrimmedCopy()
	curve := elliptic.P256()

	for inId, in := range tx.Inputs {
		prevTx := prevTXs[hex.EncodeToString(in.ID)]
		txCopy.Inputs[inId].Signature = nil
		txCopy.Inputs[inId].PubKey = prevTx.Outputs[in.Out].PubKeyHash
		txCopy.ID = txCopy.Hash()
		txCopy.Inputs[inId].PubKey = nil

		// Obtain r and s from the Transaction Input Signature
		r := big.Int{}
		s := big.Int{}
		sigLen := len(in.Signature)
		r.SetBytes(in.Signature[:(sigLen / 2)])
		s.SetBytes(in.Signature[(sigLen / 2):])

		// Obtain x and y from the Public Key of the owner of
		// the Transaction Output referenced by the input
		x := big.Int{}
		y := big.Int{}
		keyLen := len(in.PubKey)
		x.SetBytes(in.PubKey[:(keyLen / 2)])
		y.SetBytes(in.PubKey[(keyLen / 2):])

		rawPubKey := ecdsa.PublicKey{curve, &x, &y}

		// Verify the signature
		if ecdsa.Verify(&rawPubKey, in.ID, &r, &s) == false {
			return false
		}
	}
	return true
}

// function to print the transaction in desired format
func (tx *Transaction) String() string {
	var lines []string

	lines = append(lines, fmt.Sprintf("---Transaction %x: ", tx.ID))
	for i, input := range tx.Inputs {
		lines = append(lines, fmt.Sprintf("    Input %d:", i))
		lines = append(lines, fmt.Sprintf("        TxID:      %x", input.ID))
		lines = append(lines, fmt.Sprintf("        Out:       %d", input.Out))
		lines = append(lines, fmt.Sprintf("        Signature: %x", input.Signature))
		lines = append(lines, fmt.Sprintf("        PubKey:    %x", input.PubKey))
	}

	for i, output := range tx.Outputs {
		lines = append(lines, fmt.Sprintf("    Output %d:", i))
		lines = append(lines, fmt.Sprintf("        Value:  %d", output.Value))
		lines = append(lines, fmt.Sprintf("        Script: %x", output.PubKeyHash))
	}

	return strings.Join(lines, "\n")
}
