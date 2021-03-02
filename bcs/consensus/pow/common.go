package pow

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
)

// PoWConfig pow需要解析的创始块解析格式
//  根据Bitcoin推荐
//    AdjustHeightGap: 2016,
//	  MaxTarget: 0x1d00FFFF,
//    DefaultTarget: 0x207FFFFF
type PoWConfig struct {
	DefaultTarget        uint32 `json:"defaultTarget"`
	AdjustHeightGap      int32  `json:"adjustHeightGap"`
	ExpectedPeriodMilSec int32  `json:"expectedPeriod"`
	MaxTarget            uint32 `json:"maxTarget"`
}

// 目前未定义pb结构
// PoWStorage pow占用block中consensusStorage json串的格式
type PoWStorage struct {
	// TargetBits 以一个uint32类型解析，16进制格式为0xFFFFFFFF
	// 真正difficulty按照Bitcoin标准转化，将TargetBits转换为一个uint256 bits的大数
	TargetBits uint32 `json:"targetBits,omitempty"`
}

// GetCompact 将一个256bits的大数转换为一个target
func GetCompact(number *big.Int) (uint32, bool) {
	nSize := (number.BitLen() + 7) / 8
	nCompact := uint32(0)
	low64Int := new(big.Int)
	low64Int.SetUint64(0xFFFFFFFFFFFFFFFF)
	low64Int.And(low64Int, number)
	low64 := low64Int.Uint64()
	if nSize <= 3 {
		nCompact = uint32(low64 << uint64(8*(3-nSize)))
	} else {
		bn := new(big.Int)
		bn.Rsh(number, uint(8*(nSize-3)))
		low64Int.SetUint64(0xFFFFFFFFFFFFFFFF)
		low64Int.And(low64Int, bn)
		low64 := low64Int.Uint64()
		nCompact = uint32(low64)
	}
	// The 0x00800000 bit denotes the sign.
	// Thus, if it is already set, divide the mantissa by 256 and increase the exponent.
	if nCompact&0x00800000 > 0 {
		nCompact >>= 8
		nSize++
	}
	if (nCompact&0xFF800000) != 0 || nSize > 256 {
		return 0, false
	}
	nCompact |= uint32(nSize) << 24
	return nCompact, true
}

// SetCompact 将一个uint32的target转换为一个difficulty
func SetCompact(nCompact uint32) (*big.Int, bool, bool) {
	nSize := nCompact >> 24
	nWord := new(big.Int)
	u := new(big.Int)
	nCompactInt := big.NewInt(int64(nCompact))
	// 0x00800000是一个符号位，故nWord仅为后23位
	lowBits := big.NewInt(0x007fffff)
	nWord.And(nCompactInt, lowBits)
	if nSize <= 3 {
		nWord.Rsh(nWord, uint(8*(3-nSize)))
		u = nWord
	} else {
		u = nWord
		u.Lsh(u, uint(8*(nSize-3)))
	}
	pfNegative := nWord.Cmp(big.NewInt(0)) != 0 && (nCompact&0x00800000) != 0
	pfOverflow := nWord.Cmp(big.NewInt(0)) != 0 && ((nSize > 34) ||
		(nWord.Cmp(big.NewInt(0xff)) == 1 && nSize > 33) ||
		(nWord.Cmp(big.NewInt(0xffff)) == 1 && nSize > 32))
	return u, pfNegative, pfOverflow
}

func unmarshalPowConfig(input []byte) (*PoWConfig, error) {
	// 由于创世块中的配置全部使用的string，内部使用时做下转换
	// 转换配置结构到内部结构
	// 先转为interface{}
	consCfg := make(map[string]interface{})
	err := json.Unmarshal(input, &consCfg)
	if err != nil {
		return nil, err
	}
	powCfg := &PoWConfig{}
	int32Map := map[string]int32{
		"adjustHeightGap": 0,
		"expectedPeriod":  0,
	}
	uint32Map := map[string]uint32{
		"defaultTarget": 0,
		"maxTarget":     0,
	}
	for k, _ := range int32Map {
		value, err := strconv.ParseInt(consCfg[k].(string), 10, 32)
		if err != nil {
			return nil, fmt.Errorf("marshal consensus config failed key %s set error", k)
		}
		int32Map[k] = int32(value)
	}
	for k, _ := range uint32Map {
		value, err := strconv.ParseInt(consCfg[k].(string), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("marshal consensus config failed key %s set error", k)
		}
		uint32Map[k] = uint32(value)
	}
	powCfg.DefaultTarget = uint32Map["defaultTarget"]
	powCfg.MaxTarget = uint32Map["maxTarget"]
	powCfg.AdjustHeightGap = int32Map["adjustHeightGap"]
	powCfg.ExpectedPeriodMilSec = int32Map["expectedPeriod"]
	return powCfg, nil
}
