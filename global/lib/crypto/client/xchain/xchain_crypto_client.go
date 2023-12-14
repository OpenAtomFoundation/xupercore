// Package eccdefault is the default crypto client of xchain
package eccdefault

import (
	"github.com/OpenAtomFoundation/xupercore/global/lib/crypto/client/base"
	"github.com/xuperchain/crypto/client/service/xchain"
)

// make sure this plugin implemented the interface
var _ base.CryptoClient = (*XchainCryptoClient)(nil)

// XchainCryptoClient is the implementation for xchain default crypto
type XchainCryptoClient struct {
	xchain.XchainCryptoClient
}

func GetInstance() base.CryptoClient {
	xcCryptoClient := XchainCryptoClient{}
	return &xcCryptoClient
}
