package address

import (
	"fmt"
	"strings"

	"github.com/btcsuite/btcutil/base58"
	"github.com/ethereum/go-ethereum/common"
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

type EVMAddress crypto.Address

var ZeroAddress = [20]byte{}

func NewEVMAddressFromContractAccount(account string) (EVMAddress, error) {
	accountNumber := account[2:18]
	evmAddress := contractAccountPrefix + accountNumber
	return newEVMAddressFromString(evmAddress)
}

func NewEVMAddressFromXchainAK(addr string) (EVMAddress, error) {
	rawAddr := base58.Decode(addr)
	if len(rawAddr) < 21 {
		return ZeroAddress, fmt.Errorf("%s is not a valid address", addr)
	}
	ripemd160Hash := rawAddr[1:21]
	return newEVMAddressFromBytes(ripemd160Hash)
}

func NewEVMAddressFromXchainAccount(account string) (EVMAddress, error) {
	if IsContractAccount(account) {
		return NewEVMAddressFromContractAccount(account)
	} else {
		return NewEVMAddressFromXchainAK(account)
	}
}

func newEVMAddressFromString(addr string) (EVMAddress, error) {
	return newEVMAddressFromBytes([]byte(addr))
}

func newEVMAddressFromBytes(addr []byte) (EVMAddress, error) {
	burrowAddress, err := crypto.AddressFromBytes(addr)
	if err != nil {
		return ZeroAddress, err
	}
	return EVMAddress(burrowAddress), nil
}

func (a EVMAddress) Address() common.Address {
	return common.Address(a)
}

func (a EVMAddress) XchainAccount(bcName string) (string, error) {
	addr, addrType, err := DetermineEVMAddress(a.Address())
	if err != nil {
		return "", err
	}
	if addrType == ContractAccountType {
		addr = "XC" + addr + "@" + bcName
	}
	return addr, nil
}

// transfer xchain address to evm address
func XchainAKToEVMAddress(addr string) (crypto.Address, error) {
	rawAddr := base58.Decode(addr)
	if len(rawAddr) < 21 {
		return crypto.ZeroAddress, fmt.Errorf("%s is not a valid address", addr)
	}
	ripemd160Hash := rawAddr[1:21]
	return crypto.AddressFromBytes(ripemd160Hash)
}

// transfer evm address to xchain address
func EVMAddressToXchainAK(evmAddress common.Address) (string, error) {
	addrType := 1
	nVersion := uint8(addrType)
	bufVersion := []byte{byte(nVersion)}

	outputRipemd160 := evmAddress.Bytes()

	strSlice := make([]byte, len(bufVersion)+len(outputRipemd160))
	copy(strSlice, bufVersion)
	copy(strSlice[len(bufVersion):], outputRipemd160)

	checkCode := hash.DoubleSha256(strSlice)
	simpleCheckCode := checkCode[:4]
	slice := make([]byte, len(strSlice)+len(simpleCheckCode))
	copy(slice, strSlice)
	copy(slice[len(strSlice):], simpleCheckCode)

	fmt.Println("internal: ", len(slice))
	result := base58.Encode(slice)
	fmt.Println("xchain: ", len(result))
	return base58.Encode(slice), nil
}

// TODO
// transfer contract name to evm address
func ContractNameToEVMAddress(contractName string) (EVMAddress, error) {
	contractNameLength := len(contractName)
	var prefixStr string
	for i := 0; i < binary.Word160Length-contractNameLength-utils.GetContractNameMinSize(); i++ {
		prefixStr += evmAddressFiller
	}
	contractName = contractNamePrefix +prefixStr + contractName
	return newEVMAddressFromBytes([]byte(contractName))
}

// transfer evm address to contract name
func EVMAddressToContractName(evmAddr common.Address) (string, error) {
	contractNameWithPrefix := evmAddr.Bytes()
	contractNameStrWithPrefix := string(contractNameWithPrefix)
	prefixIndex := strings.LastIndex(contractNameStrWithPrefix, evmAddressFiller)
	if prefixIndex == -1 {
		return contractNameStrWithPrefix[4:], nil
	}
	return contractNameStrWithPrefix[prefixIndex+1:], nil
}

// transfer contract account to evm address
func ContractAccountToEVMAddress(contractAccount string) (crypto.Address, error) {
	contractAccountValid := contractAccount[2:18]
	contractAccountValid = contractAccountPrefix + contractAccountValid
	return crypto.AddressFromBytes([]byte(contractAccountValid))
}

// transfer evm address to contract account
func EVMAddressToContractAccount(evmAddr crypto.Address) (string, error) {
	contractNameWithPrefix := evmAddr.Bytes()
	contractNameStrWithPrefix := string(contractNameWithPrefix)
	return utils.GetAccountPrefix() + contractNameStrWithPrefix[4:] + "@xuper", nil
}

// 返回的合约账户不包括前缀XC和后缀@xuper（或其他链名）
func EVMAddressToContractAccountWithoutPrefixAndSuffix(evmAddr common.Address) (string, error) {
	contractNameWithPrefix := evmAddr.Bytes()
	contractNameStrWithPrefix := string(contractNameWithPrefix)
	return contractNameStrWithPrefix[4:], nil
}

// Deprecating, use IsContractAccount instead
func DetermineContractAccount(account string) bool {
	return IsContractAccount(account)
}

// IsContractAccount returns true for a contract account
func IsContractAccount(account string) bool {
	return utils.IsAccount(account) && strings.Contains(account, "@")
}

// Deprecating
// if error of incorrect name is needed, use `contract.ValidContractName`
// otherwise, use IsContractName
func DetermineContractName(contractName string) error {
	return contract.ValidContractName(contractName)
}

// IsContractName determine whether it is a contract name
func IsContractName(contractName string) bool {
	return contract.ValidContractName(contractName) == nil
}

// determine whether it is a contract name
func DetermineContractNameFromEVM(evmAddr common.Address) (string, error) {
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

// determine an EVM address
func DetermineEVMAddress(evmAddr common.Address) (string, string, error) {
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
		addr, err = EVMAddressToXchainAK(evmAddr)
		addrType = XchainAddrType
	}
	if err != nil {
		return "", "", err
	}

	return addr, addrType, nil
}

// determine an xchain address
func DetermineXchainAddress(xAddr string) (string, string, error) {
	var addr crypto.Address
	var addrType string
	var err error
	if IsContractAccount(xAddr) {
		addr, err = ContractAccountToEVMAddress(xAddr)
		addrType = ContractAccountType
	} else if IsContractName(xAddr) {
		//addr, err = ContractNameToEVMAddress(xAddr)
		addrType = ContractNameType
	} else {
		addr, err = XchainAKToEVMAddress(xAddr)
		addrType = XchainAddrType
	}
	if err != nil {
		return "", "", err
	}

	return addr.String(), addrType, nil
}
