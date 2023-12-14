package bls

import "math/big"

type Helper struct {
	Self           *Account
	Members        map[string]*Account
	ThresholdRatio *big.Rat
	thresholdSigns map[string]*ThresholdSign
}

func NewHelper(groupSize int) *Helper {
	h := &Helper{
		Members:        createRandomAccounts(groupSize),
		ThresholdRatio: big.NewRat(2, 3),
		thresholdSigns: make(map[string]*ThresholdSign, groupSize),
	}
	for _, value := range h.Members {
		h.Self = value
		break
	}
	return h
}

func (h *Helper) Group() (*Group, error) {
	return NewGroup(h.Members, h.ThresholdRatio)
}

func (h *Helper) ThresholdSign() (*ThresholdSign, error) {
	group, err := NewGroup(h.Members, h.ThresholdRatio)
	if err != nil {
		return nil, err
	}
	ts := NewThresholdSign(*group, h.Self)
	return &ts, nil
}

func (h *Helper) ThresholdSigns() (map[string]*ThresholdSign, error) {
	group, err := NewGroup(h.Members, h.ThresholdRatio)
	if err != nil {
		return nil, err
	}
	for index, account := range h.Members {
		ts := NewThresholdSign(*group, account)
		h.thresholdSigns[index] = &ts
	}
	return h.thresholdSigns, nil
}

func createRandomAccounts(count int) map[string]*Account {
	accounts := make(map[string]*Account, count)
	for i := 0; i < count; i++ {
		account, _ := NewAccount()
		accounts[account.Index] = account
	}
	return accounts
}
