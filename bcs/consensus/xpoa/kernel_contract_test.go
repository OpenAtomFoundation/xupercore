package xpoa

import (
	"encoding/json"
	"testing"

	"github.com/xuperchain/xupercore/kernel/consensus/mock"
)

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

func NewEditArgs() map[string][]byte {
	a := make(map[string][]byte)
	a["validates"] = []byte(`{
		"validates":"dpzuVdosQrF2kmzumhVeFQZa1aYcdgFpN;WNWk3ekXeM5M2232dY2uCJmEqWhfQiDYT;akf7qunmeaqb51Wu418d6TyPKp4jdLdpV"
	}`)
	a["rule"] = []byte("1")
	a["acceptValue"] = []byte("0.600")
	a["aksWeight"], _ = json.Marshal(&aks)
	return a
}

func NewEditM() map[string]map[string][]byte {
	a := make(map[string]map[string][]byte)
	return a
}

func TestMethodEditValidates(t *testing.T) {
	cCtx, err := prepare(getXpoaConsensusConf())
	if err != nil {
		t.Error("prepare error", "error", err)
		return
	}
	i := NewXpoaConsensus(*cCtx, getConfig(getXpoaConsensusConf()))
	xpoa, ok := i.(*xpoaConsensus)
	if !ok {
		t.Error("transfer err.")
		return
	}
	fakeCtx := mock.NewFakeKContext(NewEditArgs(), NewEditM())
	r, err := xpoa.methodEditValidates(fakeCtx)
	if err != nil {
		t.Error("methodEditValidates error", "error", err, "r", r)
		return
	}
}

func TestMethodGetValidates(t *testing.T) {
	cCtx, err := prepare(getXpoaConsensusConf())
	if err != nil {
		t.Error("prepare error", "error", err)
		return
	}
	i := NewXpoaConsensus(*cCtx, getConfig(getXpoaConsensusConf()))
	xpoa, ok := i.(*xpoaConsensus)
	if !ok {
		t.Error("transfer err.")
		return
	}
	fakeCtx := mock.NewFakeKContext(NewEditArgs(), NewEditM())
	r, err := xpoa.methodGetValidates(fakeCtx)
	if err != nil {
		t.Error("methodGetValidates error", "error", err, "r", r)
		return
	}
}

func TestIsAuthAddress(t *testing.T) {
	cCtx, err := prepare(getXpoaConsensusConf())
	if err != nil {
		t.Error("prepare error", "error", err)
		return
	}
	i := NewXpoaConsensus(*cCtx, getConfig(getXpoaConsensusConf()))
	xpoa, ok := i.(*xpoaConsensus)
	if !ok {
		t.Error("transfer err.")
		return
	}
	v1 := []string{"dpzuVdosQrF2kmzumhVeFQZa1aYcdgFpN", "WNWk3ekXeM5M2232dY2uCJmEqWhfQiDYT"}
	if !xpoa.isAuthAddress(v1, aks, 0.6, false) {
		t.Error("isAuthAddress err.")
		return
	}
	v2 := []string{"dpzuVdosQrF2kmzumhVeFQZa1aYcdgFpN"}
	if xpoa.isAuthAddress(v2, aks2, 0.6, true) {
		t.Error("isAuthAddress err.")
		return
	}
	v3 := []string{"WNWk3ekXeM5M2232dY2uCJmEqWhfQiDYT"}
	if !xpoa.isAuthAddress(v3, aks2, 0.6, true) {
		t.Error("isAuthAddress err.")
		return
	}
	v4 := []string{"dpzuVdosQrF2kmzumhVeFQZa1aYcdgFpN", "WNWk3ekXeM5M2232dY2uCJmEqWhfQiDYT"}
	if !xpoa.isAuthAddress(v4, aks2, 0.7, true) {
		t.Error("isAuthAddress err.")
	}
}
