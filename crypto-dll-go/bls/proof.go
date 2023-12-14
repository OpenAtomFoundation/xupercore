package bls

import ffi "github.com/OpenAtomFoundation/xupercore/crypto-rust/x-crypto-ffi/go-struct"

type Signature = ffi.BlsSignature
type SignaturePart = ffi.BlsSignaturePart

type Proof struct {
	Message          string
	PPrime           PublicKey
	PartIndexes      []string
	PartPublicKeySum PublicKey
}
