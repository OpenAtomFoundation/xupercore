package bls

/*
#cgo LDFLAGS: -L./lib -lxcrypto
#include <stdint.h>
#include <stdbool.h>
#include <stdlib.h>
#include "./lib/xcrypto.h"
*/
import "C"

import (
	"encoding/json"
	"fmt"
	"unsafe"

	ffi "github.com/OpenAtomFoundation/xupercore/crypto-rust/x-crypto-ffi/go-struct"
)

type BlsKey []byte
type PublicKey BlsKey
type PrivateKey BlsKey
type MkPart []byte
type Mk = MkPart

type Account struct {
	Index      string
	PublicKey  PublicKey
	PrivateKey PrivateKey
}

func NewAccount() (*Account, error) {
	outputAccount := C.create_new_bls_account()
	defer C.free(unsafe.Pointer(outputAccount))

	return NewAccountFromJson(C.GoString(outputAccount))
}

func NewAccountFromJson(data string) (*Account, error) {
	var account ffi.BLSAccount
	err := json.Unmarshal([]byte(data), &account)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal BLS account from json: %v", err)
	}

	return &Account{
		Index:      account.Index,
		PublicKey:  account.PublicKey.P,
		PrivateKey: account.PrivateKey.X,
	}, nil
}
