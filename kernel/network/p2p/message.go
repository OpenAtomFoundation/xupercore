package p2p

import (
	"errors"
	"github.com/xuperchain/xupercore/kernel/network/def"
	"hash/crc32"

	"github.com/golang/protobuf/proto"
	"github.com/golang/snappy"

	"github.com/xuperchain/xupercore/lib/utils"
	pb "github.com/xuperchain/xupercore/protos"
)

var (
	ErrMessageChecksum   = errors.New("verify checksum error")
	ErrMessageDecompress = errors.New("decompress error")
	ErrMessageUnmarshal  = errors.New("message unmarshal error")
)

// define message versions
const (
	MessageVersion1 = "1.0.0"
	MessageVersion2 = "2.0.0"
	MessageVersion3 = "3.0.0"
)

// NewMessage create P2P message instance with given params
func NewMessage(typ pb.XuperMessage_MessageType, message proto.Message, opts ...MessageOption) *pb.XuperMessage {
	msg := &pb.XuperMessage{
		Header: &pb.XuperMessage_MessageHeader{
			Version:        MessageVersion3,
			Bcname:         def.BlockChain,
			Logid:          utils.GenLogId(),
			Type:           typ,
			EnableCompress: false,
			ErrorType:      pb.XuperMessage_NONE,
		},
		Data: &pb.XuperMessage_MessageData{},
	}

	if message != nil {
		data, _ := proto.Marshal(message)
		msg.Data.MsgInfo = data
	}

	for _, op := range opts {
		op(msg)
	}

	Compress(msg)
	msg.Header.DataCheckSum = Checksum(msg)
	return msg
}

// Unmarshal unmarshal msgInfo
func Unmarshal(msg *pb.XuperMessage, message proto.Message) error {
	if !VerifyChecksum(msg) {
		return ErrMessageChecksum
	}

	data, err := Decompress(msg)
	if err != nil {
		return ErrMessageDecompress
	}

	err = proto.Unmarshal(data, message)
	if err != nil {
		return ErrMessageUnmarshal
	}

	return nil
}

type MessageOption func(*pb.XuperMessage)

func WithBCName(bcname string) MessageOption {
	return func(msg *pb.XuperMessage) {
		msg.Header.Bcname = bcname
	}
}

// WithLogId set message logId
func WithLogId(logid string) MessageOption {
	return func(msg *pb.XuperMessage) {
		msg.Header.Logid = logid
	}
}

func WithVersion(version string) MessageOption {
	return func(msg *pb.XuperMessage) {
		msg.Header.Version = version
	}
}

func WithErrorType(errorType pb.XuperMessage_ErrorType) MessageOption {
	return func(msg *pb.XuperMessage) {
		msg.Header.ErrorType = errorType
	}
}

// Checksum calculate checksum of message
func Checksum(msg *pb.XuperMessage) uint32 {
	return crc32.ChecksumIEEE(msg.GetData().GetMsgInfo())
}

// VerifyChecksum verify the checksum of message
func VerifyChecksum(msg *pb.XuperMessage) bool {
	return crc32.ChecksumIEEE(msg.GetData().GetMsgInfo()) == msg.GetHeader().GetDataCheckSum()
}

// Compressed compress msg
func Compress(msg *pb.XuperMessage) *pb.XuperMessage {
	if len(msg.GetData().GetMsgInfo()) == 0 {
		return msg
	}

	if msg == nil || msg.GetHeader().GetEnableCompress() {
		return msg
	}
	msg.Data.MsgInfo = snappy.Encode(nil, msg.Data.MsgInfo)
	msg.Header.EnableCompress = true
	return msg
}

// Decompress decompress msg
func Decompress(msg *pb.XuperMessage) ([]byte, error) {
	if msg == nil || msg.Header == nil || msg.Data == nil || msg.Data.MsgInfo == nil {
		return []byte{}, errors.New("param error")
	}

	if !msg.Header.GetEnableCompress() {
		return msg.Data.MsgInfo, nil
	}

	return snappy.Decode(nil, msg.Data.MsgInfo)
}

// VerifyMessageType 用于带返回的请求场景下验证收到的消息是否为预期的消息
func VerifyMessageType(request *pb.XuperMessage, response *pb.XuperMessage, peerID string) bool {
	if response.GetHeader().GetFrom() != peerID {
		return false
	}

	if request.GetHeader().GetLogid() != response.GetHeader().GetLogid() {
		return false
	}

	if GetRespMessageType(request.GetHeader().GetType()) != response.GetHeader().GetType() {
		return false
	}

	return true
}

// 消息类型映射
// 避免每次添加新消息都要修改，制定映射关系：request = n(偶数) => response = n+1(奇数)
var requestToResponse = map[pb.XuperMessage_MessageType]pb.XuperMessage_MessageType{
	pb.XuperMessage_GET_BLOCK:                pb.XuperMessage_GET_BLOCK_RES,
	pb.XuperMessage_GET_BLOCKCHAINSTATUS:     pb.XuperMessage_GET_BLOCKCHAINSTATUS_RES,
	pb.XuperMessage_CONFIRM_BLOCKCHAINSTATUS: pb.XuperMessage_CONFIRM_BLOCKCHAINSTATUS_RES,
	pb.XuperMessage_GET_RPC_PORT:             pb.XuperMessage_GET_RPC_PORT_RES,
	pb.XuperMessage_GET_AUTHENTICATION:       pb.XuperMessage_GET_AUTHENTICATION_RES,
}

// GetRespMessageType get the message type
func GetRespMessageType(msgType pb.XuperMessage_MessageType) pb.XuperMessage_MessageType {
	if resp, ok := requestToResponse[msgType]; ok {
		return resp
	}

	return msgType + 1
}
