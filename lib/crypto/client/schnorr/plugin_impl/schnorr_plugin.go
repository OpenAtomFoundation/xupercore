package main

import "github.com/xuperchain/xupercore/lib/crypto/client/schnorr"

// GetInstance returns the an instance of SchnorrCryptoClient
func GetInstance() interface{} {
	return &schnorr.SchnorrCryptoClient{}
}
