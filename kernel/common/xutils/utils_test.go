package xutils

import (
	"fmt"
	"testing"
)

func TestGetXRootPath(t *testing.T) {
	rt := GetXRootPath()
	fmt.Println(rt)
}

func TestGetCurRootDir(t *testing.T) {
	rt := GetCurRootDir()
	fmt.Println(rt)
}
