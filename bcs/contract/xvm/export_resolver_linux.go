// +build linux,amd64

package xvm

import (
	"bytes"
	"debug/elf"
	"io/ioutil"
	"os"
	"strings"
)

const (
	exportSymbolPrefix = "export_"
)

// return map as it is used to check whether initialize method exists
func resolveSymbols(filepath string) (map[string]struct{}, error) {
	f, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	content, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}
	file, err := elf.NewFile(bytes.NewReader(content))
	if err != nil {
		return nil, err
	}

	ret := map[string]struct{}{}

	var dynStr *elf.Section
	for _, section := range file.Sections {
		if section.Name == ".dynstr" {
			dynStr = section
		}
	}

	data, err := dynStr.Data()
	if err != nil {
		return nil, err
	}
	symbols := bytes.Split(data, []byte{0})
	for _, symbol := range symbols {
		if strings.HasPrefix(string(symbol), exportSymbolPrefix) {
			ret[string(symbol)] = struct{}{}
		}
	}

	return ret, nil
}
