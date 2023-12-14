package propose

import (
	pb "github.com/OpenAtomFoundation/xupercore/global/protos"
)

type ProposeManager interface {
	GetProposalByID(proposalID string) (*pb.Proposal, error)
}
