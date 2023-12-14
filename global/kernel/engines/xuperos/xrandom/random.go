package xrandom

import (
	"encoding/hex"
	"encoding/json"

	crypto "github.com/OpenAtomFoundation/xupercore/crypto-dll-go"
	"github.com/OpenAtomFoundation/xupercore/crypto-dll-go/bls"

	"github.com/OpenAtomFoundation/xupercore/global/service/pb"
)

type Random struct {
	Number string
	Proof  pb.Proof
}

func (r *Random) Verify() bool {
	client := crypto.NewBlsClient()

	sign, err := hex.DecodeString(r.Number)
	if err != nil {
		log.Error("decode failed", "error", err)
		return false
	}
	blsProof := bls.Proof{
		Message:          string(r.Proof.Message),
		PPrime:           bls.PublicKey(r.Proof.PPrime),
		PartIndexes:      r.Proof.Indexes,
		PartPublicKeySum: bls.PublicKey(r.Proof.PartPublicKeySum),
	}
	log.Debug("proof info",
		"message", r.Number,
		"proof", blsProof)
	return client.VerifySignatureByProof(sign, blsProof)
}

func (r *Random) toJSON() []byte {
	data, _ := json.Marshal(r)
	return data
}

func NewRandom(data []byte) (*Random, error) {
	r := &Random{}
	err := json.Unmarshal(data, r)
	return r, err
}
