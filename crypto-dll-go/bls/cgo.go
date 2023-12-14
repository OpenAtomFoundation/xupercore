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

func SumPublicKeys(keys []PublicKey) (PublicKey, error) {
	// pack input
	publicKeys := make([]ffi.PublicKey, 0, len(keys))
	for _, key := range keys {
		publicKeys = append(publicKeys, key.CStruct())
	}
	jsonKeys, _ := json.Marshal(publicKeys)
	inputKeys := C.CString(string(jsonKeys))
	defer C.free(unsafe.Pointer(inputKeys))

	// invoke function
	output := C.sum_bls_public_key(inputKeys)
	defer C.free(unsafe.Pointer(output))

	// parse output
	sum := ffi.BlsPublicKeyResponse{}
	if err := json.Unmarshal([]byte(C.GoString(output)), &sum); err != nil {
		return nil, fmt.Errorf(`unmarshal public key sum error: %v`, err)
	}
	return sum.Ok.P, nil
}

func CalcK(key, sum PublicKey) (string, error) {
	// pack input
	inputPart := C.CString(key.CString())
	defer C.free(unsafe.Pointer(inputPart))

	inputSum := C.CString(sum.CString())
	defer C.free(unsafe.Pointer(inputSum))

	// invoke function
	outputK := C.get_bls_k(inputPart, inputSum)
	defer C.free(unsafe.Pointer(outputK))

	// parse output
	return C.GoString(outputK), nil
}

func CalcMkPart(kI string, privateKeyI PrivateKey, indexJ string, pPrime PublicKey) (MkPart, error) {
	// pack input
	inputKI := C.CString(kI)
	defer C.free(unsafe.Pointer(inputKI))

	inputPrivateI := C.CString(privateKeyI.CString())
	defer C.free(unsafe.Pointer(inputPrivateI))

	inputIndexJ := C.CString(indexJ)
	defer C.free(unsafe.Pointer(inputIndexJ))

	inputPPrime := C.CString(pPrime.CString())
	defer C.free(unsafe.Pointer(inputPPrime))

	// invoking function
	outputMKPart := C.get_bls_m(inputKI, inputPrivateI, inputIndexJ, inputPPrime)
	defer C.free(unsafe.Pointer(outputMKPart))

	// parse output
	var mkPart ffi.BlsM
	if err := json.Unmarshal([]byte(C.GoString(outputMKPart)), &mkPart); err != nil {
		return nil, fmt.Errorf(`unmarshal MK part error: %v`, err)
	}
	return mkPart.P, nil
}

func SumMk(mkParts []MkPart) (Mk, error) {
	// pack input
	parts := make([]ffi.BlsM, 0, len(mkParts))
	for _, key := range mkParts {
		parts = append(parts, key.CStruct())
	}

	jsonParts, _ := json.Marshal(parts)
	inputParts := C.CString(string(jsonParts))
	defer C.free(unsafe.Pointer(inputParts))

	// invoke function
	output := C.get_bls_mk(inputParts)
	defer C.free(unsafe.Pointer(output))

	// parse output
	sum := ffi.BlsMkResponse{}
	if err := json.Unmarshal([]byte(C.GoString(output)), &sum); err != nil {
		return nil, fmt.Errorf(`unmarshal MK error: %v`, err)
	}
	return sum.Ok.P, nil
}

func VerifyMk(pPrime PublicKey, index string, MK Mk) bool {
	if len(pPrime) == 0 || len(index) == 0 || len(MK) == 0 {
		return false
	}

	// pack input
	inputPrime := C.CString(pPrime.CString())
	defer C.free(unsafe.Pointer(inputPrime))

	inputIndex := C.CString(index)
	defer C.free(unsafe.Pointer(inputIndex))

	inputMK := C.CString(MK.CString())
	defer C.free(unsafe.Pointer(inputMK))

	// invoke function
	output := C.verify_bls_mk(inputPrime, inputIndex, inputMK)

	// parse output
	return bool(output)
}

// Sign let signer signs a message returns signature as part
func Sign(signer *ffi.PartnerPrivate, message string) (ffi.BlsSignaturePart, error) {
	// pack input
	jsonSigner, _ := json.Marshal(signer)
	inputSigner := C.CString(string(jsonSigner))
	defer C.free(unsafe.Pointer(inputSigner))

	inputMessage := C.CString(message)
	defer C.free(unsafe.Pointer(inputMessage))

	// invoke function
	outputSign := C.bls_sign(inputSigner, inputMessage)
	defer C.free(unsafe.Pointer(outputSign))

	// parse output
	sign := ffi.BlsSignaturePart{}
	if err := json.Unmarshal([]byte(C.GoString(outputSign)), &sign); err != nil {
		return ffi.BlsSignaturePart{}, fmt.Errorf(`unmarshal Signature error: %v`, err)
	}
	return sign, nil
}

func CombineSignature(parts []ffi.BlsSignaturePart) (ffi.BlsSignature, error) {
	// pack input
	jsonParts, _ := json.Marshal(parts)
	inputParts := C.CString(string(jsonParts))
	defer C.free(unsafe.Pointer(inputParts))

	// invoke function
	outputSign := C.bls_combine_sign(inputParts)
	defer C.free(unsafe.Pointer(outputSign))

	// parse output
	sign := ffi.BlsSignature{}
	if err := json.Unmarshal([]byte(C.GoString(outputSign)), &sign); err != nil {
		return ffi.BlsSignature{}, fmt.Errorf(`unmarshal Signature error: %v`, err)
	}
	return sign, nil
}

// VerifyThresholdSignature verifies threshold signature
// Params:
// - pPrime: sum of public key part for group
// - sign: signature for message
// - message: message be signed
func VerifyThresholdSignature(pPrime PublicKey, sign ffi.BlsSignature, message string) (result bool) {

	if len(pPrime) == 0 || len(sign.Signature) == 0 ||
		len(sign.PartPublicKeySum) == 0 || len(sign.PartIndexes) == 0 {
		return false
	}

	// pack input
	inputPPrime := C.CString(pPrime.CString())
	defer C.free(unsafe.Pointer(inputPPrime))

	jsonSign, _ := json.Marshal(sign)
	inputSign := C.CString(string(jsonSign))
	defer C.free(unsafe.Pointer(inputSign))

	inputMessage := C.CString(message)
	defer C.free(unsafe.Pointer(inputMessage))

	// invoke function
	outputResult := C.bls_verify_sign(inputPPrime, inputSign, inputMessage)

	// parse output
	return bool(outputResult)
}

func (k PublicKey) Mul(factor string) (PublicKey, error) {
	// pack input
	inputKey := C.CString(k.CString())
	defer C.free(unsafe.Pointer(inputKey))

	inputFactor := C.CString(factor)
	defer C.free(unsafe.Pointer(inputFactor))

	// invoke function
	output := C.get_bls_public_key_part(inputKey, inputFactor)
	defer C.free(unsafe.Pointer(output))

	// parse output
	var result ffi.PublicKey
	if err := json.Unmarshal([]byte(C.GoString(output)), &result); err != nil {
		return nil, fmt.Errorf(`unmarshal plused public key error: %v`, err)
	}
	return result.P, nil
}

// CString return serialized structured public key for cgo
func (k PublicKey) CString() string {
	jsonData, _ := json.Marshal(k.CStruct())
	return string(jsonData)
}

// CStruct return public key structure for cgo
func (k PublicKey) CStruct() ffi.PublicKey {
	return ffi.PublicKey{P: k}
}

// CString return serialized structured private key for cgo
func (k PrivateKey) CString() string {
	jsonData, _ := json.Marshal(ffi.PrivateKey{X: k})
	return string(jsonData)
}

// CString return serialized structured MK for cgo
func (k Mk) CString() string {
	jsonData, _ := json.Marshal(k.CStruct())
	return string(jsonData)
}

// CStruct return MK structure for cgo
func (k Mk) CStruct() ffi.BlsM {
	return ffi.BlsM{P: k}
}
