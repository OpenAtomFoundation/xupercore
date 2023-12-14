use crate::tpl::params;

use std::collections::HashMap;
//use std::str;
use crypto::sha2::Sha256;
use crypto::digest::Digest;
use hex;

pub fn generate_tpl_signature(params_map: HashMap<String, String>) -> String {

    // 获取所有参数的key
    let mut params_key_vec: Vec<String> = params_map.to_owned().into_keys().collect();
    // 对参数key的数组进行排序
    params_key_vec.sort();
    
    // 按序遍历参数
//    let mut msg = "";
    let mut msg = String::new();
    for key in &params_key_vec {
//        msg = msg + key + "=" + params_map.get(key);
        msg.push_str(key);
        msg.push_str("=");
        msg.push_str(params_map.get(key).unwrap());
    }
    
    // 获取sk
    let tpl_params = params::get_tpl_params();
    
//    msg = msg + tpl_params.sk;
    msg.push_str(&tpl_params.sk);
    
    println!("msg for sha256 sign: {:#?}", msg);
    println!("msg bytes for sha256 sign: {:?}", msg.as_bytes());
    
    // 计算哈希结果
    let mut sha256 = Sha256::new();
    sha256.input(&msg.as_bytes());
        
    let mut hash_byte_sha256: [u8; 32] = [0; 32];
    sha256.result(&mut hash_byte_sha256);
    
    let mut sig = String::new();
    sig.push_str(&tpl_params.tpl);
    sig.push_str("_");
    
//    str::from_utf8(&hash_byte_sha256).unwrap().to_string()
//    format!("{:X}", &hash_byte_sha256)
    let sign = hex::encode(&hash_byte_sha256);
    
    println!("sha256 result: {:?}", sign);
        
    sig.push_str(&sign);
    
    sig
}