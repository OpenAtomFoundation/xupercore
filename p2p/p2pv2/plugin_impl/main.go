package main

import (
	"github.com/xuperchain/xupercore/p2p/p2pv2"
)

// GetInstance returns the an instance of SchnorrCryptoClient
func GetInstance() interface{} {
	return &p2pv2.P2PServerV2{}
}
