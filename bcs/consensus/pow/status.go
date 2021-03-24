package pow

import (
	"encoding/json"
)

// PoWStatus 实现了ConsensusStatus接口
type PoWStatus struct {
	index       int
	startHeight int64
	newHeight   int64
	miner       ValidatorsInfo
}

type ValidatorsInfo struct {
	Validators []string `json:"validators"`
}

// GetVersion 返回pow所在共识version
func (s *PoWStatus) GetVersion() int64 {
	return 0
}

// GetConsensusBeginInfo 返回该实例初始高度
func (s *PoWStatus) GetConsensusBeginInfo() int64 {
	return s.startHeight
}

// GetStepConsensusIndex 获取共识item所在consensus slice中的index
func (s *PoWStatus) GetStepConsensusIndex() int {
	return s.index
}

// GetConsensusName 获取共识类型
func (s *PoWStatus) GetConsensusName() string {
	return "pow"
}

// GetCurrentTerm 获取当前状态机term
func (s *PoWStatus) GetCurrentTerm() int64 {
	return s.newHeight
}

// GetCurrentValidatorsInfo 获取当前矿工信息
func (s *PoWStatus) GetCurrentValidatorsInfo() []byte {
	info, err := json.Marshal(s.miner)
	if err != nil {
		return nil
	}
	return info
}
