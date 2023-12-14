package rpc

import (
	"context"
	"encoding/json"
	"fmt"

	ecom "github.com/OpenAtomFoundation/xupercore/global/kernel/engines/xuperos/common"
	"github.com/OpenAtomFoundation/xupercore/global/kernel/engines/xuperos/xrandom"
	eth "github.com/OpenAtomFoundation/xupercore/global/kernel/engines/xuperos/xrandom/ecdsa"
	scom "github.com/OpenAtomFoundation/xupercore/global/service/common"
	sconf "github.com/OpenAtomFoundation/xupercore/global/service/config"
	"github.com/OpenAtomFoundation/xupercore/global/service/pb"
)

type RandomServer struct {
	pb.UnimplementedRandomServer
	cfg    *sconf.ServConf
	engine ecom.Engine
}

func newRandomService(cfg *sconf.ServConf, engine ecom.Engine) (pb.RandomServer, error) {
	if cfg == nil {
		return nil, fmt.Errorf("cfg is nil")
	}

	svr := &RandomServer{
		cfg:    cfg,
		engine: engine,
	}
	return svr, nil
}

func (s *RandomServer) QueryRandomNumber(ctx context.Context,
	req *pb.QueryRandomNumberRequest) (*pb.QueryRandomNumberResponse, error) {

	rctx := scom.ValueReqCtx(ctx)
	rctx.GetLog().Debug("QueryRandomNumber", "req", req)
	bcName := s.engine.Context().EngCfg.RootChain
	chain, err := s.engine.Get(bcName)
	if err != nil {
		rctx.GetLog().Error("get chain failed", "error", err)
		return &pb.QueryRandomNumberResponse{
			ErrorCode:    int32(ecom.ErrChainNotExist.Code),
			ErrorMessage: err.Error(),
		}, nil
	}
	state := chain.Context().State
	args := queryArgs(req)
	rctx.GetLog().Debug("queryArgs", "args", args)
	data, err := state.QueryRandom(args)
	if err != nil {
		rctx.GetLog().Error("query random failed", "error", err)
		return &pb.QueryRandomNumberResponse{
			ErrorCode:    int32(ecom.ErrContractInvokeFailed.Code),
			ErrorMessage: err.Error(),
		}, nil
	}
	random, err := xrandom.NewRandom(data)
	if err != nil {
		rctx.GetLog().Error("parse random failed", "error", err)
		return &pb.QueryRandomNumberResponse{
			ErrorCode:    int32(ecom.ErrInternal.Code),
			ErrorMessage: err.Error(),
		}, nil
	}
	beaconSign, err := eth.Sign(random.Number)
	if err != nil {
		return &pb.QueryRandomNumberResponse{
			ErrorCode:    int32(ecom.ErrInternal.Code),
			ErrorMessage: err.Error(),
		}, nil
	}

	response := &pb.QueryRandomNumberResponse{
		RandomNumber: random.Number,
		Proof:        &random.Proof,
		Sign:         beaconSign,
	}
	return response, nil
}

func queryArgs(req *pb.QueryRandomNumberRequest) map[string][]byte {
	height, _ := json.Marshal(req.Height)
	args := map[string][]byte{
		xrandom.ParamHeight:        height,
		xrandom.ParamNodePublicKey: []byte(req.NodePublicKey),
		xrandom.ParamSign:          req.Sign,
	}
	return args
}
