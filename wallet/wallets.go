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

// File where data about wallets will be stored
const walletFile = "./tmp/wallets.data"

// Structure of Wallets
type Wallets struct {
	Wallets map[string]*Wallet
}

// Save the wallets to the file
func (ws *Wallets) SaveFile() {
	var content bytes.Buffer

	gob.Register(elliptic.P256())

	encoder := gob.NewEncoder(&content)
	err := encoder.Encode(ws)
	if err != nil {
		log.Panic(err)
	}

	err = ioutil.WriteFile(walletFile, content.Bytes(), 0644)
	if err != nil {
		log.Panic(err)
	}
}

// Load wallets from the file
func (ws *Wallets) LoadFile() error {
	if _, err := os.Stat(walletFile); err != nil {
		return err
	}

	var wallets Wallets
	fileContent, err := ioutil.ReadFile(walletFile)
	if err != nil {
		return err
	}

	gob.Register(elliptic.P256())

	decoder := gob.NewDecoder(bytes.NewReader(fileContent))
	err = decoder.Decode(&wallets)
	if err != nil {
		return err
	}

	ws.Wallets = wallets.Wallets

	return nil
}

// Create wallets using existing data from the file
func CreateWallets() (*Wallets, error) {
	wallets := Wallets{}
	wallets.Wallets = make(map[string]*Wallet)

	err := wallets.LoadFile()

	return &wallets, err
}

// Get the wallet belonging to an address
func (ws *Wallets) GetWallet(address string) Wallet {
	return *ws.Wallets[address]
}

// Get data of all addresses (of all wallets) saved
func (ws *Wallets) GetAllAddresses() []string {
	var addresses []string
	for address := range ws.Wallets {
		addresses = append(addresses, address)
	}

	return addresses
}

// Add a new wallet to the wallets
func (ws *Wallets) AddWallet() string {
	wallet := MakeWallet()
	address := fmt.Sprintf("%s", wallet.Address())

	ws.Wallets[address] = wallet
	return address
}
