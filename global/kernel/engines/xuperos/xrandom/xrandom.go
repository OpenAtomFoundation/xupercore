package xrandom

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/pkg/errors"

	"github.com/OpenAtomFoundation/xupercore/global/bcs/ledger/xledger/xldgpb"
	xctx "github.com/OpenAtomFoundation/xupercore/global/kernel/common/xcontext"
	"github.com/OpenAtomFoundation/xupercore/global/kernel/contract/sandbox"
	"github.com/OpenAtomFoundation/xupercore/global/kernel/engines/xuperos/xrandom/bls"
	"github.com/OpenAtomFoundation/xupercore/global/kernel/ledger"
	"github.com/OpenAtomFoundation/xupercore/global/service/pb"
)

const (
	ContractName = "XRandom"
)

const (
	cacheSize  = 100
	freezeSize = 5
)

const (
	statusSuccess = 200
)

var (
	keyNodes     = []byte("nodes")
	keyMaxHeight = []byte("max_height")
	// wait for peer node ready for new cluster
	initOnce bool
)

// generate random for height
func GetTx(ctx xctx.XContext, height int64) (*xldgpb.Transaction, error) {
	if !initOnce {
		initOnce = true
		time.Sleep(time.Second * 3) // wait for peer handler ready for new cluster
	}
	ctx.GetLog().Debug("XRandom.GetTx() invoked", "height", height)

	h := uint64(height)
	state := manager.Ctx.ChainCtx.State

	// get seed
	preH := h - 1
	var seed string
	if preH == 0 {
		seed = manager.Ctx.ChainCtx.Ledger.GenesisBlock.GetConfig().XRandom.Seed
	} else {
		random, err := GetRandomFromDB(state.CreateXMReader(), preH)
		if err != nil {
			return nil, err
		}
		seed = random.Number
	}
	ctx.GetLog().Debug("XRandom.GetTx() get previous number",
		"previous number", seed)

	// generate random number
	sign, proof, err := bls.GenerateRandom(ctx, seed)
	if err != nil {
		return nil, err
	}
	pbProof := pb.Proof{
		Message:          []byte(proof.Message),
		PartPublicKeySum: proof.PartPublicKeySum,
		Indexes:          proof.PartIndexes,
		PPrime:           proof.PPrime,
	}

	// submit
	hData, _ := json.Marshal(h)
	proofData, _ := json.Marshal(pbProof)
	number := hex.EncodeToString(sign.Signature)
	args := map[string][]byte{
		ParamHeight:       hData,
		ParamRandomNumber: []byte(number),
		ParamProof:        proofData,
	}
	ctx.GetLog().Debug("XRandom.GetTx() generate tx",
		"height", height,
		"number", number,
		"message", proof.Message,
		"P'", proof.PPrime,
		"indexes", proof.PartIndexes,
		"publicKeySum", proof.PartPublicKeySum)
	return state.GetRandomTx(h, args)
}

func GetRandomFromDB(reader ledger.XMReader, height uint64) (*Random, error) {
	data, err := reader.Get(ContractName, bucketKeyRandom(height))
	if err != nil {
		return nil, err
	}
	return parseRandomData(data.PureData.Value)
}

func parseRandomData(value []byte) (*Random, error) {
	if len(value) == 0 {
		return nil, errors.New("get random empty")
	}

	random := new(Random)
	if err := json.Unmarshal(value, &random); err != nil {
		return nil, err
	}
	return random, nil
}

func contractBucketKeyOfRandom(height uint64) []byte {
	contractPrefix := []byte(ContractName + sandbox.BucketSeperator)
	return append(contractPrefix, bucketKeyRandom(height)...)
}

func bucketKeyRandom(height uint64) []byte {
	key := fmt.Sprintf("%d", height)
	return []byte(key)
}
