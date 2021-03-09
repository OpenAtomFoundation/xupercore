package contract

import (
	"fmt"
	"regexp"
)

var (
	contractNameRegex = regexp.MustCompile("^[a-zA-Z_]{1}[0-9a-zA-Z_.]+[0-9a-zA-Z_]$")
)

const (
	contractNameMaxSize = 16
	contractNameMinSize = 4
)

// ValidContractName return error when contractName is not a valid contract name.
func ValidContractName(contractName string) error {
	// param absence check
	// contract naming rule check
	contractSize := len(contractName)
	contractMaxSize := contractNameMaxSize
	contractMinSize := contractNameMinSize
	if contractSize > contractMaxSize || contractSize < contractMinSize {
		return fmt.Errorf("contract name length expect [%d~%d], actual: %d", contractMinSize, contractMaxSize, contractSize)
	}
	if !contractNameRegex.MatchString(contractName) {
		return fmt.Errorf("contract name does not fit the rule of contract name")
	}
	return nil
}
