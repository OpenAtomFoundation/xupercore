package p2p

import (
    "errors"
    "github.com/golang/protobuf/proto"
    "github.com/golang/snappy"
    pb "github.com/xuperchain/xupercore/kernel/network/pb"
    "github.com/xuperchain/xupercore/lib/utils"
    "hash/crc32"
)

var (
    ErrMessageChecksum      = errors.New("verify checksum error")
    ErrMessageDecompress    = errors.New("decompress error")
    ErrMessageUnmarshal     = errors.New("message unmarshal error")
)

// define message versions
const (
    MessageVersion1 = "1.0.0"
    MessageVersion2 = "2.0.0"
    MessageVersion3 = "3.0.0"
)

const (
    BlockChain = "xuper"
)

// NewMessage create P2P message instance with given params
func NewMessage(typ pb.XuperMessage_MessageType, message proto.Message, opts ...MessageOption) *pb.XuperMessage {
    msg := &pb.XuperMessage{
        Header: &pb.XuperMessage_MessageHeader{
            Version: MessageVersion3,
            Bcname: BlockChain,
            Logid: utils.GenLogId(),
            Type: typ,
            EnableCompress: false,
            ErrorType: pb.XuperMessage_NONE,
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

    err := Decompress(msg)
    if err != nil {
        return ErrMessageDecompress
    }

    err = proto.Unmarshal(msg.Data.MsgInfo, message)
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
func Decompress(msg *pb.XuperMessage) error {
    if len(msg.GetData().GetMsgInfo()) == 0 {
        return nil
    }

    originalMsg := msg.GetData().GetMsgInfo()
    var uncompressedMsg []byte
    var decodeErr error
    msgHeader := msg.GetHeader()
    if msgHeader != nil && msgHeader.GetEnableCompress() {
        uncompressedMsg, decodeErr = snappy.Decode(nil, originalMsg)
        if decodeErr != nil {
            return decodeErr
        }
    } else {
        uncompressedMsg = originalMsg
    }

    msg.Header.EnableCompress = false
    msg.Data.MsgInfo = uncompressedMsg
    return nil
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

// GetRespMessageType get the message type
func GetRespMessageType(msgType pb.XuperMessage_MessageType) pb.XuperMessage_MessageType {
    switch msgType {
    case pb.XuperMessage_GET_BLOCK:
        return pb.XuperMessage_GET_BLOCK_RES
    case pb.XuperMessage_GET_BLOCKCHAINSTATUS:
        return pb.XuperMessage_GET_BLOCKCHAINSTATUS_RES
    case pb.XuperMessage_CONFIRM_BLOCKCHAINSTATUS:
        return pb.XuperMessage_CONFIRM_BLOCKCHAINSTATUS_RES
    case pb.XuperMessage_GET_RPC_PORT:
        return pb.XuperMessage_GET_RPC_PORT_RES
    case pb.XuperMessage_GET_AUTHENTICATION:
        return pb.XuperMessage_GET_AUTHENTICATION_RES
    default:
        return pb.XuperMessage_MSG_TYPE_NONE
    }
}
