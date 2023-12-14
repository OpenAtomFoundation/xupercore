//use x_crypto::account;
//use x_crypto::wordlist::LanguageType;
//use x_crypto::crypto_config;
////use x_crypto::ecies::ecies;
//use x_crypto::core::nist_p256::ecies::ecies;
//
//use p256::pkcs8::EncodePublicKey;
//
//#[test]
//fn ecies() {
//    // 创建新账户
//    let res = account::create_new_account_with_mnemonic(LanguageType::English, account::MnemonicStrength::StrengthEasy, crypto_config::CryptoType::NistP256);
//
//    let account = res.unwrap();
//    
//    // 获取使用PEM-encoded SEC1格式编码的私钥字符串
//    let secret_key_str = account.private_key;
//
//    // 获取使用PEM-encoded SEC1格式编码的公钥字符串
//    let public_key_str = account.public_key;
//    
//    // 获取加密的密钥
//    let public_key = account::get_ecdsa_public_key_from_pem_encoded_sec1_str(public_key_str).unwrap();
//    
//    // 获取SPKI-encoded public key
//    let public_key_der = public_key.to_public_key_der().unwrap();
//    let public_key_der_bytes = public_key_der.as_bytes();
////    println!("public_key_der_bytes: {:?}", public_key_der_bytes);
//    
//    let msg = String::from("this is msg");
//    let msg_bytes = msg.as_bytes();
//    let s1 = String::from("this is s1");
//    let s1_bytes = s1.as_bytes();
//    let s2 = String::from("this is s2");
//    let s2_bytes = s2.as_bytes();
//    
//    //let enc_result = ecies::encrypt(public_key_der_bytes, crypto_config::CryptoType::NistP256, msg_bytes, s1_bytes, s2_bytes);
//    let enc_result = ecies::encrypt(public_key_der_bytes, msg_bytes, s1_bytes, s2_bytes);
//    
//    println!("ecies enc_result: {:?}", enc_result);
//    
//    let res = match enc_result {
//        Ok(_) => true,
//        Err(_) => false,
//    };
//    assert_eq!(res, true);
//    
//    // 获取解密的密钥
//    let secret_key = account::get_ecdsa_private_key_from_pem_encoded_sec1_str(secret_key_str).unwrap();
//    
//    // Serialize secret key in the SEC1 ASN.1 DER ECPrivateKey format.
//    let secret_key_der = secret_key.to_sec1_der().unwrap();
//    let secret_key_der_bytes = secret_key_der.as_ref();
//    
//    //let dec_result = ecies::decrypt(secret_key_der_bytes, crypto_config::CryptoType::NistP256, &enc_result.unwrap(), s1_bytes, s2_bytes);
//    let dec_result = ecies::decrypt(secret_key_der_bytes, &enc_result.unwrap(), s1_bytes, s2_bytes);
//    
//    println!("ecies dec_result: {:?}", dec_result);
//    
//    let res = match dec_result {
//        Ok(dec_msg) => dec_msg,
//        Err(_) => vec![0u8; 1],
//    };
//    
//    println!("ecies dec_msg: {:?}", String::from_utf8(res.clone()).unwrap());
//    assert_eq!(res, msg_bytes);
//}