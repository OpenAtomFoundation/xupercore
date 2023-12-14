use crate::utils;

use crate::bls::threshold::{
    bls_dkg,
    hash_to_curve
};

use bls12_381::{
    G1Affine,
    G1Projective,
    G2Affine,
    G2Prepared,
    G2Projective,
    multi_miller_loop,
};

use num_bigint::BigUint;

// 门限签名的碎片
#[derive(Clone, Debug)]
pub struct BlsSignaturePart {
    pub index: BigUint, // 编号id
    pub public_key: G2Affine,
    pub sig: G1Affine,
}

// verify e(G, S’) = e(P’, H(P, m))⋅e(P, H(P, i)+H(P, j)+...)
#[derive(Clone, Debug)]
pub struct BlsThresholdSignature {
    pub part_indexs: Vec<BigUint>, // i, j, ...
    pub part_public_key_sum: G2Affine, // P’
    pub sig: G1Affine, // S’
}

// BLS门限签名的DSG流程如下：
// BLS threshold signature uses a particular function, defined as:
// e(G, S') = e(P', H(P, m)) * e(P, H(P, i)+H(P, j)+H(P, ...)))
// if the indexs of the signers are i, j and ...
// then S'= S(i) + S(j) + S(...), P'= P(i) + P(j) + P(...)
// S(i) = pk(i) * H(P,m) + MK(i),
// note: P is calculated in the DKG process.
// P = sum(K(i)*P(i)) = K(1)*P(1) + K(2)*P(2) + ...,
// K(i) = Hash(P(i) || P) = Hash(P(i) || (P(1)+P(2)+...+P(n)))
//
// H is a hash function, for instance SHA256 or SM3.
// S' is the signature.
// m is the message to sign.
// pk is the private key, which can be considered as a secret big number.
//
// --------------------------------------------------------------------
//
// To verify the signature,
// 1. check the threshold requirement, which means whether the signers[i,j,...] are enough or not.
// 2. check that whether the result of
// e(P', H(P, m)) * e(P, H(P, i)+H(P, j)+H(P, ...))) is equal to e(G, S') or not.
// Which means that: e(P, H(m)) = e(G, S)
//
// G is the base point or the generator point.
// P is the public key = pk*G.
// e is a special elliptic curve pairing function which has this feature: e(x*P, Q) = e(P, x*Q).
//
// It is true because of the pairing function described above:
// for example, say the threshold requirement is signer set[i,j]
// e(G, S’) = e(G, Si+Sj)=e(G, pki*H(P, m) + MKi + pkj*H(P, m) + MKj)
// = e(G, pki*H(P, m)+pkj*H(P, m))*e(G, MKi+MKj)
// = e(pki*G+pkj*G, H(P, m))*e(P, H(P, i)+H(P, j))
// = e(P’, H(P, m))*e(P, H(P, i)+H(P, j))

// 各个节点计算出自己的BLS签名片段，也就是计算出pki×H(P, m)+MKi
pub fn sign(private_info: bls_dkg::PartnerPrivate, msg: &[u8]) -> BlsSignaturePart {
    // P || m
    let data = utils::bytes::bytes_combine(
        &private_info.threshold_public_key.p.to_uncompressed(), 
        msg
    );
    
    // H(P, m)
    let h_point_g1 = hash_to_curve::hash_to_g1_curve(&data);

    // pki×H(P, m)
    let sig_part1_point = G1Affine::from(h_point_g1 * private_info.x);
    
    // pki×H(P, m)+MKi
    let sig_part2_point = private_info.mki;
    
    let sig_point = G1Affine::from(G1Projective::from(sig_part1_point) + sig_part2_point);
    
    let signature = BlsSignaturePart {
        index: private_info.public_info.index,
        public_key: private_info.public_info.public_key.p,
        sig: sig_point,
    };

    signature
}

// 组合BLS签名片段，生成最终签名
pub fn combine_sign(bls_signature_parts: &[BlsSignaturePart]) -> BlsThresholdSignature {
    let mut part_indexs = Vec::new();
    
    part_indexs.push(bls_signature_parts[0].index.to_owned());
    
    let mut part_public_key_sum = G2Projective::from(bls_signature_parts[0].public_key);
    
    let mut sig = G1Projective::from(bls_signature_parts[0].sig);
    
    for i in 1..bls_signature_parts.len() {
        // 计算s1 + s2 + ... + sn
        part_indexs.push(bls_signature_parts[i].index.to_owned());
        
        // G2Projective
        part_public_key_sum += &bls_signature_parts[i].public_key;
        
        // G1Projective
        sig += bls_signature_parts[i].sig;
    }
    
    let signature = BlsThresholdSignature {
        part_indexs,
        part_public_key_sum: G2Affine::from(part_public_key_sum),
        sig: G1Affine::from(sig),
    };
    
    signature
}

// 验签算法如下：
// To verify the signature,
// 1. check the threshold requirement, which means whether the signers[i,j,...] are enough or not.
// 2. check that whether the result of
// e(P', H(P, m)) * e(P, H(P, i)+H(P, j)+H(P, ...))) is equal to e(G, S') or not.
//
// G is the base point or the generator point.
// P is the public key = pk*G.
// e is a special elliptic curve pairing function which has this feature: e(x*P, Q) = e(P, x*Q).
//
// It is true because of the pairing function described above:
// for example, say the threshold requirement is signer set[i,j]
// e(G, S’) = e(G, Si+Sj)=e(G, pk1*H(P, m) + MKi + pk3*H(P, m) + MKj)
// = e(G, pki*H(P, m)+pkj*H(P, m))*e(G, MKi+MKj)
// = e(pki*G+pkj*G, H(P, m))*e(P, H(P, i)+H(P, j))
// = e(P’, H(P, m))*e(P, H(P, i)+H(P, j))
pub fn verify_sign(public_key: bls_dkg::BlsPublicKey, t_sig: BlsThresholdSignature, msg: &[u8]) -> bool {
    // step 1: e(G, S) = e(S, G)
    //let left = pairing(&t_sig.sig, &G2Affine::generator());
    let left = multi_miller_loop(&[(&t_sig.sig, &G2Prepared::from(G2Affine::generator()))]).final_exponentiation();
    
    // step 2: e(P’, H(P, m))*e(P, H(P, i)+H(P, j)+...)

    // step 2.1: e(P’, H(P, m)) = e(H(P, m), P’)
    // step 2.1.1: H(P, m)
    let data = utils::bytes::bytes_combine(
        &public_key.p.to_uncompressed(), 
        msg
    );

    let h_point1_g1 = hash_to_curve::hash_to_g1_curve(&data);
    
    // 为multi_miller_loop做准备
    let mut terms = Vec::new();
    
    // step 2.1.2: e(P’, H(P, m)) = e(H(P, m), P’)
    //let right1 = pairing(&h_point1_g1, &t_sig.part_public_key_sum);
    let point1_g2 = G2Prepared::from(t_sig.part_public_key_sum);
    terms.push((&h_point1_g1, &point1_g2));
    
    // step 2.2: e(P, H(P, i)+H(P, j)+ ...) = e(H(P, i)+H(P, j)+ ..., P)
    // step 2.2.1: H(P, i)+H(P, j)+ ...
    if t_sig.part_indexs.is_empty() {
        return false
    }
    
    // P' || i
    let data_pi = utils::bytes::bytes_combine(&public_key.p.to_uncompressed(), &t_sig.part_indexs[0].to_bytes_be());
    
    // 计算H(P' || i)
    let mut pi_sum_point_g1 = G1Projective::from(hash_to_curve::hash_to_g1_curve(&data_pi));
    
    for i in 1..t_sig.part_indexs.len() {
        // 计算s1 + s2 + ... + sn
        let data_pi = utils::bytes::bytes_combine(&public_key.p.to_uncompressed(), &t_sig.part_indexs[i].to_bytes_be());
        
        // 计算H(P' || i)
        pi_sum_point_g1 += hash_to_curve::hash_to_g1_curve(&data_pi);
    }
    
    // step 2.2.2: e(P, H(P, i)+H(P, j)+ ...) = e(H(P, i)+H(P, j)+ ..., P)
    //let right2 = pairing(&G1Affine::from(pi_sum_point_g1), &public_key.p);
    let point2_g1 = G1Affine::from(pi_sum_point_g1);
    let point2_g2 = G2Prepared::from(public_key.p);
    terms.push((&point2_g1, &point2_g2));
    
    // step 2.3: rp = e(P’, H(P, m))*e(P, H(P, i)+H(P, j)+...)
    //let right = right1 * right2;
    let right = multi_miller_loop(&terms[..]).final_exponentiation();
    
    left == right
}
