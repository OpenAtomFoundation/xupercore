package xpoa

import "testing"

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

func TestParseVersion(t *testing.T) {
	strVersion := `{
		"version": "2"
	}`
	v, err := ParseVersion(strVersion)
	if err != nil {
		t.Error("ParseVersion err, err: ", err)
		return
	}
	if v != 2 {
		t.Error("ParseVersion err, v: ", v)
		return
	}
	intVersion := `{
		"version": 3
	}`
	v, err = ParseVersion(intVersion)
	if err != nil {
		t.Error("ParseVersion err, err: ", err)
		return
	}
	if v != 3 {
		t.Error("ParseVersion err, v: ", v)
		return
	}
	empryVersion := `{}`
	v, err = ParseVersion(empryVersion)
	if err != nil {
		t.Error("ParseVersion err, err: ", err)
		return
	}
	if v != 0 {
		t.Error("ParseVersion err, v: ", v)
		return
	}
}
