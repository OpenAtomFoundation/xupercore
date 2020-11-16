package p2pv1

import (
	"errors"
	"io"
	"strconv"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"

	"github.com/xuperchain/xupercore/kernel/network/config"
	nctx "github.com/xuperchain/xupercore/kernel/network/context"
	"github.com/xuperchain/xupercore/kernel/network/p2p"
	pb "github.com/xuperchain/xupercore/kernel/network/pb"
	"github.com/xuperchain/xupercore/lib/logs"
)

type Conn struct {
	ctx    nctx.DomainCtx
	log    logs.Logger
	config *config.Config

	id   string // addr:"IP:Port"
	conn *grpc.ClientConn
}

// NewConn create new connection with addr
func NewConn(ctx nctx.DomainCtx, addr string) (*Conn, error) {
	c := &Conn{
		id:     addr,
		config: ctx.GetP2PConf(),
		log:    ctx.GetLog(),
	}

	if err := c.newConn(); err != nil {
		ctx.GetLog().Error("NewConn error", "error", err.Error())
		return nil, err
	}

	return c, nil
}

func (c *Conn) newClient() (pb.P2PServiceClient, error) {
	state := c.conn.GetState()
	if state != connectivity.Ready {
		c.log.Error("newClient conn state not ready", "id", c.id, "state", state.String())
		c.Close()
		err := c.newConn()
		if err != nil {
			c.log.Error("newClient newGrpcConn error", "id", c.id, "error", err)
			return nil, err
		}
	}

	return pb.NewP2PServiceClient(c.conn), nil
}

func (c *Conn) newConn() error {
	conn := &grpc.ClientConn{}
	options := append([]grpc.DialOption{}, grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(int(c.config.MaxMessageSize)<<20)))
	if c.config.IsTls {
		creds, err := p2p.NewTLS(c.config.KeyPath, c.config.ServiceName)
		if err != nil {
			return err
		}
		options = append(options, grpc.WithTransportCredentials(creds))
	} else {
		options = append(options, grpc.WithInsecure())
	}

	conn, err := grpc.Dial(c.id, options...)
	if err != nil {
		c.log.Error("newGrpcConn error", "error", err, "id", c.id)
		return errors.New("New grpcs conn error")
	}

	c.conn = conn
	return nil
}

// SendMessage send message to a peer
func (c *Conn) SendMessage(ctx nctx.OperateCtx, msg *pb.XuperMessage) error {
	client, err := c.newClient()
	if err != nil {
		ctx.GetLog().Error("SendMessage new client error", "error", err.Error(), "id", c.id)
		return err
	}

	stream, err := client.SendP2PMessage(ctx)
	if err != nil {
		ctx.GetLog().Error("SendMessage new stream error", "error", err.Error(), "id", c.id)
		return err
	}
	defer stream.CloseSend()

	ctx.GetLog().Trace("SendMessage", "log_id", msg.GetHeader().GetLogid(),
		"type", msg.GetHeader().GetType(), "checksum", msg.GetHeader().GetDataCheckSum(), "id", c.id)

	msg.Header.From = strconv.Itoa(int(c.config.Port))
	err = stream.Send(msg)
	if err != nil {
		ctx.GetLog().Error("SendMessage Send error", "error", err.Error(), "id", c.id)
		return err
	}
	if err == io.EOF {
		return nil
	}

	return err
}

// SendMessageWithResponse send message to a peer with responce
func (c *Conn) SendMessageWithResponse(ctx nctx.OperateCtx, msg *pb.XuperMessage) (*pb.XuperMessage, error) {
	client, err := c.newClient()
	if err != nil {
		ctx.GetLog().Error("SendMessageWithResponse new client error", "error", err.Error(), "id", c.id)
		return nil, err
	}

	stream, err := client.SendP2PMessage(ctx)
	if err != nil {
		ctx.GetLog().Error("SendMessageWithResponse new stream error", "error", err.Error(), "id", c.id)
		return nil, err
	}
	defer stream.CloseSend()

	ctx.GetLog().Trace("SendMessageWithResponse", "log_id", msg.GetHeader().GetLogid(),
		"type", msg.GetHeader().GetType(), "checksum", msg.GetHeader().GetDataCheckSum(), "id", c.id)

	msg.Header.From = strconv.Itoa(int(c.config.Port))
	err = stream.Send(msg)
	if err != nil {
		ctx.GetLog().Error("SendMessageWithResponse error", "error", err.Error(), "id", c.id)
		return nil, err
	}

	resp, err := stream.Recv()
	if err != nil {
		ctx.GetLog().Error("SendMessageWithResponse Recv error", "error", err.Error())
		return nil, err
	}

	ctx.GetLog().Trace("SendMessageWithResponse return", "log_id", resp.GetHeader().GetLogid(), c.id)
	return resp, nil
}

// Close close this conn
func (c *Conn) Close() {
	c.log.Info("Conn Close", "id", c.id)
	c.conn.Close()
}

// GetConnID return conn id
func (c *Conn) PeerID() string {
	return c.id
}

func NewConnPool(ctx nctx.DomainCtx) (*ConnPool, error) {
	return &ConnPool{
		ctx: ctx,
	}, nil
}

// ConnPool manage all the connection
type ConnPool struct {
	ctx  nctx.DomainCtx
	pool sync.Map // map[peerID]*conn
}

func (p *ConnPool) Get(addr string) (*Conn, error) {
	if v, ok := p.pool.Load(addr); ok {
		return v.(*Conn), nil
	}

	conn, err := NewConn(p.ctx, addr)
	if err != nil {
		return nil, err
	}

	p.pool.LoadOrStore(addr, conn)
	return conn, nil
}
