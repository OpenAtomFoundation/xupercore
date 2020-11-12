package utils

import (
	"fmt"
	"strings"
)

func IsAccount(name string) int {
	if name == "" {
		return -1
	}
	if !strings.HasPrefix(name, GetAccountPrefix()) {
		return 0
	}
	prefix := strings.Split(name, GetAccountBcnameSep())[0]
	prefix = prefix[len(GetAccountPrefix()):]
	if err := ValidRawAccount(prefix); err != nil {
		return 0
	}
	return 1
}

// ValidRawAccount validate account number
func ValidRawAccount(accountName string) error {
	// param absence check
	if accountName == "" {
		return fmt.Errorf("invoke NewAccount failed, account name is empty")
	}
	// account naming rule check
	if len(accountName) != GetAccountSize() {
		return fmt.Errorf("invoke NewAccount failed, account name length expect %d, actual: %d", GetAccountSize(), len(accountName))
	}
	for i := 0; i < GetAccountSize(); i++ {
		if accountName[i] >= '0' && accountName[i] <= '9' {
			continue
		} else {
			return fmt.Errorf("invoke NewAccount failed, account name expect continuous %d number", GetAccountSize())
		}
	}
	return nil
}
