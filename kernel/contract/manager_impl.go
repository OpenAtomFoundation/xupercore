package contract

type managerImpl struct {
}

func newManagerImpl(core ChainCore) (Manager, error) {
	return new(managerImpl), nil
}

func (m *managerImpl) NewContext(_ *ContextConfig) (Context, error) {
	return new(contextImpl), nil
}

type contextImpl struct {
}

func (c *contextImpl) Invoke(method string, args map[string][]byte) (*Response, error) {
	return &Response{
		Status:  500,
		Message: "not implemented",
	}, nil
}

func (c *contextImpl) ResourceUsed() Limits {
	return Limits{}
}

func (c *contextImpl) Release() error {
	return nil
}

func init() {
	Register("default", newManagerImpl)
}
