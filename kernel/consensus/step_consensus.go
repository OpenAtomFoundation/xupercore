package consensus

import "sync"

// stepConsensus 封装了可插拔共识需要的共识数组
type stepConsensus struct {
	cons []ConsensusImplInterface
	// 共识升级切换开关
	switchConsensus bool
	// mutex保护StepConsensus数据结构cons的读写操作
	mutex sync.Mutex
}

// 获取共识切换开关
func (sc *stepConsensus) getSwitch() bool {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()
	return sc.switchConsensus
}

// 设置共识切换开关
func (sc *stepConsensus) setSwitch(s bool) {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()
	sc.switchConsensus = s
}

// 向可插拔共识数组put item
func (sc *stepConsensus) put(con ConsensusImplInterface) error {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()
	sc.cons = append(sc.cons, con)
	return nil
}

// 获取最新的共识实例
func (sc *stepConsensus) tail() ConsensusImplInterface {
	//getCurrentConsensusComponent
	sc.mutex.Lock()
	defer sc.mutex.Unlock()
	if len(sc.cons) == 0 {
		return nil
	}
	return sc.cons[len(sc.cons)-1]
}

// 获取倒数第二个共识实例
func (sc *stepConsensus) preTail() ConsensusImplInterface {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()
	if len(sc.cons) < 2 {
		return nil
	}
	return sc.cons[len(sc.cons)-2]
}

// 获取共识实例长度
func (sc *stepConsensus) len() int {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()
	return len(sc.cons)
}
