package xevidence

import (
	"errors"
	"fmt"

	"github.com/spf13/viper"
	"github.com/xuperchain/xupercore/kernel/contract"
)

const (
	ConfigName = "engine.yaml"
)

const (
	XEvidence = "$XEvidence"

	Success = 200

	Save = "Save"
	Get  = "Get"

	AddAdmins   = "AddAdmins"
	DelAdmins   = "DelAdmins"
	QueryAdmins = "QueryAdmins"
	SetFee      = "SetFee"
	GetFee      = "GetFee"
)

type XEvidenceConfig struct {
	// 通过配置升级过程中，可以接受 LastSaveMethodFeeConfig
	// 网络升级结束后，使用 CurrentSaveMethodFeeConfig
	XEvidenceSaveMethodFeeConfig *SaveMethodFeeConfig
	XEvidenceAdmins              map[string]bool
	XEvidenceMethodFee           map[string]int64
}

type SaveMethodFeeConfig struct {
	// content 字段长度阈值，小于此值手续费为 FeeForLengthThreshold
	LengthThreshold       int64 `json:"lengthThreshold"`
	FeeForLengthThreshold int64 `json:"feeForLengthThreshold"`

	// content 字段长度超过 LengthThreshold 时，长度每增加 LengthIncrement，手续费增加 FeeIncrement
	LengthIncrement int64 `json:"lengthIncrement"`
	FeeIncrement    int64 `json:"feeIncrement"`

	// content 字段最大长度
	MaxLength int64 `json:"maxLength"`
}

type Manager struct {
	Ctx *Context
}

func NewManager(ctx *Context) (*Manager, error) {
	if ctx == nil || ctx.Contract == nil {
		return nil, errors.New("xtoken contract ctx set error")
	}

	var (
		cfg *XEvidenceConfig
		err error
	)
	genesisCfg := ctx.ChainCtx.Ledger.GenesisBlock.GetConfig().XEvidence
	if len(genesisCfg.XEvidenceAdmins) != 0 || len(genesisCfg.XEvidenceMethodFee) != 0 {
		// 如果创世配置了这两个字段，以创世字段为主，后期也通过交易修改 admins 和 fee
		cfg = &XEvidenceConfig{
			XEvidenceSaveMethodFeeConfig: &SaveMethodFeeConfig{
				LengthThreshold:       genesisCfg.LengthThreshold,
				FeeForLengthThreshold: genesisCfg.FeeForLengthThreshold,
				LengthIncrement:       genesisCfg.LengthIncrement,
				FeeIncrement:          genesisCfg.FeeIncrement,
				MaxLength:             genesisCfg.MaxLength,
			},
			XEvidenceAdmins:    genesisCfg.XEvidenceAdmins,
			XEvidenceMethodFee: genesisCfg.XEvidenceMethodFee,
		}
	} else {
		// 如果创世没有配置这两个字段，则从配置文件读取
		cfg, err = loadConfig(ctx.ChainCtx.EngCtx.EnvCfg.GenConfFilePath(ConfigName))
		if err != nil {
			return nil, err
		}
	}

	ctx.XLog.Debug("XEvidence Config Loaded:", *cfg)

	x := NewContract(ctx, cfg)

	register := ctx.Contract.GetKernRegistry()
	kMethods := map[string]contract.KernMethod{
		// 存证接口
		Save: x.Save,
		Get:  x.Get,

		// admins 和 fee 接口
		AddAdmins:   x.AddAdmins,
		DelAdmins:   x.DelAdmins,
		QueryAdmins: x.QueryAdmins,
		SetFee:      x.SetFee,
		GetFee:      x.GetFee,
	}

	for method, f := range kMethods {
		// 系统合约名字前缀为$
		if _, err := register.GetKernMethod(XEvidence, method); err != nil {
			register.RegisterKernMethod(XEvidence, method, f)
		}
	}
	return nil, nil
}

func loadConfig(fname string) (*XEvidenceConfig, error) {
	viperObj := viper.New()
	viperObj.SetConfigFile(fname)
	err := viperObj.ReadInConfig()
	if err != nil {
		return nil, fmt.Errorf("read config failed.path:%s,err:%v", fname, err)
	}

	cfg := &XEvidenceConfig{}
	if err = viperObj.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmatshal config failed.path:%s,err:%v", fname, err)
	}
	return cfg, nil
}
