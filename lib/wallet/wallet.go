package wallet

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"

	// "fmt"

	// "tensor/lib/blockchain"

	"golang.org/x/crypto/ripemd160"
)


type Wallet struct {
	PrivateKey						ecdsa.PrivateKey
	PublicKey						[]byte
}


const (
	checksumLength = 4
	version =   byte(0x00)
)


func NewKeyPair() (ecdsa.PrivateKey, []byte){
	curve := elliptic.P256()

	private, err := ecdsa.GenerateKey(curve, rand.Reader)
	HandleError(err)

	pub := append(private.PublicKey.X.Bytes(), private.PublicKey.Y.Bytes()...)

	return *private, pub
}


func MakeWallet() *Wallet {
	private, public := NewKeyPair()
	wallet := Wallet{PrivateKey: private, PublicKey: public}

	return &wallet
}


func PubKeyHash(pubKey []byte) []byte {
	pubHash := sha256.Sum256(pubKey)

	hasher := ripemd160.New()
	_, err := hasher.Write(pubHash[:])
	HandleError(err)

	publicRipMD := hasher.Sum(nil)

	return publicRipMD
}


func Checksum(payload []byte) []byte {
	firstHash := sha256.Sum256(payload)
	secondHash := sha256.Sum256(firstHash[:])

	return secondHash[:checksumLength]
}


func (wallet *Wallet) Address() []byte {
	pubHash := PubKeyHash(wallet.PublicKey)

	vasionedHash := append([]byte{version}, pubHash...)

	checksum := Checksum(vasionedHash)

	fullHash := append(vasionedHash, checksum...)

	address := Base58Encode(fullHash)

	// fmt.Printf("PublicKey:   %x\n", wallet.PublicKey)
	// fmt.Printf("PubHash:   %x\n", pubHash)
	// fmt.Printf("Address:   %x\n", address)

	return address
}


func ValidateAddress(address string) bool {
	PubKeyHash := Base58Decode([]byte(address))
	actualChecksum := PubKeyHash[len(PubKeyHash)-checksumLength:]
	version := PubKeyHash[0]
	PubKeyHash = PubKeyHash[1: len(PubKeyHash)-checksumLength]
	targetChecksum := Checksum(append([]byte{version}, PubKeyHash...))

	return bytes.Compare(actualChecksum, targetChecksum) == 0
}
