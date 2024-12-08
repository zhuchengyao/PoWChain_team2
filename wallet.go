package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"math/big"

	"golang.org/x/crypto/ripemd160"
)

const version = byte(0x00)
const addressChecksumLen = 4

type Wallet struct {
	PrivateKey []byte // 仅存储私钥的字节形式
	PublicKey  []byte // 仅存储公钥X,Y拼接的字节形式
}

func NewWallet() *Wallet {
	private, public := newKeyPair()
	w := &Wallet{PrivateKey: private, PublicKey: public}
	return w
}

func newKeyPair() ([]byte, []byte) {
	curve := elliptic.P256()
	priv, _ := ecdsa.GenerateKey(curve, rand.Reader)

	// 私钥D是大整数，需要转换为32字节定长切片（P-256曲线的大小）
	dBytes := priv.D.Bytes()
	privBytes := make([]byte, 32)
	copy(privBytes[32-len(dBytes):], dBytes)

	// 公钥X,Y各32字节，拼接后长度64字节
	xBytes := priv.X.Bytes()
	yBytes := priv.Y.Bytes()
	pubBytes := make([]byte, 64)
	copy(pubBytes[32-len(xBytes):32], xBytes)
	copy(pubBytes[64-len(yBytes):], yBytes)

	return privBytes, pubBytes
}

func (w Wallet) GetAddress() []byte {
	pubHash := HashPubKey(w.PublicKey)
	versionedPayload := append([]byte{version}, pubHash...)
	checksum := checksum(versionedPayload)
	fullPayload := append(versionedPayload, checksum...)
	address := Base58Encode(fullPayload)
	return address
}

func HashPubKey(pubKey []byte) []byte {
	pubHash := sha256.Sum256(pubKey)
	hasher := ripemd160.New()
	hasher.Write(pubHash[:])
	publicRIPEMD160 := hasher.Sum(nil)
	return publicRIPEMD160
}

func checksum(payload []byte) []byte {
	firstHash := sha256.Sum256(payload)
	secondHash := sha256.Sum256(firstHash[:])
	return secondHash[:addressChecksumLen]
}

func ValidateAddress(address string) bool {
	pubKeyHash := Base58Decode([]byte(address))
	actualChecksum := pubKeyHash[len(pubKeyHash)-addressChecksumLen:]
	version := pubKeyHash[0]
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-addressChecksumLen]
	targetChecksum := checksum(append([]byte{version}, pubKeyHash...))
	return bytesEqual(actualChecksum, targetChecksum)
}

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func (w *Wallet) getPrivateKey() ecdsa.PrivateKey {
	curve := elliptic.P256()

	// 公钥X,Y各32字节
	x := new(big.Int).SetBytes(w.PublicKey[:32])
	y := new(big.Int).SetBytes(w.PublicKey[32:64])
	d := new(big.Int).SetBytes(w.PrivateKey)

	priv := ecdsa.PrivateKey{
		PublicKey: ecdsa.PublicKey{Curve: curve, X: x, Y: y},
		D:         d,
	}

	return priv
}
