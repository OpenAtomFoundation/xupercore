package p2p

import (
	"testing"

	pb "github.com/xuperchain/xupercore/protos"
)

func TestMessage(t *testing.T) {
	data := &pb.XuperMessage{
		Data: &pb.XuperMessage_MessageData{
			MsgInfo: []byte("hello world"),
		},
	}

	cases := []*pb.XuperMessage{
		NewMessage(pb.XuperMessage_GET_BLOCK, data),
		NewMessage(pb.XuperMessage_GET_BLOCKCHAINSTATUS, data),
	}

	for i, req := range cases {
		var data pb.XuperMessage
		err := Unmarshal(req, &data)
		if err != nil {
			t.Errorf("case[%d]: unmarshal message error: %v", i, err)
			continue
		}

		respType := GetRespMessageType(req.GetHeader().GetType())
		resp := NewMessage(respType, &data)
		if VerifyMessageType(req, resp, "") {
			t.Errorf("case[%d]: verify message type error", i)
			continue
		}
	}

}
