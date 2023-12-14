//#![crate_type = "dylib"]

extern crate libc;

use x_crypto::bls::threshold::{
    bls_dkg,
    bls_dsg,
};

use crate::bls::threshold::bls_dkg::BlsPublicKey;

use std::ffi::{CStr, CString};

use serde_json;
use serde_bytes;
use serde::{
    Serialize, 
    //Serializer, 
    Deserialize, 
    //Deserializer
};

use bls12_381::{
    G1Affine,
//    G1Projective,
    G2Affine,
//    G2Prepared,
//    G2Projective,
    Scalar
};

use std::collections::HashMap;
use std::str::FromStr;

use base64::{
    Engine as _, 
    engine::general_purpose,
};

use num_bigint::BigUint;

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct PartnerShares {
    pub partner_info: PartnerPublic,
    //pub shares: HashMap<BigUint, BigUint>, // key: partner index，也就是x坐标, value: 实际数值，也就是y坐标
    pub shares: HashMap<String, String>, // key: partner index，也就是x坐标, value: 实际数值，也就是y坐标
    //pub mis: Vec<G1Affine>,
    pub mis: Vec<String>,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct PartnerPublic {
    //pub index: BigUint, // 编号id
    pub index: String, // 编号id
    pub public_key: BlsPublicKey,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct PartnerPrivate {
    pub public_info: PartnerPublic,
    pub threshold_public_key: BlsPublicKey,
    //pub x: Scalar,
    #[serde(with = "serde_bytes")]
    pub x: Vec<u8>,
    //pub mki: G1Affine,
    #[serde(with = "serde_bytes")]
    pub mki: Vec<u8>,
}

// 门限签名的碎片
#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct BlsSignaturePart {
    //pub index: BigUint, // 编号id
    pub index: String,
    //pub public_key: G2Affine,
    #[serde(with = "serde_bytes")]
    pub public_key: Vec<u8>,
    //pub sig: G1Affine,
    #[serde(with = "serde_bytes")]
    pub sig: Vec<u8>,
}

// verify e(G, S’) = e(P’, H(P, m))⋅e(P, H(P, i)+H(P, j)+...)
#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct BlsThresholdSignature {
    //pub part_indexs: Vec<BigUint>, // i, j, ...
    pub part_indexs: Vec<String>, // i, j, ...
    //pub part_public_key_sum: G2Affine, // P’
    #[serde(with = "serde_bytes")]
    pub part_public_key_sum: Vec<u8>, // P’
    //pub sig: G1Affine, // S’
    #[serde(with = "serde_bytes")]
    pub sig: Vec<u8>, // S’
}

// 各个节点计算出自己的BLS签名片段，也就是计算出pki×H(P, m)+MKi
#[no_mangle]
pub extern "C" fn bls_sign(raw_private_info: *const libc::c_char, raw_msg: *const libc::c_char) -> *const libc::c_char {
    // ---
    let cstr_private_info = unsafe { CStr::from_ptr(raw_private_info) };
    let str_private_info = cstr_private_info.to_str().unwrap().to_string();
    
    println!("str_private_info is: {:?}", str_private_info);
    let de_private_info: PartnerPrivate = serde_json::from_str(&str_private_info).unwrap();
    println!("de_private_info is: {:?}", de_private_info);
    
    // public_key
    let p_bytes = general_purpose::STANDARD.decode(&de_private_info.public_info.public_key.p).unwrap();
    let public_key = bls_dkg::BlsPublicKey {
        // 把一个`&Vec<u8>`转化为 `&[u8; 96]`
        p: G2Affine::from_compressed(p_bytes.as_slice().try_into().unwrap()).unwrap(),
    };
    
    // index
    let index = BigUint::from_str(&de_private_info.public_info.index).unwrap();
    
    // threshold_public_key
    let p_bytes = general_purpose::STANDARD.decode(&de_private_info.threshold_public_key.p).unwrap();
    let threshold_public_key = bls_dkg::BlsPublicKey {
        // 把一个`&Vec<u8>`转化为 `&[u8; 96]`
        p: G2Affine::from_compressed(p_bytes.as_slice().try_into().unwrap()).unwrap(),
    };
    
    // scalar
    let x_bytes = general_purpose::STANDARD.decode(&de_private_info.x).unwrap();
    let x = Scalar::from_bytes(&x_bytes.as_slice().try_into().unwrap()).unwrap();
    
    // mki
    let mki_bytes = general_purpose::STANDARD.decode(&de_private_info.mki).unwrap();
    let mki = G1Affine::from_compressed(mki_bytes.as_slice().try_into().unwrap()).unwrap();
    
    // 组装调用参数
    let public_info = bls_dkg::PartnerPublic {
        index,
        public_key,
    };
    
    let private_info = bls_dkg::PartnerPrivate {
        public_info,
        threshold_public_key,
        x,
        mki,
    };
    
    // ---
    
    let cstr_msg = unsafe { CStr::from_ptr(raw_msg) };
    let str_msg = cstr_msg.to_str().unwrap().to_string();
    
    let msg = str_msg.as_bytes();
    
    let signature = bls_dsg::sign(private_info, msg);
    
    println!("bls sign result: {:#?}", signature);
    
    let response = BlsSignaturePart {
        index: signature.index.to_string(),
        public_key: signature.public_key.to_compressed().to_vec(),
        sig: signature.sig.to_compressed().to_vec(),
    };
    
    //let res_string = serde_json::to_string(&Ok::<bls_dkg::BlsAccount, String>(bls_account)).unwrap();
    let res_string = serde_json::to_string(&response).unwrap();
    
    //println!("res_string for create_new_bls_account is: {:?}", res_string);
    
    let c_res = CString::new(res_string).unwrap().into_raw();

    c_res
}

// 组合BLS签名片段，生成最终签名
#[no_mangle]
pub extern "C" fn bls_combine_sign(raw_bls_signature_parts: *const libc::c_char) -> *const libc::c_char {
    // ---
    let cstr_bls_signature_parts = unsafe { CStr::from_ptr(raw_bls_signature_parts) };
    let str_bls_signature_parts = cstr_bls_signature_parts.to_str().unwrap().to_string();
    
    println!("str_bls_signature_parts is: {:?}", str_bls_signature_parts);
    let de_bls_signature_parts: Vec<BlsSignaturePart> = serde_json::from_str(&str_bls_signature_parts).unwrap();
    println!("de_bls_signature_parts is: {:?}", de_bls_signature_parts);
    
    let mut bls_signature_parts = Vec::new();
    
    for i in 0..de_bls_signature_parts.len() {
        // index
        let index = BigUint::from_str(&de_bls_signature_parts[i].index).unwrap();
        
        // public_key
        let p_bytes = general_purpose::STANDARD.decode(&de_bls_signature_parts[i].public_key).unwrap();
        let public_key = G2Affine::from_compressed(p_bytes.as_slice().try_into().unwrap()).unwrap();
        
        // sig
        let p_bytes = general_purpose::STANDARD.decode(&de_bls_signature_parts[i].sig).unwrap();
        let sig = G1Affine::from_compressed(p_bytes.as_slice().try_into().unwrap()).unwrap();
        
        let bls_signature_part = bls_dsg::BlsSignaturePart {
            index,
            public_key,
            sig,
        };
        
        bls_signature_parts.push(bls_signature_part);
    }
    
    let signature = bls_dsg::combine_sign(&bls_signature_parts);
    
    println!("bls combine_sign result: {:#?}", signature);
    
    let mut bls_signature_part_indexs = Vec::new();
    for i in 0..signature.part_indexs.len() {
        // index
        let index = signature.part_indexs[i].to_string();
        
        bls_signature_part_indexs.push(index);
    }
    
    let response = BlsThresholdSignature {
        part_indexs: bls_signature_part_indexs,
        part_public_key_sum: signature.part_public_key_sum.to_compressed().to_vec(),
        sig: signature.sig.to_compressed().to_vec(),
    };
    
    let res_string = serde_json::to_string(&response).unwrap();
    
    //println!("res_string for bls combine_sign is: {:?}", res_string);
    
    let c_res = CString::new(res_string).unwrap().into_raw();

    c_res
}

// 组合BLS签名片段，生成最终签名
#[no_mangle]
pub extern "C" fn bls_verify_sign(raw_public_key: *const libc::c_char,
 raw_t_sig: *const libc::c_char, raw_msg: *const libc::c_char) -> bool {
    let cstr_public_key = unsafe { CStr::from_ptr(raw_public_key) };
    let str_public_key = cstr_public_key.to_str().unwrap().to_string();
    
    let cstr_t_sig = unsafe { CStr::from_ptr(raw_t_sig) };
    let str_t_sig = cstr_t_sig.to_str().unwrap().to_string();
    
    let cstr_msg = unsafe { CStr::from_ptr(raw_msg) };
    let str_msg = cstr_msg.to_str().unwrap().to_string();
    
    // 反序列化
    
    // public_key
    println!("str_public_key is: {:?}", str_public_key);
    let de_public_key: BlsPublicKey = serde_json::from_str(&str_public_key).unwrap();
    println!("de_public_key is: {:?}", de_public_key);
    
    let p_bytes = general_purpose::STANDARD.decode(&de_public_key.p).unwrap();
    let public_key = bls_dkg::BlsPublicKey {
        // 把一个`&Vec<u8>`转化为 `&[u8; 96]`
        p: G2Affine::from_compressed(p_bytes.as_slice().try_into().unwrap()).unwrap(),
    };
    
    // t_sig
    println!("str_t_sig is: {:?}", str_t_sig);
    let de_t_sig: BlsThresholdSignature = serde_json::from_str(&str_t_sig).unwrap();
    println!("de_t_sig is: {:?}", de_t_sig);
    
    // t_sig part_indexs
    let mut part_indexs = Vec::new();
    
    for i in 0..de_t_sig.part_indexs.len() {
        // index
        let index = BigUint::from_str(&de_t_sig.part_indexs[i]).unwrap();
        
        part_indexs.push(index);
    }
    
    // t_sig part_public_key_sum
    //let p_bytes = general_purpose::STANDARD.decode(&de_t_sig.part_public_key_sum).unwrap();
    let p_bytes = &de_t_sig.part_public_key_sum;
    let part_public_key_sum = G2Affine::from_compressed(p_bytes.as_slice().try_into().unwrap()).unwrap();
    
    // t_sig sig
    //let p_bytes = general_purpose::STANDARD.decode(&de_t_sig.sig).unwrap();
    let p_bytes = &de_t_sig.sig;
    let sig = G1Affine::from_compressed(p_bytes.as_slice().try_into().unwrap()).unwrap();
    
    let bls_threshold_signature = bls_dsg::BlsThresholdSignature {
        part_indexs,
        part_public_key_sum,
        sig,
    };
    
    // msg    
    let msg = str_msg.as_bytes();
    
    // 调用verify_sign
    let verify_result = bls_dsg::verify_sign(public_key, bls_threshold_signature, msg);
    
    println!("bls verify_sign result: {:#?}", verify_result);
    
    verify_result
}