package crypto_dll_go

import (
	"encoding/json"
	"fmt"
	"testing"

	ffi "github.com/OpenAtomFoundation/xupercore/crypto-rust/x-crypto-ffi/go-struct"
	"github.com/stretchr/testify/assert"

	"github.com/OpenAtomFoundation/xupercore/crypto-dll-go/bls"
)

const (
	testAccount1 = `{"index":"5068161082419015240502520392764951298362816233167274727352107679212581830087892411256674283175582004835800509618391184401826892729061875636925728556009094","public_key":{"p":[164,221,255,34,120,66,140,120,163,230,208,249,96,42,222,231,194,36,30,244,137,96,213,46,176,93,46,89,68,243,12,96,47,240,60,88,234,72,58,247,142,105,57,139,4,34,199,166,21,218,230,130,179,163,111,135,245,255,146,42,84,213,78,12,203,60,198,79,49,4,168,66,220,234,78,122,153,164,15,62,154,215,52,45,226,9,170,8,82,26,241,74,132,7,27,158]},"private_key":{"x":[126,216,212,87,252,248,31,247,22,253,251,174,250,126,133,143,126,198,80,240,71,33,187,76,120,15,42,84,195,63,242,10]}}`
	testAccount2 = `{"index":"257006940103443962868532947373235556938271693750136125808684034232840884830638208732593735565637276170886442912667767381105455605885467612946853679655161","public_key":{"p":[169,63,130,44,124,41,185,231,49,229,169,230,236,191,119,183,4,112,139,183,168,59,194,72,98,180,198,46,210,24,250,22,148,209,179,146,156,26,249,90,15,187,58,205,155,3,160,80,6,42,138,63,246,4,214,33,158,33,27,171,56,221,230,3,26,237,14,14,2,118,110,250,255,90,48,117,217,188,141,5,17,2,51,250,168,16,86,155,139,33,151,233,182,100,29,132]},"private_key":{"x":[191,221,254,233,125,198,203,133,10,143,8,111,70,134,181,224,108,192,24,15,179,45,55,110,159,120,218,127,214,121,57,16]}}`
	testAccount3 = `{"index":"1368445478336047788671563167816769065025819906297195306486206640016117886676338763178411004319413299193570518260109002737706123789987243019974525949913989","public_key":{"p":[184,218,106,191,140,15,62,11,237,138,34,150,245,179,186,91,63,162,67,110,202,20,162,205,58,212,132,189,215,247,6,53,236,133,53,9,194,85,222,183,0,8,255,214,152,3,243,169,7,19,149,12,36,48,231,150,100,133,58,242,212,122,201,127,57,216,170,26,7,150,6,140,38,199,56,134,94,195,102,91,237,165,77,145,238,189,125,173,203,233,14,246,157,165,101,94]},"private_key":{"x":[167,8,106,250,211,151,167,45,169,99,223,47,114,152,242,220,254,243,216,36,205,110,87,211,42,64,177,205,71,244,232,32]}}`

	testIndex1 = "5068161082419015240502520392764951298362816233167274727352107679212581830087892411256674283175582004835800509618391184401826892729061875636925728556009094"
	testIndex2 = "257006940103443962868532947373235556938271693750136125808684034232840884830638208732593735565637276170886442912667767381105455605885467612946853679655161"
	testIndex3 = "1368445478336047788671563167816769065025819906297195306486206640016117886676338763178411004319413299193570518260109002737706123789987243019974525949913989"

	m11Data = `[176,154,54,219,26,170,77,59,4,33,64,14,125,245,238,126,47,235,36,229,103,132,104,43,141,122,112,105,21,255,29,96,221,194,37,90,248,230,231,1,80,27,91,90,131,191,244,36]`
	m21Data = `[177,67,89,158,248,146,78,40,202,75,245,127,164,186,17,246,135,9,158,149,161,137,74,105,63,39,107,133,162,173,132,219,207,127,199,212,200,220,159,190,163,147,122,234,19,55,253,4]`
	m31Data = `[145,38,78,132,62,54,110,225,174,110,232,153,185,27,154,97,131,129,210,133,70,91,163,216,134,136,59,236,253,182,116,44,146,178,64,97,7,2,141,217,149,225,208,96,13,119,185,82]`

	testMsg = "msg for bls sign"
)

var testAccounts = []string{testAccount1, testAccount2, testAccount3}

func TestNewBlsClient(t *testing.T) {
	client := NewBlsClient()
	assert.NotNil(t, client, "client not created")
	assert.NotNil(t, client.ThresholdRatio, "threshold ratio not initialized.")
}

func TestBlsClient_UpdateGroup_givenCase(t *testing.T) {
	// prepare
	accounts := loadGivenAccounts()
	client := NewBlsClient()
	for _, firstAccount := range accounts {
		client.Account = firstAccount
		break
	}

	// invoke
	peers := accounts
	err := client.UpdateGroup(peers)
	assert.Nil(t, err, "error updating group")

	// testing:
	// test P
	pData := `[176,118,190,38,138,106,96,189,173,197,167,218,215,119,118,0,89,250,171,101,228,37,17,196,98,58,128,47,253,28,169,81,252,101,5,58,24,136,209,69,187,78,1,10,198,136,70,61,24,85,196,10,104,233,235,219,20,111,148,206,12,50,192,51,123,59,247,236,160,96,60,118,231,14,193,54,175,72,181,100,112,133,67,233,73,228,226,246,54,224,255,182,222,15,210,3]`
	var expectP bls.PublicKey
	_ = json.Unmarshal([]byte(pData), &expectP)
	assert.Equal(t, expectP, client.Group.P, "expected group P got %v", client.Group.P)

	// test K
	k1 := `[140,37,229,3,19,243,96,246,122,72,200,180,52,0,9,179,19,143,244,94,173,162,87,36,197,29,38,250,225,74,109,30]`
	k2 := `[64,5,229,211,128,239,1,117,214,132,30,54,135,103,75,207,146,58,221,141,164,165,166,166,198,55,217,183,99,105,50,45]`
	k3 := `[17,235,183,101,130,152,244,119,179,103,210,226,225,234,175,154,67,188,110,193,22,102,81,189,73,175,165,51,86,185,167,79]`
	expectK := map[string]string{
		testIndex1: k1,
		testIndex2: k2,
		testIndex3: k3,
	}
	assert.Equal(t, expectK, client.Group.K, "expected group's K got %", client.Group.K)

	// test public key Part
	keyPartData1 := `[144,107,202,68,218,60,144,121,71,4,48,142,101,193,13,253,251,119,200,106,212,137,133,11,124,242,114,134,133,54,19,56,27,59,199,200,145,156,18,219,185,159,236,20,239,153,221,203,25,24,28,48,139,165,230,111,245,55,237,155,67,25,103,100,176,140,9,107,197,192,178,30,159,171,236,233,27,215,47,17,160,210,246,240,137,111,239,218,218,194,71,145,167,147,65,43]`
	keyPartData2 := `[140,91,1,59,43,210,222,10,6,106,41,166,145,201,102,122,53,61,248,248,132,185,57,235,12,144,187,196,238,114,212,47,144,71,31,233,140,144,223,246,188,219,215,157,174,117,76,152,13,221,31,63,127,39,140,40,81,142,115,177,27,153,74,51,215,118,208,87,10,27,243,237,227,195,248,237,243,157,17,26,116,153,8,190,106,140,215,236,18,139,84,23,68,242,193,25]`
	keyPartData3 := `[143,22,71,145,250,9,189,255,235,144,51,175,92,243,53,152,129,13,148,241,36,186,185,45,92,50,37,172,179,156,36,208,189,134,175,66,130,212,208,39,59,30,123,231,174,6,152,19,11,4,94,145,16,168,49,127,47,178,147,158,26,67,239,214,235,132,104,249,222,139,44,247,105,118,78,76,44,190,221,212,89,152,89,79,23,230,163,93,149,35,197,49,157,40,53,104]`
	var expectKeyPart1, expectKeyPart2, expectKeyPart3 bls.PublicKey
	_ = json.Unmarshal([]byte(keyPartData1), &expectKeyPart1)
	_ = json.Unmarshal([]byte(keyPartData2), &expectKeyPart2)
	_ = json.Unmarshal([]byte(keyPartData3), &expectKeyPart3)
	expectKeyPart := map[string]bls.PublicKey{
		testIndex1: expectKeyPart1,
		testIndex2: expectKeyPart2,
		testIndex3: expectKeyPart3,
	}
	assert.Equal(t, expectKeyPart, client.Group.WeightedPublicKeys,
		"expected group's public key part got %v", client.Group.WeightedPublicKeys)

	// test P'
	pPrimeData := `[171,178,166,167,33,194,144,195,141,131,49,92,199,137,40,205,194,99,220,198,105,244,206,210,95,38,186,87,88,26,238,161,52,234,216,234,255,143,150,65,183,143,42,140,229,140,33,136,23,249,35,152,52,199,73,160,250,200,135,42,141,63,9,14,138,207,168,154,182,3,167,87,169,131,193,34,43,75,174,123,28,240,144,82,12,170,148,89,101,249,145,185,93,14,98,157]`
	var expectPPrime bls.PublicKey
	_ = json.Unmarshal([]byte(pPrimeData), &expectPPrime)
	assert.Equal(t, expectPPrime, client.Group.PPrime, "expected group P' got %v", client.Group.PPrime)
}

func TestBlsClient_GenerateMKParts_randomCase(t *testing.T) {
	client, err := NewHelper(3).ClientWithGroup()
	assert.NoError(t, err)

	mkParts, err := client.GenerateMkParts()
	assert.Nil(t, err, "error generating MK parts")
	assert.Equal(t, client.Group.Size(), len(mkParts), "MK parts count not match")

	t.Log(mkParts)
}

func TestBlsClient_GenerateMKParts_givenCase(t *testing.T) {

	// prepare
	accounts := loadGivenAccounts()
	client := NewBlsClient()
	client.Account = accounts[testIndex1]

	peers := accounts
	err := client.UpdateGroup(peers)
	assert.Nil(t, err, "error updating group")

	// invoke
	mkParts, err := client.GenerateMkParts()
	assert.Nil(t, err, "error generating MK parts")

	// testing
	t.Log(mkParts)
	var expectM11 bls.MkPart
	_ = json.Unmarshal([]byte(m11Data), &expectM11)
	assert.Equal(t, expectM11, mkParts[testIndex1], "expected MK part 1->1 got %v", mkParts)
}

func TestBlsClient_UpdateMK_randomCase(t *testing.T) {
	groupSize := 3
	clients, err := NewHelper(groupSize).ClientsWithGroup()
	assert.NoError(t, err)
	mkParts := make(map[string]map[string]bls.MkPart, groupSize)

	// generate MK parts
	for fromIndex, client := range clients {
		partsFromIndex, err := client.GenerateMkParts()
		assert.Nil(t, err, "error generating MK parts")
		assert.Equal(t, groupSize, len(partsFromIndex), "MK parts count not match")
		mkParts[fromIndex] = partsFromIndex
	}

	// exchange MK parts with peer
	for toIndex, client := range clients {
		partsToIndex := make([]bls.MkPart, 0, groupSize)
		for fromIndex := range mkParts {
			if fromIndex == toIndex {
				continue
			}
			partsToIndex = append(partsToIndex, mkParts[fromIndex][toIndex])
		}

		// update MK
		err := client.UpdateMk(partsToIndex)
		assert.Nil(t, err, "error updating MK part #%v", toIndex)
		assert.NotNil(t, client.Mk, "MK part not updated!")
	}
}

func TestBlsClient_UpdateMk_givenCase(t *testing.T) {
	clients := loadGroup()
	mkParts := make(map[string]map[string]bls.MkPart)

	// generate MK parts
	var expectM11, expectM21, expectM31 bls.MkPart
	_ = json.Unmarshal([]byte(m11Data), &expectM11)
	_ = json.Unmarshal([]byte(m21Data), &expectM21)
	_ = json.Unmarshal([]byte(m31Data), &expectM31)
	expectMKPart := map[string]bls.MkPart{
		testIndex1: expectM11,
		testIndex2: expectM21,
		testIndex3: expectM31,
	}
	for fromIndex, client := range clients {
		partsFromIndex, err := client.GenerateMkParts()
		assert.Nil(t, err, "error generating MK parts")
		assert.Equal(t, len(clients), len(partsFromIndex), "MK parts count not match")
		assert.Equal(t, expectMKPart[fromIndex], partsFromIndex[testIndex1],
			"expected MK part *->1 got %v", mkParts)
		mkParts[fromIndex] = partsFromIndex
	}

	// exchange MK parts with peer
	for toIndex, client := range clients {
		partsToIndex := make([]bls.MkPart, 0, len(clients))
		for fromIndex := range mkParts {
			if fromIndex == toIndex {
				continue
			}
			partsToIndex = append(partsToIndex, mkParts[fromIndex][toIndex])
		}

		// invoke: update MK
		err := client.UpdateMk(partsToIndex)
		assert.Nil(t, err, "error updating MK part #%v", toIndex)
		assert.NotNil(t, client.Mk, "MK part not updated!")
	}

	// testing
	mk1Data := `[138,213,55,153,23,180,53,142,94,173,156,106,58,2,64,148,75,59,224,182,10,102,96,186,2,201,77,149,141,167,95,30,255,241,176,29,70,104,59,59,49,25,184,198,19,145,158,103]`
	var expectMK1 bls.Mk
	_ = json.Unmarshal([]byte(mk1Data), &expectMK1)
	assert.Equal(t, expectMK1, clients[testIndex1].Mk,
		"expected MK got: %v", clients[testIndex1].Mk)
}

func TestBlsClient_verifyMk(t *testing.T) {
	groupSize := 3
	helper := NewHelper(groupSize)
	clients, err := helper.ClientsWithGroup()
	assert.NoError(t, err)
	for index, client := range clients {
		result := client.verifyMk()
		assert.False(t, result, "error verifying MK #%v", index)
	}

	clients, err = helper.ClientsWithMk()
	assert.Nil(t, err, "error updating MK")
	for index, client := range clients {
		result := client.verifyMk()
		assert.True(t, result, "error verifying MK #%v", index)
	}
}

func TestBlsClient_verifySignature(t *testing.T) {
	groupSize := 3
	helper := NewHelper(groupSize)
	clients, err := helper.ClientsWithMk()
	assert.Equal(t, groupSize, len(clients))
	assert.NoError(t, err)

	signParts, err := helper.SignMessage(testMsg)
	assert.Nil(t, err, "error signing message")

	accountIndexes := make([]string, 0, groupSize)
	for index := range clients {
		accountIndexes = append(accountIndexes, index)
	}

	type args struct {
		partIndexes []int
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "12",
			args: args{
				partIndexes: []int{0, 1},
			},
			want: true,
		},
		{
			name: "13",
			args: args{
				partIndexes: []int{1, 2},
			},
			want: true,
		},
		{
			name: "23",
			args: args{
				partIndexes: []int{2, 0},
			},
			want: true,
		},
		{
			name: "123",
			args: args{
				partIndexes: []int{0, 1, 2},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// prepare combination for signature parts
			parts := make(map[string]ffi.BlsSignaturePart)
			for _, i := range tt.args.partIndexes {
				account := accountIndexes[i]
				parts[account] = signParts[account]
			}
			client := clients.One()

			// combine signature
			sign, err := client.CombineSignatureParts(parts)
			assert.Nil(t, err, "error combining signature parts")

			// invoke verify
			result := client.VerifySignature(testMsg, sign)
			assert.Equal(t, tt.want, result,
				"error verifying signature for parts #%v", tt.args.partIndexes)
		})
	}
}

func TestBlsClient_VerifySignature(t *testing.T) {
	groupSize := 3
	helper := NewHelper(groupSize)
	clients, err := helper.ClientsWithMk()
	assert.NoError(t, err)
	sign, err := helper.ThresholdSignMessage(testMsg)
	assert.Nil(t, err, "error threshold signing message")

	oneClient := clients.One()
	rightPPrime := oneClient.Group.PPrime
	type args struct {
		message string
		pPrime  bls.PublicKey
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "pass",
			args: args{
				message: testMsg,
				pPrime:  rightPPrime,
			},
			want: true,
		},
		{
			name: "empty message",
			args: args{
				message: "",
				pPrime:  rightPPrime,
			},
			want: false,
		},
		{
			name: "wrong message",
			args: args{
				message: "wrong message",
				pPrime:  rightPPrime,
			},
			want: false,
		},
		{
			name: "empty P'",
			args: args{
				message: testMsg,
				pPrime:  nil,
			},
			want: false,
		},
		// TODO: wrong P' as panic
		//{
		//	name: "random P'",
		//	args: args{
		//		message: testMsg,
		//		pPrime:  []byte("mock wrong P'"),
		//	},
		//	want: false,
		//},
		{
			name: "wrong P'",
			args: args{
				message: testMsg,
				pPrime:  oneClient.Account.PublicKey,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// invoke verify
			t.Log("VerifyThresholdSignature verify signature with",
				"\nP':", tt.args.pPrime, "\nsign:", sign, "\nmessage:", tt.args.message)
			result := bls.VerifyThresholdSignature(tt.args.pPrime, sign, tt.args.message)
			assert.Equal(t, tt.want, result,
				"error verifying threshold signature for parts #%v", tt.name)
		})
	}
}

func loadGivenAccounts() map[string]*bls.Account {
	accounts := make(map[string]*bls.Account, len(testAccounts))
	for _, testAccount := range testAccounts {
		account, _ := bls.NewAccountFromJson(testAccount)
		accounts[account.Index] = account
	}
	return accounts
}

type ClientGroup map[string]*BlsClient

type Helper struct {
	bls.Helper
	Clients ClientGroup
}

func NewHelper(groupSize int) *Helper {
	return &Helper{
		Helper:  *bls.NewHelper(groupSize),
		Clients: make(ClientGroup, groupSize),
	}
}

func (h *Helper) ClientWithGroup() (*BlsClient, error) {
	client := NewBlsClient()
	client.Account = h.Self
	err := client.UpdateGroup(h.Members)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func (h *Helper) ClientsWithGroup() (ClientGroup, error) {
	for index, account := range h.Members {
		client := NewBlsClient()
		client.Account = account
		err := client.UpdateGroup(h.Members)
		if err != nil {
			return nil, err
		}
		h.Clients[index] = client
	}
	return h.Clients, nil
}

func (h *Helper) ClientsWithMk() (ClientGroup, error) {
	if len(h.Clients) == 0 {
		_, err := h.ClientsWithGroup()
		if err != nil {
			return nil, err
		}
	}

	// generate MK parts
	groupSize := len(h.Clients)
	mkParts := make(map[string][]bls.MkPart, groupSize)
	for fromIndex, client := range h.Clients {
		partsFromIndex, err := client.GenerateMkParts()

		// manage MK parts for exchange with peer
		if err != nil {
			return nil, fmt.Errorf(`error generating MK parts for client #%v: %v`, fromIndex, err)
		}
		for toIndex, mkPart := range partsFromIndex {
			if toIndex == fromIndex {
				continue
			}
			mkParts[toIndex] = append(mkParts[toIndex], mkPart)
		}
	}

	// update MK
	for toIndex, client := range h.Clients {
		err := client.UpdateMk(mkParts[toIndex])
		if err != nil {
			return nil, fmt.Errorf("error updating MK part #%v: %v", toIndex, err)
		}
	}
	return h.Clients, nil
}

// SignMessage signs message by each client in group
// call after MK initialized
func (h *Helper) SignMessage(message string) (map[string]ffi.BlsSignaturePart, error) {
	parts := make(map[string]ffi.BlsSignaturePart, len(h.Clients))

	for index, client := range h.Clients {
		sigPart, err := client.Sign([]byte(message))
		if err != nil {
			return nil, fmt.Errorf("error signing message #%v: %v", index, err)
		}
		parts[index] = sigPart
	}

	return parts, nil
}

// ThresholdSignMessage sign message by group with threshold
// call after MK initialized
func (h *Helper) ThresholdSignMessage(message string) (ffi.BlsSignature, error) {
	// distributed sign
	signAllParts, err := h.SignMessage(message)
	if err != nil {
		return ffi.BlsSignature{}, err
	}

	// exchange signatures
	oneClient := h.Clients.One()
	threshold := oneClient.Group.Threshold
	signParts := make(map[string]ffi.BlsSignaturePart, threshold)
	for index, part := range signAllParts {
		signParts[index] = part
		if len(signParts) >= threshold {
			break
		}
	}

	// combine signatures
	return oneClient.CombineSignatureParts(signParts)
}

func (g ClientGroup) One() *BlsClient {
	for _, client := range g {
		return client
	}
	return nil
}

func loadGroup() ClientGroup {
	clients := make(ClientGroup)
	accounts := loadGivenAccounts()
	for index, account := range accounts {
		client := NewBlsClient()
		client.Account = account
		clients[index] = client
	}

	// update group
	for _, client := range clients {
		_ = client.UpdateGroup(accounts)
	}

	return clients
}
