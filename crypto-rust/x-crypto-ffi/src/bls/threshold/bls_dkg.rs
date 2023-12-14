//#![crate_type = "dylib"]

extern crate libc;

use x_crypto::bls::threshold::bls_dkg;

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

//use std::collections::HashMap;
//use std::convert::TryInto;
use std::str::FromStr;

use base64::{
    Engine as _, 
    engine::general_purpose,
};

use num_bigint::BigUint;

//#[derive(Clone, Debug, Serialize, Deserialize)]
//pub struct BlsPrivateKey {
//    //pub x: Scalar,
//    #[serde(serialize_with = "serialize_x", deserialize_with = "deserialize_x")]
//    pub x: [u8; 32],
//}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct BlsPrivateKey {
    //pub x: Scalar,
    #[serde(with = "serde_bytes")]
    pub x: Vec<u8>,
}

//fn serialize_x<S>(data: &[u8; 32], serializer: S) -> Result<S::Ok, S::Error>
//where
//    S: Serializer,
//{
//    serializer.serialize_bytes(data)
//}

//fn deserialize_x<'de, D>(deserializer: D) -> Result<[u8; 32], D::Error>
//where
//    D: Deserializer<'de>,
//{
//    let bytes = Vec::deserialize(deserializer)?;
//    if bytes.len() != 32 {
//        return Err(serde::de::Error::custom("expected an array of 96 bytes"));
//    }
//    let mut array = [0u8; 32];
//    array.copy_from_slice(&bytes);
//    Ok(array)
//}

//#[derive(Clone, Debug, Serialize, Deserialize)]
//pub struct BlsPublicKey {
//    //pub p: G2Affine,
//    #[serde(serialize_with = "serialize_p", deserialize_with = "deserialize_p")]
//    pub p: [u8; 96],
//}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct BlsPublicKey {
    #[serde(with = "serde_bytes")]
    pub p: Vec<u8>,
}

//fn serialize_p<S>(data: &[u8; 96], serializer: S) -> Result<S::Ok, S::Error>
//where
//    S: Serializer,
//{
//    serializer.serialize_bytes(data)
//}

//fn deserialize_p<'de, D>(deserializer: D) -> Result<[u8; 96], D::Error>
//where
//    D: Deserializer<'de>,
//{
//    let bytes = Vec::deserialize(deserializer)?;
//    if bytes.len() != 96 {
//        return Err(serde::de::Error::custom("expected an array of 96 bytes"));
//    }
//    let mut array = [0u8; 96];
//    array.copy_from_slice(&bytes);
//    Ok(array)
//}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct BlsM {
    #[serde(with = "serde_bytes")]
    pub p: Vec<u8>,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct BlsAccount {
    //pub index: BigUint,
    pub index: String,
    pub public_key: BlsPublicKey,
    pub private_key: BlsPrivateKey,
}

#[no_mangle]
pub extern "C" fn create_new_bls_account() -> *const libc::c_char {
    let bls_account = bls_dkg::create_new_bls_account();
    
    let index = bls_account.index.to_string();
    
    let public_key = BlsPublicKey {
        //p: String::from_utf8(bls_account.public_key.p.to_compressed().to_vec()).unwrap(),
        //p: bls_account.public_key.p.to_compressed(),
        p: bls_account.public_key.p.to_compressed().to_vec(),
    };
    
    let private_key = BlsPrivateKey {
        //x: String::from_utf8(bls_account.private_key.x.to_bytes().to_vec()).unwrap(),
        //x: bls_account.private_key.x.to_bytes(),
        x: bls_account.private_key.x.to_bytes().to_vec(),
    };
    
    let bls_new_account = BlsAccount {
        index,
        public_key,
        private_key,
    };
    
    //let res_string = serde_json::to_string(&Ok::<bls_dkg::BlsAccount, String>(bls_account)).unwrap();
    let res_string = serde_json::to_string(&bls_new_account).unwrap();
    
    //println!("res_string for create_new_bls_account is: {:?}", res_string);
    
    let c_res = CString::new(res_string).unwrap().into_raw();

    c_res
}

#[no_mangle]
pub extern "C" fn sum_bls_public_key(str_public_keys: *const libc::c_char) -> *const libc::c_char {
    let cstr_public_keys = unsafe { CStr::from_ptr(str_public_keys) };
    let str_public_keys = cstr_public_keys.to_str().unwrap().to_string();
    
    // 反序列化
    // testing...
    //let test_str = r#"[{"p":"pqnJ7itFhywC/gbb09butIg3G2/1wOcVHCVo5AMur40CkwjfhDVq38bdqi6a0b2nF6LDIP4VsZs02SggXv8ATWBc9iQmLxMszgP609O6TSoZc4Fw5/mVF2F1uIQofvvf"},{"p":"pLg7mYisylvac0QNvn+10/iZzYS9NxI0abz1lr7jSBvE2RNpom36frO+ZII664dHFvsyCE0AOXIJ2eRBTvA3vbcgiW4l7NfihD+QzgUceIJPGoHK4osaU0iGwZvf+ha6"},{"p":"mPC9QYOzFhDwNAxdYeL1aYXu54bnmtAh7A5+IowthNeaGXrICcE5NVaVYQHexC7HGBuaiouCKyf8c39n91HcYTVyUc2xRgppaMJsf7mauJ1Q0wAy/D1hyzpoSpgsi5/q"}]"#;
    //let test_public_keys: Vec<BlsPublicKey> = serde_json::from_str(&test_str).unwrap();
    //println!("test_public_keys is: {:?}", test_public_keys);
    
    println!("str_public_keys is: {:?}", str_public_keys);
    let raw_public_keys: Vec<BlsPublicKey> = serde_json::from_str(&str_public_keys).unwrap();
    println!("raw_public_keys is: {:?}", raw_public_keys);
    
    let mut public_keys = Vec::new();
    
    for i in 0..raw_public_keys.len() {
        println!("raw_public_keys[{:?}] length is: {:?}", i, raw_public_keys[i].p.len());
        
        //let p_bytes = base64::decode(raw_public_keys[i].p).unwrap();
        let p_bytes = general_purpose::STANDARD.decode(&raw_public_keys[i].p).unwrap();
        println!("p_bytes[{:?}] length is: {:?}", i, p_bytes.len());
        
        let public_key = bls_dkg::BlsPublicKey {
            //p: G2Affine::from_compressed(&raw_public_keys[i].p).unwrap(),
            // 把一个`&Vec<u8>`转化为 `&[u8; 96]`
            //p: G2Affine::from_compressed(raw_public_keys[i].p.as_slice().try_into().unwrap()).unwrap(),
            p: G2Affine::from_compressed(p_bytes.as_slice().try_into().unwrap()).unwrap(),
        };
        
        public_keys.push(public_key);
    }
    
    let response_result = bls_dkg::sum_bls_public_key(&public_keys);
    
    println!("sum_bls_public_key result: {:#?}", response_result);

    let result = match response_result {
        Ok(resp) => {
            let response = BlsPublicKey {
                //p: resp.p.to_compressed(),
                p: resp.p.to_compressed().to_vec(),
            };
            
            serde_json::to_string(&Ok::<BlsPublicKey, String>(response)).unwrap()
        }
        Err(error) => {
            serde_json::to_string(&Err::<BlsPublicKey, String>(error.to_string())).unwrap()
        },
    };
    
    let c_res = CString::new(result).unwrap().into_raw();

    c_res
}

#[no_mangle]
pub extern "C" fn get_bls_k(str_public_key: *const libc::c_char, str_public_key_sum: *const libc::c_char) -> *const libc::c_char {
    let cstr_public_key = unsafe { CStr::from_ptr(str_public_key) };
    let str_public_key = cstr_public_key.to_str().unwrap().to_string();
    
    let cstr_public_key_sum = unsafe { CStr::from_ptr(str_public_key_sum) };
    let str_public_key_sum = cstr_public_key_sum.to_str().unwrap().to_string();
    
    // 反序列化
    // testing...
    //let test_str = r#"[{"p":"pqnJ7itFhywC/gbb09butIg3G2/1wOcVHCVo5AMur40CkwjfhDVq38bdqi6a0b2nF6LDIP4VsZs02SggXv8ATWBc9iQmLxMszgP609O6TSoZc4Fw5/mVF2F1uIQofvvf"},{"p":"pLg7mYisylvac0QNvn+10/iZzYS9NxI0abz1lr7jSBvE2RNpom36frO+ZII664dHFvsyCE0AOXIJ2eRBTvA3vbcgiW4l7NfihD+QzgUceIJPGoHK4osaU0iGwZvf+ha6"},{"p":"mPC9QYOzFhDwNAxdYeL1aYXu54bnmtAh7A5+IowthNeaGXrICcE5NVaVYQHexC7HGBuaiouCKyf8c39n91HcYTVyUc2xRgppaMJsf7mauJ1Q0wAy/D1hyzpoSpgsi5/q"}]"#;
    //let test_public_key: BlsPublicKey = serde_json::from_str(&test_str).unwrap();
    //println!("test_public_key is: {:?}", test_public_key);
    
    println!("str_public_key is: {:?}", str_public_key);
    let raw_public_key: BlsPublicKey = serde_json::from_str(&str_public_key).unwrap();
    println!("raw_public_key is: {:?}", raw_public_key);
    
    println!("str_public_key_sum is: {:?}", str_public_key_sum);
    let raw_public_key_sum: BlsPublicKey = serde_json::from_str(&str_public_key_sum).unwrap();
    println!("raw_public_key_sum is: {:?}", raw_public_key_sum);
    
    let p_bytes = general_purpose::STANDARD.decode(&raw_public_key.p).unwrap();
    let public_key = bls_dkg::BlsPublicKey {
        // 把一个`&Vec<u8>`转化为 `&[u8; 96]`
        p: G2Affine::from_compressed(p_bytes.as_slice().try_into().unwrap()).unwrap(),
    };
    
    let p_bytes_sum = general_purpose::STANDARD.decode(&raw_public_key_sum.p).unwrap();
    let public_key_sum = bls_dkg::BlsPublicKey {
        // 把一个`&Vec<u8>`转化为 `&[u8; 96]`
        p: G2Affine::from_compressed(p_bytes_sum.as_slice().try_into().unwrap()).unwrap(),
    };
    
    let k = bls_dkg::get_k(public_key, public_key_sum);
    
    println!("get_bls_k result: {:#?}", k);

    let res_string = serde_json::to_string(&k.to_bytes()).unwrap();
    
    //println!("res_string for get_bls_k is: {:?}", res_string);
    
    let c_res = CString::new(res_string).unwrap().into_raw();

    c_res
}

#[no_mangle]
pub extern "C" fn get_bls_public_key_part(str_public_key: *const libc::c_char, str_k: *const libc::c_char) -> *const libc::c_char {
    let cstr_public_key = unsafe { CStr::from_ptr(str_public_key) };
    let str_public_key = cstr_public_key.to_str().unwrap().to_string();
    
    let cstr_k = unsafe { CStr::from_ptr(str_k) };
    let str_k = cstr_k.to_str().unwrap().to_string();
    
    // 反序列化
    println!("str_public_key is: {:?}", str_public_key);
    let raw_public_key: BlsPublicKey = serde_json::from_str(&str_public_key).unwrap();
    println!("raw_public_key is: {:?}", raw_public_key);
    
    println!("str_k is: {:?}", str_k);
    let raw_k: [u8; 32] = serde_json::from_str(&str_k).unwrap();
    println!("raw_k is: {:?}", raw_k);
    
    let p_bytes = general_purpose::STANDARD.decode(&raw_public_key.p).unwrap();
    let public_key = bls_dkg::BlsPublicKey {
        // 把一个`&Vec<u8>`转化为 `&[u8; 96]`
        p: G2Affine::from_compressed(p_bytes.as_slice().try_into().unwrap()).unwrap(),
    };
    
    let k = Scalar::from_bytes(&raw_k).unwrap();
    
    let public_key_part = bls_dkg::get_public_key_part(public_key, k);
    
    println!("get_public_key_part result: {:#?}", public_key_part);
    
    let response = BlsPublicKey {
        p: public_key_part.p.to_compressed().to_vec(),
    };

    let res_string = serde_json::to_string(&response).unwrap();
    
    //println!("res_string for get_public_key_part is: {:?}", res_string);
    
    let c_res = CString::new(res_string).unwrap().into_raw();

    c_res
}

#[no_mangle]
pub extern "C" fn get_bls_m(str_k: *const libc::c_char, str_private_key: *const libc::c_char,
 str_index: *const libc::c_char, str_public_key: *const libc::c_char) -> *const libc::c_char {
    let cstr_k = unsafe { CStr::from_ptr(str_k) };
    let str_k = cstr_k.to_str().unwrap().to_string();
    
    let cstr_private_key = unsafe { CStr::from_ptr(str_private_key) };
    let str_private_key = cstr_private_key.to_str().unwrap().to_string();
    
    let cstr_index = unsafe { CStr::from_ptr(str_index) };
    let str_index = cstr_index.to_str().unwrap().to_string();
    
    let cstr_public_key = unsafe { CStr::from_ptr(str_public_key) };
    let str_public_key = cstr_public_key.to_str().unwrap().to_string();
    
    // 反序列化
    println!("str_k is: {:?}", str_k);
    let raw_k: [u8; 32] = serde_json::from_str(&str_k).unwrap();
    println!("raw_k is: {:?}", raw_k);
    
    println!("str_private_key is: {:?}", str_private_key);
    let raw_private_key: BlsPrivateKey = serde_json::from_str(&str_private_key).unwrap();
    println!("raw_private_key is: {:?}", raw_private_key);
    
    println!("str_public_key is: {:?}", str_public_key);
    let raw_public_key: BlsPublicKey = serde_json::from_str(&str_public_key).unwrap();
    println!("raw_public_key is: {:?}", raw_public_key);
    
    // k
    let k = Scalar::from_bytes(&raw_k).unwrap();
    
    // private_key
    let x_bytes = general_purpose::STANDARD.decode(&raw_private_key.x).unwrap();
    let private_key = bls_dkg::BlsPrivateKey {
        // 把一个`&Vec<u8>`转化为 `&[u8; 32]`
        x: Scalar::from_bytes(&x_bytes.as_slice().try_into().unwrap()).unwrap(),
    };
    
    // index
    let index = BigUint::from_str(&str_index).unwrap();
    
    // public_key
    let p_bytes = general_purpose::STANDARD.decode(&raw_public_key.p).unwrap();
    let public_key = bls_dkg::BlsPublicKey {
        // 把一个`&Vec<u8>`转化为 `&[u8; 96]`
        p: G2Affine::from_compressed(p_bytes.as_slice().try_into().unwrap()).unwrap(),
    };
    
    
    let bls_m = bls_dkg::get_m(k, private_key.x, index, public_key);
    
    println!("get_m result: {:#?}", bls_m);
    
    let response = BlsM {
        p: bls_m.p.to_compressed().to_vec(),
    };

    let res_string = serde_json::to_string(&response).unwrap();
    
    //println!("res_string for get_m is: {:?}", res_string);
    
    let c_res = CString::new(res_string).unwrap().into_raw();

    c_res
}

#[no_mangle]
pub extern "C" fn get_bls_mk(str_ms: *const libc::c_char) -> *const libc::c_char {
    let cstr_ms = unsafe { CStr::from_ptr(str_ms) };
    let str_ms = cstr_ms.to_str().unwrap().to_string();
    
    // 反序列化
    println!("str_m is: {:?}", str_ms);
    let raw_ms: Vec<BlsM> = serde_json::from_str(&str_ms).unwrap();
    println!("raw_ms is: {:?}", raw_ms);
    
    let mut bls_ms = Vec::new();
    
    for i in 0..raw_ms.len() {
        println!("raw_ms[{:?}] length is: {:?}", i, raw_ms[i].p.len());
        
        let p_bytes = general_purpose::STANDARD.decode(&raw_ms[i].p).unwrap();
        println!("p_bytes[{:?}] length is: {:?}", i, p_bytes.len());
        
        let bls_m = bls_dkg::BlsM {
            // 把一个`&Vec<u8>`转化为 `&[u8; 96]`
            p: G1Affine::from_compressed(p_bytes.as_slice().try_into().unwrap()).unwrap(),
        };
        
        bls_ms.push(bls_m);
    }
    
    let response_result = bls_dkg::get_mk(&bls_ms);
    
    println!("get_mk result: {:#?}", response_result);

    let result = match response_result {
        Ok(resp) => {
            let response = BlsM {
                //p: resp.p.to_compressed(),
                p: resp.p.to_compressed().to_vec(),
            };
            
            serde_json::to_string(&Ok::<BlsM, String>(response)).unwrap()
        }
        Err(error) => {
            serde_json::to_string(&Err::<BlsM, String>(error.to_string())).unwrap()
        },
    };
    
    let c_res = CString::new(result).unwrap().into_raw();

    c_res
}

#[no_mangle]
pub extern "C" fn verify_bls_mk(raw_public_key: *const libc::c_char,
 raw_index: *const libc::c_char, raw_mk: *const libc::c_char) -> bool {
    let cstr_public_key = unsafe { CStr::from_ptr(raw_public_key) };
    let str_public_key = cstr_public_key.to_str().unwrap().to_string();
    
    let cstr_index = unsafe { CStr::from_ptr(raw_index) };
    let str_index = cstr_index.to_str().unwrap().to_string();
    
    let cstr_mk = unsafe { CStr::from_ptr(raw_mk) };
    let str_mk = cstr_mk.to_str().unwrap().to_string();
    
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
    
    // index
    let index = BigUint::from_str(&str_index).unwrap();
    
    // mk
    println!("str_mk is: {:?}", str_mk);
    let de_mk: BlsM = serde_json::from_str(&str_mk).unwrap();
    println!("de_mk is: {:?}", de_mk);
    
    let p_bytes = general_purpose::STANDARD.decode(&de_mk.p).unwrap();
    let mk = bls_dkg::BlsM {
        // 把一个`&Vec<u8>`转化为 `&[u8; 96]`
        p: G1Affine::from_compressed(p_bytes.as_slice().try_into().unwrap()).unwrap(),
    };
    
    let response_result = bls_dkg::verify_mk(public_key, index, mk);
    
    println!("verify_mk result: {:#?}", response_result);

    response_result
}