#[allow(dead_code)]
#[derive(Clone, Debug, Copy)]
pub enum CryptoType {
    // 也可以被称为secp256r1曲线
    // 如果用了这个曲线，则签名算法限定ECDSA
    NistP256,
    
    // secp256k1曲线
    // 如果用了这个曲线，则签名算法限定ECDSA
    Secp256k1,
    
    // 国密曲线
    // 如果用了这个曲线，则签名算法限定SM2
    Gm,
    
    // Curve25519椭圆曲线，密码学家Daniel J. Bernstein推出，被设计用于椭圆曲线迪菲-赫尔曼（ECDH）密钥交换方法，可用作提供256比特的安全密钥。
    // 它是不被任何已知专利覆盖的最快ECC曲线之一，且具有较强的安全性。
    // 如果用了这个曲线，则签名算法限定EDDSA，也就是ED25519
    Curve25519,
}
