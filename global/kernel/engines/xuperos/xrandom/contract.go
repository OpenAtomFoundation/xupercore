package xrandom

import (
	"encoding/json"
	"fmt"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/pkg/errors"

	"github.com/OpenAtomFoundation/xupercore/global/kernel/contract"
	"github.com/OpenAtomFoundation/xupercore/global/kernel/contract/sandbox"
	"github.com/OpenAtomFoundation/xupercore/global/lib/cache"
	"github.com/OpenAtomFoundation/xupercore/global/lib/storage/kvdb"
)

type Contract struct {
	RandomNumbers *cache.LRUCache
	ExecNodes     map[string]bool

	contractCtx *Context
	Admins      map[string]bool
}

func NewContract(admins []string, ctx *Context) *Contract {
	c := &Contract{
		RandomNumbers: cache.NewLRUCache(cacheSize),
		contractCtx:   ctx,
		Admins:        map[string]bool{},
	}
	for _, admin := range admins {
		c.Admins[admin] = true
	}
	return c
}

// AddNode adds potential miner node of execution layer
func (c Contract) AddNode(ctx contract.KContext) (*contract.Response, error) {
	if !c.isAdmin(ctx.Initiator()) {
		return nil, errors.New("permission denied")
	}

	// parse param
	addNode, err := parseNodeAddress(ctx.Args())
	if err != nil {
		return nil, err
	}

	nodes, err := c.getNodes(ctx)
	if err != nil {
		return nil, err
	}

	// add new node
	if len(nodes) == 0 {
		nodes = map[string]bool{addNode: true}
	} else if _, exist := nodes[addNode]; exist {
		// exist already
		return &contract.Response{
			Status: statusSuccess,
		}, nil
	} else {
		nodes[addNode] = true
	}

	// update nodes data
	err = c.setNodes(ctx, nodes)
	if err != nil {
		return nil, err
	}
	return &contract.Response{
		Status: statusSuccess,
	}, nil
}

// DeleteNode removes potential miner node of execution layer
func (c Contract) DeleteNode(ctx contract.KContext) (*contract.Response, error) {
	if !c.isAdmin(ctx.Initiator()) {
		return nil, errors.New("permission denied")
	}

	// parse param
	delNode, err := parseNodeAddress(ctx.Args())
	if err != nil {
		return nil, err
	}

	nodes, err := c.getNodes(ctx)
	if err != nil {
		return nil, err
	}

	// delete node
	if len(nodes) == 0 || !nodes[delNode] {
		return nil, errors.New("node not exist")
	}
	delete(nodes, delNode)

	// update nodes data
	err = c.setNodes(ctx, nodes)
	if err != nil {
		return nil, err
	}
	return &contract.Response{
		Status: statusSuccess,
	}, nil
}

func (c Contract) QueryAccessList(ctx contract.KContext) (*contract.Response, error) {
	nodes, err := c.getNodes(ctx)
	if err != nil {
		return nil, err
	}

	// convert nodes to list
	nodeList := make([]string, 0, len(nodes))
	for node := range nodes {
		nodeList = append(nodeList, node)
	}
	data, _ := json.Marshal(nodeList)

	return &contract.Response{
		Status: statusSuccess,
		Body:   data,
	}, nil
}

func (c Contract) SubmitRandomNumber(ctx contract.KContext) (*contract.Response, error) {
	// parse params
	height, err := parseHeight(ctx.Args())
	if err != nil {
		return nil, err
	}

	random, err := parseRandom(ctx.Args())
	if err != nil {
		return nil, err
	}

	log.Debug("SubmitRandomNumber parse random done",
		"random", random)
	// check height
	maxHeight, err := c.maxHeight(ctx)
	if err != nil {
		log.Debug("SubmitRandomNumber parse random failed",
			"error", err)
		return nil, err
	}
	if height != maxHeight+1 {
		return nil, errors.New("height is not valid")
	}
	log.Debug("SubmitRandomNumber parse height done",
		"height", height)

	// update random number
	err = c.updateRandom(ctx, height, random)
	log.Debug("SubmitRandomNumber",
		"height", height,
		"random", random,
		"error", err)
	if err != nil {
		return nil, err
	}
	return &contract.Response{
		Status: statusSuccess,
	}, nil
}

func (c Contract) QueryRandomNumber(ctx contract.KContext) (*contract.Response, error) {
	// parse params
	queryNode, err := parseNodePublicKey(ctx.Args())
	if err != nil {
		return nil, err
	}
	height, err := parseHeightWithSign(ctx.Args())
	if err != nil {
		return nil, err
	}

	// check permission
	permitted, err := c.checkPermission(ctx, queryNode, height)
	if err != nil {
		return nil, err
	}
	if !permitted {
		return nil, fmt.Errorf("permission denied")
	}

	// get random
	random, err := c.getRandom(ctx, height)
	if err != nil {
		return nil, err
	}
	return &contract.Response{
		Status: statusSuccess,
		Body:   random.toJSON(),
	}, nil
}

func (c Contract) isAdmin(initiator string) bool {
	return c.Admins[initiator]
}

func (c Contract) getNodes(ctx contract.KContext) (map[string]bool, error) {
	if len(c.ExecNodes) > 0 {
		return c.ExecNodes, nil
	}

	// get from ctx
	value, err := ctx.Get(ContractName, keyNodes)
	if err != nil && !kvdb.ErrNotFound(err) && !errors.Is(err, sandbox.ErrHasDel) {
		return nil, errors.Wrap(err, "get nodes failed")
	}
	if len(value) == 0 {
		return nil, nil
	}
	nodes := make(map[string]bool)
	if err := json.Unmarshal(value, &nodes); err != nil {
		return nil, err
	}
	c.ExecNodes = nodes

	return c.ExecNodes, nil
}

func (c Contract) setNodes(ctx contract.KContext, nodes map[string]bool) error {
	// save nodes data
	value, err := json.Marshal(nodes)
	if err != nil {
		return err
	}
	err = ctx.Put(ContractName, keyNodes, value)
	if err != nil {
		return errors.Wrap(err, "set nodes failed")
	}

	c.ExecNodes = nodes
	return nil
}

func (c Contract) maxHeight(ctx contract.KContext) (uint64, error) {
	value, err := ctx.Get(ContractName, keyMaxHeight)
	if err != nil && !kvdb.ErrNotFound(err) && !errors.Is(err, sandbox.ErrHasDel) {
		return 0, errors.Wrap(err, "get max height failed")
	}
	if len(value) == 0 {
		return 0, nil
	}
	var height uint64
	err = json.Unmarshal(value, &height)
	return height, errors.Wrap(err, "parse max height failed")
}

func (c Contract) updateRandom(ctx contract.KContext, height uint64, random *Random) error {
	// save random data
	kRandom := bucketKeyRandom(height)
	randomData, err := json.Marshal(random)
	if err != nil {
		log.Error("marshall random failed", "error", err)
		return err
	}
	err = ctx.Put(ContractName, kRandom, randomData)
	if err != nil {
		log.Error("save random failed", "error", err)
		return err
	}
	_ = c.RandomNumbers.Add(height, random)

	// save max height data
	heightData, _ := json.Marshal(height)
	err = ctx.Put(ContractName, keyMaxHeight, heightData)
	if err != nil {
		log.Error("save height failed", "error", err)
		return err
	}
	return nil
}

func (c Contract) getRandom(ctx contract.KContext, height uint64) (*Random, error) {
	if random, exist := c.RandomNumbers.Get(height); exist {
		return random.(*Random), nil
	}

	// get from ctx
	kRandom := bucketKeyRandom(height)
	value, err := ctx.Get(ContractName, kRandom)
	if err != nil && !kvdb.ErrNotFound(err) && !errors.Is(err, sandbox.ErrHasDel) {
		return nil, errors.Wrap(err, "get random failed")
	}
	if len(value) == 0 {
		return nil, errors.Wrap(err, "get random empty")
	}
	random := new(Random)
	if err := json.Unmarshal(value, &random); err != nil {
		return nil, err
	}
	_ = c.RandomNumbers.Add(height, random)

	return random, nil
}

func (c Contract) checkPermission(ctx contract.KContext, node string, height uint64) (bool, error) {

	// parse node
	address, err := publicKeyToAddress(node)
	if err != nil {
		return false, errors.Wrap(err, "derive public key to address failed")
	}

	// check node
	nodes, err := c.getNodes(ctx)
	if err != nil {
		return false, errors.Wrap(err, "get access list failed")
	}
	if !nodes[address] {
		return false, nil
	}

	// check height
	maxHeight, err := c.maxHeight(ctx)
	if err != nil {
		return false, errors.Wrap(err, "get max height failed")
	}
	maxQueryableHeight := maxHeight - freezeSize
	return height <= maxQueryableHeight, nil
}

// publicKeyToAddress 将压缩的公钥转换为地址
func publicKeyToAddress(compressed string) (string, error) {
	bytes, err := hexutil.Decode(compressed)
	if err != nil {
		return "", errors.Wrap(err, "decode failed")
	}
	publicKey, err := crypto.DecompressPubkey(bytes)
	if err != nil {
		return "", errors.Wrap(err, "decompress public key failed")
	}
	return crypto.PubkeyToAddress(*publicKey).String(), nil
}
