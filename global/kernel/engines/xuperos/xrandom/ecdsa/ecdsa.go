package ecdsa

import (
	"crypto/ecdsa"
	"fmt"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/pkg/errors"
)

var (
	privateKey *ecdsa.PrivateKey
)

func SetAccount(private *ecdsa.PrivateKey) {
	privateKey = private
}

func PublicKeyFromECDSA(pub *ecdsa.PublicKey) string {
	return fmt.Sprintf("%#x", crypto.CompressPubkey(pub))
}

func Sign(msg string) ([]byte, error) {
	if privateKey == nil {
		return nil, errors.New("private key not inited")
	}
	message := crypto.Keccak256([]byte(msg))
	return crypto.Sign(message, privateKey)
}

func Verify(pubKey, msg, sign []byte) bool {
	publicKey, err := hexutil.Decode(string(pubKey))
	if err != nil {
		return false
	}
	message := crypto.Keccak256(msg)
	if len(sign) < crypto.RecoveryIDOffset {
		return false
	}
	sign = sign[:crypto.RecoveryIDOffset]
	return crypto.VerifySignature(publicKey, message, sign)
}
