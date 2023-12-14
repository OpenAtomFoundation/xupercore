package bls

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	crypto "github.com/OpenAtomFoundation/xupercore/crypto-dll-go"
	"github.com/OpenAtomFoundation/xupercore/crypto-dll-go/bls"
	xctx "github.com/OpenAtomFoundation/xupercore/global/kernel/common/xcontext"
	"github.com/OpenAtomFoundation/xupercore/global/kernel/engines/xuperos/common"
	"github.com/OpenAtomFoundation/xupercore/global/kernel/network"
	"github.com/OpenAtomFoundation/xupercore/global/kernel/network/p2p"
	"github.com/OpenAtomFoundation/xupercore/global/lib/logs"
	"github.com/OpenAtomFoundation/xupercore/global/protos"
	"github.com/OpenAtomFoundation/xupercore/global/service/pb"
)

var (
	thresholdSign *ThresholdSign
	initOnce      sync.Once
)

func init() {
	initOnce.Do(func() {
		thresholdSign = &ThresholdSign{
			client: crypto.NewBlsClient(),
		}
		initGroup := bls.Group{
			Members: make(map[string]*bls.Account),
		}
		thresholdSign.ts = bls.NewThresholdSign(initGroup, thresholdSign.client.Account)
	})
}

// ThresholdSign control threshold sign process
type ThresholdSign struct {
	client *crypto.BlsClient
	engine *common.EngineCtx
	ts     bls.ThresholdSign

	height uint64 // height for current BLS threshold sign
	sign   bls.Signature
	proof  bls.Proof

	groupUpdating  chan bool
	indexToID      sync.Map
	indexToAccount sync.Map
}

func (s *ThresholdSign) updateGroupMember(ctx xctx.XContext, req pb.BroadcastPublicKeyRequest, peerId string) (*pb.BroadcastPublicKeyResponse, error) {

	// parse peer account
	peer := bls.Account{
		Index:     req.Index,
		PublicKey: req.PublicKey,
	}
	s.indexToID.LoadOrStore(peer.Index, peerId)
	s.indexToAccount.LoadOrStore(peer.Index, peer)
	expectGroup := req.Group

	// update group
	ctx.GetLog().Debug("updateGroupMember stored index", "peerIndex", peer.Index)
	controlSignal := len(expectGroup) > 0
	if controlSignal && !s.isGroupRemain(expectGroup) {
		if s.groupUpdating == nil {
			s.groupUpdating = make(chan bool)
		}
		go func() {
			err := s.updateGroup(ctx, expectGroup, false)
			if err != nil {
				ctx.GetLog().Error("passive update group failed", "error", err)
			}
			err = s.exchangeMemberSigns(ctx)
			if err != nil {
				ctx.GetLog().Error("passive exchange member sign failed", "error", err)
			}
		}()
	}

	// return self account info
	me := s.client.Account
	resp := &pb.BroadcastPublicKeyResponse{
		Success:   true,
		ErrorCode: CodeSucc,
		Index:     me.Index,
		PublicKey: me.PublicKey,
	}
	return resp, nil
}

func (s *ThresholdSign) updateMemberSignPart(req pb.SendMemberSignRequest) (*pb.SendMemberSignResponse, error) {
	// parse peer member sign to self
	peerSignMe := req.MemberSign
	peer := req.GetIndex()
	client := s.client
	group := s.ts.Group
	if s.groupUpdating != nil {
		select {
		case _ = <-s.groupUpdating:
		case <-time.After(time.Second * 1):
			return nil, &common.Error{
				Status: common.ErrStatusInternalErr,
				Code:   CodeErrMk,
				Msg:    "group updating not finish",
			}
		}
	}
	if _, isMember := group.Members[peer]; !isMember {
		return nil, common.ErrParameter
	}

	// update MK part
	go func() {
		enough, err := s.ts.CollectMkParts(peer, peerSignMe)
		log, _ := logs.NewLogger("", "threshold")
		log.Debug("update peer",
			"peer index", peer,
			"enough", enough,
			"error", err)
	}()

	// return self member sign to peer
	meSignPeer := s.ts.MkPartsTo[peer]
	resp := &pb.SendMemberSignResponse{
		Success:    true,
		ErrorCode:  CodeSucc,
		Index:      client.Account.Index,
		MemberSign: meSignPeer,
	}
	return resp, nil
}

func (s *ThresholdSign) updateMessageSignPart(ctx xctx.XContext, req pb.BroadcastMessageSignRequest) (*pb.BroadcastMessageSignResponse, error) {
	// parse peer message sign
	peer := req.Index
	if req.Message == "" {
		ctx.GetLog().Error("no message")
		return nil, common.ErrParameter
	}
	client := s.client
	group := s.ts.Group
	if _, isMember := group.Members[peer]; !isMember {
		ctx.GetLog().Error("unexpect peer",
			"peer", peer)
		return nil, common.ErrParameter
	}
	peerSign := bls.SignaturePart{}
	err := json.Unmarshal(req.MessageSign, &peerSign)
	if err != nil {
		ctx.GetLog().Error("invalid message sign",
			"error", err)
		return nil, common.ErrParameter
	}

	// update message sign part
	err = s.ts.WaitMk(time.Second * 2)
	if err != nil {
		return nil, &common.Error{
			Status: common.ErrStatusInternalErr,
			Code:   CodeErrSign,
			Msg:    err.Error(),
		}
	}
	meSign, err := s.ts.SignByAccount([]byte(req.Message))
	if err != nil {
		return nil, &common.Error{
			Status: common.ErrStatusInternalErr,
			Code:   CodeErrSign,
			Msg:    err.Error(),
		}
	}
	ctx.GetLog().Debug("combine message signature part",
		"peerIndex", req.Index,
		"signPart", peerSign)
	go s.ts.CollectSignaturePart(peerSign)

	// return self message sign
	meSignData, err := json.Marshal(meSign)
	if err != nil {
		return nil, &common.Error{
			Status: common.ErrStatusInternalErr,
			Code:   CodeErrSign,
			Msg:    err.Error(),
		}
	}
	go s.sendMessageSignOnce(ctx, req.Message, meSignData)
	resp := &pb.BroadcastMessageSignResponse{
		Success:     true,
		ErrorCode:   CodeSucc,
		Index:       client.Account.Index,
		Message:     req.Message,
		MessageSign: meSignData,
		Height:      s.height,
	}
	return resp, nil
}

// updateGroup update group by p2p network
// invoke when group changed
// Returns:
// - error
func (s *ThresholdSign) updateGroup(ctx xctx.XContext, expectGroup []string, sync bool) error {
	ctx.GetLog().Debug("ThresholdSign.updateGroup() invoked",
		"expectGroup", expectGroup,
		"sync", sync)

	if s.groupUpdating != nil {
		s.groupUpdating = make(chan bool)
	}

	req := pb.BroadcastPublicKeyRequest{
		Index:     s.client.Account.Index,
		PublicKey: s.client.Account.PublicKey,
	}
	if sync {
		req.Group = expectGroup
	}

	chainName := s.engine.ChainM.GetChains()[0]
	msgOpts := []p2p.MessageOption{
		p2p.WithBCName(chainName),
	}
	msg := p2p.NewMessage(protos.XuperMessage_BLS_GET_PUBLIC_KEY, &req, msgOpts...)
	responses, err := s.net().SendMessageWithResponse(ctx, msg, p2p.WithPeerIDs(expectGroup))
	if err != nil {
		ctx.GetLog().Debug("send message",
			"error", err)
		return common.ErrSendMessageFailed
	}

	receiveGroup := map[string]*bls.Account{
		s.client.Account.Index: {
			Index:     s.client.Account.Index,
			PublicKey: s.client.Account.PublicKey,
		},
	}
	for _, response := range responses {
		var resp pb.BroadcastPublicKeyResponse
		ctx.GetLog().Debug("receive public key",
			"peerId", response.Header.From)
		err := p2p.Unmarshal(response, &resp)
		if err != nil {
			ctx.GetLog().Error("unmarshal error", "chain", chainName, "error", err)
			return err
		}
		ctx.GetLog().Debug("receive public key",
			"index", resp.Index)
		if !resp.Success {
			ctx.GetLog().Error("broadcast fail", "chain", chainName, "error", resp.ErrorMessage)
			return fmt.Errorf("response with error: %s", resp.ErrorMessage)
		}
		receiveGroup[resp.Index] = &bls.Account{
			Index:     resp.Index,
			PublicKey: resp.PublicKey,
		}
		peerID := response.Header.From
		s.indexToID.LoadOrStore(resp.Index, peerID)
	}

	ctx.GetLog().Debug("UpdateGroup()",
		"receiveGroup", len(receiveGroup))
	return s.client.UpdateGroup(receiveGroup)
}

func (s *ThresholdSign) net() network.Network {
	return s.engine.Net
}

func (s *ThresholdSign) peerIDsFromNet() []string {
	self := s.net().PeerInfo()
	peers := make([]string, 0, len(self.Peer))
	for _, peer := range self.Peer {
		peers = append(peers, peer.Id)
	}
	return peers
}

func (s *ThresholdSign) peerIDsFromGroup() []string {
	peerIDs := make([]string, 0, s.ts.Group.Size())
	for index := range s.ts.Group.Members {
		if index == s.client.Account.Index {
			continue
		}
		id, ok := s.indexToID.Load(index)
		if !ok {
			s.engine.GetLog().Warn("group member's index not maintained",
				"index", index)
			continue
		}
		peerIDs = append(peerIDs, id.(string))
	}
	return peerIDs
}

func (s *ThresholdSign) electGroup() []string {
	self := s.net().PeerInfo()
	peers := s.peerIDsFromNet()
	account := s.client.Account
	s.indexToID.LoadOrStore(account.Index, self.Id)
	s.indexToAccount.LoadOrStore(account.Index, account)
	return append(peers, self.Id)
}

func (s *ThresholdSign) exchangeMemberSigns(ctx xctx.XContext) error {
	ctx.GetLog().Debug("ThresholdSign.exchangeMemberSigns() invoked",
		"chan", s.groupUpdating)

	s.ts = s.client.ThresholdSign()
	ctx.GetLog().Debug("new thresholdSign", "P'", s.ts.Group.PPrime)
	mkPartsTo, err := s.ts.MkPartsByAccount()
	ctx.GetLog().Debug("exchangeMemberSigns mkPartsTo generated")
	if err != nil {
		return err
	}
	if s.groupUpdating != nil {
		ctx.GetLog().Debug("close chan for group updating",
			"chan", s.groupUpdating)
		close(s.groupUpdating)
		s.groupUpdating = nil
	}

	wg := sync.WaitGroup{}
	for peerIndex, mkPart := range mkPartsTo {
		if peerIndex == s.client.Account.Index {
			continue
		}
		wg.Add(1)
		go func(peer string, sign bls.MkPart) {
			err := s.exchangeMemberSign(ctx, peer, sign)
			if err != nil {
				ctx.GetLog().Error("exchangeMemberSigns failed",
					"peer", peer,
					"error", err)
			}
			wg.Done()
		}(peerIndex, mkPart)
	}
	wg.Wait()

	return nil
}

func (s *ThresholdSign) exchangeMemberSign(ctx xctx.XContext, peerIndex string, part bls.MkPart) error {

	peerID, exist := s.indexToID.Load(peerIndex)
	if !exist {
		return fmt.Errorf("peer ID not marked for index: %s", peerIndex)
	}
	ctx.GetLog().Debug("ThresholdSign.exchangeMemberSign() invoked",
		"peerIndex", peerIndex,
		"peerID", peerID,
		"MK part", part)
	req := pb.SendMemberSignRequest{
		Index:      s.client.Account.Index,
		MemberSign: part,
	}

	chainName := s.engine.ChainM.GetChains()[0]
	msgOpts := []p2p.MessageOption{
		p2p.WithBCName(chainName),
	}
	msg := p2p.NewMessage(protos.XuperMessage_BLS_GET_MEMBER_SIGN_PART, &req, msgOpts...)
	responses, err := s.net().SendMessageWithResponse(ctx, msg, p2p.WithPeerIDs([]string{peerID.(string)}))
	ctx.GetLog().Debug("send member sign with response",
		"chain", chainName,
		"peerIndex", peerIndex,
		"peerID", peerID,
		"error", err,
		"peers", s.peerIDsFromNet(),
		"response len:", len(responses))
	if err != nil {
		return err
	}

	for _, response := range responses {
		resp := pb.SendMemberSignResponse{}
		err := p2p.Unmarshal(response, &resp)
		if err != nil {
			ctx.GetLog().Error("unmarshal error", "error", err)
			return err
		}
		if len(resp.MemberSign) == 0 {
			err := fmt.Errorf("empty member sign")
			ctx.GetLog().Error(err.Error())
			return err
		}
		done, err := s.ts.CollectMkParts(resp.Index, resp.MemberSign)
		if err != nil {
			ctx.GetLog().Error("update MK part error", "error", err)
			return err
		}
		if done {
			ctx.GetLog().Info("update MK part done")
			return nil
		}
	}
	return nil
}

func (s *ThresholdSign) exchangeMessageSign(ctx xctx.XContext, message string) error {
	ctx.GetLog().Debug("ThresholdSign.exchangeMessageSign() invoked",
		"message", message,
		"MK", s.ts.Mk)

	sign, err := s.ts.SignByAccount([]byte(message))
	if err != nil {
		ctx.GetLog().Error("sign message error", "error", err)
		return err
	}
	signData, err := json.Marshal(sign)
	if err != nil {
		ctx.GetLog().Error("marshal message sign error", "error", err)
		return err
	}

	responses, err := s.sendMessageSignOnce(ctx, message, signData)
	if err != nil {
		ctx.GetLog().Error("send message sign error", "error", err)
		return err
	}

	for _, response := range responses {
		var resp pb.BroadcastMessageSignResponse
		err = p2p.Unmarshal(response, &resp)
		if err != nil {
			ctx.GetLog().Warn("unmarshal error", "error", err)
			continue
		}
		if !resp.Success {
			ctx.GetLog().Warn("broadcast fail", "error", resp.ErrorMessage)
			continue
		}
		peerSign := bls.SignaturePart{}
		err = json.Unmarshal(resp.MessageSign, &peerSign)
		if err != nil {
			return common.ErrParameter
		}
		ctx.GetLog().Debug("combine message signature part",
			"peerIndex", resp.Index,
			"peerId", response.Header.From,
			"signPart", resp.MessageSign)
		_, partsEnough, err := s.ts.CollectSignaturePart(peerSign)
		if err != nil {
			ctx.GetLog().Error("update message sign error", "error", err)
		}
		if partsEnough {
			ctx.GetLog().Info("update message sign done")
			return nil
		}
	}
	return nil
}

func (s *ThresholdSign) sendMessageSignOnce(ctx xctx.XContext, message string,
	signData []byte) ([]*protos.XuperMessage, error) {

	var err error
	var resps []*protos.XuperMessage
	if s.ts.SignSend != nil {
		s.ts.SignSend.Do(
			func() {
				req := pb.BroadcastMessageSignRequest{
					Index:       s.client.Account.Index,
					Message:     message,
					MessageSign: signData,
				}
				chainName := s.engine.ChainM.GetChains()[0]
				msgOpts := []p2p.MessageOption{
					p2p.WithBCName(chainName),
				}
				msg := p2p.NewMessage(protos.XuperMessage_BLS_GET_MESSAGE_SIGN_PART, &req, msgOpts...)
				resps, err = s.net().SendMessageWithResponse(ctx, msg, p2p.WithPeerIDs(s.peerIDsFromGroup()))
			})
	}
	return resps, err
}

func (s *ThresholdSign) isGroupRemain(currentIDs []string) bool {
	if s.ts.Group.Size() == 0 {
		// must change for init
		return false
	}

	if s.ts.Group.Size() != len(currentIDs) {
		return false
	}

	// construct map for searching ID within previous group
	previousGroup := make(map[string]bool, len(currentIDs))
	for index := range s.ts.Group.Members {
		id, exist := s.indexToID.Load(index)
		if !exist {
			return false
		}
		previousGroup[id.(string)] = true
	}

	// check all peer in previous group
	for _, currentID := range currentIDs {
		_, exist := previousGroup[currentID]
		if !exist {
			return false
		}
	}
	return true
}
