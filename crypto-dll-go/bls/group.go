package bls

import (
	"fmt"
	"math/big"
)

type Group struct {
	Members map[string]*Account

	Threshold int

	// K is member weight in group
	// map[memberIndex]Weight;
	// weight is a jsonify []byte
	K map[string]string
	// WeightedPublicKeys is part of P'
	// map[memberIndex]weightedPublicKey;
	// weightedPublicKey = K * publicKey
	WeightedPublicKeys map[string]PublicKey

	// P is sum of public keys for all member
	P PublicKey
	// P' is sum of weighted public keys for all member
	PPrime PublicKey
}

func NewGroup(members map[string]*Account, thresholdRatio *big.Rat) (*Group, error) {
	// check member
	group := &Group{
		Members: make(map[string]*Account),
	}
	keys := make([]PublicKey, 0, len(members))
	for _, m := range members {
		keys = append(keys, m.PublicKey)
		group.Members[m.Index] = m
	}
	if len(group.Members) != len(members) {
		return nil, fmt.Errorf("exist duplicated members")
	}

	// init
	if err := group.calcThreshold(thresholdRatio); err != nil {
		return nil, fmt.Errorf(`calc Threshold: %v`, err)
	}

	sum, err := SumPublicKeys(keys)
	if err != nil {
		return nil, fmt.Errorf(`SumPublicKeys: %v`, err)
	}
	group.P = sum

	if err := group.calcKs(); err != nil {
		return nil, fmt.Errorf(`calculate K: %v`, err)
	}
	if err := group.calcPublicKeyParts(); err != nil {
		return nil, fmt.Errorf(`calculate public key parts: %v`, err)
	}
	if err := group.calcPPrime(); err != nil {
		return nil, fmt.Errorf(`calculate P': %v`, err)
	}

	return group, nil
}

func (g *Group) calcThreshold(ratio *big.Rat) error {
	if ratio.Cmp(big.NewRat(0, 1)) <= 0 || ratio.Cmp(big.NewRat(1, 1)) > 0 {
		return fmt.Errorf("threshold ratio must between [0, 1]")
	}

	num := int(ratio.Num().Int64())
	denom := int(ratio.Denom().Int64())

	threshold := g.Size() * num / denom
	reminder := (g.Size() * num) % denom

	if reminder == 0 {
		g.Threshold = threshold
	} else {
		g.Threshold = threshold + 1
	}
	return nil
}

func (g *Group) calcKs() error {
	k := make(map[string]string, g.Size())
	for i, member := range g.Members {
		ki, err := CalcK(member.PublicKey, g.P)
		if err != nil {
			return fmt.Errorf(`CalcK: %v`, err)
		}
		k[i] = ki
	}
	g.K = k
	return nil
}

func (g *Group) calcPublicKeyParts() error {
	weightedPublicKeys := make(map[string]PublicKey, g.Size())
	for index, member := range g.Members {
		weightedKey, err := member.PublicKey.Mul(g.K[index])
		if err != nil {
			return fmt.Errorf(`Mul: %v`, err)
		}
		weightedPublicKeys[index] = weightedKey
	}
	g.WeightedPublicKeys = weightedPublicKeys
	return nil
}

func (g *Group) calcPPrime() error {

	keys := make([]PublicKey, 0, len(g.WeightedPublicKeys))
	for _, key := range g.WeightedPublicKeys {
		keys = append(keys, key)
	}

	sum, err := SumPublicKeys(keys)
	if err != nil {
		return fmt.Errorf(`SumPublicKeys: %v`, err)
	}
	g.PPrime = sum
	return nil
}

func (g *Group) CalcMkParts(signer *Account) (map[string]MkPart, error) {
	mkParts := make(map[string]MkPart, g.Size())
	for index := range g.Members {
		mkPart, err := CalcMkPart(g.K[signer.Index], signer.PrivateKey, index, g.PPrime)
		if err != nil {
			return nil, fmt.Errorf(`CalcMkPart: %v`, err)
		}
		mkParts[index] = mkPart
	}
	return mkParts, nil
}

func (g *Group) Size() int {
	return len(g.Members)
}
