package bls

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewThresholdSign(t *testing.T) {
	t.Parallel()

	helper := NewHelper(3)
	group, err := helper.Group()
	assert.NoError(t, err)

	ts := NewThresholdSign(*group, helper.Self)
	assert.False(t, ts.Mk.IsReady())
}

func TestThresholdSign_MkPartsByAccount(t *testing.T) {
	t.Parallel()

	groupSize := 3
	helper := NewHelper(groupSize)
	ts, err := helper.ThresholdSign()
	assert.NoError(t, err)

	mkParts, err := ts.MkPartsByAccount()
	assert.NoError(t, err)
	assert.Equal(t, groupSize, len(mkParts))
}

func TestThresholdSign_CollectMkParts(t *testing.T) {
	t.Parallel()

	groupSize := 3
	helper := NewHelper(groupSize)
	helper.ThresholdRatio = big.NewRat(2, 3)
	tss, err := helper.ThresholdSigns()
	assert.NoError(t, err)

	// first MK part
	miner := tss[helper.Self.Index]
	_, err = miner.MkPartsByAccount()
	assert.NoError(t, err)
	mkCount := 1

	// second MK part
	for index, ts := range tss {
		if index == helper.Self.Index {
			continue
		}
		mkParts, err := ts.MkPartsByAccount()
		assert.NoError(t, err)

		enough, err := miner.CollectMkParts(index, mkParts[index])
		mkCount++
		assert.NoError(t, err)
		assert.Equal(t, mkCount == groupSize, enough)
	}
}