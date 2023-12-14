package contract

import (
	"errors"
	"fmt"
	"regexp"
)

var (
	contractNameRegex = regexp.MustCompile("^[a-zA-Z_]{1}[0-9a-zA-Z_.]+[0-9a-zA-Z_]$")

	// xkernel contract bucket
	blacklistContractName = map[string]bool{
		"XCContract2Account": true,
		"XCAccount2Contract": true,
		"contract":           true,
		"governToken":        true,
		"proposal":           true,
		"timer":              true,
		"XToken":             true,
		"XCAccount":          true,
		"XCAK2Account":       true,
		"XCContract":         true,
	}
)

const (
	contractNameMaxSize = 16
	contractNameMinSize = 4
)

// ValidContractName return error when contractName is not a valid contract name.
func ValidContractName(name string) error {

	// check blacklist
	if blacklistContractName[name] {
		return errors.New("the contract name is in the blacklist, please change the name and try again")
	}
	// check name size
	nameSize := len(name)
	if nameSize > contractNameMaxSize || nameSize < contractNameMinSize {
		return fmt.Errorf("contract name length expect [%d~%d], actual: %d",
			contractNameMinSize, contractNameMaxSize, nameSize)
	}

	// check name pattern
	if !contractNameRegex.MatchString(name) {
		return fmt.Errorf("contract name does not fit the rule of contract name")
	}
	return nil
}
