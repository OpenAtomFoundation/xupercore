package tdpos

import (
	"encoding/json"
	"testing"
)

func TestGetCurrentValidatorsInfo(t *testing.T) {
	cStr := getTdposConsensusConf()
	tdposCfg, err := buildConfigs([]byte(cStr))
	if err != nil {
		t.Error("Config unmarshal err", "err", err)
	}
	cCtx, err := prepare(getTdposConsensusConf())
	if err != nil {
		t.Error("prepare error", "error", err)
		return
	}
	s := NewSchedule(tdposCfg, cCtx.XLog, cCtx.Ledger, 1)
	status := TdposStatus{
		Version:     1,
		StartHeight: 1,
		Index:       0,
		election:    s,
	}
	b := status.GetCurrentValidatorsInfo()
	var addrs ValidatorsInfo
	if err := json.Unmarshal(b, &addrs); err != nil {
		t.Error("GetCurrentValidatorsInfo error", "error", err)
	}
}
