package def

import (
	"errors"
	"testing"
)

func TestNormalizedKVError(t *testing.T) {
	kvErr := errors.New("Key not found")
	err := NormalizedKVError(kvErr)
	if err != nil {
		t.Log(err)
	}
	p2pErr := errors.New("invalid stream")
	err = NormalizedKVError(p2pErr)
	if err != nil {
		t.Log(err)
	}
}
