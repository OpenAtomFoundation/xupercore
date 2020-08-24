package main

import (
	"github.com/xuperchain/xupercore/p2p/p2pv1"
)

// GetInstance returns the an instance of SchnorrCryptoClient
func GetInstance() interface{} {
	return &p2pv1.P2PServerV1{}
}
