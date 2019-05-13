package wallet

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"log"

	"golang.org/x/crypto/ripemd160"
)

const (
	ChecksumLength = 4          // Length of checksum
	version        = byte(0x00) // Version of the protocol
)

// Structure for a Wallet
type Wallet struct {
	PrivateKey ecdsa.PrivateKey
	PublicKey  []byte
}

// Validate the address of a user
func ValidateAddress(address string) bool {
	pubKeyHash := Base58Decode([]byte(address))
	actualChecksum := pubKeyHash[len(pubKeyHash)-ChecksumLength:]
	version := pubKeyHash[0]
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-ChecksumLength]
	targetChecksum := Checksum(append([]byte{version}, pubKeyHash...))

	return bytes.Compare(actualChecksum, targetChecksum) == 0
}

// Generate a new Key Pair using ECDSA curve
func NewKeyPair() (ecdsa.PrivateKey, []byte) {
	curve := elliptic.P256()
	private, err := ecdsa.GenerateKey(curve, rand.Reader)

	if err != nil {
		log.Panic(err)
	}

	pub := append(private.PublicKey.X.Bytes(), private.PublicKey.Y.Bytes()...)
	return *private, pub
}

// Make a new wallet
func MakeWallet() *Wallet {
	private, public := NewKeyPair()
	wallet := Wallet{private, public}
	return &wallet
}

// Obtain the hash of a Public Key
func PublicKeyHash(pubkey []byte) []byte {
	pubHash := sha256.Sum256(pubkey) // First, obtain the SHA256 hash of the public key

	// Now, use RipeMD160 to hash the above hash
	hasher := ripemd160.New()
	_, err := hasher.Write(pubHash[:])
	if err != nil {
		log.Panic(err)
	}

	publicRipMD := hasher.Sum(nil)

	return publicRipMD
}

// Get the checksum for a given payload
func Checksum(payload []byte) []byte {
	// Use SHA256 to hash the payload twice
	firstHash := sha256.Sum256(payload)
	secondHash := sha256.Sum256(firstHash[:])

	// The first few bytes of the final hash
	// (as decided by the checksumLength) defines the checksum
	return secondHash[:ChecksumLength]
}

// Generate the Address for a wallet
func (w Wallet) Address() []byte {

	// Obtain the Hash of the Public Key owning the wallet
	pubHash := PublicKeyHash(w.PublicKey)

	// Append the version of protocol to the hash and obtain a checksum
	versionedHash := append([]byte{version}, pubHash...)
	checksum := Checksum(versionedHash)

	// Use the checksum to get the full hash
	fullHash := append(versionedHash, checksum...)

	// Encode the full hash in base58 to obtain the required address
	address := Base58Encode(fullHash)

	// fmt.Printf("Public Key: %x\n", w.PublicKey)
	// fmt.Printf("Public Hash: %x\n", pubHash)
	// fmt.Printf("Address: %x\n", address)

	return address
}
