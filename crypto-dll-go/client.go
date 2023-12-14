package crypto_dll_go

import (
	"errors"
	"fmt"
	"math/big"

	ffi "github.com/OpenAtomFoundation/xupercore/crypto-rust/x-crypto-ffi/go-struct"

	"github.com/OpenAtomFoundation/xupercore/crypto-dll-go/bls"
)

// struct for BLS client
type BlsClient struct {
	Account        *bls.Account
	Group          *bls.Group
	ThresholdRatio *big.Rat

	// MkPartsTo manages MK parts signed by client for others
	// map[otherIndex]MKPart
	MkPartsTo map[string]bls.MkPart
	Mk        bls.Mk
}

func NewBlsClient() *BlsClient {
	return &BlsClient{
		ThresholdRatio: big.NewRat(2, 3),
	}
}

// CreateAccount creates new account from BLS client
func (c *BlsClient) CreateAccount() error {
	if c.Account != nil {
		return fmt.Errorf("account already set")
	}

	account, err := bls.NewAccount()
	if err != nil {
		return err
	}
	c.Account = account
	return nil
}

// UpdateGroup updates threshold signature group
func (c *BlsClient) UpdateGroup(accounts map[string]*bls.Account) error {
	if c.Account == nil {
		return fmt.Errorf("account not set")
	}
	if len(accounts) == 0 {
		return fmt.Errorf("empty group")
	}
	accounts[c.Account.Index] = c.Account

	group, err := bls.NewGroup(accounts, c.ThresholdRatio)
	if err != nil {
		return err
	}
	c.Group = group
	return nil
}

// UpdateThresholdRatio updates threshold ratio for threshold signature
func (c *BlsClient) UpdateThresholdRatio(threshold *big.Rat) error {
	c.ThresholdRatio = threshold
	return nil
}

// ThresholdSign creates a process for threshold sign
func (c *BlsClient) ThresholdSign() bls.ThresholdSign {
	return bls.NewThresholdSign(*c.Group, c.Account)
}

// GenerateMkParts generates MK parts for all member of group signed by client account
func (c *BlsClient) GenerateMkParts() (map[string]bls.MkPart, error) {
	mkParts, err := c.Group.CalcMkParts(c.Account)
	if err != nil {
		return nil, err
	}
	c.MkPartsTo = mkParts
	return mkParts, nil
}

// UpdateMk update client account MK by parts signed by other accounts
func (c *BlsClient) UpdateMk(mkParts []bls.MkPart) error {
	if len(mkParts)+1 != c.Group.Size() {
		return fmt.Errorf("MK parts count not enough")
	}
	mkParts = append(mkParts, c.MkPartsTo[c.Account.Index])

	mk, err := bls.SumMk(mkParts)
	if err != nil {
		return fmt.Errorf(`sum MK: %v`, err)
	}
	c.Mk = mk
	return nil
}

// Sign signs a message with current account and group
// which result in a threshold signature part
func (c *BlsClient) Sign(message []byte) (bls.SignaturePart, error) {
	signer := &ffi.PartnerPrivate{
		PublicInfo: ffi.PartnerPublic{
			Index:     c.Account.Index,
			PublicKey: c.Account.PublicKey.CStruct(),
		},
		ThresholdPublicKey: c.Group.PPrime.CStruct(),
		X:                  c.Account.PrivateKey,
		Mki:                c.Mk,
	}

	// sign
	return bls.Sign(signer, string(message))
}

// CombineSignatureParts combines signature parts for threshold signature
func (c *BlsClient) CombineSignatureParts(parts map[string]bls.SignaturePart) (bls.Signature, error) {
	if c.Group == nil {
		return bls.Signature{}, fmt.Errorf("group not set")
	}
	if len(parts) < c.Group.Threshold {
		return bls.Signature{}, fmt.Errorf(`the number of signature parts has not reached the threshold`)
	}
	signParts := make([]bls.SignaturePart, 0, len(parts))
	for index, part := range parts {
		if index != part.Index || c.Group.Members[index] == nil {
			return bls.Signature{}, fmt.Errorf(`index %v is not a member of group`, index)
		}
		signParts = append(signParts, part)
	}
	return bls.CombineSignature(signParts)
}

// Proof check signature for itr message and generate a proof for VRF
func (c *BlsClient) Proof(message string, sign bls.Signature) (bls.Proof, error) {
	if !c.VerifySignature(message, sign) {
		return bls.Proof{}, errors.New("signature not valid")
	}

	proof := bls.Proof{
		Message:          message,
		PPrime:           c.Group.PPrime,
		PartIndexes:      sign.PartIndexes,
		PartPublicKeySum: bls.PublicKey(sign.PartPublicKeySum),
	}
	return proof, nil
}

// VerifySignature verifies signature with current group
func (c *BlsClient) VerifySignature(message string, sign bls.Signature) bool {
	return bls.VerifyThresholdSignature(c.Group.PPrime, sign, message)
}

// VerifySignatureByProof verifies signature with proof
func (c *BlsClient) VerifySignatureByProof(signData bls.BlsKey, proof bls.Proof) bool {
	sing := bls.Signature{
		PartIndexes:      proof.PartIndexes,
		PartPublicKeySum: ffi.BlsData(proof.PartPublicKeySum),
		Signature:        ffi.BlsData(signData),
	}
	return bls.VerifyThresholdSignature(proof.PPrime, sing, proof.Message)
}

// verifyMk verifies client's current MK, check its correctness for group
func (c *BlsClient) verifyMk() bool {
	if c.Group == nil || c.Account == nil {
		return false
	}
	return bls.VerifyMk(c.Group.PPrime, c.Account.Index, c.Mk)
}
