package tdpos

import (
	"encoding/json"
	"testing"
)

func Test(t *testing.T) {
	// map[string]map[string]int64
	resRaw := NewNominateValue()
	testValue := make(map[string]int64)
	testValue["NodeB"] = 1
	resRaw["NodeA"] = testValue
	res, err := json.Marshal(&resRaw)
	if err != nil {
		t.Error("Marshal error ", err)
		return
	}

	nominateValue := NewNominateValue()
	if err := json.Unmarshal(res, &nominateValue); err != nil {
		t.Error("Unmarshal err ", err)
		return
	}
	t.Log("nominateValue: ", nominateValue)
}
