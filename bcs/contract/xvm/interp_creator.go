package xvm

import (
	"bytes"
	"io/ioutil"

	"github.com/xuperchain/wagon/wasm"
	"github.com/xuperchain/xvm/runtime/wasi"

	"github.com/xuperchain/xvm/exec"
	"github.com/xuperchain/xvm/runtime/emscripten"
	gowasm "github.com/xuperchain/xvm/runtime/go"

	"github.com/xuperchain/xupercore/kernel/contract/bridge"
)

type xvmInterpCreator struct {
	cm     *codeManager
	config bridge.InstanceCreatorConfig
}

func newXVMInterpCreator(creatorConfig *bridge.InstanceCreatorConfig) (bridge.InstanceCreator, error) {
	creator := &xvmInterpCreator{
		config: *creatorConfig,
	}
	var err error
	creator.cm, err = newCodeManager(creator.config.Basedir,
		creator.compileCode, creator.makeExecCode)
	if err != nil {
		return nil, err
	}
	return creator, nil
}

func (x *xvmInterpCreator) compileCode(buf []byte, outputPath string) error {
	return ioutil.WriteFile(outputPath, buf, 0600)
}

func (x *xvmInterpCreator) makeExecCode(codepath string) (exec.Code, bool, error) {
	codebuf, err := ioutil.ReadFile(codepath)
	if err != nil {
		return nil, false, err
	}
	resolver := exec.NewMultiResolver(
		gowasm.NewResolver(),
		emscripten.NewResolver(),
		newSyscallResolver(x.config.SyscallService),
		wasi.NewResolver(),
		builtinResolver,
	)
	// not good to dependency wagon direct in xupercore,but no better solution
	legacy, err := isLegacyInterp(codebuf)
	if err != nil {
		return nil, false, err
	}
	code, err := exec.NewInterpCode(codebuf, resolver)
	return code, legacy, err
}

func (x *xvmInterpCreator) CreateInstance(ctx *bridge.Context, cp bridge.ContractCodeProvider) (bridge.Instance, error) {
	code, err := x.cm.GetExecCode(ctx.ContractName, cp)
	if err != nil {
		return nil, err
	}
	return createInstance(ctx, code, x.config.SyscallService)
}

func (x *xvmInterpCreator) RemoveCache(contractName string) {
	x.cm.RemoveCode(contractName)
}

func isLegacyInterp(codebuf []byte) (bool, error) {
	module, err := wasm.DecodeModule(bytes.NewBuffer(codebuf))
	if err != nil {
		return false, err
	}

	if module.Import != nil {
		for _, entry := range module.Export.Entries {
			if entry.FieldStr == currentContractMethodInitialize {
				return false, nil
			}
		}
	}
	return true, nil
}
func init() {
	bridge.Register(bridge.TypeWasm, "ixvm", newXVMInterpCreator)
}
