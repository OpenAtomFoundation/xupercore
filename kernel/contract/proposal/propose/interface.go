package propose

import (
	pb "github.com/xuperchain/xupercore/protos"
)

type ProposeManager interface {
	GetProposalByID(proposalID string) (*pb.Proposal, error)
}
