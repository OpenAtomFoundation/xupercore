package sandbox

import (
	"bytes"

	"github.com/xuperchain/xupercore/kernel/contract"
	"github.com/xuperchain/xupercore/kernel/ledger"
)

// peekIterator用来辅助multiIterator更容易实现
type peekIterator struct {
	next  bool
	key   []byte
	value *ledger.VersionedData

	iter ledger.XMIterator
}

func newPeekIterator(iter ledger.XMIterator) *peekIterator {
	p := &peekIterator{
		iter: iter,
	}
	p.fill()
	return p
}

func (p *peekIterator) fill() {
	ok := p.iter.Next()
	if !ok {
		p.next = false
		p.key = nil
		p.value = nil
		return
	}
	p.next = true
	p.key = p.iter.Key()
	p.value = p.iter.Value()
}

func (p *peekIterator) HasNext() bool {
	return p.next
}

func (p *peekIterator) Next() ([]byte, *ledger.VersionedData) {
	if !p.HasNext() {
		return nil, nil
	}
	key := p.key
	value := p.value
	p.fill()
	return key, value
}

// Peek向前查询key, value的值但不移动迭代器的指针
func (p *peekIterator) Peek() ([]byte, *ledger.VersionedData) {
	if !p.HasNext() {
		return nil, nil
	}
	return p.key, p.value
}

func (p *peekIterator) Error() error {
	return p.iter.Error()
}

func (p *peekIterator) Close() {
	p.next = false
	p.key = nil
	p.value = nil
	p.iter.Close()
}

// multiIterator 按照归并排序合并两个XMIterator
// 如果两个XMIterator在某次迭代返回同样的Key，选取front的Value
type multiIterator struct {
	front *peekIterator
	back  *peekIterator

	key   []byte
	value *ledger.VersionedData
}

func newMultiIterator(front, back ledger.XMIterator) ledger.XMIterator {
	m := &multiIterator{
		front: newPeekIterator(front),
		back:  newPeekIterator(back),
	}
	return m
}

func (m *multiIterator) Key() []byte {
	return m.key
}

func (m *multiIterator) Value() *ledger.VersionedData {
	return m.value
}

func (m *multiIterator) Next() bool {
	if !m.front.HasNext() {
		ok := m.back.HasNext()
		m.key, m.value = m.back.Next()
		return ok
	}
	if !m.back.HasNext() {
		ok := m.front.HasNext()
		m.key, m.value = m.front.Next()
		return ok
	}

	k1, _ := m.front.Peek()
	k2, _ := m.back.Peek()
	ret := compareBytes(k1, k2)
	switch ret {
	case 0:
		m.key, m.value = m.front.Next()
		m.back.Next()
	case -1:
		m.key, m.value = m.front.Next()
	case 1:
		m.key, m.value = m.back.Next()
	default:
		panic("unexpected compareBytes return")
	}

	return true
}

func (m *multiIterator) Error() error {
	err := m.front.Error()
	if err != nil {
		return err
	}

	err = m.back.Error()
	if err != nil {
		return err
	}
	return nil
}

// Iterator 必须在使用完毕后关闭
func (m *multiIterator) Close() {
	m.front.Close()
	m.back.Close()
}

// rsetIterator 把迭代到的Key记录到读集里面
type rsetIterator struct {
	bucket string
	mc     *XMCache
	ledger.XMIterator
	err error
}

func newRsetIterator(bucket string, iter ledger.XMIterator, mc *XMCache) ledger.XMIterator {
	return &rsetIterator{
		bucket:     bucket,
		mc:         mc,
		XMIterator: iter,
	}
}

func (r *rsetIterator) Next() bool {
	if r.err != nil {
		return false
	}
	ok := r.XMIterator.Next()
	if !ok {
		return false
	}
	// fill read set
	r.mc.Get(r.bucket, r.XMIterator.Key())
	return true
}

func (r *rsetIterator) Error() error {
	if r.err != nil {
		return r.err
	}
	return r.XMIterator.Error()
}

// ContractIterator 把contract.XMIterator转换成contract.Iterator
type ContractIterator struct {
	ledger.XMIterator
}

func newContractIterator(xmiter ledger.XMIterator) contract.Iterator {
	return &ContractIterator{
		XMIterator: xmiter,
	}
}

func (c *ContractIterator) Value() []byte {
	v := c.XMIterator.Value()
	return v.GetPureData().GetValue()
}

// stripDelIterator 从迭代器里剔除删除标注和空版本
type stripDelIterator struct {
	ledger.XMIterator
}

func newStripDelIterator(xmiter ledger.XMIterator) ledger.XMIterator {
	return &stripDelIterator{
		XMIterator: xmiter,
	}
}

func (s *stripDelIterator) Next() bool {
	for s.XMIterator.Next() {
		v := s.Value()
		if IsDelFlag(v.PureData.Value) {
			continue
		}
		return true
	}
	return false
}

// compareBytes like bytes.Compare but treats nil as max value
func compareBytes(k1, k2 []byte) int {
	if k1 == nil && k2 == nil {
		return 0
	}
	if k1 == nil {
		return 1
	}
	if k2 == nil {
		return -1
	}
	return bytes.Compare(k1, k2)
}
