package address

import (
	"fmt"
	"strings"

	"github.com/btcsuite/btcutil/base58"
	"github.com/hyperledger/burrow/binary"
	"github.com/hyperledger/burrow/crypto"

	"github.com/xuperchain/xupercore/kernel/contract"
	"github.com/xuperchain/xupercore/kernel/permission/acl/utils"
	"github.com/xuperchain/xupercore/lib/crypto/hash"
)

const (
	evmAddressFiller = "-"

	contractNamePrefix    = "1111"
	contractAccountPrefix = "1112"
)

const (
	XchainAddrType      = "xchain"
	ContractNameType    = "contract-name"
	ContractAccountType = "contract-account"
)

// XchainToEVMAddress transfer xchain address to evm address
func XchainToEVMAddress(addr string) (crypto.Address, error) {
	rawAddr := base58.Decode(addr)
	if len(rawAddr) < 21 {
		return crypto.ZeroAddress, fmt.Errorf("%s is not a valid address", addr)
	}
	ripemd160Hash := rawAddr[1:21]
	return crypto.AddressFromBytes(ripemd160Hash)
}

// EVMAddressToXchain transfer evm address to xchain address
func EVMAddressToXchain(evmAddress crypto.Address) (string, error) {
	addrType := 1
	nVersion := uint8(addrType)
	bufVersion := []byte{nVersion}

	outputRipemd160 := evmAddress.Bytes()

	strSlice := make([]byte, len(bufVersion)+len(outputRipemd160))
	copy(strSlice, bufVersion)
	copy(strSlice[len(bufVersion):], outputRipemd160)

	checkCode := hash.DoubleSha256(strSlice)
	simpleCheckCode := checkCode[:4]
	slice := make([]byte, len(strSlice)+len(simpleCheckCode))
	copy(slice, strSlice)
	copy(slice[len(strSlice):], simpleCheckCode)

	return base58.Encode(slice), nil
}

// ContractNameToEVMAddress transfer contract name to evm address
func ContractNameToEVMAddress(contractName string) (crypto.Address, error) {
	contractNameLength := len(contractName)
	var prefixStr string
	for i := 0; i < binary.Word160Length-contractNameLength-utils.GetContractNameMinSize(); i++ {
		prefixStr += evmAddressFiller
	}
	contractName = prefixStr + contractName
	contractName = contractNamePrefix + contractName
	return crypto.AddressFromBytes([]byte(contractName))
}

// EVMAddressToContractName transfer evm address to contract name
func EVMAddressToContractName(evmAddr crypto.Address) (string, error) {
	contractNameWithPrefix := evmAddr.Bytes()
	contractNameStrWithPrefix := string(contractNameWithPrefix)
	prefixIndex := strings.LastIndex(contractNameStrWithPrefix, evmAddressFiller)
	if prefixIndex == -1 {
		return contractNameStrWithPrefix[4:], nil
	}
	return contractNameStrWithPrefix[prefixIndex+1:], nil
}

// ContractAccountToEVMAddress transfer contract account to evm address
func ContractAccountToEVMAddress(contractAccount string) (crypto.Address, error) {
	contractAccountValid := contractAccount[2:18]
	contractAccountValid = contractAccountPrefix + contractAccountValid
	return crypto.AddressFromBytes([]byte(contractAccountValid))
}

// EVMAddressToContractAccount transfer evm address to contract account
func EVMAddressToContractAccount(evmAddr crypto.Address) (string, error) {
	contractNameWithPrefix := evmAddr.Bytes()
	contractNameStrWithPrefix := string(contractNameWithPrefix)
	return utils.GetAccountPrefix() + contractNameStrWithPrefix[4:] + "@xuper", nil
}

// EVMAddressToContractAccountWithoutPrefixAndSuffix 返回的合约账户不包括前缀XC和后缀@xuper（或其他链名）
func EVMAddressToContractAccountWithoutPrefixAndSuffix(evmAddr crypto.Address) (string, error) {
	contractNameWithPrefix := evmAddr.Bytes()
	contractNameStrWithPrefix := string(contractNameWithPrefix)
	return contractNameStrWithPrefix[4:], nil
}

// IsContractAccount returns true for a contract account
func IsContractAccount(account string) bool {
	return utils.IsAccount(account) && strings.Contains(account, "@")
}

// IsContractName determine whether it is a contract name
func IsContractName(contractName string) bool {
	return contract.ValidContractName(contractName) == nil
}

// DetermineContractNameFromEVM determine whether it is a contract name
func DetermineContractNameFromEVM(evmAddr crypto.Address) (string, error) {
	var addr string
	var err error

	evmAddrWithPrefix := evmAddr.Bytes()
	evmAddrStrWithPrefix := string(evmAddrWithPrefix)
	if evmAddrStrWithPrefix[0:4] != contractNamePrefix {
		return "", fmt.Errorf("not a valid contract name from evm")
	} else {
		addr, err = EVMAddressToContractName(evmAddr)
	}

	if err != nil {
		return "", err
	}

	return addr, nil
}

// DetermineEVMAddress determine an EVM address
func DetermineEVMAddress(evmAddr crypto.Address) (string, string, error) {
	evmAddrWithPrefix := evmAddr.Bytes()
	evmAddrStrWithPrefix := string(evmAddrWithPrefix)

	var addr, addrType string
	var err error
	if evmAddrStrWithPrefix[0:4] == contractAccountPrefix {
		// 此时 addr 不包括前缀和后缀！
		addr, err = EVMAddressToContractAccountWithoutPrefixAndSuffix(evmAddr)
		addrType = ContractAccountType
	} else if evmAddrStrWithPrefix[0:4] == contractNamePrefix {
		addr, err = EVMAddressToContractName(evmAddr)
		addrType = ContractNameType
	} else {
		addr, err = EVMAddressToXchain(evmAddr)
		addrType = XchainAddrType
	}
	if err != nil {
		return "", "", err
	}

	return addr, addrType, nil
}

// DetermineXchainAddress determine an xchain address
func DetermineXchainAddress(xAddr string) (string, string, error) {
	var addr crypto.Address
	var addrType string
	var err error
	if IsContractAccount(xAddr) {
		addr, err = ContractAccountToEVMAddress(xAddr)
		addrType = ContractAccountType
	} else if IsContractName(xAddr) {
		addr, err = ContractNameToEVMAddress(xAddr)
		addrType = ContractNameType
	} else {
		addr, err = XchainToEVMAddress(xAddr)
		addrType = XchainAddrType
	}
	if err != nil {
		return "", "", err
	}

	return addr.String(), addrType, nil
}
