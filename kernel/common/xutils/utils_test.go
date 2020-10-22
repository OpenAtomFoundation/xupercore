package xutils

import (
	"fmt"
	"testing"
)

func TestGetXRootPath(t *testing.T) {
	rt := GetXRootPath()
	fmt.Println(rt)
}
