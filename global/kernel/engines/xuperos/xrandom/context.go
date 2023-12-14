package xrandom

import (
	"encoding/json"
	"fmt"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"

	"github.com/OpenAtomFoundation/xupercore/global/kernel/common/xcontext"
	"github.com/OpenAtomFoundation/xupercore/global/kernel/contract"
	"github.com/OpenAtomFoundation/xupercore/global/kernel/engines/xuperos/common"
	"github.com/OpenAtomFoundation/xupercore/global/kernel/engines/xuperos/xrandom/ecdsa"
	"github.com/OpenAtomFoundation/xupercore/global/lib/logs"
	"github.com/OpenAtomFoundation/xupercore/global/lib/timer"
	"github.com/OpenAtomFoundation/xupercore/global/service/pb"
)

type Context struct {
	// 基础上下文
	xcontext.BaseCtx

	BcName string

	Contract contract.Manager
	ChainCtx *common.ChainCtx
}

const (
	ParamNodePublicKey = "node_public_key" // compressed public key
	ParamNodeAddress   = "node_address"
	ParamHeight        = "height"
	ParamSign          = "sign"
	ParamRandomNumber  = "random_number"
	ParamProof         = "proof"
)

func NewCtx(ctx *common.ChainCtx) (*Context, error) {
	if ctx == nil {
		return nil, fmt.Errorf("new ctx failed because param error")
	}

	log, err := logs.NewLogger("", ContractName)
	if err != nil {
		return nil, fmt.Errorf("new ctx failed because new logger error. err:%v", err)
	}

	c := new(Context)
	c.XLog = log
	c.Timer = timer.NewXTimer()
	c.BcName = ctx.BCName
	c.Contract = ctx.Contract
	c.ChainCtx = ctx

	return c, nil
}

func parseNodePublicKey(args map[string][]byte) (string, error) {
	return parseNonEmptyString(args, ParamNodePublicKey)
}

func parseNodeAddress(args map[string][]byte) (string, error) {
	nodeAddrByte, ok := args[ParamNodeAddress]
	if !ok {
		return "", errors.New("param node_address is empty")
	}
	nodeAddr := string(nodeAddrByte)
	if !ethcommon.IsHexAddress(nodeAddr) {
		return "", errors.New("param node_address is not hex address")
	}

	return ethcommon.HexToAddress(nodeAddr).String(), nil
}

func parseHeight(args map[string][]byte) (uint64, error) {
	value, ok := args[ParamHeight]
	if !ok {
		return 0, errors.New("param height is empty")
	}

	var height uint64
	err := json.Unmarshal(value, &height)
	if height == 0 {
		return 0, errors.New("height 0 is invalid")
	}
	return height, err
}

func parseHeightWithSign(args map[string][]byte) (uint64, error) {
	if !ecdsa.Verify(args[ParamNodePublicKey], args[ParamHeight], args[ParamSign]) {
		return 0, errors.New("signature error")
	}
	return parseHeight(args)
}

// parseRandom return verified random number and its proof
func parseRandom(args map[string][]byte) (*Random, error) {
	randomNumber, err := parseRandomNumber(args)
	if err != nil {
		return nil, err
	}
	proof, err := parseRandomProof(args)
	if err != nil {
		return nil, err
	}

	random := &Random{
		Number: randomNumber,
		Proof:  proof,
	}
	if !random.Verify() {
		return nil, errors.New("random number verify failed")
	}
	return random, nil
}

func parseRandomProof(args map[string][]byte) (pb.Proof, error) {
	proof := pb.Proof{}

	value := args[ParamProof]
	if len(value) == 0 {
		return proof, errors.New("proof param empty")
	}

	err := json.Unmarshal(value, &proof)
	return proof, err
}

func parseRandomNumber(args map[string][]byte) (string, error) {
	return parseNonEmptyString(args, ParamRandomNumber)
}

func parseNonEmptyString(args map[string][]byte, key string) (string, error) {
	value := args[key]
	if len(value) == 0 {
		return "", fmt.Errorf("param empty for key: %s", key)
	}
	return string(value), nil
}
