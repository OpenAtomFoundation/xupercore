use crate::crypto_config;
//use crate::my_error::{ErrorKind,CryptoError,CryptoResult};

#[derive(Clone, Debug, Copy)]
pub struct ECIESParams {
    // block size of symmetric cipher
//    pub block_size: i32,
    pub block_size: usize,
    // length of symmetric key
//    pub key_len: i32,
    pub key_len: usize,
}

/// 返回曲线相关参数
pub fn get_params_for_ecies(cryptography: crypto_config::CryptoType) -> ECIESParams {
    // 根据密码学标记位来判断椭圆曲线，并返回相应的ECIES参数
    // todo：例如NIST-P256，key_len是16；NIST-P384，key_len是32；NIST-P512，key_len是32
    let ecies_params: ECIESParams = match cryptography {
        // NISTP256，也是secp256r1
        crypto_config::CryptoType::NistP256 => ECIESParams{
            block_size: 16,
            key_len: 16,
        },
        // Secp256k1
        crypto_config::CryptoType::Secp256k1 => ECIESParams{
            block_size: 16,
            key_len: 16,
        },
        // GM国密系列
        crypto_config::CryptoType::Gm => ECIESParams{
            block_size: 16,
            key_len: 16,
        },
        // Curve25519
        crypto_config::CryptoType::Curve25519 => ECIESParams{
            block_size: 16,
            key_len: 16,
        },
    };
    
    ecies_params
}