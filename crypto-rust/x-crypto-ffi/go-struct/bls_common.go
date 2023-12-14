package go_struct

import "encoding/json"

type BLSAccount struct {
	Index      string     `json:"index"`
	PublicKey  PublicKey  `json:"public_key"`
	PrivateKey PrivateKey `json:"private_key"`
}

type PublicKey struct {
	P []byte `json:"p"`
}

type PrivateKey struct {
	X []byte `json:"x"`
}

type BlsPublicKeyResponse struct {
	Ok struct {
		P []byte `json:"p"`
	} `json:"Ok"`
}

type BlsM struct {
	P []byte `json:"p"`
}

type BlsMkResponse struct {
	Ok struct {
		P []byte `json:"p"`
	} `json:"Ok"`
}

type PartnerPrivate struct {
	PublicInfo         PartnerPublic `json:"public_info"`
	ThresholdPublicKey PublicKey     `json:"threshold_public_key"`
	X                  []byte        `json:"x"`
	Mki                []byte        `json:"mki"`
}

type PartnerPublic struct {
	Index     string    `json:"index"`
	PublicKey PublicKey `json:"public_key"`
}

type BlsSignaturePart struct {
	Index     string `json:"index"`
	PublicKey []byte `json:"public_key"`
	Sig       []byte `json:"sig"`
}

// BlsSignature is combined signature
type BlsSignature struct {
	PartIndexes      []string `json:"part_indexs"`
	PartPublicKeySum BlsData  `json:"part_public_key_sum"`
	Signature        BlsData  `json:"sig"`
}

type BlsData []byte

// MarshalJSON serialized to number array, but not default string
func (b BlsData) MarshalJSON() ([]byte, error) {
	result := make([]int, len(b))
	for i, v := range b {
		result[i] = int(v)
	}
	return json.Marshal(result)
}
