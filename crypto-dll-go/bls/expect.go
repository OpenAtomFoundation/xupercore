package bls

import (
	"errors"
	"sync"
	"time"
)

type ExpectValue struct {
	Value interface{}

	Ch   chan struct{}
	once sync.Once
}

func NewExpectValue() ExpectValue {
	return ExpectValue{
		once: sync.Once{},
		Ch:   make(chan struct{}),
	}
}

// Set expect value can only be set once
// it also broadcasts the signal for set behaviour
func (v *ExpectValue) Set(value interface{}) {
	v.once.Do(func() {
		v.Value = value
		close(v.Ch)
	})
}

// Wait waits for expect value until timeout
func (v *ExpectValue) Wait(timeout time.Duration) error {
	if v.IsReady() {
		return nil
	}

	// wait until ready or timeout
	select {
	case <-v.Ch:
	case <-time.After(timeout):
		return errors.New("timeout")
	}
	return nil
}

// IsReady return true after expect value set
func (v *ExpectValue) IsReady() bool {
	return v.Value != nil
}
