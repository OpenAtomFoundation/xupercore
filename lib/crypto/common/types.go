package common

import (
	"math/big"
)

// 定义签名中所包含的标记符的值，及其所对应的签名算法的类型
const (
	// ECDSA签名算法
	ECDSA = "ECDSA"
	// Schnorr签名算法，EDDSA的前身
	Schnorr = "Schnorr"
	// Schnorr环签名算法
	SchnorrRing = "SchnorrRing"
	// 多重签名算法
	MultiSig = "MultiSig"
)

// --- 签名数据结构相关 start ---

// XuperSignature 统一的签名结构
type XuperSignature struct {
	SigType    string
	SigContent []byte
}

// ECDSASignature ECDSA签名
type ECDSASignature struct {
	R, S *big.Int
}

// SchnorrSignature Schnorr签名，EDDSA的前身
type SchnorrSignature struct {
	E, S *big.Int
}

// --- Schnorr环签名的数据结构定义 start ---

// PublicKeyFactor 公钥元素
type PublicKeyFactor struct {
	X, Y *big.Int
}

// RingSignature Schnorr环签名
type RingSignature struct {
	//	elliptic.Curve
	CurveName string
	Members   []*PublicKeyFactor
	E         *big.Int
	S         []*big.Int
}

// --- Schnorr环签名的数据结构定义 end ---

// MultiSignature 多重签名
type MultiSignature struct {
	S []byte
	R []byte
}

// MultiSigCommon 多重签名中间公共结果，C是公共公钥，R是公共随机数
type MultiSigCommon struct {
	C []byte
	R []byte
}

// --- 签名数据结构相关 end ---

// 定义创建账户时产生的助记词中的标记符的值，及其所对应的椭圆曲线密码学算法的类型
const (
	// 不同语言标准不一样，也许这里用const直接定义值还是好一些
	_ = iota
	// NIST
	Nist // = 1
	// 国密
	Gm // = 2
	// P-256 + schnorr
	NistSN
)

// 定义创建账户时产生的助记词中的标记符的值，及其所对应的预留标记位的类型
const (
	// 不同语言标准不一样，也许这里用const直接定义值还是好一些
	_ = iota
	// 预留标记位的类型1
	ReservedType1
	// 预留标记位的类型2
	ReservedType2
)

// 定义公私钥中所包含的标记符的值，及其所对应的椭圆曲线密码学算法的类型
const (
	// 美国Federal Information Processing Standards的椭圆曲线
	CurveNist = "P-256"
	// 国密椭圆曲线
	CurveGm = "SM2-P-256"
	// Nist P256 + schnorr
	CurveNistSN = "P-256-SN"
)

// IsValidCryptoType 判断是否支持的加密类型
func IsValidCryptoType(ctype byte) bool {
	valid := true
	switch ctype {
	case Nist:
	case Gm:
	case NistSN:
	default:
		valid = false
	}
	return valid
}