package govern_token

import pb "github.com/OpenAtomFoundation/xupercore/global/protos"

type GovManager interface {
	GetGovTokenBalance(accountName string) (*pb.GovernTokenBalance, error)
	DetermineGovTokenIfInitialized() (bool, error)
}
