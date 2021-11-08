package xpoa

import "testing"

var (
	aks = map[string]float64{
		"dpzuVdosQrF2kmzumhVeFQZa1aYcdgFpN": 0.5,
		"WNWk3ekXeM5M2232dY2uCJmEqWhfQiDYT": 0.5,
	}
	aks2 = map[string]float64{
		"dpzuVdosQrF2kmzumhVeFQZa1aYcdgFpN": 0.5,
		"WNWk3ekXeM5M2232dY2uCJmEqWhfQiDYT": 0.6,
	}
	aks3 = map[string]float64{
		"dpzuVdosQrF2kmzumhVeFQZa1aYcdgFpN": 0.4,
		"WNWk3ekXeM5M2232dY2uCJmEqWhfQiDYT": 0.6,
	}
)

func TestIsAuthAddress(t *testing.T) {
	v1 := []string{"dpzuVdosQrF2kmzumhVeFQZa1aYcdgFpN", "WNWk3ekXeM5M2232dY2uCJmEqWhfQiDYT"}
	if !IsAuthAddress(aks, 0.6, v1, false) {
		t.Error("isAuthAddress err.")
	}
	v2 := []string{"dpzuVdosQrF2kmzumhVeFQZa1aYcdgFpN"}
	if IsAuthAddress(aks2, 0.6, v2, true) {
		t.Error("isAuthAddress err.")
	}
	v3 := []string{"WNWk3ekXeM5M2232dY2uCJmEqWhfQiDYT"}
	if !IsAuthAddress(aks2, 0.6, v3, true) {
		t.Error("isAuthAddress err.")
	}
	v4 := []string{"dpzuVdosQrF2kmzumhVeFQZa1aYcdgFpN", "WNWk3ekXeM5M2232dY2uCJmEqWhfQiDYT"}
	if !IsAuthAddress(aks2, 0.7, v4, true) {
		t.Error("isAuthAddress err.")
	}
}

func TestCalFault(t *testing.T) {
	if CalFault(0, 1) {
		t.Error("TestCalFault error 1.")
		return
	}
	if !CalFault(2, 2) {
		t.Error("TestCalFault error 2.")
		return
	}
	if CalFault(1, 4) {
		t.Error("TestCalFault error 3.")
		return
	}
	if !CalFault(3, 7) {
		t.Error("TestCalFault error 4.")
	}
}

func TestFind(t *testing.T) {
	if !Find("a", []string{"a", "b"}) {
		t.Error("TestFind error 1.")
		return
	}
	if Find("c", []string{"a", "b"}) {
		t.Error("TestFind error 2.")
	}
}

func TestLoadValidatorsMultiInfo(t *testing.T) {
	if _, err := loadValidatorsMultiInfo([]byte{}); err == nil {
		t.Error("TestLoadValidatorsMultiInfo error 1.")
		return
	}
	b := []byte(`{
		"validators":["a","b"]
	}`)
	if _, err := loadValidatorsMultiInfo(b); err != nil {
		t.Error("TestLoadValidatorsMultiInfo error 2.")
	}
}
