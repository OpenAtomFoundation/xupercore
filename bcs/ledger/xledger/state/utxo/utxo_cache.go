package utxo

import (
	"bytes"
	"container/list"
	"errors"
	"github.com/xuperchain/xupercore/kernel/contract"
	"github.com/xuperchain/xupercore/protos"
	"math/big"
	"sync"
)

type CacheItem struct {
	UtxoItem
	ele *list.Element
}

type CacheFiller struct {
	hooks []func()
}

func (cf *CacheFiller) Commit() {
	for _, f := range cf.hooks {
		f()
	}
}

func (cf *CacheFiller) Add(f func()) {
	cf.hooks = append(cf.hooks, f)
}

// UtxoCache is a in-memory cache of UTXO
type UtxoCache struct {
	// <ADDRESS, <UTXO_KEY, UTXO_ITEM>>
	Available map[string]map[string]*CacheItem
	All       map[string]map[string]*CacheItem
	List      *list.List
	Limit     int
	mutex     *sync.Mutex
}

// NewUtxoCache create instance of UtxoCache
func NewUtxoCache(limit int) *UtxoCache {
	return &UtxoCache{
		Available: map[string]map[string]*CacheItem{},
		All:       map[string]map[string]*CacheItem{},
		List:      list.New(),
		Limit:     limit,
		mutex:     &sync.Mutex{},
	}
}

// Insert insert/update utxo cache
func (uv *UtxoCache) Insert(addr string, utxoKey string, item *UtxoItem) {
	uv.mutex.Lock()
	defer uv.mutex.Unlock()
	if _, exist := uv.All[addr]; !exist {
		uv.Available[addr] = map[string]*CacheItem{}
		uv.All[addr] = map[string]*CacheItem{}
	}
	ele := uv.List.PushFront([]string{addr, utxoKey})
	cacheItem := &CacheItem{UtxoItem{Amount: item.Amount, FrozenHeight: item.FrozenHeight}, ele}
	uv.Available[addr][utxoKey] = cacheItem
	uv.All[addr][utxoKey] = cacheItem
	if uv.List.Len() > uv.Limit {
		oldEle := uv.List.Back() //最近最少使用的address
		addressUtxoKey := oldEle.Value.([]string)
		uv.remove(addressUtxoKey[0], addressUtxoKey[1])
	}
}

// Use mark a utxo key as used
func (uv *UtxoCache) Use(address string, utxoKey string) {
	if l2, exist := uv.Available[address]; exist {
		delete(l2, utxoKey)
	}
}

func (uv *UtxoCache) remove(address string, utxoKey string) {
	if l2, exist := uv.All[address]; exist {
		cacheItem, _ := l2[utxoKey]
		if cacheItem != nil {
			uv.List.Remove(cacheItem.ele)
			delete(l2, utxoKey)
		}
		if len(l2) == 0 { //这个address的utxo删完了
			delete(uv.All, address)
			delete(uv.Available, address)
		}
	}
	if l2, exist := uv.Available[address]; exist {
		delete(l2, utxoKey)
	}
}

// Remove delete uxto key from cache
func (uv *UtxoCache) Remove(address string, utxoKey string) {
	uv.mutex.Lock()
	defer uv.mutex.Unlock()
	uv.remove(address, utxoKey)
}

// Lock used to lock cache
func (uv *UtxoCache) Lock() {
	uv.mutex.Lock()
}

// Unlock used to unlock cache
func (uv *UtxoCache) Unlock() {
	uv.mutex.Unlock()
}

type UTXOSandbox struct {
	inputCache  []*protos.TxInput
	outputCache []*protos.TxOutput
	inputIdx    int
	Penetrate   bool
	utxovm      contract.UtxoVM
}

func NewUTXOSandbox(vm contract.UtxoVM, inputs []*protos.TxInput, Penetrate bool) *UTXOSandbox {
	return &UTXOSandbox{
		utxovm:      vm,
		inputCache:  inputs,
		outputCache: []*protos.TxOutput{},
		Penetrate:   Penetrate,
		inputIdx:    0,
	}
}

func (u *UTXOSandbox) selectUtxos(from string, amount *big.Int) (*big.Int, error) {
	if u.Penetrate {
		inputs, _, total, err := u.utxovm.SelectUtxos(from, amount, true, false)
		if err != nil {
			return nil, err
		}
		u.inputCache = append(u.inputCache, inputs...)
		return total, nil
	}

	fromBytes := []byte(from)
	inputCache := u.inputCache[u.inputIdx:]
	sum := new(big.Int)
	n := 0
	for _, input := range inputCache {
		n++
		// Since contract calls bridge serially, a mismatched from address is an error
		if !bytes.Equal(input.GetFromAddr(), fromBytes) {
			return nil, errors.New("from address mismatch in utxo cache")
		}
		sum.Add(sum, new(big.Int).SetBytes(input.GetAmount()))
		if sum.Cmp(amount) >= 0 {
			break
		}
	}
	if sum.Cmp(amount) < 0 {
		return nil, errors.New("utxo not enough in utxo cache")
	}
	u.inputIdx += n
	return sum, nil
}

func (u *UTXOSandbox) Transfer(from, to string, amount *big.Int) error {
	if amount.Cmp(new(big.Int)) == 0 {
		return nil
	}
	total, err := u.selectUtxos(from, amount)
	if err != nil {
		return err
	}
	u.outputCache = append(u.outputCache, &protos.TxOutput{
		Amount: amount.Bytes(),
		ToAddr: []byte(to),
	})
	// make change
	if total.Cmp(amount) > 0 {
		u.outputCache = append(u.outputCache, &protos.TxOutput{
			Amount: new(big.Int).Sub(total, amount).Bytes(),
			ToAddr: []byte(from),
		})
	}
	return nil
}

func (uc *UTXOSandbox) GetUTXORWSets() ([]*protos.TxInput, []*protos.TxOutput) {

	if uc.Penetrate {
		return uc.inputCache, uc.outputCache
	}
	return uc.inputCache[:uc.inputIdx], uc.outputCache
}
