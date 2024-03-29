package xvm

import (
	"os"

	"github.com/xuperchain/xupercore/kernel/contract/bridge"
	"github.com/xuperchain/xvm/exec"
	"github.com/xuperchain/xvm/runtime/emscripten"
	gowasm "github.com/xuperchain/xvm/runtime/go"
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
	return os.WriteFile(outputPath, buf, 0600)
}

func (x *xvmInterpCreator) makeExecCode(codepath string) (exec.Code, error) {
	codebuf, err := os.ReadFile(codepath)
	if err != nil {
		return nil, err
	}
	resolver := exec.NewMultiResolver(
		gowasm.NewResolver(),
		emscripten.NewResolver(),
		newSyscallResolver(x.config.SyscallService),
		builtinResolver,
	)
	return exec.NewInterpCode(codebuf, resolver)
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

func init() {
	bridge.Register(bridge.TypeWasm, "ixvm", newXVMInterpCreator)
}
