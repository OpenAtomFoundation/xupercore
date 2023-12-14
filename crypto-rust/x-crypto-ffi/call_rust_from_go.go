package main

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
	"log"
	"unsafe"

	g "github.com/xuperchain/yogurt-chain/crypto-rust/x-crypto-ffi/go-struct"
)

func main() {
	// --------- create_new_bls_account

	// account 1
	o := C.create_new_bls_account()
	defer C.free(unsafe.Pointer(o))

	output := C.GoString(o)
	fmt.Printf("create_new_bls_account1: %s\n", output)

	// parse account to Go struct
	var account1 g.BLSAccount
	err := json.Unmarshal([]byte(output), &account1)
	if err != nil {
		fmt.Println("unmarshal account failed")
	} else {
		fmt.Printf("BLSAccount: %+v\n", account1)
	}

	// account 2
	o = C.create_new_bls_account()
	defer C.free(unsafe.Pointer(o))

	output = C.GoString(o)
	fmt.Printf("create_new_bls_account2: %s\n", output)

	// parse account to Go struct
	var account2 g.BLSAccount
	err = json.Unmarshal([]byte(output), &account2)

	// account 3
	o = C.create_new_bls_account()
	defer C.free(unsafe.Pointer(o))

	output = C.GoString(o)
	fmt.Printf("create_new_bls_account3: %s\n", output)

	// parse account to Go struct
	var account3 g.BLSAccount
	err = json.Unmarshal([]byte(output), &account3)

	// --------- sum_bls_public_key

	// 创建数组
	var blsPublicKeys []g.PublicKey
	blsPublicKeys = append(blsPublicKeys, account1.PublicKey)
	blsPublicKeys = append(blsPublicKeys, account2.PublicKey)
	blsPublicKeys = append(blsPublicKeys, account3.PublicKey)

	jsonBlsPublicKeys, _ := json.Marshal(blsPublicKeys)

	log.Printf("jsonBlsPublicKeys is %s", jsonBlsPublicKeys)

	inputBlsPublicKeys := C.CString(string(jsonBlsPublicKeys))
	defer C.free(unsafe.Pointer(inputBlsPublicKeys))

	o = C.sum_bls_public_key(inputBlsPublicKeys)
	defer C.free(unsafe.Pointer(o))

	output = C.GoString(o)
	fmt.Printf("sum_bls_public_key: %s\n", output)

	// --------- get_bls_k
	sumBlsPublicKeyResponse := g.BlsPublicKeyResponse{}
	err = json.Unmarshal([]byte(output), &sumBlsPublicKeyResponse)
	if err != nil {
		return
	}

	inputBlsPublicKeySumJson, _ := json.Marshal(sumBlsPublicKeyResponse.Ok)
	inputBlsPublicKeySum := C.CString(string(inputBlsPublicKeySumJson))
	defer C.free(unsafe.Pointer(inputBlsPublicKeySum))

	// k1
	jsonBlsPublicKey1, _ := json.Marshal(account1.PublicKey)
	inputBlsPublicKey1 := C.CString(string(jsonBlsPublicKey1))
	defer C.free(unsafe.Pointer(inputBlsPublicKey1))

	account1KOutput := C.get_bls_k(inputBlsPublicKey1, inputBlsPublicKeySum)
	defer C.free(unsafe.Pointer(account1KOutput))

	account1K := C.GoString(account1KOutput)
	fmt.Printf("get_bls_k account1K: %s\n", account1K)

	// k2
	jsonBlsPublicKey2, _ := json.Marshal(account2.PublicKey)
	inputBlsPublicKey2 := C.CString(string(jsonBlsPublicKey2))
	defer C.free(unsafe.Pointer(inputBlsPublicKey2))

	account2KOutput := C.get_bls_k(inputBlsPublicKey2, inputBlsPublicKeySum)
	defer C.free(unsafe.Pointer(account2KOutput))

	account2K := C.GoString(account2KOutput)
	fmt.Printf("get_bls_k account2K: %s\n", account2K)

	// k3
	jsonBlsPublicKey3, _ := json.Marshal(account3.PublicKey)
	inputBlsPublicKey3 := C.CString(string(jsonBlsPublicKey3))
	defer C.free(unsafe.Pointer(inputBlsPublicKey3))

	account3KOutput := C.get_bls_k(inputBlsPublicKey3, inputBlsPublicKeySum)
	defer C.free(unsafe.Pointer(account3KOutput))

	account3K := C.GoString(account3KOutput)
	fmt.Printf("get_bls_k account3K: %s\n", account3K)

	// --------- get_bls_public_key_part

	// -- bls_account1_public_key_part
	inputAccount1BlsK := C.CString(string(account1K))
	defer C.free(unsafe.Pointer(inputAccount1BlsK))

	o = C.get_bls_public_key_part(inputBlsPublicKey1, inputAccount1BlsK)
	defer C.free(unsafe.Pointer(o))

	bls_account1_public_key_part := C.GoString(o)
	fmt.Printf("get_bls_public_key_part account1: %s\n", bls_account1_public_key_part)

	// -- bls_account2_public_key_part
	inputAccount2BlsK := C.CString(string(account2K))
	defer C.free(unsafe.Pointer(inputAccount2BlsK))

	o = C.get_bls_public_key_part(inputBlsPublicKey2, inputAccount2BlsK)
	defer C.free(unsafe.Pointer(o))

	bls_account2_public_key_part := C.GoString(o)
	fmt.Printf("get_bls_public_key_part account2: %s\n", bls_account2_public_key_part)

	// -- bls_account3_public_key_part
	inputAccount3BlsK := C.CString(string(account3K))
	defer C.free(unsafe.Pointer(inputAccount3BlsK))

	o = C.get_bls_public_key_part(inputBlsPublicKey3, inputAccount3BlsK)
	defer C.free(unsafe.Pointer(o))

	bls_account3_public_key_part := C.GoString(o)
	fmt.Printf("get_bls_public_key_part account3: %s\n", bls_account3_public_key_part)

	// --------- get share_public_key 门限公钥

	// parse public_key_part to Go struct
	var publicKeyPart1 g.PublicKey
	err = json.Unmarshal([]byte(bls_account1_public_key_part), &publicKeyPart1)

	var publicKeyPart2 g.PublicKey
	err = json.Unmarshal([]byte(bls_account2_public_key_part), &publicKeyPart2)

	var publicKeyPart3 g.PublicKey
	err = json.Unmarshal([]byte(bls_account3_public_key_part), &publicKeyPart3)

	// 创建数组
	var blsPublicShareKeys []g.PublicKey
	blsPublicShareKeys = append(blsPublicShareKeys, publicKeyPart1)
	blsPublicShareKeys = append(blsPublicShareKeys, publicKeyPart2)
	blsPublicShareKeys = append(blsPublicShareKeys, publicKeyPart3)

	jsonBlsPublicShareKeys, _ := json.Marshal(blsPublicShareKeys)

	log.Printf("jsonBlsPublicShareKeys is %s", jsonBlsPublicShareKeys)

	inputBlsPublicShareKeys := C.CString(string(jsonBlsPublicShareKeys))
	defer C.free(unsafe.Pointer(inputBlsPublicShareKeys))

	o = C.sum_bls_public_key(inputBlsPublicShareKeys)
	defer C.free(unsafe.Pointer(o))

	output = C.GoString(o)
	fmt.Printf("sum_bls_public_key for PublicShareKey: %s\n", output)

	sumBlsPublicKeyResponse = g.BlsPublicKeyResponse{}
	err = json.Unmarshal([]byte(output), &sumBlsPublicKeyResponse)
	if err != nil {
		return
	}

	inputBlsPublicKeySumJson, _ = json.Marshal(sumBlsPublicKeyResponse.Ok)
	inputPublicShareKey := C.CString(string(inputBlsPublicKeySumJson))
	defer C.free(unsafe.Pointer(inputPublicShareKey))

	// --------- get_bls_m

	// account1 m1
	jsonBlsPrivateKey1, _ := json.Marshal(account1.PrivateKey)
	inputBlsPrivateKey1 := C.CString(string(jsonBlsPrivateKey1))
	defer C.free(unsafe.Pointer(inputBlsPrivateKey1))

	inputIndex1 := C.CString(string(account1.Index))
	defer C.free(unsafe.Pointer(inputIndex1))

	inputAccount1K := C.CString(account1K)
	defer C.free(unsafe.Pointer(inputAccount1K))

	//inputPublicShareKey := C.CString(publicShareKey)
	//defer C.free(unsafe.Pointer(inputPublicShareKey))

	o = C.get_bls_m(inputAccount1K, inputBlsPrivateKey1, inputIndex1, inputPublicShareKey)
	defer C.free(unsafe.Pointer(o))

	output = C.GoString(o)
	fmt.Printf("get_bls_m account1 m1: %s\n", output)

	blsAccount1M1 := output

	// account1 m2
	inputIndex2 := C.CString(string(account2.Index))
	defer C.free(unsafe.Pointer(inputIndex2))

	o = C.get_bls_m(inputAccount1K, inputBlsPrivateKey1, inputIndex2, inputPublicShareKey)
	defer C.free(unsafe.Pointer(o))

	output = C.GoString(o)
	fmt.Printf("get_bls_m account1 m2: %s\n", output)

	blsAccount1M2 := output

	// account1 m3
	inputIndex3 := C.CString(string(account3.Index))
	defer C.free(unsafe.Pointer(inputIndex3))

	o = C.get_bls_m(inputAccount1K, inputBlsPrivateKey1, inputIndex3, inputPublicShareKey)
	defer C.free(unsafe.Pointer(o))

	output = C.GoString(o)
	fmt.Printf("get_bls_m account1 m3: %s\n", output)

	blsAccount1M3 := output

	// -- account2 m1
	jsonBlsPrivateKey2, _ := json.Marshal(account2.PrivateKey)
	inputBlsPrivateKey2 := C.CString(string(jsonBlsPrivateKey2))
	defer C.free(unsafe.Pointer(inputBlsPrivateKey2))

	inputAccount2K := C.CString(account2K)
	defer C.free(unsafe.Pointer(inputAccount2K))

	o = C.get_bls_m(inputAccount2K, inputBlsPrivateKey2, inputIndex1, inputPublicShareKey)
	defer C.free(unsafe.Pointer(o))

	output = C.GoString(o)
	fmt.Printf("get_bls_m account2 m1: %s\n", output)

	blsAccount2M1 := output

	// -- account2 m2
	o = C.get_bls_m(inputAccount2K, inputBlsPrivateKey2, inputIndex2, inputPublicShareKey)
	defer C.free(unsafe.Pointer(o))

	output = C.GoString(o)
	fmt.Printf("get_bls_m account2 m2: %s\n", output)

	blsAccount2M2 := output

	// -- account2 m3
	o = C.get_bls_m(inputAccount2K, inputBlsPrivateKey2, inputIndex3, inputPublicShareKey)
	defer C.free(unsafe.Pointer(o))

	output = C.GoString(o)
	fmt.Printf("get_bls_m account2 m3: %s\n", output)

	blsAccount2M3 := output

	// -- account3 m1
	jsonBlsPrivateKey3, _ := json.Marshal(account3.PrivateKey)
	inputBlsPrivateKey3 := C.CString(string(jsonBlsPrivateKey3))
	defer C.free(unsafe.Pointer(inputBlsPrivateKey3))

	inputAccount3K := C.CString(account3K)
	defer C.free(unsafe.Pointer(inputAccount3K))

	o = C.get_bls_m(inputAccount3K, inputBlsPrivateKey3, inputIndex1, inputPublicShareKey)
	defer C.free(unsafe.Pointer(o))

	output = C.GoString(o)
	fmt.Printf("get_bls_m account3 m1: %s\n", output)

	blsAccount3M1 := output

	// -- account3 m2
	o = C.get_bls_m(inputAccount3K, inputBlsPrivateKey3, inputIndex2, inputPublicShareKey)
	defer C.free(unsafe.Pointer(o))

	output = C.GoString(o)
	fmt.Printf("get_bls_m account3 m2: %s\n", output)

	blsAccount3M2 := output

	// -- account3 m3
	o = C.get_bls_m(inputAccount3K, inputBlsPrivateKey3, inputIndex3, inputPublicShareKey)
	defer C.free(unsafe.Pointer(o))

	output = C.GoString(o)
	fmt.Printf("get_bls_m account3 m3: %s\n", output)

	blsAccount3M3 := output

	// --------- get_bls_mk
	// parse blsM to Go struct

	// M1
	var account1M1 g.BlsM
	err = json.Unmarshal([]byte(blsAccount1M1), &account1M1)

	var account2M1 g.BlsM
	err = json.Unmarshal([]byte(blsAccount2M1), &account2M1)

	var account3M1 g.BlsM
	err = json.Unmarshal([]byte(blsAccount3M1), &account3M1)

	// M2
	var account1M2 g.BlsM
	err = json.Unmarshal([]byte(blsAccount1M2), &account1M2)

	var account2M2 g.BlsM
	err = json.Unmarshal([]byte(blsAccount2M2), &account2M2)

	var account3M2 g.BlsM
	err = json.Unmarshal([]byte(blsAccount3M2), &account3M2)

	// M3
	var account1M3 g.BlsM
	err = json.Unmarshal([]byte(blsAccount1M3), &account1M3)

	var account2M3 g.BlsM
	err = json.Unmarshal([]byte(blsAccount2M3), &account2M3)

	var account3M3 g.BlsM
	err = json.Unmarshal([]byte(blsAccount3M3), &account3M3)

	// MK1
	// 创建数组
	var blsM1s []g.BlsM
	blsM1s = append(blsM1s, account1M1)
	blsM1s = append(blsM1s, account2M1)
	blsM1s = append(blsM1s, account3M1)

	jsonBlsM1s, _ := json.Marshal(blsM1s)

	log.Printf("jsonBlsM1s is %s", jsonBlsM1s)

	inputBlsM1s := C.CString(string(jsonBlsM1s))
	defer C.free(unsafe.Pointer(inputBlsM1s))

	mk1 := C.get_bls_mk(inputBlsM1s)
	defer C.free(unsafe.Pointer(mk1))

	output = C.GoString(mk1)
	fmt.Printf("get_bls_mk1 : %s\n", output)

	var mki1Response g.BlsMkResponse
	_ = json.Unmarshal([]byte(output), &mki1Response)

	// MK2
	var blsM2s []g.BlsM
	blsM2s = append(blsM2s, account1M2)
	blsM2s = append(blsM2s, account2M2)
	blsM2s = append(blsM2s, account3M2)

	jsonBlsM2s, _ := json.Marshal(blsM2s)

	log.Printf("jsonBlsM2s is %s", jsonBlsM2s)

	inputBlsM2s := C.CString(string(jsonBlsM2s))
	defer C.free(unsafe.Pointer(inputBlsM2s))

	mk2 := C.get_bls_mk(inputBlsM2s)
	defer C.free(unsafe.Pointer(mk2))

	output = C.GoString(mk2)
	fmt.Printf("get_bls_mk2: %s\n", output)

	var mki2Response g.BlsMkResponse
	_ = json.Unmarshal([]byte(output), &mki2Response)

	// MK3
	var blsM3s []g.BlsM
	blsM3s = append(blsM3s, account1M3)
	blsM3s = append(blsM3s, account2M3)
	blsM3s = append(blsM3s, account3M3)

	jsonBlsM3s, _ := json.Marshal(blsM3s)

	log.Printf("jsonBlsM3s is %s", jsonBlsM3s)

	inputBlsM3s := C.CString(string(jsonBlsM3s))
	defer C.free(unsafe.Pointer(inputBlsM3s))

	mk3 := C.get_bls_mk(inputBlsM3s)
	defer C.free(unsafe.Pointer(mk3))

	output = C.GoString(mk3)
	fmt.Printf("get_bls_mk3: %s\n", output)

	var mki3Response g.BlsMkResponse
	_ = json.Unmarshal([]byte(output), &mki3Response)

	// --------- verify_bls_mk

	// verify mk1
	jsonMki1, _ := json.Marshal(mki1Response.Ok)
	inputMki1 := C.CString(string(jsonMki1))
	defer C.free(unsafe.Pointer(inputMki1))

	verifyResult := C.verify_bls_mk(inputPublicShareKey, inputIndex1, inputMki1)

	verifyOutput := bool(verifyResult)
	fmt.Printf("verify_bls_mk mk1 : %s\n", verifyOutput)

	// verify mk2
	jsonMki2, _ := json.Marshal(mki2Response.Ok)
	inputMki2 := C.CString(string(jsonMki2))
	defer C.free(unsafe.Pointer(inputMki2))

	verifyResult = C.verify_bls_mk(inputPublicShareKey, inputIndex2, inputMki2)

	verifyOutput = bool(verifyResult)
	fmt.Printf("verify_bls_mk mk2 : %s\n", verifyOutput)

	// verify mk3
	jsonMki3, _ := json.Marshal(mki3Response.Ok)
	inputMki3 := C.CString(string(jsonMki3))
	defer C.free(unsafe.Pointer(inputMki3))

	verifyResult = C.verify_bls_mk(inputPublicShareKey, inputIndex3, inputMki3)

	verifyOutput = bool(verifyResult)
	fmt.Printf("verify_bls_mk mk3 : %s\n", verifyOutput)

	// --------- bls_sign

	// account1 sign
	var partnerPublic1 g.PartnerPublic
	partnerPublic1.Index = account1.Index
	partnerPublic1.PublicKey = account1.PublicKey

	var sharePublicKey g.PublicKey
	sharePublicKey.P = sumBlsPublicKeyResponse.Ok.P

	//var account1Mki1 BlsM
	//var account1Mki1 BlsMkResponse
	//_ = json.Unmarshal([]byte(C.GoString(account1mk1)), &account1Mki1)

	var partnerPrivate1 g.PartnerPrivate
	partnerPrivate1.PublicInfo = partnerPublic1
	partnerPrivate1.ThresholdPublicKey = sharePublicKey
	partnerPrivate1.X = account1.PrivateKey.X
	partnerPrivate1.Mki = mki1Response.Ok.P

	jsonPartnerPrivate1, _ := json.Marshal(partnerPrivate1)
	inputPartnerPrivate1 := C.CString(string(jsonPartnerPrivate1))
	defer C.free(unsafe.Pointer(inputPartnerPrivate1))

	msg := "msg for bls sign"

	inputMsg := C.CString(msg)
	defer C.free(unsafe.Pointer(inputMsg))

	o = C.bls_sign(inputPartnerPrivate1, inputMsg)
	defer C.free(unsafe.Pointer(o))

	blsSignPart1 := C.GoString(o)
	fmt.Printf("bls_sign for account1: %s\n", blsSignPart1)

	// account2 sign
	var partnerPublic2 g.PartnerPublic
	partnerPublic2.Index = account2.Index
	partnerPublic2.PublicKey = account2.PublicKey

	var partnerPrivate2 g.PartnerPrivate
	partnerPrivate2.PublicInfo = partnerPublic2
	partnerPrivate2.ThresholdPublicKey = sharePublicKey
	partnerPrivate2.X = account2.PrivateKey.X
	partnerPrivate2.Mki = mki2Response.Ok.P

	jsonPartnerPrivate2, _ := json.Marshal(partnerPrivate2)
	inputPartnerPrivate2 := C.CString(string(jsonPartnerPrivate2))
	defer C.free(unsafe.Pointer(inputPartnerPrivate2))

	o = C.bls_sign(inputPartnerPrivate2, inputMsg)
	defer C.free(unsafe.Pointer(o))

	blsSignPart2 := C.GoString(o)
	fmt.Printf("bls_sign for account2: %s\n", blsSignPart2)

	// combine sign

	// parse blsSignPart to Go struct
	var blsSignaturePart1 g.BlsSignaturePart
	err = json.Unmarshal([]byte(blsSignPart1), &blsSignaturePart1)

	var blsSignaturePart2 g.BlsSignaturePart
	err = json.Unmarshal([]byte(blsSignPart2), &blsSignaturePart2)

	var blsSignatureParts []g.BlsSignaturePart
	blsSignatureParts = append(blsSignatureParts, blsSignaturePart1)
	blsSignatureParts = append(blsSignatureParts, blsSignaturePart2)

	jsonBlsSignatureParts, _ := json.Marshal(blsSignatureParts)

	log.Printf("jsonBlsSignatureParts is %s", jsonBlsSignatureParts)

	inputBlsSignatureParts := C.CString(string(jsonBlsSignatureParts))
	defer C.free(unsafe.Pointer(inputBlsSignatureParts))

	o = C.bls_combine_sign(inputBlsSignatureParts)
	defer C.free(unsafe.Pointer(o))

	// parse BlsSignature to Go struct
	output = C.GoString(o)
	var blsSignature g.BlsSignature
	err = json.Unmarshal([]byte(output), &blsSignature)
	if err != nil {
		fmt.Errorf(" parse bls combined signature error: %s", err)
		return
	}
	fmt.Printf("bls_combine_sign result is: %s\n", output)

	// verify sign
	blsThresholdSig, _ := json.Marshal(blsSignature)

	inputBlsThresholdSig := C.CString(string(blsThresholdSig))
	defer C.free(unsafe.Pointer(inputBlsThresholdSig))

	verifyResult = C.bls_verify_sign(inputPublicShareKey, inputBlsThresholdSig, inputMsg)

	verifyOutput = bool(verifyResult)
	fmt.Printf("bls_verify_sign result is: %s\n", verifyOutput)
}
