package permission

import (
	"sort"
	"sync"

	"github.com/xuperchain/xupercore/kernel/contract"
	"github.com/xuperchain/xupercore/kernel/contract/kernel"
	"github.com/xuperchain/xupercore/kernel/permission/base"
	pctx "github.com/xuperchain/xupercore/kernel/permission/context"
	"github.com/xuperchain/xupercore/kernel/permission/pb"
)

// 创建Permission实例方法
type NewPermissionFunc func(pCtx pctx.PermissionCtx) base.PermissionImpl

var (
	servsMu  sync.RWMutex
	services = make(map[string]NewPermissionFunc)
)

// Register makes a driver available by the provided name.
// If Register is called twice with the same name or if driver is nil,it panics.
func Register(name string, f NewPermissionFunc) {
	servsMu.Lock()
	defer servsMu.Unlock()

	if f == nil {
		panic("permission: Register new func is nil")
	}
	if _, dup := services[name]; dup {
		panic("permission: Register called twice for func " + name)
	}
	services[name] = f
}

func Drivers() []string {
	servsMu.RLock()
	defer servsMu.RUnlock()
	list := make([]string, 0, len(services))
	for name := range services {
		list = append(list, name)
	}
	sort.Strings(list)
	return list
}

func createPermission(name string, pCtx pctx.PermissionCtx) base.PermissionImpl {
	servsMu.RLock()
	defer servsMu.RUnlock()

	if f, ok := services[name]; ok {
		return f(pCtx)
	}

	return nil
}

// permission对外提供的接口
type Permission interface {
	GetAccountACL(ctx pctx.PermissionCtx, accountName string) (*pb.Acl, error)
	GetAccountACLWithConfirmed(ctx pctx.PermissionCtx, accountName string) (*pb.Acl, bool, error)
	GetContractMethodACL(ctx pctx.PermissionCtx, contractName, methodName string) (*pb.Acl, error)
	GetContractMethodACLWithConfirmed(ctx pctx.PermissionCtx, contractName, methodName string) (*pb.Acl, bool, error)
	NewAccount(ctx kernel.KContext) (*contract.Response, error)
	SetAccountACL(ctx kernel.KContext) (*contract.Response, error)
	SetMethodACL(ctx kernel.KContext) (*contract.Response, error)
}
