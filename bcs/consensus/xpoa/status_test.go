package xpoa

import (
	"encoding/json"
	"testing"
)

func TestGetCurrentValidatorsInfo(t *testing.T) {
	s, err := NewSchedule("dpzuVdosQrF2kmzumhVeFQZa1aYcdgFpN", InitValidators, true)
	if err != nil {
		t.Error("newSchedule error.")
		return
	}
	status := XpoaStatus{
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
