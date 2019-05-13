package blockchain

import (
	"bytes"

	"github.com/tezansahu/golang_blockchain/wallet"
)

type Transaction struct {
	ID      []byte
	Inputs  []TxInput
	Outputs []TxOutput
}

type TxOutput struct {
	Value      int
	PubKeyHash []byte
}

type TxInput struct {
	ID        []byte
	Out       int
	Signature []byte
	PubKey    []byte
}

func (in *TxInput) UsesKey(pubKeyHash []byte) bool {
	lockingHash := wallet.PublicKeyHash(in.PubKey)

	return bytes.Compare(lockingHash, pubKeyHash) == 0
}

func (out *TxOutput) Lock(address []byte) {
	pubKeyHash := wallet.Base58Decode(address)
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-wallet.ChecksumLength]

	out.PubKeyHash = pubKeyHash
}

func (out *TxOutput) IsLockedWithKey(pubKeyHash []byte) bool {
	return bytes.Compare(out.PubKeyHash, pubKeyHash) == 0
}

func NewTXOutput(value int, address string) *TxOutput {
	txo := TxOutput{value, nil}

	txo.Lock([]byte(address))
	return &txo
}
