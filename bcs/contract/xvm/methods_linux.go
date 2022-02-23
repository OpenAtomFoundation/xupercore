// +build linux

package xvm

import (
	"debug/elf"
	"io"
)

// return map as it is used to check whether initialize method exists
func symbols(r io.ReaderAt) (map[string]struct{}, error) {
	file, err := elf.NewFile(r)
	if err != nil {
		return nil, err
	}
	ret := map[string]struct{}{}
	symbols, err := file.Symbols()
	if err != nil {
		return nil, err
	}
	for _, symbol := range symbols {
		ret[symbol.Name] = struct{}{}

	}
	return ret, nil
}
