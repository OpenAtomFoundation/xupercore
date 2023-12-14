package bls

import (
	"bytes"
	"errors"
	"fmt"
	"sync"
	"time"

	ffi "github.com/OpenAtomFoundation/xupercore/crypto-rust/x-crypto-ffi/go-struct"
)

// ThresholdSign controls threshold sign process for one message with given group
type ThresholdSign struct {
	Group   Group
	Account *Account

	message []byte

	MkPartsFrom sync.Map          // map[sender]MKPart records MK part from sender to Self
	MkPartsTo   map[string]MkPart // map[receiver]MKPart records MK part from Self to receiver
	Mk          ExpectValue       // collected and combined MK

	SignParts sync.Map    // map[signer]SignaturePart records message signature part by signer
	Sign      ExpectValue // collected and combined message signature
	SignSend  *sync.Once  // controls Self message signature part has been broadcast once
}

func NewThresholdSign(group Group, account *Account) ThresholdSign {
	return ThresholdSign{
		Group:   group,
		Account: account,
		Mk:      NewExpectValue(),
	}
}

// MkPartsByAccount generates MK parts for all member of group signed by client account
func (ts *ThresholdSign) MkPartsByAccount() (map[string]MkPart, error) {
	mkPartsTo, err := ts.Group.CalcMkParts(ts.Account)
	if err != nil {
		return nil, err
	}
	ts.MkPartsTo = mkPartsTo

	ts.MkPartsFrom.LoadOrStore(ts.Account.Index, mkPartsTo[ts.Account.Index])
	return mkPartsTo, nil
}

// CollectMkParts collects MK parts signed by other client account
// Returns:
//   - bool: true for enough MK parts
//   - err: error if failed to update MK parts
func (ts *ThresholdSign) CollectMkParts(account string, part MkPart) (bool, error) {
	if ts.Group.Members[account] == nil {
		return false, fmt.Errorf(`index %v is not a member of group`, account)
	}
	if ts.Mk.IsReady() {
		return true, nil
	}

	// update MK parts info
	_, exist := ts.MkPartsFrom.LoadOrStore(account, part)
	if exist {
		return ts.Mk.IsReady(), nil
	}

	// try merge MK
	mkParts := make([]MkPart, 0, ts.Group.Size())
	ts.MkPartsFrom.Range(func(_, v any) bool {
		mkParts = append(mkParts, v.(MkPart))
		return true
	})
	if len(mkParts) != ts.Group.Size() {
		return false, nil
	}
	mk, err := SumMk(mkParts)
	if err != nil {
		return false, err
	}
	ts.Mk.Set(mk)
	return true, nil
}

func (ts *ThresholdSign) WaitMk(timeout time.Duration) error {
	return ts.Mk.Wait(timeout)
}

// SignByAccount signs a message with current account and group
// which result in a threshold signature part
func (ts *ThresholdSign) SignByAccount(message []byte) (SignaturePart, error) {
	if !ts.Mk.IsReady() {
		return SignaturePart{}, errors.New("MK not initialized")
	}
	if len(message) == 0 {
		return SignaturePart{}, errors.New("message is empty")
	}
	if bytes.Equal(message, ts.message) {
		sign, ok := ts.SignParts.Load(ts.Account.Index)
		if !ok {
			return SignaturePart{}, errors.New("sign not exist")
		}
		return sign.(SignaturePart), nil
	}
	ts.message = message
	ts.SignParts = sync.Map{}
	ts.Sign = NewExpectValue()
	ts.SignSend = new(sync.Once)

	signer := &ffi.PartnerPrivate{
		PublicInfo: ffi.PartnerPublic{
			Index:     ts.Account.Index,
			PublicKey: ts.Account.PublicKey.CStruct(),
		},
		ThresholdPublicKey: ts.Group.PPrime.CStruct(),
		X:                  ts.Account.PrivateKey,
		Mki:                ts.Mk.Value.(Mk),
	}

	// sign
	sign, err := Sign(signer, string(message))
	if err != nil {
		return SignaturePart{}, err
	}
	ts.SignParts.LoadOrStore(ts.Account.Index, sign)
	return sign, nil
}

func (ts *ThresholdSign) GetMessage() []byte {
	return ts.message
}

// CombineSignaturePart combines signature parts for threshold signature
// Returns:
//   - Signature: non-nil if parts enough, which is combined signatures
//   - bool: true if parts enough
//   - error: non-nil if input part is not valid
func (ts *ThresholdSign) CollectSignaturePart(signPart SignaturePart) (*Signature, bool, error) {
	if ts.Group.Members[signPart.Index] == nil {
		return nil, false, fmt.Errorf(`index %v is not a member of group`, signPart.Index)
	}
	if ts.Sign.IsReady() {
		return ts.Sign.Value.(*Signature), true, nil
	}

	// save sign part
	_, exist := ts.SignParts.LoadOrStore(signPart.Index, signPart)
	if exist {
		return nil, ts.Sign.IsReady(), nil
	}

	// try to combine threshold signature
	signPartList := make([]SignaturePart, 0, ts.Group.Size())
	ts.SignParts.Range(func(_, v interface{}) bool {
		signPartList = append(signPartList, v.(SignaturePart))
		return true
	})
	if len(signPartList) < ts.Group.Threshold {
		return nil, false, nil
	}
	sign, err := CombineSignature(signPartList)
	if err != nil {
		return nil, false, err
	}
	ts.Sign.Set(&sign)
	return &sign, true, nil
}

func (ts *ThresholdSign) WaitSign(timeout time.Duration) error {
	return ts.Sign.Wait(timeout)
}

// Proof check signature for itr message and generate a proof for VRF
func (ts *ThresholdSign) Proof(sign Signature) (Proof, error) {
	message := string(ts.message)
	if !VerifyThresholdSignature(ts.Group.PPrime, sign, message) {
		return Proof{}, errors.New("signature not valid")
	}

	proof := Proof{
		Message:          message,
		PPrime:           ts.Group.PPrime,
		PartIndexes:      sign.PartIndexes,
		PartPublicKeySum: PublicKey(sign.PartPublicKeySum),
	}
	return proof, nil
}
