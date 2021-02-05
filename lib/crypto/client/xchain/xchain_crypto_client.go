// Package eccdefault is the default crypto client of xchain
package eccdefault

import (
	"github.com/xuperchain/crypto/client/service/xchain"
	"github.com/xuperchain/xupercore/lib/crypto/client/base"
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
