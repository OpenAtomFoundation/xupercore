package p2pv2

import (
	"fmt"

	"github.com/libp2p/go-libp2p-core/peer"
)

type blankValidator struct{}

func (blankValidator) Validate(_ string, _ []byte) error        { return nil }
func (blankValidator) Select(_ string, _ [][]byte) (int, error) { return 0, nil }

func Key(account string) string {
	return fmt.Sprintf("/%s/account/%s", namespace, account)
}

func GenAccountKey(account string) string {
	return fmt.Sprintf("/%s/account/%s", namespace, account)
}

func GenPeerIDKey(id peer.ID) string {
	return fmt.Sprintf("/%s/id/%s", namespace, id)
}
