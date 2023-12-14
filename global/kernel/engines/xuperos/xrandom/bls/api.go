package bls

import (
	"fmt"
	"time"

	"github.com/OpenAtomFoundation/xupercore/crypto-dll-go/bls"
	xctx "github.com/OpenAtomFoundation/xupercore/global/kernel/common/xcontext"
	"github.com/OpenAtomFoundation/xupercore/global/kernel/engines/xuperos/common"
)

func SetAccount(account *bls.Account) *bls.Account {
	if account != nil {
		thresholdSign.client.Account = account
		return account
	}

	if thresholdSign.client.Account == nil {
		_ = thresholdSign.client.CreateAccount()
	}
	return thresholdSign.client.Account
}

func SetEngine(engine *common.EngineCtx) error {
	thresholdSign.engine = engine
	return RegisterEvent(engine)
}

func GenerateRandom(ctx xctx.XContext, message string) (*bls.Signature, *bls.Proof, error) {
	ctx.GetLog().Debug("GenerateRandom() invoked", "message", message)

	// get public key of peers
	expectGroup := thresholdSign.electGroup()
	groupChanged := !thresholdSign.isGroupRemain(expectGroup)

	// exchange member sign
	if groupChanged {
		ctx.GetLog().Debug("group changed",
			"expectGroup", expectGroup,
			"currentGroup", thresholdSign.ts.Group.Members)
		for index := range thresholdSign.ts.Group.Members {
			id, exist := thresholdSign.indexToID.Load(index)
			ctx.GetLog().Debug("ID map to index",
				"index", index,
				"ID", id,
				"exist", exist)
		}
		err := thresholdSign.updateGroup(ctx, expectGroup, true)
		if err != nil {
			return nil, nil, err
		}

		err = thresholdSign.exchangeMemberSigns(ctx)
		if err != nil {
			return nil, nil, err
		}
	}

	// exchange message sign
	if err := thresholdSign.ts.WaitMk(time.Second); err != nil {
		ctx.GetLog().Error("exchange member sign but not enough")
		return nil, nil, err
	}
	if err := thresholdSign.exchangeMessageSign(ctx, message); err != nil {
		return nil, nil, err
	}

	// get sign
	if err := thresholdSign.ts.WaitSign(time.Second); err != nil {
		ctx.GetLog().Error("exchange message sign but not enough")
		return nil, nil, err
	}
	if !thresholdSign.ts.Sign.IsReady() {
		return nil, nil, fmt.Errorf("threshold not reached for message sign")
	}
	sign := thresholdSign.ts.Sign.Value.(*bls.Signature)
	proof, err := thresholdSign.ts.Proof(*sign)
	ctx.GetLog().Debug("Proof() invoked", "sign", sign)
	if err != nil {
		return nil, nil, fmt.Errorf("proof error: %s", err)
	}
	return sign, &proof, nil
}
