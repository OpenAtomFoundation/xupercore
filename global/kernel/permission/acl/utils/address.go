package utils

import (
	"fmt"
	"strings"
)

type AddressType int

const (
	AddressAK AddressType = iota
	AddressAccount
)

// ParseAddressType returns address type when address is valid
// A valid address is not empty which is AK or Account.
// A valid Account pattern is `<AccountPrefix><AccountNumber>[@<BlockChainName>]`
func ParseAddressType(address string) (addressType AddressType, isValid bool) {
	if address == "" {
		// invalid address
		return
	}
	isValid = true
	addressType = AddressAK

	// check if address matches account
	// check account prefix
	if !strings.HasPrefix(address, GetAccountPrefix()) {
		return
	}
	// check account number
	number := strings.Split(address, GetAccountBcNameSep())[0]
	number = number[len(GetAccountPrefix()):]
	if err := ValidAccountNumber(number); err != nil {
		return
	}

	addressType = AddressAccount
	return
}

// IsAccount returns true for a valid account
func IsAccount(address string) bool {
	t, isValid := ParseAddressType(address)
	return isValid && t == AddressAccount
}

// IsAK returns true for a valid AK
func IsAK(address string) bool {
	t, isValid := ParseAddressType(address)
	return isValid && t == AddressAK
}

// Deprecating, use ValidAccountNumber instead
func ValidRawAccount(number string) error {
	return ValidAccountNumber(number)
}

// ValidAccountNumber validate account number
// a valid account number pattern is `[0-9]{16}`
func ValidAccountNumber(number string) error {
	// param absence check
	if number == "" {
		return fmt.Errorf("invoke NewAccount failed, account number is empty")
	}

	// check number size
	if len(number) != GetAccountNumberSize() {
		return fmt.Errorf("invoke NewAccount failed, account number length expect %d, actual: %d",
			GetAccountNumberSize(), len(number))
	}

	// check number's digit
	for _, digit := range number {
		if digit < '0' || digit > '9' {
			return fmt.Errorf("invoke NewAccount failed, account number expect continuous %d digits",
				GetAccountNumberSize())
		}
	}
	return nil
}
