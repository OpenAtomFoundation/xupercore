package govern_token

type GovManager interface {
	GetGovTokenBalance(accountName string) (string, error)
	DetermineGovTokenIfInitialized() (bool, error)
}
