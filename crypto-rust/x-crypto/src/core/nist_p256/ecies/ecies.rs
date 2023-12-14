//use crate::crypto_config;
//
//use crate::my_error::{ErrorKind,CryptoError,CryptoResult};
//use crate::utils;
//use crate::ecies::params;
//use crate::aes::gcm_aes;
//
//
////use ecdsa::elliptic_curve;
////use ecdsa::elliptic_curve::sec1::{ToEncodedPoint, FromEncodedPoint};
//use p256::elliptic_curve::sec1::{ToEncodedPoint, FromEncodedPoint};
//
//use p256;
//use p256::PublicKey;
//use p256::SecretKey;
//use p256::ecdh::SharedSecret;
//use p256::ecdh::EphemeralSecret;
////use rand_core::OsRng;
//use rand::rngs::OsRng;
//
////use std::option::Option;
//
////use crypto::sha2::Sha256;
//use sha2::Sha256 as EcdhSha256;
//
//use crypto::sha2::Sha256;
//use crypto::digest::Digest;
//
//use generic_array;
//use sec1;
//
//use subtle::ConstantTimeEq;
//
////use elliptic_curve;
////use elliptic_curve::pkcs8::DecodePublicKey;
//
//use p256::elliptic_curve;
//use p256::elliptic_curve::pkcs8::DecodePublicKey;
//
//#[derive(Clone, Debug)]
//pub struct EccPoint {
//    pub x: Vec<u8>,
//    pub y: Vec<u8>,
//}
//
//
//// Encrypt encrypts a message using ECIES as specified in SEC 1, 5.1.
////
//// s1 and s2 contain shared information that is not part of the resulting
//// ciphertext. s1 is fed into key derivation, s2 is fed into the MAC. If the
//// shared information parameters aren't being used, they should be nil.
//// 
//// other_side_public_key_der_bytes: `Elliptic-Curve-Point-to-Octet-String` encoded key
//// described in SEC 1: Elliptic Curve Cryptography (Version 2.0) section 2.3.3 (page 10).
//pub fn encrypt(other_side_public_key_der_bytes: &[u8], msg: &[u8], s1: &[u8], s2: &[u8]) -> CryptoResult<Vec<u8>> {
//    // 生成一个随机椭圆曲线公私钥
//    // 生成NistP256的私钥
//    let ran_secret_key = EphemeralSecret::random(&mut OsRng);
//    
//    // 根据随机生成的私钥推导出公钥
//    let ran_public_key = PublicKey::from(&ran_secret_key);
//    
//    // Decode [`PublicKey`] (compressed or uncompressed) from the
//    // `Elliptic-Curve-Point-to-Octet-String` encoding described in
//    // SEC 1: Elliptic Curve Cryptography (Version 2.0) section
//    // 2.3.3 (page 10).
////    let other_side_public_key_result = PublicKey::from_sec1_bytes(other_side_public_key_der_bytes);
//    
//    let other_side_public_key_result = elliptic_curve::PublicKey::from_public_key_der(other_side_public_key_der_bytes);
//    
//    // 判断是否成功的恢复出了公钥
//    let other_side_public_key = match other_side_public_key_result {
//        Ok(key) => key,
//        Err(_) => return Err(
//            CryptoError {
//                kind: ErrorKind::EciesPublickeyRetrieveError,
//                message: ErrorKind::EciesPublickeyRetrieveError.to_string()
//            }
//        ),
//    };
//    
////    println!("ecies other_side_public_key get.....");
//
//    let cryptography = crypto_config::CryptoType::NistP256;
//    
//    let ecies_params = params::get_params_for_ecies(cryptography);
//    
////    let shared_secret_key_result = generate_p256_shared_secret_key(ran_secret_key, other_side_public_key, ecies_params.key_len, ecies_params.key_len);
//    let shared_secret_key_result = generate_raw_p256_shared_secret_key(ran_secret_key, other_side_public_key, ecies_params.key_len, ecies_params.key_len);
//
//    let shared_secret_key = match shared_secret_key_result {
//        Ok(secret) => secret,
//        Err(error) => return Err(error),
//    };
//
//    // Use HKDF (HMAC-based Extract-and-Expand Key Derivation Function)
//    // to extract entropy from this shared secret.
//    // This method can be used to transform the shared secret into
//    // uniformly random values which are suitable as key material.
////    let salt = String::from("jingbo is handsome.").as_bytes();
//    let salt = s1;
//    let hkdf = shared_secret_key.extract::<EcdhSha256>(Some(salt));
//    
////    let mut okm = Vec::with_capacity(ecies_params.key_len * 2);
//    let mut okm = vec![0u8; ecies_params.key_len * 2];
//    
////    let hkdf_result = hkdf.expand(&msg, &mut okm);
//    let info = Vec::with_capacity(1);
//    let hkdf_result = hkdf.expand(&info, &mut okm);
//    
////    println!("hkdf_result: {:?}", hkdf_result);
//     
//    if !hkdf_result.is_ok() {
//        return Err(
//            CryptoError {
//                kind: ErrorKind::EciesHkdfInvalidKeyLength,
//                message: ErrorKind::EciesHkdfInvalidKeyLength.to_string()
//            }
//        );
//    }
//    
//    // 启动AES对称加密流程
//    // 从hkdf的输出结果中，截取前一半作为aes对称加密密钥
//    // Extracts a slice containing the entire vector
//    // and then returns the subslice corresponding to the specified range
////    println!("okm: {:?}", okm);
//    let aes_key = &okm[..].get(0..ecies_params.key_len).unwrap();
//    
////    println!("aes_encrypting...");
//    let aes_result = gcm_aes::aes_encrypt(aes_key, &msg, ecies_params);
////    println!("aes_result: {:?}", aes_result);
//    
//    // 判断是否成功的完成了AES加密
//    let mut aes_enc_res = match aes_result {
//        Ok(res) => res,
//        Err(_) => return Err(
//            CryptoError {
//                kind: ErrorKind::EciesEncryptError,
//                message: ErrorKind::EciesEncryptError.to_string()
//            }
//        ),
//    };
//    
//    println!("ecies encrypt real cipher_bytes: {:?}", aes_enc_res);
//    
//    // 从hkdf的输出结果中，截取后一半作为Message Tag计算的一部分
//    let msg_key = &okm[..].get(ecies_params.key_len..).unwrap();
//    
////    let mut msg_tag = generate_msg_tag(msg_key, msg, shared_secret_key.raw_secret_bytes(), s2);
//    let mut msg_tag = generate_msg_tag(msg_key, &aes_enc_res, shared_secret_key.raw_secret_bytes(), s2);
//    println!("ecies encrypt msg_tag: {:?}", msg_tag);
//    
//    let ran_encoded_point = ran_public_key.to_encoded_point(false);
//    
//    let x_coordinates = ran_encoded_point.x().unwrap();
//    let y_coordinates = ran_encoded_point.y().unwrap();
//    
//    println!("x_coordinates: {:?}", x_coordinates);
//    println!("y_coordinates: {:?}", y_coordinates);
//    
//    let mut ran_public_key_bytes = marshal_nist_p256_point(x_coordinates, y_coordinates);
//    
//    // 组装密文
//    let mut cipher_bytes = Vec::with_capacity(ran_public_key_bytes.len() + aes_enc_res.len() + msg_tag.len());
//    
//    cipher_bytes.append(&mut ran_public_key_bytes);
//    cipher_bytes.append(&mut aes_enc_res);
//    cipher_bytes.append(&mut msg_tag);
//    
//    println!("ecies cipher_bytes: {:?}", cipher_bytes);
//
//    Ok(cipher_bytes)
//}
//
//// Decrypt decrypts an ECIES ciphertext.
////
//// s1 and s2 contain shared information that is not part of the resulting
//// ciphertext. s1 is fed into key derivation, s2 is fed into the MAC. If the
//// shared information parameters aren't being used, they should be nil.
//pub fn decrypt(secret_key_der_bytes: &[u8], cipher_text: &[u8], s1: &[u8], s2: &[u8]) -> CryptoResult<Vec<u8>> {
//    if cipher_text.len() == 0 {
//        return Err(
//            CryptoError {
//                kind: ErrorKind::EciesDecryptError,
//                message: ErrorKind::EciesDecryptError.to_string()
//            }
//        );
//    }
//    
//    //现在基于NistP256实现
//    // 恢复公钥    
//    
//    //    let secret_key = SecretKey::from_sec1_pem(&secret_key_bytes);
//    // Deserialize secret key encoded in the SEC1 ASN.1 DER ECPrivateKey format.
//    let secret_key_result = SecretKey::from_sec1_der(secret_key_der_bytes);
//    
//    // 判断是否成功的恢复出了私钥
//    let secret_key = match secret_key_result {
//        Ok(key) => key,
//        Err(_) => return Err(
//            CryptoError {
//                kind: ErrorKind::EllipticCurveNotSupportedYet,
//                message: ErrorKind::EllipticCurveNotSupportedYet.to_string()
//            }
//        ),
//    };
//    
//    println!("ecies decrypt SecretKey get.....");
//    
//    let cryptography = crypto_config::CryptoType::NistP256;
//    
//    let ecies_params = params::get_params_for_ecies(cryptography);
//    
//    let chk_res = check_ecies_cipher_text_by_nist_p256_decrypt_key(cipher_text, ecies_params.key_len);
//    
//    if chk_res == false {
//        println!("ecies check_ecies_cipher_text_by_nist_p256_decrypt_key result is false");
//        return Err(
//            CryptoError {
//                kind: ErrorKind::EllipticCurveNotSupportedYet,
//                message: ErrorKind::EllipticCurveNotSupportedYet.to_string()
//            }
//        )
//    }
//    
//    let msg_start = (256 + 7) / 4;
//    
//    let ecc_point_result = unmarshal_nist_p256_point(&cipher_text[..msg_start]);
//    let ecc_point = match ecc_point_result {
//        Ok(point) => point,
//        Err(error) => return Err(error),
//    };
//    
//    let x_array = generic_array::GenericArray::from_slice(&ecc_point.x);
//    let y_array = generic_array::GenericArray::from_slice(&ecc_point.y);
//    
//    println!("x_coordinates: {:?}", x_array);
//    println!("y_coordinates: {:?}", y_array);
//    
//    // Available on crate feature sec1 only.
//    let encoded_point = sec1::EncodedPoint::from_affine_coordinates(x_array, y_array, false);
//    
//    let other_side_public_key_result = PublicKey::from_encoded_point(&encoded_point);
//    if other_side_public_key_result.is_some().unwrap_u8() == 0{
//        return Err(
//            CryptoError {
//                kind: ErrorKind::EciesDecryptError,
//                message: ErrorKind::EciesDecryptError.to_string()
//            }
//        )
//    }
//    
//    let other_side_public_key = other_side_public_key_result.unwrap();
//    
//    let shared_secret_key_result = generate_raw_p256_shared_secret_key_by_ecc(secret_key, other_side_public_key, ecies_params.key_len, ecies_params.key_len);
//
//    let shared_secret_key = match shared_secret_key_result {
//        Ok(secret) => secret,
//        Err(error) => return Err(error),
//    };
//    
//    // Use HKDF (HMAC-based Extract-and-Expand Key Derivation Function)
//    // to extract entropy from this shared secret.
//    // This method can be used to transform the shared secret into
//    // uniformly random values which are suitable as key material.
////    let salt = String::from("jingbo is handsome.").as_bytes();
//    let salt = s1;
//    let hkdf = shared_secret_key.extract::<EcdhSha256>(Some(salt));
//    
////    let mut okm = Vec::with_capacity(ecies_params.key_len * 2);
//    let mut okm = vec![0u8; ecies_params.key_len * 2];
//    
////    let hkdf_result = hkdf.expand(&msg, &mut okm);
//    
////    let hkdf_result = hkdf.expand(&cipher_text, &mut okm);
//    let info = Vec::with_capacity(1);
//    let hkdf_result = hkdf.expand(&info, &mut okm);
//     
//    if !hkdf_result.is_ok() {
//        return Err(
//            CryptoError {
//                kind: ErrorKind::EciesHkdfInvalidKeyLength,
//                message: ErrorKind::EciesHkdfInvalidKeyLength.to_string()
//            }
//        );
//    }
//    
//    // 启动AES对称解密流程
//    // 从hkdf的输出结果中，截取前一半作为aes对称解密密钥
//    // Extracts a slice containing the entire vector
//    // and then returns the subslice corresponding to the specified range
//    let aes_key = &okm[..].get(0..ecies_params.key_len).unwrap();
//    
//    // 前面选择的是EcdhSha256哈希算法，所以hash结果的size是256/8=32
//    let msg_end = cipher_text.len() - 32;
//    
//    let real_cipher = cipher_text[msg_start..msg_end].to_vec();
//    
//    // 从hkdf的输出结果中，截取后一半作为Message Tag计算的一部分
//    let msg_key = &okm[..].get(ecies_params.key_len..).unwrap();
//    
//    println!("ecies real_cipher: {:?}", real_cipher);
//    
//    let msg_tag = generate_msg_tag(msg_key, &real_cipher, shared_secret_key.raw_secret_bytes(), s2);
//    let msg_tag_retrieved = cipher_text[msg_end..].to_vec();
//    
//    // 基于ConstantTimeEq抵御侧信道计时攻击
//    if constant_time_compare(&msg_tag, &msg_tag_retrieved) == false {
//        println!("ecies decrypt msg_tag: {:?}", msg_tag);
//        println!("ecies decrypt msg_tag_retrieved: {:?}", msg_tag_retrieved);
//        return Err(
//            CryptoError {
//                kind: ErrorKind::EciesDecryptError,
//                message: ErrorKind::EciesDecryptError.to_string()
//            }
//        )
//    }
//    
//    let aes_result = gcm_aes::aes_decrypt(aes_key, &real_cipher, ecies_params);
//    
//    // 判断是否成功的完成了AES加密
//    let aes_dec_res = match aes_result {
//        Ok(res) => res,
//        Err(_) => return Err(
//            CryptoError {
//                kind: ErrorKind::EciesDecryptError,
//                message: ErrorKind::EciesDecryptError.to_string()
//            }
//        ),
//    };
//    
//    Ok(aes_dec_res)
//}
//
//fn constant_time_compare(x: &[u8], y: &[u8]) -> bool {
//    if x.len() != y.len() {
//        return false
//    }
//
//    let mut v = 0u8;
//    
//    let mut index = 0;
//    while index < x.len() {
//        // ^ 是位运算符<异或>，返回值是0或1，相同位不相同则返回1，否则返回0
//        // | 是逻辑运算符<或>
//        //
//        // 假设A(u8类型)的二进制格式为0 0 0 0 0 0 1 0
//        // 假设B(u8类型)的二进制格式为0 0 0 0 0 0 0 1
//        // 则 A^B = 1
//        v |= x[index] ^ y[index];
//        
//        index += 1;
//    }
//
//    // 如果最终结果为0，说明两个u8数组完全相等
//    if v.ct_eq(&0u8).unwrap_u8() == 1 {
//        return true
//    } else {
//        return false
//    }
//}
//
//// messageTag computes the MAC of a message (called the tag) as per SEC 1, 3.5.
//fn generate_msg_tag(msg_key: &[u8], msg: &[u8], shared_secret_key: &[u8], shared_secret: &[u8]) -> Vec<u8> {
//    let mut sha256 = Sha256::new();
//    
//    // SHA256 once
//    sha256.input(&msg_key);
//    sha256.input(&msg);
//    sha256.input(&shared_secret_key);
//    sha256.input(&shared_secret);
//    
//    let mut hash_byte_sha256: [u8; 32] = [0; 32];
//    sha256.result(&mut hash_byte_sha256);
//    
//    sha256.reset();
//    
//    hash_byte_sha256.to_vec()
//}
//
//// ECDH key agreement method, which is used for establishing secret keys for future aes encryption.
//fn generate_raw_p256_shared_secret_key_by_ecc(one_side_secret_key: SecretKey, other_side_public_key: PublicKey, sk_len: usize, mac_len: usize) -> CryptoResult<SharedSecret> {
//    // 获取p256曲线支持的最长key长度
//    let p256_max_key_length = get_nist_p256_max_shared_key_length();
//    
//    if sk_len + mac_len > p256_max_key_length {
//        return Err(
//            CryptoError {
//                kind: ErrorKind::EciesSharedKeyTooBig,
//                message: ErrorKind::EciesSharedKeyTooBig.to_string()
//            }
//        );
//    }
//    
//    let shared_secret_key = elliptic_curve::ecdh::diffie_hellman(one_side_secret_key.to_nonzero_scalar(), other_side_public_key.as_affine());
//    
//    Ok(shared_secret_key)
//}
//
//// ECDH key agreement method, which is used for establishing secret keys for future aes encryption.
//fn generate_raw_p256_shared_secret_key(one_side_secret_key: EphemeralSecret, other_side_public_key: PublicKey, sk_len: usize, mac_len: usize) -> CryptoResult<SharedSecret> {
//    // 获取p256曲线支持的最长key长度
//    let p256_max_key_length = get_nist_p256_max_shared_key_length();
//    
//    if sk_len + mac_len > p256_max_key_length {
//        return Err(
//            CryptoError {
//                kind: ErrorKind::EciesSharedKeyTooBig,
//                message: ErrorKind::EciesSharedKeyTooBig.to_string()
//            }
//        );
//    }
//    
//    let shared_secret_key = one_side_secret_key.diffie_hellman(&other_side_public_key);
//    
//    Ok(shared_secret_key)
//}
//
//#[allow(dead_code)]
//// ECDH key agreement method, which is used for establishing secret keys for future aes encryption.
//fn generate_p256_shared_secret_key(one_side_secret_key: EphemeralSecret, other_side_public_key: PublicKey, sk_len: usize, mac_len: usize) -> CryptoResult<Vec<u8>> {
//    // 获取p256曲线支持的最长key长度
//    let p256_max_key_length = get_nist_p256_max_shared_key_length();
//    
////    let mut key_len_valid = true;
//    if sk_len + mac_len > p256_max_key_length {
////        key_len_valid = false;
//        return Err(
//            CryptoError {
//                kind: ErrorKind::EciesSharedKeyTooBig,
//                message: ErrorKind::EciesSharedKeyTooBig.to_string()
//            }
//        );
//    }
//    
//    let shared_secret_key = one_side_secret_key.diffie_hellman(&other_side_public_key);
//    
////    sk = make([]byte, skLen+macLen)
////    skBytes := x.Bytes()
////    copy(sk[len(sk)-len(skBytes):], skBytes)
//
//    let sk_bytes = shared_secret_key.raw_secret_bytes();
//    
////    let sk_mac_sum_len = sk_len + mac_len;
//    
//    let bytes_shared_secret_key = utils::bytes::bytes_pad(sk_bytes.to_vec(), sk_len + mac_len);
//    
//    Ok(bytes_shared_secret_key)
//}
//
//// See FIPS 186-3, section D.2.3
//// p256.CurveParams = &CurveParams{Name: "P-256"}
//// p256.P, _ = new(big.Int).SetString("115792089210356248762697446949407573530086143415290314195533631308867097853951", 10)
//// p256.N, _ = new(big.Int).SetString("115792089210356248762697446949407573529996955224135760342422259061068512044369", 10)
//// p256.B, _ = new(big.Int).SetString("5ac635d8aa3a93e7b3ebbd55769886bc651d06b0cc53b0f63bce3c3e27d2604b", 16)
//// p256.Gx, _ = new(big.Int).SetString("6b17d1f2e12c4247f8bce6e563a440f277037d812deb33a0f4a13945d898c296", 16)
//// p256.Gy, _ = new(big.Int).SetString("4fe342e2fe1a7f9b8ee7eb4a7c0f9e162bce33576b315ececbb6406837bf51f5", 16)
//// p256.BitSize = 256
//fn get_nist_p256_max_shared_key_length() -> usize {
//    // 根据bitsize来计算max_key_length
//    let max_key_length = (256 + 7) / 8;
//    
//    max_key_length
//}
//
//// See FIPS 186-3, section D.2.3
//// p256.CurveParams = &CurveParams{Name: "P-256"}
//// p256.P, _ = new(big.Int).SetString("115792089210356248762697446949407573530086143415290314195533631308867097853951", 10)
//// p256.N, _ = new(big.Int).SetString("115792089210356248762697446949407573529996955224135760342422259061068512044369", 10)
//// p256.B, _ = new(big.Int).SetString("5ac635d8aa3a93e7b3ebbd55769886bc651d06b0cc53b0f63bce3c3e27d2604b", 16)
//// p256.Gx, _ = new(big.Int).SetString("6b17d1f2e12c4247f8bce6e563a440f277037d812deb33a0f4a13945d898c296", 16)
//// p256.Gy, _ = new(big.Int).SetString("4fe342e2fe1a7f9b8ee7eb4a7c0f9e162bce33576b315ececbb6406837bf51f5", 16)
//// p256.BitSize = 256
//fn check_ecies_cipher_text_by_nist_p256_decrypt_key(cipher_text: &[u8], mac_len: usize) -> bool {
//    if cipher_text[0] != 2u8 && cipher_text[0] != 3u8 && cipher_text[0] != 4u8 {
////        println!("cipher_text[0]: {:?}", cipher_text[0]);
//        return false;
//    }
//    
//    // 根据bitsize来计算r_length
//    let r_length = (256 + 7) / 4;
//    
////    println!("cipher_text.len(): {:?}", cipher_text.len());
////    println!("r_length + mac_len + 1: {:?}", r_length + mac_len + 1);
//    
//    if cipher_text.len() < (r_length + mac_len + 1) {
//        return false
//    }
//    
//    true
//}
//
//// Marshal converts a point on the curve into the uncompressed form specified in
//// section 4.3.6 of ANSI X9.62.
//fn marshal_nist_p256_point(x: &[u8], y: &[u8]) -> Vec<u8> {
//    // 根据bitsize来计算byte_length
//    let byte_length = (256 + 7) / 8;
//    
////    let mut ret_bytes = Vec::with_capacity(1 + byte_length*2);
//    let mut ret_bytes = vec![0u8; 1];
//    ret_bytes[0] = 4u8; // means using uncompressed form
//    
////    let mut ret_bytes_left = Vec::with_capacity(byte_length);
//    let mut ret_bytes_left = vec![0u8; byte_length];
//    ret_bytes_left.copy_from_slice(&utils::bytes::bytes_pad(x.to_vec(), byte_length));
//    
////    let mut ret_bytes_right = Vec::with_capacity(byte_length);
//    let mut ret_bytes_right = vec![0u8; byte_length];
//    ret_bytes_right.copy_from_slice(&utils::bytes::bytes_pad(y.to_vec(), byte_length));
//    
//    ret_bytes.append(&mut ret_bytes_left);
//    ret_bytes.append(&mut ret_bytes_right);
//    
//    ret_bytes
//}
//
//// Unmarshal converts a point, serialized by Marshal, into an x, y pair.
//// It is an error if the point is not in uncompressed form or is not on the curve.
//fn unmarshal_nist_p256_point(data: &[u8]) -> CryptoResult<EccPoint> {
//    // 根据bitsize来计算byte_length
//    let byte_length = (256 + 7) / 8;
//    
//    if data.len() != 1 + 2*byte_length {
//        return Err(
//            CryptoError {
//                kind: ErrorKind::EciesDecryptError,
//                message: ErrorKind::EciesDecryptError.to_string()
//            }
//        );
//    }
//    
//    // uncompressed form
//    if data[0] != 4 {
//        return Err(
//            CryptoError {
//                kind: ErrorKind::EciesDecryptError,
//                message: ErrorKind::EciesDecryptError.to_string()
//            }
//        );
//    }
//    
////    let mut x = Vec::with_capacity(byte_length);
//    let mut x = vec![0u8; byte_length];
//    x.copy_from_slice(&data[1..1+byte_length]);
//    
////    let mut y = Vec::with_capacity(byte_length);
//    let mut y = vec![0u8; byte_length];
//    y.copy_from_slice(&data[1+byte_length..]);
//    
//    let ecc_point = EccPoint{
//        x,
//        y,
//    };
//    
//    // TODO: isOnCurve的判断后续补上
//    Ok(ecc_point)
//}