package chained_bft

// PacemakerInterface is the interface of Pacemaker. It responsible for generating a new round.
// We assume Pacemaker in all correct replicas will have synchronized leadership after GST.
// Safty is entirely decoupled from liveness by any potential instantiation of Packmaker.
// Different consensus have different pacemaker implement
type PacemakerInterface interface {
	// CurrentView return current view of this node.
	GetCurrentView() int64
	// 原NextNewProposal，generate new proposal directly.
	AdvanceView(qc QuorumCertInterface) (bool, error)
	// TODO: 原有的NextNewView，NewView是我们的Chained-Hotstuff的特殊实现之一，后面可以考虑如何优化掉
	PrepareAdvance(viewNum int64, proposer string) error
}

// DefaultPaceMaker 是一个PacemakerInterface的默认实现，我们与PacemakerInterface放置在一起，方便查看
// PacemakerInterface的新实现直接直接替代DefaultPaceMaker即可
// The Pacemaker keeps track of votes and of time.
// TODO:  the Pacemaker broadcasts a TimeoutMsg notification.
type DefaultPaceMaker struct {
	StartView   int64
	currentView int64
	// timeout int64
}

// PrepareAdvance
func (p *DefaultPaceMaker) PrepareAdvance(viewNum int64, proposer string) error {
	return nil
}

func (p *DefaultPaceMaker) AdvanceView(qc QuorumCertInterface) (bool, error) {
	r := qc.GetProposalView()
	if r+1 > p.currentView {
		p.currentView = r + 1
	}
	return true, nil
}

func (p *DefaultPaceMaker) GetCurrentView() int64 {
	return p.currentView
}
