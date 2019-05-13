package blockchain

import (
	"bytes"

	"github.com/tezansahu/golang_blockchain/wallet"
)

// Structure of a transaction in a block
type Transaction struct {
	ID      []byte
	Inputs  []TxInput
	Outputs []TxOutput
}

// Structure of a Transaction Output
type TxOutput struct {
	Value      int    // Number of tokens that the output accounts for
	PubKeyHash []byte // Hash of Public Key of owner of the output tokens
}

// Structure of a Transaction Input
type TxInput struct {
	ID        []byte // ID of the Transaction whose output is being referenced
	Out       int    // Output Index of the Transaction Outputs slice being referenced
	Signature []byte // Owner's sign to acknowledge the usage of referenced output as input for this txn
	PubKey    []byte // Public Key of owner of the tokens
}

// Check if a Transaction Input uses Public Key of calling user
func (in *TxInput) UsesKey(pubKeyHash []byte) bool {
	lockingHash := wallet.PublicKeyHash(in.PubKey)

	return bytes.Compare(lockingHash, pubKeyHash) == 0
}

// Lock a Transaction Output using Public Key Hash of calling user
func (out *TxOutput) Lock(address []byte) {
	// Get the public key hash of the user from his wallet data
	pubKeyHash := wallet.Base58Decode(address)
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-wallet.ChecksumLength]

	// Lock the transaction output using this public key hash
	out.PubKeyHash = pubKeyHash
}

// Check if a Transaction Output is locked with Public Key Hash of calling user
func (out *TxOutput) IsLockedWithKey(pubKeyHash []byte) bool {
	return bytes.Compare(out.PubKeyHash, pubKeyHash) == 0
}

// Create a new Transaction Output
func NewTXOutput(value int, address string) *TxOutput {
	txo := TxOutput{value, nil}

	txo.Lock([]byte(address))
	return &txo
}
