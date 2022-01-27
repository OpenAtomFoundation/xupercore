package consensus

import (
	cctx "github.com/xuperchain/xupercore/kernel/consensus/context"
	"github.com/xuperchain/xupercore/kernel/consensus/def"
)

var consensusMap = make(map[string]NewStepConsensus)

type NewStepConsensus func(cCtx cctx.ConsensusCtx, cCfg def.ConsensusConfig) ConsensusImplInterface

// Register 不同类型的共识需要提前完成注册
func Register(name string, f NewStepConsensus) error {
	if f == nil {
		panic("Pluggable Consensus::Register::new function is nil")
	}
	if _, dup := consensusMap[name]; dup {
		panic("Pluggable Consensus::Register::called twice for func " + name)
	}
	consensusMap[name] = f
	return nil
}

// NewPluginConsensus 新建可插拔共识实例
func NewPluginConsensus(cCtx cctx.ConsensusCtx, cCfg def.ConsensusConfig) (ConsensusImplInterface, error) {
	if cCfg.ConsensusName == "" {
		return nil, EmptyConsensusName
	}
	if cCfg.StartHeight < 0 {
		return nil, BeginBlockIdErr
	}
	if f, ok := consensusMap[cCfg.ConsensusName]; ok {
		return f(cCtx, cCfg), nil
	}
	return nil, ConsensusNotRegister
}
