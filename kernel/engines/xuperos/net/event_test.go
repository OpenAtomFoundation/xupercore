package net

import (
	"errors"
	"testing"

	xconf "github.com/xuperchain/xupercore/kernel/common/xconfig"
	xctx "github.com/xuperchain/xupercore/kernel/common/xcontext"
	"github.com/xuperchain/xupercore/kernel/engines/xuperos/common"
	nctx "github.com/xuperchain/xupercore/kernel/network/context"
	"github.com/xuperchain/xupercore/kernel/network/p2p"
	"github.com/xuperchain/xupercore/lib/logs"
	"github.com/xuperchain/xupercore/protos"
)

type mockEngine struct{}

func (m mockEngine) Init(conf *xconf.EnvConf) error {
	panic("implement me")
}

func (m mockEngine) Run() {
	panic("implement me")
}

func (m mockEngine) Exit() {
	panic("implement me")
}

func (m mockEngine) Get(s string) (common.Chain, error) {
	panic("implement me")
}

func (m mockEngine) GetChains() []string {
	panic("implement me")
}

func (m mockEngine) LoadChain(s string) error {
	panic("implement me")
}

func (m mockEngine) Stop(s string) error {
	panic("implement me")
}

func (m mockEngine) Context() *common.EngineCtx {
	return &common.EngineCtx{
		Net: new(mockNet),
	}
}

func (m mockEngine) SetRelyAgent(agent common.EngineRelyAgent) error {
	panic("implement me")
}

type mockNet struct{}

func (m mockNet) Start() {
	panic("implement me")
}

func (m mockNet) Stop() {
	panic("implement me")
}

func (m mockNet) SendMessage(_ xctx.XContext, message *protos.XuperMessage, _ ...p2p.OptionFunc) error {
	if message == nil {
		return errors.New("nil message")
	}
	return nil
}

func (m mockNet) SendMessageWithResponse(context xctx.XContext, message *protos.XuperMessage,
	optionFunc ...p2p.OptionFunc) ([]*protos.XuperMessage, error) {
	panic("implement me")
}

func (m mockNet) NewSubscriber(messageType protos.XuperMessage_MessageType, i interface{},
	option ...p2p.SubscriberOption) p2p.Subscriber {
	panic("implement me")
}

func (m mockNet) Register(subscriber p2p.Subscriber) error {
	panic("implement me")
}

func (m mockNet) UnRegister(subscriber p2p.Subscriber) error {
	panic("implement me")
}

func (m mockNet) Context() *nctx.NetCtx {
	panic("implement me")
}

func (m mockNet) PeerInfo() protos.PeerInfo {
	panic("implement me")
}

func TestNetEvent_sendMessage(t *testing.T) {
	type fields struct {
		log      logs.Logger
		engine   common.Engine
		msgChan  chan *protos.XuperMessage
		exitChan chan bool
	}
	type args struct {
		ctx xctx.XContext
		msg *protos.XuperMessage
		of  []p2p.OptionFunc
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "send succ",
			fields: fields{
				engine: new(mockEngine),
			},
			args: args{
				msg: new(protos.XuperMessage),
			},
		},
		{
			name: "send fail",
			fields: fields{
				engine: new(mockEngine),
			},
			args: args{
				msg: nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &Event{
				log:      tt.fields.log,
				engine:   tt.fields.engine,
				msgChan:  tt.fields.msgChan,
				exitChan: tt.fields.exitChan,
			}
			e.sendMessage(tt.args.ctx, tt.args.msg, tt.args.of...)
		})
	}
}
