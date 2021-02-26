package single

import (
	"encoding/json"
	"sync"
)

type ValidatorsInfo struct {
	Validators []string `json:"validators"`
}

type SingleStatus struct {
	startHeight int64
	mutex       sync.RWMutex
	newHeight   int64
	index       int
	config      *SingleConfig
}

// GetVersion 返回pow所在共识version
func (s *SingleStatus) GetVersion() int64 {
	return s.config.Version
}

// GetConsensusBeginInfo 返回该实例初始高度
func (s *SingleStatus) GetConsensusBeginInfo() int64 {
	return s.startHeight
}

// GetStepConsensusIndex 获取共识item所在consensus slice中的index
func (s *SingleStatus) GetStepConsensusIndex() int {
	return s.index
}

// GetConsensusName 获取共识类型
func (s *SingleStatus) GetConsensusName() string {
	return "single"
}

// GetCurrentTerm 获取当前状态机term
func (s *SingleStatus) GetCurrentTerm() int64 {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.newHeight
}

// GetCurrentValidatorsInfo 获取当前矿工信息
func (s *SingleStatus) GetCurrentValidatorsInfo() []byte {
	miner := ValidatorsInfo{
		Validators: []string{s.config.Miner},
	}
	m, _ := json.Marshal(miner)
	return m
}
