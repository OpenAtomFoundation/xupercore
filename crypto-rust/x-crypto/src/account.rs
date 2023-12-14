use crate::my_error::{ErrorKind,CryptoError,CryptoResult};
use crate::wordlist;
use crate::crypto_config;
use crate::seed;
use crate::utils;
use crate::mnemonic;

//use std::mem;

use num_bigint::{BigInt, Sign};

use p256;
use p256::elliptic_curve::bigint::Encoding;
use p256::elliptic_curve::sec1::ToEncodedPoint;
use p256::elliptic_curve::zeroize::Zeroizing;

use p256::ecdsa;
use p256::ecdsa::{SigningKey, Signature, VerifyingKey};
use p256::ecdsa::signature::{Signer, Verifier};

//use ecdsa::{SigningKey, VerifyingKey};
//use ecdsa::signature::{Signer, Verifier};

use crypto::ripemd160::Ripemd160;
use crypto::sha2::Sha256;
use crypto::digest::Digest;

use base58::ToBase58;

use std::str::FromStr;

//use ecdsa;
//use ecdsa::Signature;

#[derive(Clone, Debug)]
pub struct Account {
    // 随机熵
    pub entropy: Vec<u8>,
    // 助记词
    pub mnemonic: String,
    pub address: String,
//    pub json_private_key: String,
//    pub json_public_key: String,
    // 私钥，EM-encoded SEC1格式的私钥，开头标志：-----BEGIN EC PRIVATE KEY-----
    pub private_key: String,
    // 公钥，EM-encoded SEC1格式的公钥，开头标志：-----BEGIN EC PUBLIC KEY-----
    pub public_key: String,
//    // 椭圆曲线类型
//    pub elliptic_curve: EllipticCurve,
}

#[derive(Debug)]
pub struct NistP256Account {
    // 私钥，EM-encoded SEC1格式的私钥，开头标志：-----BEGIN EC PRIVATE KEY-----
    pub private_key: String,
    // 公钥，EM-encoded SEC1格式的公钥，开头标志：-----BEGIN EC PUBLIC KEY-----
    pub public_key: String,
    
    public_key_sec1_encoded_bytes: Vec<u8>,
}

#[derive(Clone, Debug, Copy)]
pub enum MnemonicStrength {
    // 助记词安全强度弱 12个助记词
    StrengthEasy,
    // 助记词安全强度中 18个助记词
    StrengthMiddle,
    // 助记词安全强度高 24个助记词
    StrengthHard,
}

//#[derive(Clone, Debug, Copy)]
//pub enum EllipticCurve {
//    // National Institute of Standards and Technology 美国国家标准与技术研究院发布的P256曲线。
//    // 爱德华·斯诺登所泄漏的棱镜计划的内部摘要暗示，NSA在双椭圆曲线确定性随机比特生成器标准中加入后门可能存在逻辑陷阱，通过伪随机数可以直接计算出私钥。
//    // 因此密码学家在致力于推出更多的设计更透明的椭圆曲线。
//    NistP256,
//    // Curve25519椭圆曲线，密码学家Daniel J. Bernstein推出，被设计用于椭圆曲线迪菲-赫尔曼（ECDH）密钥交换方法，可用作提供256比特的安全密钥。
//    // 它是不被任何已知专利覆盖的最快ECC曲线之一，且具有较强的安全性。
//    Curve25519,
//}

/// 创建带有助记词的新的区块链账户
/// todo：国密系列尚未完成支持，ED25519集成中
pub fn create_new_account_with_mnemonic(language_type: wordlist::LanguageType, strength: MnemonicStrength, cryptography: crypto_config::CryptoType) -> CryptoResult<Account> {
    // 根据强度来判断随机数长度
    // 预留出8个bit用来指定当使用助记词时来恢复私钥时所需要的密码学算法组合
    let entropy_bit_length = match strength {
        // 弱 12个助记词
        MnemonicStrength::StrengthEasy => 120,
        // 中 18个助记词
        MnemonicStrength::StrengthMiddle => 184,
        // 高 24个助记词
        MnemonicStrength::StrengthHard => 248,
    };
    
    // 产生随机熵
    let entropy_byte_result = seed::generate_entropy(entropy_bit_length);
    
    // 判断是否成功的产生了符合长度要求的随机熵
    let mut entropy_byte = match entropy_byte_result {
        Ok(bytes) => bytes,
        Err(error) => return Err(error),
    };
    
    // 设置密码学标记位
    let cryptography_tag: usize = match cryptography {
        // NIST
        crypto_config::CryptoType::NistP256 => 1,
        // Secp256k1
        crypto_config::CryptoType::Secp256k1 => 2,
        // GM国密系列
        crypto_config::CryptoType::Gm => 3,
        // Curve25519
        crypto_config::CryptoType::Curve25519 => 4,
    };
    
//    let cryptography_bit: u8 = cryptography_tag.to_be();
    
    // 把带有密码学标记位的byte数组转化为一个bigint，方便后续做比特位运算（主要是移位操作）
    let tag_bigint = BigInt::from(cryptography_tag);
    
    // 1111 - 4个1，当一个大的bigint和它进行“And”比特运算的时候，就会获得大的bigint最右边4位的比特位
    let last_4_bits_mask = BigInt::from(15);

    // 10000 - 乘以这个带有4个0的数等于左移4个比特位，除以这个带有4个0的数等于右移4个比特位，
    let left_shift_4_bits_multiplier = BigInt::from(16);
    
    // 综合标记位获取密码学标记位最右边的4个比特
    let tag_bigint = tag_bigint & &last_4_bits_mask;
    
    // 将综合标记位左移4个比特
    let tag_bigint = tag_bigint * left_shift_4_bits_multiplier;
    
    // 定义预留标记位，暂时是0，后面可以改
    let reserved_tag_bigint = BigInt::from(0);
    
    // 综合标记位获取预留标记位最右边的4个比特
    let reserved_tag_bigint = reserved_tag_bigint & last_4_bits_mask;
    
    // 合并密码学标记位和预留标记位
    let tag_bigint = tag_bigint | reserved_tag_bigint;
    
    // 把密码学标记位的比特补齐为1个字节
    let (_, tag_bytes) = tag_bigint.to_bytes_be();
    let mut tag_byte = utils::bytes::bytes_pad(tag_bytes, 1);
    
    entropy_byte.append(&mut tag_byte);
    
//    println!("entropy_byte is: {:?}", entropy_byte);
//    println!("entropy_byte length is: {}", entropy_byte.len());
    
    // 将随机熵转为指定语言的助记词
    let mnemonic_sentence =  mnemonic::generate_mnemonic_sentence_from_entropy(entropy_byte.clone(), language_type).unwrap();
    
    // 生产私钥所需的seed
    // 将助记词转为伪随机数种子
    let password = String::from("jingbo is handsome!");
    let e_seed = seed::generate_seed_from_mnemonic(mnemonic_sentence.clone(), password);
    
    // 创建带有NistP256曲线的区块链公私钥账户
    let nistp256_account = create_new_account_nistp256(e_seed);
    let str_private_key = nistp256_account.private_key;
    let str_public_key = nistp256_account.public_key;
    let public_key_bytes = nistp256_account.public_key_sec1_encoded_bytes;
    
    // 补齐address相关的代码
    let str_address = get_address_from_public_key(public_key_bytes, cryptography);

    let account = Account {
        entropy:  entropy_byte,
        mnemonic: mnemonic_sentence,
        address: str_address,
        private_key: str_private_key,
        public_key: str_public_key,
    };
    
//    println!("account:{:?}", account);
    
    Ok(account)
}

/// 创建带有NistP256曲线的区块链公私钥账户
pub fn create_new_account_nistp256(e_seed: [u8; 40]) -> NistP256Account {
    // 通过随机数种子来生成椭圆曲线加密的私钥
    let seed = BigInt::from_bytes_be(Sign::Plus, &e_seed);
    let n = p256::U256::from_be_hex("ffffffff00000000ffffffffffffffffbce6faada7179e84f3b9cac2fc632551");
    let n_bytes = n.to_be_bytes();
    let order = BigInt::from_bytes_be(Sign::Plus, &n_bytes) - 1;
    
    let remainder = seed % order;
    let remainder: BigInt = remainder + 1;
    
    let (_, r_seed) = remainder.to_bytes_be();
    
//    println!("r_seed:{:?} and size:{}", r_seed, r_seed.len());
    
//    let r_seed = utils::bytes::bytes_pad(r_seed, mem::size_of::<usize>() * 8);
    let r_seed = utils::bytes::bytes_pad(r_seed, 32);
    
//    println!("r_seed:{:?} and size:{}", r_seed, r_seed.len());

    // 通过随机数种子来生成椭圆曲线加密的私钥
    let private_key = p256::SecretKey::from_be_bytes(&r_seed).unwrap();
    let str_private_key: Zeroizing<String> = private_key.to_pem(Default::default()).unwrap();    
    let str_private_key: &str = str_private_key.as_ref();
    let str_private_key = str_private_key.to_string();
    
    let public_key = private_key.public_key();
    let str_public_key = public_key.to_string();
    
    // 把公钥编码为NIST SEC1标准的一个椭圆曲线上的点
    // false表示不采用压缩模式
    let encoded_point = public_key.to_encoded_point(false);
    
    let encoded_point_bytes = encoded_point.as_bytes().to_vec();
    
    let account = NistP256Account {
        private_key: str_private_key,
        public_key: str_public_key,
        public_key_sec1_encoded_bytes: encoded_point_bytes,
    };
    
    account
}

/// 根据助记词恢复出区块链账户
pub fn retrieve_account_by_mnemonic(mnemonic_sentence: String, language_type: wordlist::LanguageType) -> CryptoResult<Account> {
    // 从助记词中提取密码学算法
    let cryptography_result = get_crypto_byte_from_mnemonic(mnemonic_sentence.clone(), language_type);
    
    let cryptography_tag = match cryptography_result {
        Ok(cryptography) => cryptography,
        Err(error) => return Err(error),
    };

    // 生产私钥所需的seed
    // 将助记词转为伪随机数种子
    let password = String::from("jingbo is handsome!");
    let e_seed = seed::generate_seed_from_mnemonic(mnemonic_sentence.clone(), password);
    
//    println!("e_seed:{:?} and size:{}", e_seed, e_seed.len());

    // 创建带有NistP256曲线的区块链公私钥账户
    let nistp256_account = create_new_account_nistp256(e_seed);
    let str_private_key = nistp256_account.private_key;
    let str_public_key = nistp256_account.public_key;
    let public_key_bytes = nistp256_account.public_key_sec1_encoded_bytes;
    
    // 补齐address相关的代码
    let str_address = get_address_from_public_key(public_key_bytes, cryptography_tag);
    
//    // 通过随机数种子来生成椭圆曲线加密的私钥
//    let seed = BigInt::from_bytes_be(Sign::Plus, &e_seed);
//    let n = p256::U256::from_be_hex("ffffffff00000000ffffffffffffffffbce6faada7179e84f3b9cac2fc632551");
//    
//    let n_bytes = n.to_be_bytes();
//    let order = BigInt::from_bytes_be(Sign::Plus, &n_bytes) - 1;
//    
////    println!("order:{:?}", order); 
//    
//    let remainder = seed % order;
//    let remainder: BigInt = remainder + 1;
//
////    println!("remainder:{:?}", remainder); 
//    
//    let (_, r_seed) = remainder.to_bytes_be();
//    
////    println!("r_seed:{:?} and size:{}", r_seed, r_seed.len());
//    
////    let r_seed = utils::bytes::bytes_pad(r_seed, mem::size_of::<usize>() * 8);
//    let r_seed = utils::bytes::bytes_pad(r_seed, 32);
//    
////    println!("r_seed:{:?} and size:{}", r_seed, r_seed.len());
//
//    // 通过随机数种子来生成椭圆曲线加密的私钥
//    let private_key = p256::SecretKey::from_be_bytes(&r_seed).unwrap();
//    let str_private_key: Zeroizing<String> = private_key.to_pem(Default::default()).unwrap();    
//    let str_private_key: &str = str_private_key.as_ref();
//    let str_private_key = str_private_key.to_string();
//    
//    let public_key = private_key.public_key();
//    let str_public_key = public_key.to_string();    
//    
//    // 补齐address相关的代码
//    let str_address = get_address_from_public_key(public_key, cryptography_tag);
    
//    println!("get_address_from_public_key -- str_address:{:?}", str_address);
    
    // 上文检查过助记词有效性了，这里不可能抛出错误
    let entropy_byte = mnemonic::get_entropy_from_mnemonic_sentence(mnemonic_sentence.clone(), language_type).unwrap();

//    println!("get_entropy_from_mnemonic_sentence -- entropy_byte:{:?}", entropy_byte);

    let account = Account {
        entropy:  entropy_byte,
        mnemonic: mnemonic_sentence,
        address: str_address,
        private_key: str_private_key,
        public_key: str_public_key,
    };
    
//    println!("account:{:?}", account);
    
    Ok(account)
}

/// 从助记词中提取出密码学标记位
pub fn get_crypto_byte_from_mnemonic(mnemonic: String, language_type: wordlist::LanguageType) -> CryptoResult<crypto_config::CryptoType> {
    let entrophy_result = mnemonic::get_entropy_from_mnemonic_sentence(mnemonic, language_type);
    
//    println!("get_crypto_byte_from_mnemonic -- get_entropy_from_mnemonic_sentence -- entrophy_result:{:?}", entrophy_result);
    
    // 判断是否成功的恢复出了随机熵
    let entropy_vec = match entrophy_result {
        Ok(entropy) => entropy,
        Err(error) => return Err(error),
    };
    
    // 从熵中提取综合标记位
    let tag_byte = entropy_vec[entropy_vec.len() - 1];
    
    let tag_bytes = tag_byte.to_be_bytes();
    
    // 将一个 big endian formatted u8 vec 转换为一个 big int
    let mut tag_bigint =  BigInt::from_bytes_be(Sign::Plus, &tag_bytes);
    
    // 10000 - 乘以这个带有4个0的数等于左移4个比特位，除以这个带有4个0的数等于右移4个比特位，
    let shift4_bits_factor = BigInt::from(16);
    
    // 将熵右移4个比特
    tag_bigint = tag_bigint / shift4_bits_factor;
    
    // 1111 - 4个1，当一个大的bigint和它进行“And”比特运算的时候，就会获得大的bigint最右边4位的比特位
    let last4_bits_mask = BigInt::from(15);
    
    // 从综合标记位获取密码学标记位最右边的4个比特
    let cryptography_bigint = tag_bigint & &last4_bits_mask;
    
    let (_, cryptography_bytes) = cryptography_bigint.to_bytes_be();
    
    // vec转定长array
    // If the length doesn’t match, the input comes back in Err, and the program will panic,
    // but it won't happen!
    let word_bytes_array: [u8; 1] = cryptography_bytes.try_into().unwrap();
    
    // 设置密码学标记位
    let cryptography_tag = match word_bytes_array[0] {
        // NIST
        1 => crypto_config::CryptoType::NistP256,
        // GM国密系列
        2 => crypto_config::CryptoType::Gm,
        // Curve25519
        3 => crypto_config::CryptoType::Curve25519,
        // 错误的标记位
        _ => return Err(
            CryptoError {
                kind: ErrorKind::MnemonicWordInvalid,
                message: ErrorKind::MnemonicWordInvalid.to_string()
            }
        ),
    };
    
    Ok(cryptography_tag)
}

/// 将公钥转换为地址
//pub fn get_address_from_public_key(public_key: p256::PublicKey, cryptography: crypto_config::CryptoType) -> String {
pub fn get_address_from_public_key(public_key_sec1_encoded_bytes: Vec<u8>, cryptography: crypto_config::CryptoType) -> String {
    let encoded_point_bytes = public_key_sec1_encoded_bytes;
    
//    println!("encoded_point_bytes:{:?}", encoded_point_bytes);
    
    let mut sha256 = Sha256::new();
    
    // SHA256 once
    sha256.input(&encoded_point_bytes);
    let mut hash_byte_sha256: [u8; 32] = [0; 32];
    sha256.result(&mut hash_byte_sha256);
    
    sha256.reset();
    
    let mut ripemd160 = Ripemd160::new();
    
    ripemd160.input(&mut hash_byte_sha256);
    
    // initialize an array of 20*8 = 160 length
    let mut hash_byte: [u8; 20] = [0; 20];
    
    ripemd160.result(&mut hash_byte);
    
//    println!("hash_byte after ripemd160:{:?}", hash_byte);
    
    // 设置密码学标记位
    let cryptography_tag: u8 = match cryptography {
        // NIST
        crypto_config::CryptoType::NistP256 => 1,
        // Secp256k1
        crypto_config::CryptoType::Secp256k1 => 2,
        // GM国密系列
        crypto_config::CryptoType::Gm => 3,
        // Curve25519
        crypto_config::CryptoType::Curve25519 => 4,
    };
    
    // Return the memory representation of this integer as a byte array in big-endian (network) byte order.
    let n_version_byte: [u8; 1] = cryptography_tag.to_be_bytes();
    
    let mut version_byte_vec = n_version_byte.to_vec();
    let mut hash_byte_vec = hash_byte.to_vec();
    
    version_byte_vec.append(&mut hash_byte_vec);
    
    let mut original_vec = version_byte_vec.clone();
    
    // 加入校验位
    // using double SHA256 for future risks
    let mut sha256 = Sha256::new();
    
    // SHA256 once
    sha256.input(&version_byte_vec);
    let mut hash_byte: [u8; 32] = [0; 32];
    sha256.result(&mut hash_byte);
    
    sha256.reset();
    
    // SHA256 twice -- double SHA256
    sha256.input(&hash_byte);
    let mut hash_byte: [u8; 32] = [0; 32];
    sha256.result(&mut hash_byte);
    
    // 拿到校验位
    let simple_check_code: [u8; 4] = hash_byte[0..4].try_into().unwrap();
    let mut simple_check_code_vec = simple_check_code.to_vec();
    
//    let mut hash_byte_vec = hash_byte.to_vec();
    
    original_vec.append(&mut simple_check_code_vec);
    
    let str_enc = &original_vec.to_base58();
    
    str_enc.to_string()
}

/// 将使用PEM-encoded SEC1格式编码的私钥字符串转化为ECC私钥
/// Parse SecretKey from PEM-encoded SEC1 ECC PrivateKey format.
pub fn get_ecdsa_private_key_from_pem_encoded_sec1_str(key_str: String) -> CryptoResult<p256::SecretKey> {
    let private_key_parse_result = p256::SecretKey::from_sec1_pem(&key_str);
    
    // 判断是否成功的恢复出了ECC私钥
    let private_key = match private_key_parse_result {
        Ok(private_key_parse_result) => private_key_parse_result,
        Err(_) => return Err(
            CryptoError {
                kind: ErrorKind::ErrInvalidStringFormat,
                message: ErrorKind::ErrInvalidStringFormat.to_string(),
            }
        ),
    };
    
    Ok(private_key)

}

/// 将使用PEM-encoded SEC1格式编码的公钥字符串转化为ECC私钥
/// Parse PublicKey from PEM-encoded SEC1 ECC PublicKey format.
pub fn get_ecdsa_public_key_from_pem_encoded_sec1_str(key_str: String) -> CryptoResult<p256::PublicKey> {
    // 引入std::str::FromStr，并开启crate feature "pem"
    let public_key_parse_result = p256::PublicKey::from_str(&key_str);
    
    // 判断是否成功的恢复出了ECC私钥
    let public_key = match public_key_parse_result {
        Ok(public_key_parse_result) => public_key_parse_result,
        Err(_) => return Err(
            CryptoError {
                kind: ErrorKind::ErrInvalidStringFormat,
                message: ErrorKind::ErrInvalidStringFormat.to_string(),
            }
        ),
    };
    
    Ok(public_key)

}

///// 将使用PEM-encoded SEC1格式编码的私钥字符串转化为ECC私钥
///// Parse SecretKey from PEM-encoded SEC1 ECC Signing Key format.
//pub fn get_ecdsa_signing_key_from_pem_encoded_sec1_str(key_str: String) -> CryptoResult<ecdsa::SigningKey> {
//    let private_key_parse_result = p256::SecretKey::from_sec1_pem(&key_str);
//    
//    // 判断是否成功的恢复出了ECC私钥
//    let private_key = match private_key_parse_result {
//        Ok(private_key_parse_result) => private_key_parse_result,
//        Err(_) => return Err(
//            CryptoError {
//                kind: ErrorKind::ErrInvalidStringFormat,
//                message: ErrorKind::ErrInvalidStringFormat.to_string(),
//            }
//        ),
//    };
//    
//    Ok(private_key)
//
//}

/// ECDSA Sign
//pub fn ecdsa_sign(secret_key: String, data: String) -> CryptoResult<ecdsa::Signature<p256::NistP256>> {
    pub fn ecdsa_sign(secret_key: String, data: String) -> CryptoResult<String> {
    let private_key_result = get_ecdsa_private_key_from_pem_encoded_sec1_str(secret_key);
    
    // 判断是否成功的恢复出了ECC私钥
    let private_key = match private_key_result {
        Ok(private_key) => private_key,
        Err(error) => return Err(error),
    };
    
    // convert ecc SecretKey to ecdsa SigningKey
    let signing_key = SigningKey::from(private_key);
    
//    let sig = SigningKey::sign(data);
    let sig: Signature = signing_key.sign(data.as_bytes());
    
//    let sig_bytes = sig.as_bytes();

    let sig_str = sig.to_string();
    
    Ok(sig_str)
}

/// ECDSA Sign
pub fn ecdsa_verify(public_key: String, data: String, signature: String) -> CryptoResult<bool> {
    let public_key_result = get_ecdsa_public_key_from_pem_encoded_sec1_str(public_key);
    
    // 判断是否成功的恢复出了ECC私钥
    let public_key = match public_key_result {
        Ok(public_key) => public_key,
        Err(error) => return Err(error),
    };
    
    // convert ecc publicKey to ecdsa VerifyingKey
    let verifying_key = VerifyingKey::from(public_key);
    
    let sig = ecdsa::Signature::from_str(&signature).unwrap();
    
    let verify_result = verifying_key.verify(data.as_bytes(), &sig);
    
    let result = match verify_result {
        Ok(_) => true,
        Err(_) => return Err(
            CryptoError {
                kind: ErrorKind::ErrInvalidEcdsaSig,
                message: ErrorKind::ErrInvalidEcdsaSig.to_string(),
            }
        ),
    };
    
    Ok(result)
}

///// ECDSA Sign
//pub fn ecdsa_sign(secret_key: p256::SecretKey, data: String) -> CryptoResult<p256::PublicKey> {
//    let private_key = 
//}
//
///// ECDSA Verify
//pub fn ecdsa_verify(public_key: p256::PublicKey, data: String) -> bool {
//    
//}