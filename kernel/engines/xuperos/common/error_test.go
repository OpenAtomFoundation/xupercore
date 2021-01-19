package common

import (
	"fmt"
	"testing"
)

func TestEqual(t *testing.T) {
	var err error
	err = getError()
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("err is nil")
}

func getError() error {
	return nil
}
