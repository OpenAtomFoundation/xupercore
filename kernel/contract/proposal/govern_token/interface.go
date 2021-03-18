package govern_token

import pb "github.com/xuperchain/xupercore/protos"

type GovManager interface {
	GetGovTokenBalance(accountName string) (*pb.GovernTokenBalance, error)
	DetermineGovTokenIfInitialized() (bool, error)
}
