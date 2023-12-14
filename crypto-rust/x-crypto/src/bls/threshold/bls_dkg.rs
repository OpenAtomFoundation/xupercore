use crate::my_error::{
    CryptoError,
    CryptoResult,
    ErrorKind,
};

use crate::utils;

use crate::bls::threshold::hash_to_curve;

use bls12_381::{
    G1Affine,
    G1Projective,
    G2Affine,
    G2Prepared,
    G2Projective,
    multi_miller_loop,
    Scalar
};

use crypto::{
    digest::Digest,
    sha2::Sha512,
};

use ff::Field;
use num_bigint::BigUint;
use rand::rngs::OsRng;
use std::collections::HashMap;

//use serde::{Serialize, Deserialize};

#[derive(Copy, Clone, Debug)]
pub struct BlsPrivateKey {
    pub x: Scalar,
}

#[derive(Copy, Clone, Debug)]
pub struct BlsPublicKey {
    pub p: G2Affine,
}

#[derive(Copy, Clone, Debug)]
pub struct BlsM {
    pub p: G1Affine,
}

#[derive(Clone, Debug)]
pub struct PartnerShares {
    pub partner_info: PartnerPublic,
    pub shares: HashMap<BigUint, BigUint>, // key: partner index，也就是x坐标, value: 实际数值，也就是y坐标
    pub mis: Vec<G1Affine>,
}

#[derive(Clone, Debug)]
pub struct PartnerPublic {
    pub index: BigUint, // 编号id
    pub public_key: BlsPublicKey,
}

#[derive(Clone, Debug)]
pub struct PartnerPrivate {
    pub public_info: PartnerPublic,
    pub threshold_public_key: BlsPublicKey,
    pub x: Scalar,
    pub mki: G1Affine,
}

#[derive(Clone, Debug)]
pub struct BlsAccount {
    pub index: BigUint,
    pub public_key: BlsPublicKey,
    pub private_key: BlsPrivateKey,
}

// BLS门限签名的DKG流程如下：
// 1. 各方分别生成自己的公私钥对，公钥P(i)，私钥X(i)，i是自己的独一无二的编号
// 2. 各方广播，交换彼此的公钥P(i)和编号i，从而计算出公钥P = sum(P(i)) = P(1)+P(2)+...+P(n)
// 3. 各方分别为自己和其它人生成对应的偏离系数K(i) = Hash(P(i) || P) = Hash(P(i) || (P(1)+P(2)+...+P(n)))
// 4. 各方分别为自己和其它人计算对应的公钥碎片P'(i) = K(i)*P(i)
// 5. 各方均有能力计算出最终公钥P' = sum(P'(i)) = sum(K(i)*P(i)) = K(1)*P(1) + K(2)*P(2) + ...
// 6. 各方分别用自己的K和X，为网络中的每一个i计算自己的签名碎片M(i) = K*X*H(P' || i), i=1,2,...,n
// 7. 各方广播，交换彼此的签名碎片群M，也就是每方的M(1),M(2),...,M(n)。例如，每方都将自己的M(1)发送给编号为1的节点
// 8. 各方分别收集其它方广播的与自己的编号i相关的的签名碎片M(i)，再将这些M(i)累加，得到MK(i)。例如，编号为1的节点聚合所有的M(1)，得到MK(1)
// 最终，每方都得到了最重要的两个参数，最终公共公钥P'和自己的MK(i)。至此，BLS的DKG过程结束。

// 1. 各方分别生成自己的公私钥对，公钥P(i)，私钥X(i)，i是自己的独一无二的编号
pub fn create_new_bls_account() -> BlsAccount {
    // 产生随机熵
//    let entropy_bytes = (0..512/8).map(|_| { rand::random::<u8>() }).collect();
    let entropy_bytes: &[u8] = &(0..512/8).map(|_| rand::random::<u8>()).collect::<Vec<u8>>();
    // 产生编号
    let index =  BigUint::from_bytes_be(entropy_bytes);

    // 1.1 生成随机公私钥
    let (private_key, public_key) = generate_key_pair();

    let account = BlsAccount{
        public_key,
        private_key,
        index,
    };

    account
}

// 1.1 Generate a BLS key pair
fn generate_key_pair() -> (BlsPrivateKey, BlsPublicKey) {
    let mut rng = OsRng {};
    
    let private_key  = Scalar::random(&mut rng);
    let public_key = G2Affine::from(G2Affine::generator() * private_key);
    
    let bls_private_key = BlsPrivateKey
    {
        x: private_key
    };
    
    let bls_public_key = BlsPublicKey
    {
        p: public_key
    };
    
    (bls_private_key, bls_public_key)
}

// 2. 各方广播，交换彼此的公钥P(i)和编号i，从而计算出公钥P = sum(P(i)) = P(1)+P(2)+...+P(n)
// 根据所有潜在节点发布的公钥P(i)，计算出公共公钥P（公钥P(i)之和）
// or
// 5. 各方均有能力计算出最终公钥P' = sum(P'(i)) = sum(K(i)*P(i)) = K(1)*P(1) + K(2)*P(2) + ...
pub fn sum_bls_public_key(public_keys: &[BlsPublicKey]) -> CryptoResult<BlsPublicKey>
{
    if public_keys.is_empty() {
        return Err(
            CryptoError {
                kind: ErrorKind::ErrEmptyArray,
                message: ErrorKind::ErrEmptyArray.to_string(),
            }
        );
    }
    
    // 将所有潜在节点的公钥相加
    let mut public_point_g2 = G2Projective::from(public_keys[0].p);

    for i in 1..public_keys.len() {
        //public_point.add_assign(&public_keys[i].p);
        public_point_g2 += &public_keys[i].p;
    }

    let public_key_sum = BlsPublicKey { 
        p: G2Affine::from(public_point_g2)
    };

    Ok(public_key_sum)
}

// 3. 各方分别为自己和其它人生成对应的偏离系数K(i) = Hash(P(i) || P) = Hash(P(i) || (P(1)+P(2)+...+P(n)))
pub fn get_k(public_key: BlsPublicKey, public_key_sum: BlsPublicKey) -> Scalar {
    // bytes combine P(i) || P
    let data = utils::bytes::bytes_combine(&public_key.p.to_uncompressed(), &public_key_sum.p.to_uncompressed());
    
    let mut sha512 = Sha512::new();
    sha512.input(&data);
    
    let mut hash_byte_sha512: [u8; 64] = [0; 64];
    sha512.result(&mut hash_byte_sha512);
    
    // Converts a 512-bit little endian integer into a Scalar by reducing by the modulus.
    let scalar_factor = Scalar::from_bytes_wide(&hash_byte_sha512);
    
    scalar_factor
}

// 4. 各方分别为自己和其它人计算对应的公钥碎片P'(i) = K(i)*P(i)
pub fn get_public_key_part(public_key: BlsPublicKey, k: Scalar) -> BlsPublicKey {
    let public_key = G2Affine::from(public_key.p * k);
    
    let bls_public_key = BlsPublicKey
    {
        p: public_key
    };
    
    bls_public_key
}

// 6. 各方分别用自己的K和X，为网络中的每一个i计算自己的签名碎片M(i) = K*X*H(P' || i), i=1,2,...,n
pub fn get_m(k: Scalar, x: Scalar, index: BigUint, public_key: BlsPublicKey) -> BlsM {
    // bytes combine P' || i
    let data = utils::bytes::bytes_combine(&public_key.p.to_uncompressed(), &index.to_bytes_be());
    
    // 计算H(P' || i)
    let h_point_g1 = hash_to_curve::hash_to_g1_curve(&data);
    
    // 计算M(i) = K*X*H(P' || i)
    let kx = k * x;
    
    let m = G1Affine::from(h_point_g1 * kx);
    
    let bls_m = BlsM
    {
        p: m
    };
    
    bls_m
}

// 7. 各方广播，交换彼此的签名碎片群M，也就是每方的M(1),M(2),...,M(n)。例如，每方都将自己的M(1)发送给编号为1的节点
// 8. 各方分别收集其它方广播的与自己的编号i相关的的签名碎片M(i)，再将这些M(i)累加，得到MK(i)。例如，编号为1的节点聚合所有的M(1)，得到MK(1)
pub fn get_mk(ms: &[BlsM]) -> CryptoResult<BlsM>
{
    if ms.is_empty() {
        return Err(
            CryptoError {
                kind: ErrorKind::ErrEmptyArray,
                message: ErrorKind::ErrEmptyArray.to_string(),
            }
        );
    }
    
    // 将所有潜在节点的公钥相加
    let mut mk = G1Projective::from(ms[0].p);

    for i in 1..ms.len() {
        mk += &ms[i].p;
    }

    let bls_m = BlsM {
        p: G1Affine::from(mk)
    };

    Ok(bls_m)
}

// e(G, MKi)=e(P, H(P,i))
// 通过判断等式是否成立，来判断mk计算是否正确
pub fn verify_mk(public_key: BlsPublicKey, index: BigUint, mk: BlsM) -> bool {
    // step 1: e(G, MKi) = e(MKi, G)
    let left = bls12_381::pairing(&mk.p, &G2Affine::generator());
    
    // step 2: e(P, H(P,i)) = e(H(P,i), P)
    
    // bytes combine P || i
    let data = utils::bytes::bytes_combine(&public_key.p.to_uncompressed(), &index.to_bytes_be());
    
    // 计算H(P || i)
    let h_point_g1 = hash_to_curve::hash_to_g1_curve(&data);
    
    let right = bls12_381::pairing(&h_point_g1, &public_key.p);
    
    // step 3: verify e(G, MKi)=e(P, H(P,i))
    left == right
}

// e(G, MKi)=e(P, H(P,i))
// 通过判断等式是否成立，来判断mk计算是否正确
pub fn verify_mk_by_multi_miller_loop(public_key: BlsPublicKey, index: BigUint, mk: BlsM) -> bool {
    // step 1: e(G, MKi) = e(MKi, G)
//    let left = bls12_381::pairing(&mk.p, &G2Affine::generator());
//    println!("left is by pairing: {:?}", left);
    let left = multi_miller_loop(&[(&mk.p, &G2Prepared::from(G2Affine::generator()))]).final_exponentiation();
    println!("left is by multi_miller_loop: {:?}", left);
    // step 2: e(P, H(P,i)) = e(H(P,i), P)
    
    // bytes combine P || i
    let data = utils::bytes::bytes_combine(&public_key.p.to_uncompressed(), &index.to_bytes_be());
    
    // 计算H(P || i)
    let h_point_g1 = hash_to_curve::hash_to_g1_curve(&data);
    
    let right = bls12_381::pairing(&h_point_g1, &public_key.p);
    
    // step 3: verify e(G, MKi)=e(P, H(P,i))
    left == right
}