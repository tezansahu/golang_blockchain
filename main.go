package main

import (
	"fmt"
	"strconv"

	"github.com/tezansahu/golang_blockchain/blockchain"
)

func main() {
	chain := blockchain.InitBlockchain()
	chain.AddBlock("First Block")
	chain.AddBlock("Second Block")

	for _, block := range chain.Blocks {
		fmt.Printf("Previous Hash: %x\n", block.PrevHash)
		fmt.Printf("Data: %s\n", block.Data)
		fmt.Printf("Hash: %x\n", block.Hash)
		fmt.Printf("Nonce: %d\n", block.Nonce)

		pow := blockchain.NewProof(block)
		fmt.Printf("PoW: %s\n", strconv.FormatBool(pow.Validate()))
		fmt.Println()
	}
}
