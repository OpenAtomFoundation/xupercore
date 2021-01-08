package def

import (
	"errors"
	"strings"
)

var (
	ErrKVNotFound = errors.New("Key not found")
	ErrP2PError   = errors.New("invalid stream")
)

func NormalizedKVError(err error) error {
	if err == nil {
		return err
	}
	if strings.HasSuffix(err.Error(), "not found") {
		return ErrKVNotFound
	}
	if isInvalidStream(err.Error()) {
		return ErrP2PError
	}
	return err
}

func isInvalidStream(err string) bool {
	if strings.HasSuffix(err, "stream reset") {
		return true
	}
	if strings.HasSuffix(err, "connection reset by peer") {
		return true
	}
	if strings.HasSuffix(err, "stream closed") {
		return true
	}
	if strings.HasSuffix(err, "stream not valid") {
		return true
	}
	return false
}
