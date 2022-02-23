package xvm

import (
	"debug/macho"
	"fmt"
	"io"
)

func Symbols(reader io.ReaderAt) error {
	file, err := macho.NewFile(reader)
	if err != nil {
		return err
	}
	symbols := file.Symtab.Syms
	for _, sym := range symbols {
		fmt.Println(sym.Name, sym.Type)
	}
	return nil
}
