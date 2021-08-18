package p2pv1

import (
	"errors"
	"io"
	"sync"

	xctx "github.com/xuperchain/xupercore/kernel/common/xcontext"

	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"

	"github.com/xuperchain/xupercore/kernel/network/config"
	nctx "github.com/xuperchain/xupercore/kernel/network/context"
	"github.com/xuperchain/xupercore/kernel/network/p2p"
	"github.com/xuperchain/xupercore/lib/logs"
	pb "github.com/xuperchain/xupercore/protos"
)

type Conn struct {
	log    logs.Logger
	config *config.NetConf

	id   string // addr:"IP:Port"
	conn *grpc.ClientConn
}

// NewConn create new connection with addr
func NewConn(ctx *nctx.NetCtx, addr string) (*Conn, error) {
	c := &Conn{
		id:     addr,
		config: ctx.P2PConf,
		log:    ctx.GetLog(),
	}

	if err := c.newConn(); err != nil {
		ctx.GetLog().Error("NewConn error", "error", err)
		return nil, err
	}

	return c, nil
}

func (c *Conn) newClient() (pb.P2PServiceClient, error) {
	state := c.conn.GetState()
	if state == connectivity.TransientFailure || state == connectivity.Shutdown {
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
		c.log.Error("newGrpcConn error", "error", err, "peerID", c.id)
		return errors.New("new grpc conn error")
	}

	c.conn = conn
	return nil
}

// SendMessage send message to a peer
func (c *Conn) SendMessage(ctx xctx.XContext, msg *pb.XuperMessage) error {
	client, err := c.newClient()
	if err != nil {
		c.log.Error("SendMessage new client error", "log_id", msg.GetHeader().GetLogid(), "error", err, "peerID", c.id)
		return err
	}

	stream, err := client.SendP2PMessage(ctx)
	if err != nil {
		c.log.Error("SendMessage new stream error", "log_id", msg.GetHeader().GetLogid(), "error", err, "peerID", c.id)
		return err
	}
	defer stream.CloseSend()

	c.log.Trace("SendMessage", "log_id", msg.GetHeader().GetLogid(),
		"type", msg.GetHeader().GetType(), "checksum", msg.GetHeader().GetDataCheckSum(), "peerID", c.id)

	msg.Header.From = c.config.Address
	err = stream.Send(msg)
	if err != nil {
		c.log.Error("SendMessage Send error", "log_id", msg.GetHeader().GetLogid(), "error", err, "peerID", c.id)
		return err
	}
	if err == io.EOF {
		return nil
	}

	return err
}

// SendMessageWithResponse send message to a peer with responce
func (c *Conn) SendMessageWithResponse(ctx xctx.XContext, msg *pb.XuperMessage) (*pb.XuperMessage, error) {
	client, err := c.newClient()
	if err != nil {
		c.log.Error("SendMessageWithResponse new client error", "log_id", msg.GetHeader().GetLogid(), "error", err, "peerID", c.id)
		return nil, err
	}

	stream, err := client.SendP2PMessage(ctx)
	if err != nil {
		c.log.Error("SendMessageWithResponse new stream error", "log_id", msg.GetHeader().GetLogid(), "error", err, "peerID", c.id)
		return nil, err
	}
	defer stream.CloseSend()

	c.log.Trace("SendMessageWithResponse", "log_id", msg.GetHeader().GetLogid(),
		"type", msg.GetHeader().GetType(), "checksum", msg.GetHeader().GetDataCheckSum(), "peerID", c.id)

	msg.Header.From = c.config.Address
	err = stream.Send(msg)
	if err != nil {
		c.log.Error("SendMessageWithResponse error", "log_id", msg.GetHeader().GetLogid(), "error", err, "peerID", c.id)
		return nil, err
	}

	resp, err := stream.Recv()
	if err != nil {
		c.log.Error("SendMessageWithResponse Recv error", "log_id", resp.GetHeader().GetLogid(), "error", err.Error())
		return nil, err
	}

	c.log.Trace("SendMessageWithResponse return", "log_id", resp.GetHeader().GetLogid(), "peerID", c.id)
	return resp, nil
}

// Close close this conn
func (c *Conn) Close() {
	c.log.Info("Conn Close", "peerID", c.id)
	c.conn.Close()
}

// GetConnID return conn id
func (c *Conn) PeerID() string {
	return c.id
}

func NewConnPool(ctx *nctx.NetCtx) (*ConnPool, error) {
	p := ConnPool{
		ctx:           ctx,
		staticRouter:  make(map[string]string),
		staticNodeSet: make(map[string]struct{}),
	}
	p.staticModeOn = len(ctx.P2PConf.StaticNodes) > 0 && len(ctx.P2PConf.BootNodes) <= 0
	for _, addresses := range ctx.P2PConf.StaticNodes {
		for _, address := range addresses {
			if _, ok := p.staticNodeSet[address]; ok {
				continue
			}
			p.staticNodeSet[address] = struct{}{}
		}
	}
	return &p, nil
}

// ConnPool manage all the connection
type ConnPool struct {
	ctx  *nctx.NetCtx
	pool sync.Map // map[peerID]*conn

	staticModeOn  bool
	staticNodeSet map[string]struct{} // 标记静态节点
	staticRouter  map[string]string   // map[peerId]peerID
	mutex         sync.Mutex
}

func (p *ConnPool) staticRouterInsert(addr string, mapping string) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.staticRouter[addr] = mapping
}

func (p *ConnPool) getStaticRouter(addr string) string {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	v, ok := p.staticRouter[addr]
	if !ok {
		return ""
	}
	return v
}

func (p *ConnPool) poolGet(addr string, m pb.XuperMessage_MessageType) (*Conn, error) {
	if p.staticModeOn && m != pb.XuperMessage_GET_PEER_INFO {
		addr = func() string {
			if _, ok := p.staticNodeSet[addr]; ok {
				// 该地址在静态列表里，直接转发
				return addr
			}
			v := p.getStaticRouter(addr)
			if v != "" {
				// 该地址在路由表里，转发
				return v
			}
			// 路由表查不到就转向默认出口
			addr = p.getStaticRouter("default")
			return v
		}()
	}
	conn, err := p.Get(addr)
	if err != nil {
		return nil, err
	}
	return conn, nil
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

func (p *ConnPool) GetAll() map[string]string {
	remotePeer := make(map[string]string, 32)
	p.pool.Range(func(key, value interface{}) bool {
		addr := key.(string)
		conn := value.(*Conn)
		if conn.conn.GetState() == connectivity.Ready {
			remotePeer[conn.PeerID()] = addr
		}
		return true
	})

	return remotePeer
}
