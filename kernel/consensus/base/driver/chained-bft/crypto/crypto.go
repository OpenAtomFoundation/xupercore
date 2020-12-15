package crypto

import (
	"bytes"
	"encoding/json"
	"errors"

	"github.com/xuperchain/crypto/core/hash"
	pb "github.com/xuperchain/xupercore/kernel/consensus/base/driver/chained-bft/pb"
	cctx "github.com/xuperchain/xupercore/kernel/consensus/context"
)

type CBFTCrypto struct {
	Address      *cctx.Address
	CryptoClient cctx.CryptoClient
}

func NewCBFTCrypto(addr *cctx.Address, c cctx.CryptoClient) *CBFTCrypto {
	return &CBFTCrypto{
		Address:      addr,
		CryptoClient: c,
	}
}

func (c *CBFTCrypto) SignProposalMsg(msg *pb.ProposalMsg) (*pb.ProposalMsg, error) {
	msgDigest, err := MakeProposalMsgDigest(msg)
	if err != nil {
		return nil, err
	}
	msg.MsgDigest = msgDigest
	sign, err := c.CryptoClient.SignECDSA(c.Address.PrivateKey, msgDigest)
	if err != nil {
		return nil, err
	}
	msg.Sign = &pb.QuorumCertSign{
		Address:   c.Address.Address,
		PublicKey: c.Address.PublicKeyStr,
		Sign:      sign,
	}
	return msg, nil
}

// MakePhaseMsgDigest make ChainedBftPhaseMessage Digest
func MakeProposalMsgDigest(msg *pb.ProposalMsg) ([]byte, error) {
	msgEncoder, err := encodeProposalMsg(msg)
	if err != nil {
		return nil, err
	}
	msg.MsgDigest = hash.DoubleSha256(msgEncoder)
	return hash.DoubleSha256(msgEncoder), nil
}

func encodeProposalMsg(msg *pb.ProposalMsg) ([]byte, error) {
	var msgBuf bytes.Buffer
	encoder := json.NewEncoder(&msgBuf)
	if err := encoder.Encode(msg.ProposalView); err != nil {
		return nil, err
	}
	if err := encoder.Encode(msg.ProposalId); err != nil {
		return nil, err
	}
	if err := encoder.Encode(msg.Timestamp); err != nil {
		return nil, err
	}
	if err := encoder.Encode(msg.JustifyQC); err != nil {
		return nil, err
	}
	return msgBuf.Bytes(), nil
}

// SignVoteMsg make ChainedBftVoteMessage sign
func (c *CBFTCrypto) SignVoteMsg(msg []byte) (*pb.QuorumCertSign, error) {
	sign, err := c.CryptoClient.SignECDSA(c.Address.PrivateKey, msg)
	if err != nil {
		return nil, err
	}
	return &pb.QuorumCertSign{
		Address:   c.Address.Address,
		PublicKey: c.Address.PublicKeyStr,
		Sign:      sign,
	}, nil
}

func (c *CBFTCrypto) VerifyVoteMsgSign(sig *pb.QuorumCertSign, msg []byte) (bool, error) {
	ak, err := c.CryptoClient.GetEcdsaPublicKeyFromJsonStr(sig.GetPublicKey())
	if err != nil {
		return false, err
	}
	addr, err := c.CryptoClient.GetAddressFromPublicKey(ak)
	if err != nil {
		return false, err
	}
	if addr != sig.GetAddress() {
		return false, errors.New("VerifyVoteMsgSign error, addr not match pk: " + addr)
	}
	return c.CryptoClient.VerifyECDSA(ak, sig.GetSign(), msg)
}
