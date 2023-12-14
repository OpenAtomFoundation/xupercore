package parachain

// Group 平行链群组
type Group struct {
	GroupID    string   `json:"name,omitempty"`       // group name which is the same as its parachain name
	Admin      []string `json:"admin,omitempty"`      // admin addresses
	Identities []string `json:"identities,omitempty"` // accessible addresses
	Status     int      `json:"status,omitempty"`     // parachain status
}

// hasAccessAuth returns true when address has access to the parachain
func (g *Group) hasAccessAuth(address string) bool {
	return contains(g.Admin, address) || contains(g.Identities, address)
}

// hasAdminAuth returns true when address is one of admin for parachain
func (g *Group) hasAdminAuth(address string) bool {
	return contains(g.Admin, address)
}

// IsParaChainEnable returns true when status is `start`
func (g *Group) IsParaChainEnable() bool {
	return g.Status == ParaChainStatusStart
}

// contains returns true when target string item in string list
func contains(items []string, item string) bool {
	for _, eachItem := range items {
		if eachItem == item {
			return true
		}
	}
	return false
}
