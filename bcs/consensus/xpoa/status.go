package xpoa

import (
	"encoding/json"
	"time"
)

type ValidatorsInfo struct {
	Validators []string `json:"validators"`
	Miner      string   `json:"miner"`
}

// xpoaStatus 实现了ConsensusStatus接口
type XpoaStatus struct {
	Name        string
	Version     int64 `json:"version"`
	StartHeight int64 `json:"startHeight"`
	Index       int   `json:"index"`
	election    *xpoaSchedule
}

// 获取共识版本号
func (x *XpoaStatus) GetVersion() int64 {
	return x.Version
}

// 共识起始高度
func (x *XpoaStatus) GetConsensusBeginInfo() int64 {
	return x.StartHeight
}

// 获取共识item所在consensus slice中的index
func (x *XpoaStatus) GetStepConsensusIndex() int {
	return x.Index
}

// 获取共识类型
func (x *XpoaStatus) GetConsensusName() string {
	return x.Name
}

// 获取当前状态机term
func (x *XpoaStatus) GetCurrentTerm() int64 {
	term, _, _ := x.election.minerScheduling(time.Now().UnixNano(), len(x.election.validators))
	return term
}

// 获取当前矿工信息
func (x *XpoaStatus) GetCurrentValidatorsInfo() []byte {
	i := ValidatorsInfo{
		Validators: x.election.validators,
		Miner:      x.election.miner,
	}
	b, _ := json.Marshal(i)
	return b
}
