package p2pv2

import "fmt"

type blankValidator struct{}

func (blankValidator) Validate(_ string, _ []byte) error        { return nil }
func (blankValidator) Select(_ string, _ [][]byte) (int, error) { return 0, nil }

func Key(account string) string {
	return fmt.Sprintf("/%s/account/%s", namespace, account)
}
